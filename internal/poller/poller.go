// internal/poller/poller.go
package poller

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// Client abstracts Modbus operations needed by the poller.
// Geometry only: no semantics.
type Client interface {
	ReadCoils(addr, qty uint16) ([]bool, error)              // FC 1
	ReadDiscreteInputs(addr, qty uint16) ([]bool, error)     // FC 2
	ReadHoldingRegisters(addr, qty uint16) ([]uint16, error) // FC 3
	ReadInputRegisters(addr, qty uint16) ([]uint16, error)   // FC 4
}

// Config is the minimal runtime config the poller needs.
// Each ReadBlock carries its own Interval; there is no device-level interval.
type Config struct {
	UnitID string
	Reads  []ReadBlock
}

// Poller reads from a field device via a Client.
// It reuses the client while healthy and discards it when the connection is dead.
// No retries are performed inside a poll cycle.
// A future tick may create a new client via factory.
type Poller struct {
	cfg Config

	schedules []readSchedule

	client  Client
	factory func() (Client, error)

	// Transport lifetime instrumentation (passive only)
	counters TransportCounters
}

// New creates a poller with immutable config.
// - client is the initial connected client (may be nil)
// - factory creates a new connected client when the current one is missing/dead
func New(cfg Config, client Client, factory func() (Client, error)) (*Poller, error) {
	if cfg.UnitID == "" {
		return nil, errors.New("poller: unit id required")
	}
	if len(cfg.Reads) == 0 {
		return nil, errors.New("poller: at least one read block required")
	}
	for i, rb := range cfg.Reads {
		if rb.Interval <= 0 {
			return nil, fmt.Errorf("poller: reads[%d].interval must be > 0", i)
		}
	}

	now := time.Now()
	schedules := make([]readSchedule, len(cfg.Reads))
	for i, rb := range cfg.Reads {
		schedules[i] = readSchedule{cfg: rb, nextExec: now}
	}

	return &Poller{
		cfg:       cfg,
		schedules: schedules,
		client:    client,
		factory:   factory,
	}, nil
}

// executeSingleRead performs exactly one Modbus read for the given block.
// It manages client lifecycle and updates transport counters.
//
// PollResult semantics: each call produces a result for one read block.
// Connection policy:
// - reuse existing client while healthy
// - if client is nil, try to create once via factory
// - on a "dead connection" error, discard client (so next tick can recreate)
func (p *Poller) executeSingleRead(rb ReadBlock) PollResult {

	// Increment request attempt (one per read)
	p.counters.RequestsTotal++

	res := PollResult{
		UnitID: p.cfg.UnitID,
		At:     time.Now(),
	}

	// Ensure we have a client for this attempt.
	if p.client == nil {
		if p.factory == nil {
			res.Err = errors.New("poller: client is nil and no factory provided")
			p.recordFailure(res.Err)
			return res
		}
		c, err := p.factory()
		if err != nil {
			res.Err = err
			p.recordFailure(err)
			return res
		}
		p.client = c
	}

	var br BlockResult

	switch rb.FC {
	case 1:
		bits, err := p.client.ReadCoils(rb.Address, rb.Quantity)
		if err != nil {
			p.maybeInvalidateClient(err)
			res.Err = err
			p.recordFailure(err)
			return res
		}
		br = BlockResult{FC: rb.FC, Address: rb.Address, Quantity: rb.Quantity, Bits: bits}

	case 2:
		bits, err := p.client.ReadDiscreteInputs(rb.Address, rb.Quantity)
		if err != nil {
			p.maybeInvalidateClient(err)
			res.Err = err
			p.recordFailure(err)
			return res
		}
		br = BlockResult{FC: rb.FC, Address: rb.Address, Quantity: rb.Quantity, Bits: bits}

	case 3:
		regs, err := p.client.ReadHoldingRegisters(rb.Address, rb.Quantity)
		if err != nil {
			p.maybeInvalidateClient(err)
			res.Err = err
			p.recordFailure(err)
			return res
		}
		br = BlockResult{FC: rb.FC, Address: rb.Address, Quantity: rb.Quantity, Registers: regs}

	case 4:
		regs, err := p.client.ReadInputRegisters(rb.Address, rb.Quantity)
		if err != nil {
			p.maybeInvalidateClient(err)
			res.Err = err
			p.recordFailure(err)
			return res
		}
		br = BlockResult{FC: rb.FC, Address: rb.Address, Quantity: rb.Quantity, Registers: regs}

	default:
		res.Err = errors.New("poller: unsupported function code")
		p.recordFailure(res.Err)
		return res
	}

	res.Blocks = []BlockResult{br}

	// Successful read
	p.recordSuccess()

	return res
}

// recordSuccess updates counters for a successful read.
func (p *Poller) recordSuccess() {
	p.counters.ResponsesValidTotal++
	p.counters.ConsecutiveFailCurr = 0
}

// recordFailure updates counters for a failed read.
func (p *Poller) recordFailure(err error) {
	if err == nil {
		return
	}

	// Classify timeout separately
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		p.counters.TimeoutsTotal++
	} else {
		p.counters.TransportErrorsTotal++
	}

	p.counters.ConsecutiveFailCurr++

	if p.counters.ConsecutiveFailCurr > p.counters.ConsecutiveFailMax {
		p.counters.ConsecutiveFailMax = p.counters.ConsecutiveFailCurr
	}
}

// Counters returns a snapshot copy of the transport counters.
func (p *Poller) Counters() TransportCounters {
	return p.counters
}

// maybeInvalidateClient discards the current client only when the error indicates
// the underlying TCP connection is dead.
//
// This is NOT a retry loop.
// It simply prevents a known-dead client from poisoning future ticks.
func (p *Poller) maybeInvalidateClient(err error) {
	if err == nil {
		return
	}
	if !isDeadConnErr(err) {
		return
	}

	// Close if the concrete client supports it.
	if c, ok := p.client.(interface{ Close() error }); ok {
		_ = c.Close()
	}

	p.client = nil
}

// isDeadConnErr is a conservative classifier for transport-death errors.
// If it returns true, reusing the same client is very likely to fail forever.
func isDeadConnErr(err error) bool {
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return false
	}

	s := strings.ToLower(err.Error())

	if strings.Contains(s, "eof") {
		return true
	}
	if strings.Contains(s, "broken pipe") {
		return true
	}
	if strings.Contains(s, "connection reset") {
		return true
	}
	if strings.Contains(s, "connection aborted") {
		return true
	}
	if strings.Contains(s, "use of closed network connection") {
		return true
	}
	if strings.Contains(s, "forcibly closed by the remote host") {
		return true
	}
	if strings.Contains(s, "wsasend") || strings.Contains(s, "wsarecv") {
		return true
	}

	return false
}
