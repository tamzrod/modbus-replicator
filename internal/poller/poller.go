// internal/poller/poller.go
package poller

import (
	"errors"
	"time"
)

// Client abstracts Modbus operations needed by the poller.
// The poller depends on geometry only.
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

// Poller is a dumb, clock-driven reader.
type Poller struct {
	cfg    Config
	client Client
}

// New creates a poller with immutable config.
func New(cfg Config, client Client) (*Poller, error) {
	if cfg.UnitID == "" {
		return nil, errors.New("poller: unit id required")
	}
	if cfg.Interval <= 0 {
		return nil, errors.New("poller: interval must be > 0")
	}
	if len(cfg.Reads) == 0 {
		return nil, errors.New("poller: at least one read block required")
	}
	return &Poller{cfg: cfg, client: client}, nil
}

// PollOnce performs exactly one poll cycle.
// All-or-nothing: any failure aborts the cycle.
func (p *Poller) PollOnce() PollResult {
	res := PollResult{
		UnitID: p.cfg.UnitID,
		At:     time.Now(),
	}

	var blocks []BlockResult

	for _, rb := range p.cfg.Reads {
		switch rb.FC {
		case 1:
			bits, err := p.client.ReadCoils(rb.Address, rb.Quantity)
			if err != nil {
				res.Err = err
				return res
			}
			blocks = append(blocks, BlockResult{
				FC: rb.FC, Address: rb.Address, Quantity: rb.Quantity, Bits: bits,
			})

		case 2:
			bits, err := p.client.ReadDiscreteInputs(rb.Address, rb.Quantity)
			if err != nil {
				res.Err = err
				return res
			}
			blocks = append(blocks, BlockResult{
				FC: rb.FC, Address: rb.Address, Quantity: rb.Quantity, Bits: bits,
			})

		case 3:
			regs, err := p.client.ReadHoldingRegisters(rb.Address, rb.Quantity)
			if err != nil {
				res.Err = err
				return res
			}
			blocks = append(blocks, BlockResult{
				FC: rb.FC, Address: rb.Address, Quantity: rb.Quantity, Registers: regs,
			})

		case 4:
			regs, err := p.client.ReadInputRegisters(rb.Address, rb.Quantity)
			if err != nil {
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
