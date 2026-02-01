// cmd/replicator/main.go
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/tamzrod/modbus-replicator/internal/config"
	"github.com/tamzrod/modbus-replicator/internal/poller"
	"github.com/tamzrod/modbus-replicator/internal/status"
	"github.com/tamzrod/modbus-replicator/internal/writer"
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

		// ---- writer plan ----
		plan, err := writer.BuildPlan(unit)
		if err != nil {
			log.Fatalf("writer plan failed (unit=%s): %v", unit.ID, err)
		}

		// ---- writer clients ----
		clients, closeWriters, err := writer.BuildEndpointClients(unit)
		if err != nil {
			log.Fatalf("writer clients failed (unit=%s): %v", unit.ID, err)
		}
		defer closeWriters()

		dataWriter := writer.New(plan, clients)
		statusWriters := writer.NewDeviceStatusWriters(plan, clients)

		out := make(chan poller.PollResult)

		// ---- orchestrator ----
		go func(unitID string) {
			snap := status.Snapshot{
				Health:         status.HealthUnknown,
				LastErrorCode:  0,
				SecondsInError: 0,
			}

			secTicker := time.NewTicker(time.Second)
			defer secTicker.Stop()

			// initial full assert
			for _, sw := range statusWriters {
				_ = sw.WriteStatus(snap)
			}

			for {
				select {
				case <-ctx.Done():
					return

				case res := <-out:
					if err := dataWriter.Write(res); err != nil {
						log.Printf("writer error (unit=%s): %v", unitID, err)
					}

					if len(statusWriters) == 0 {
						continue
					}

					if res.Err == nil {
						changed := false
						if snap.Health != status.HealthOK {
							snap.Health = status.HealthOK
							changed = true
						}
						if snap.LastErrorCode != 0 {
							snap.LastErrorCode = 0
							changed = true
						}
						if snap.SecondsInError != 0 {
							snap.SecondsInError = 0
							changed = true
						}

						if changed {
							for _, sw := range statusWriters {
								_ = sw.WriteStatus(snap)
							}
						}
					} else {
						changed := false
						if snap.Health != status.HealthError {
							snap.Health = status.HealthError
							changed = true
						}

						code := errorCode(res.Err)
						if snap.LastErrorCode != code {
							snap.LastErrorCode = code
							changed = true
						}

						if changed {
							for _, sw := range statusWriters {
								_ = sw.WriteStatus(snap)
							}
						}
					}

				case <-secTicker.C:
					if len(statusWriters) == 0 {
						continue
					}
					if snap.Health != status.HealthOK && snap.SecondsInError < 65535 {
						snap.SecondsInError++
						for _, sw := range statusWriters {
							_ = sw.WriteStatus(snap)
						}
					}
				}
			}
		}(unit.ID)

		go p.Run(ctx, out)
	}

	// daemon block
	for {
		time.Sleep(time.Hour)
	}
}

func errorCode(err error) uint16 {
	if err == nil {
		return 0
	}

	type coderA interface{ Code() uint16 }
	type coderB interface{ ErrorCode() uint16 }
	type coderC interface{ ModbusCode() uint16 }

	var a coderA
	if errors.As(err, &a) {
		return a.Code()
	}
	var b coderB
	if errors.As(err, &b) {
		return b.ErrorCode()
	}
	var c coderC
	if errors.As(err, &c) {
		return c.ModbusCode()
	}

	return 1
}
