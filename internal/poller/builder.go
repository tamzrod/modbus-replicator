// internal/poller/builder.go
package poller

import (
	"time"

	cfg "github.com/tamzrod/modbus-replicator/internal/config"
	pmodbus "github.com/tamzrod/modbus-replicator/internal/poller/modbus"
)

// Build constructs a Poller without touching the network.
// No dialing at startup. Device availability is runtime state.
func Build(u cfg.UnitConfig) (*Poller, func() error, error) {

	factory := func() (Client, error) {
		return pmodbus.New(pmodbus.Config{
			Endpoint: u.Source.Endpoint,
			UnitID:   u.Source.UnitID,
			Timeout:  time.Duration(u.Source.TimeoutMs) * time.Millisecond,
		})
	}

	reads := make([]ReadBlock, 0, len(u.Reads))
	for _, r := range u.Reads {
		rb := ReadBlock{
			FC:       r.FC,
			Address:  r.Address,
			Quantity: r.Quantity,
		}
		// Invert is only applicable to digital FCs (1 and 2).
		// Silently ignore it for all other function codes.
		if r.FC == 1 || r.FC == 2 {
			rb.Invert = r.Invert
		}
		reads = append(reads, rb)
	}

	p, err := New(
		Config{
			UnitID:   u.ID,
			Interval: time.Duration(u.Poll.IntervalMs) * time.Millisecond,
			Reads:    reads,
		},
		nil,      // no initial client
		factory,  // lazy connection
	)
	if err != nil {
		return nil, nil, err
	}

	return p, func() error { return nil }, nil
}