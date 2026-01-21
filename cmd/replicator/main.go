// cmd/replicator/main.go
package main

import (
	"context"
	"log"
	"os"
	"time"

	"modbus-replicator/internal/config"
	"modbus-replicator/internal/poller"
	"modbus-replicator/internal/writer"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: replicator <config.yaml>")
	}

	cfgPath := os.Args[1]

	// --------------------
	// Load + validate config
	// --------------------

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	if err := config.Validate(cfg); err != nil {
		log.Fatalf("config validation failed: %v", err)
	}

	ctx := context.Background()

	// --------------------
	// Build per-unit pipelines
	// --------------------

	for _, unit := range cfg.Replicator.Units {

		// ---- poller ----
		p, closePoller, err := poller.Build(unit)
		if err != nil {
			log.Fatalf("poller build failed (unit=%s): %v", unit.ID, err)
		}
		defer closePoller()

		// ---- writer ----
		plan, err := writer.BuildPlan(unit)
		if err != nil {
			log.Fatalf("writer plan failed (unit=%s): %v", unit.ID, err)
		}

		clients, closeWriters, err := writer.BuildEndpointClients(unit)
		if err != nil {
			log.Fatalf("writer clients failed (unit=%s): %v", unit.ID, err)
		}
		defer closeWriters()

		w := writer.New(plan, clients)

		// ---- channel between poller and writer ----
		out := make(chan poller.PollResult)

		// writer consumer
		go func(unitID string) {
			for res := range out {
				if err := w.Write(res); err != nil {
					log.Printf("writer error (unit=%s): %v", unitID, err)
				}
			}
		}(unit.ID)

		// poller producer
		go p.Run(ctx, out)
	}

	// --------------------
	// Block forever (daemon-safe, no deadlock)
	// --------------------
	for {
		time.Sleep(time.Hour)
	}
}
