// internal/poller/poller.go
package poller

import (
	"errors"
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
type Config struct {
	UnitID   string
	Interval time.Duration
	Reads    []ReadBlock
}

// Poller reads from a field device via a Client.
// It reuses the client while healthy and discards it when the connection is dead.
// No retries are performed inside a poll cycle.
// A future tick may create a new client via factory.
type Poller struct {
	cfg Config

	client  Client
	factory func() (Client, error)
}

// New creates a poller with immutable config.
// - client is the initial connected client (may be nil)
// - factory creates a new connected client when the current one is missing/dead
func New(cfg Config, client Client, factory func() (Client, error)) (*Poller, error) {
	if cfg.UnitID == "" {
		return nil, errors.New("poller: unit id required")
	}
	if cfg.Interval <= 0 {
		return nil, errors.New("poller: interval must be > 0")
	}
	if len(cfg.Reads) == 0 {
		return nil, errors.New("poller: at least one read block required")
	}

	return &Poller{
		cfg:     cfg,
		client:  client,
		factory: factory,
	}, nil
}

// PollOnce performs exactly one poll cycle.
// All-or-nothing: any failure aborts the cycle.
//
// Connection policy:
// - reuse existing client while healthy
// - if client is nil, try to create once via factory
// - on a "dead connection" error, discard client (so next tick can recreate)
func (p *Poller) PollOnce() PollResult {
	res := PollResult{
		UnitID: p.cfg.UnitID,
		At:     time.Now(),
	}

	// Ensure we have a client for this attempt.
	if p.client == nil {
		if p.factory == nil {
			res.Err = errors.New("poller: client is nil and no factory provided")
			return res
		}
		c, err := p.factory()
		if err != nil {
			res.Err = err
			return res
		}
		p.client = c
	}

	var blocks []BlockResult

	for _, rb := range p.cfg.Reads {
		switch rb.FC {
		case 1:
			bits, err := p.client.ReadCoils(rb.Address, rb.Quantity)
			if err != nil {
				p.maybeInvalidateClient(err)
				res.Err = err
				return res
			}
			blocks = append(blocks, BlockResult{
				FC: rb.FC, Address: rb.Address, Quantity: rb.Quantity, Bits: bits,
			})

		case 2:
			bits, err := p.client.ReadDiscreteInputs(rb.Address, rb.Quantity)
			if err != nil {
				p.maybeInvalidateClient(err)
				res.Err = err
				return res
			}
			blocks = append(blocks, BlockResult{
				FC: rb.FC, Address: rb.Address, Quantity: rb.Quantity, Bits: bits,
			})

		case 3:
			regs, err := p.client.ReadHoldingRegisters(rb.Address, rb.Quantity)
			if err != nil {
				p.maybeInvalidateClient(err)
				res.Err = err
				return res
			}
			blocks = append(blocks, BlockResult{
				FC: rb.FC, Address: rb.Address, Quantity: rb.Quantity, Registers: regs,
			})

		case 4:
			regs, err := p.client.ReadInputRegisters(rb.Address, rb.Quantity)
			if err != nil {
				p.maybeInvalidateClient(err)
				res.Err = err
				return res
			}
			blocks = append(blocks, BlockResult{
				FC: rb.FC, Address: rb.Address, Quantity: rb.Quantity, Registers: regs,
			})

		default:
			res.Err = errors.New("poller: unsupported function code")
			return res
		}
	}

	// Commit only if all reads succeeded
	res.Blocks = blocks
	return res
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
	// First: if it is a net.Error timeout, do NOT automatically mark dead.
	// Timeouts can be transient.
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return false
	}

	s := strings.ToLower(err.Error())

	// Cross-platform dead-transport markers.
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

	// Windows-specific: this is your exact error class.
	// Example:
	// "wsasend: an existing connection was forcibly closed by the remote host."
	if strings.Contains(s, "forcibly closed by the remote host") {
		return true
	}
	if strings.Contains(s, "wsasend") || strings.Contains(s, "wsarecv") {
		return true
	}

	return false
}
