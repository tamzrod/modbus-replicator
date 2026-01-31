// internal/poller/builder.go
package poller

import (
	"time"

	cfg "github.com/tamzrod/modbus-replicator/internal/config"
	pmodbus "github.com/tamzrod/modbus-replicator/internal/poller/modbus"
)

// Build constructs a Poller and wires Modbus client lifecycle.
// Connection is reused while healthy.
// On transport death, Poller discards the client and uses factory on a future tick.
// No retries, no loops, no semantics.
func Build(u cfg.UnitConfig) (*Poller, func() error, error) {
	// client factory: ONE attempt per call
	factory := func() (Client, error) {
		return pmodbus.New(pmodbus.Config{
			Endpoint: u.Source.Endpoint,
			UnitID:   u.Source.UnitID,
			Timeout:  time.Duration(u.Source.TimeoutMs) * time.Millisecond,
		})
	}

	// initial client (fail fast at startup)
	client, err := factory()
	if err != nil {
		return nil, nil, err
	}

	reads := make([]ReadBlock, 0, len(u.Reads))
	for _, r := range u.Reads {
		reads = append(reads, ReadBlock{
			FC:       r.FC,
			Address:  r.Address,
			Quantity: r.Quantity,
		})
	}

	p, err := New(
		Config{
			UnitID:   u.ID,
			Interval: time.Duration(u.Poll.IntervalMs) * time.Millisecond,
			Reads:    reads,
		},
		client,
		factory,
	)
	if err != nil {
		return nil, nil, err
	}

	// No-op closer: poller handles client lifecycle internally
	return p, func() error { return nil }, nil
}
