// internal/writer/builder.go
package writer

import (
	"errors"
	"time"

	cfg "modbus-replicator/internal/config"
	wmodbus "modbus-replicator/internal/writer/modbus"
)

// BuildPlan converts one unit config into a Writer Plan.
// Assumes config has already passed conflict validation.
func BuildPlan(u cfg.UnitConfig) (Plan, error) {
	if u.ID == "" {
		return Plan{}, errors.New("writer: unit.id required")
	}

	plan := Plan{UnitID: u.ID}

	for _, t := range u.Targets {
		ep := TargetEndpoint{
			TargetID: uint32(t.ID),
			Endpoint: t.Endpoint,
		}

		for _, m := range t.Memories {
			md := MemoryDest{
				MemoryID: m.MemoryID,
				Offsets:  m.Offsets, // map[int]uint16 (delta map)
			}
			ep.Memories = append(ep.Memories, md)
		}

		plan.Targets = append(plan.Targets, ep)
	}

	return plan, nil
}

// BuildEndpointClients creates one TCP client per unique endpoint.
func BuildEndpointClients(u cfg.UnitConfig) (map[string]*wmodbus.EndpointClient, func() error, error) {
	unique := map[string]struct{}{}
	for _, t := range u.Targets {
		unique[t.Endpoint] = struct{}{}
	}

	clients := make(map[string]*wmodbus.EndpointClient)
	var closers []func() error

	for endpoint := range unique {
		c, err := wmodbus.NewEndpointClient(wmodbus.Config{
			Endpoint: endpoint,
			Timeout:  time.Duration(u.Source.TimeoutMs) * time.Millisecond,
		})
		if err != nil {
			for _, fn := range closers {
				_ = fn()
			}
			return nil, nil, err
		}
		clients[endpoint] = c
		closers = append(closers, c.Close)
	}

	closeAll := func() error {
		var last error
		for _, fn := range closers {
			if err := fn(); err != nil {
				last = err
			}
		}
		return last
	}

	return clients, closeAll, nil
}
