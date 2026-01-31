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
		if plan.Status != nil {
    	plan.Status.Endpoint = cfg.Replicator.StatusMemory.Endpoint
		}

		if err != nil {
			log.Fatalf("writer plan failed (unit=%s): %v", unit.ID, err)
		}

		// ---- writer clients (DATA + STATUS) ----
		clients, closeWriters, err := writer.BuildEndpointClients(
			unit,
			cfg.Replicator.StatusMemory.Endpoint,
		)
		if err != nil {
			log.Fatalf("writer clients failed (unit=%s): %v", unit.ID, err)
		}
		defer closeWriters()

		dataWriter := writer.New(plan, clients)

		// Status writer (optional per unit)
		statusWriter, statusEnabled := writer.NewDeviceStatusWriter(plan, clients)

		// ---- channel between poller and writer ----
		out := make(chan poller.PollResult)

		// Orchestrator (runner-owned state + 1Hz seconds ticker)
		go func(unitID string) {
			var snap status.Snapshot

			// Default snapshot state on start.
			snap.Health = status.HealthUnknown
			snap.LastErrorCode = 0
			snap.SecondsInError = 0

			secTicker := time.NewTicker(time.Second)
			defer secTicker.Stop()

			// Full block write on start (identity re-assert) if enabled.
			if statusEnabled {
				if err := statusWriter.WriteStatus(snap); err != nil {
					log.Printf("status write failed on start (unit=%s): %v", unitID, err)
				}
			}

			for {
				select {
				case <-ctx.Done():
					return

				case res := <-out:
					// --- data delivery ---
					if err := dataWriter.Write(res); err != nil {
						log.Printf("writer error (unit=%s): %v", unitID, err)
					}

					// --- status update (device-level truth) ---
					if !statusEnabled {
						continue
					}

					if res.Err == nil {
						// Recovery / OK
						changed := false

						if snap.Health != status.HealthOK {
							snap.Health = status.HealthOK
							changed = true
						}
						// Reset last error code when healthy.
						if snap.LastErrorCode != 0 {
							snap.LastErrorCode = 0
							changed = true
						}
						// Reset seconds-in-error on recovery.
						if snap.SecondsInError != 0 {
							snap.SecondsInError = 0
							changed = true
						}

						if changed {
							if err := statusWriter.WriteStatus(snap); err != nil {
								log.Printf("status write failed (unit=%s): %v", unitID, err)
							}
						}
					} else {
						// Error
						changed := false

						if snap.Health != status.HealthError {
							snap.Health = status.HealthError
							changed = true
						}

						// Set raw-ish error code (best-effort pass-through).
						code := errorCode(res.Err)
						if snap.LastErrorCode != code {
							snap.LastErrorCode = code
							changed = true
						}

						// NOTE: seconds_in_error increments on the 1Hz ticker only.
						// No increment here.

						if changed {
							if err := statusWriter.WriteStatus(snap); err != nil {
								log.Printf("status write failed (unit=%s): %v", unitID, err)
							}
						}
					}

				case <-secTicker.C:
					if !statusEnabled {
						continue
					}

					// Tick 1 Hz while not OK.
					if snap.Health != status.HealthOK {
						if snap.SecondsInError < 65535 {
							snap.SecondsInError++
							if err := statusWriter.WriteStatus(snap); err != nil {
								log.Printf("status seconds tick write failed (unit=%s): %v", unitID, err)
							}
						}
					}
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

// errorCode extracts a best-effort uint16 code from an error without assuming concrete types.
// If the error does not expose a code, returns 1 (generic error).
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
