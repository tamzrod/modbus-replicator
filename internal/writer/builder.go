// internal/writer/builder.go
package writer

import (
	"errors"
	"time"

	cfg "github.com/tamzrod/modbus-replicator/internal/config"
	ingest "github.com/tamzrod/modbus-replicator/internal/writer/ingest"
)

// BuildPlan converts one unit config into a Writer Plan.
// Assumes config has already passed validation.
func BuildPlan(u cfg.UnitConfig) (Plan, error) {
	if u.ID == "" {
		return Plan{}, errors.New("writer: unit.id required")
	}

	plan := Plan{
		UnitID: u.ID,
	}

	// ------------------------------------------------------------
	// STATUS PLAN (OPT-IN)
	// ------------------------------------------------------------
	if u.Source.StatusSlot != nil {
		plan.Status = &StatusPlan{
			// Endpoint is resolved via Status_Memory at higher level
			Endpoint:   "",
			UnitID:     uint16(u.Source.UnitID),
			BaseSlot:   *u.Source.StatusSlot,
			DeviceName: u.Source.DeviceName,
		}
	}

	// ------------------------------------------------------------
	// DATA TARGETS
	// ------------------------------------------------------------
	for _, t := range u.Targets {
		ep := TargetEndpoint{
			TargetID: uint32(t.ID),
			Endpoint: t.Endpoint,
		}

		for _, m := range t.Memories {
			ep.Memories = append(ep.Memories, MemoryDest{
				Offsets: m.Offsets,
			})
		}

		plan.Targets = append(plan.Targets, ep)
	}

	return plan, nil
}

// BuildEndpointClients creates Raw Ingest clients and returns them
// as writer.endpointClient interfaces.
func BuildEndpointClients(
	u cfg.UnitConfig,
	statusEndpoint string, // <<< ADD THIS
) (map[string]endpointClient, func() error, error) {

	unique := map[string]struct{}{}

	// ------------------------------------------------------------
	// TARGET ENDPOINTS
	// ------------------------------------------------------------
	for _, t := range u.Targets {
		unique[t.Endpoint] = struct{}{}
	}

	// ------------------------------------------------------------
	// STATUS ENDPOINT (if enabled)
	// ------------------------------------------------------------
	if u.Source.StatusSlot != nil && statusEndpoint != "" {
		unique[statusEndpoint] = struct{}{}
	}

	clients := make(map[string]endpointClient)
	var closers []func() error

	for endpoint := range unique {
		c, err := ingest.NewEndpointClient(ingest.Config{
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
