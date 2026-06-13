package writer

import (
	"fmt"
	"log"

	"github.com/tamzrod/modbus-replicator/internal/status"
)

const healthOutputAreaCoil byte = 1

type HealthOutputWriter interface {
	PublishHealth(health uint16) error
}

type healthOutputWriter struct {
	plan *HealthOutputPlan
	cli  endpointClient

	hasLast bool
	last    uint16
}

func NewHealthOutputWriters(plan Plan, clients map[string]endpointClient) []HealthOutputWriter {
	var out []HealthOutputWriter

	for _, hp := range plan.HealthOutputs {
		cli := clients[hp.Endpoint]
		if cli == nil {
			continue
		}

		out = append(out, &healthOutputWriter{
			plan: &hp,
			cli:  cli,
		})
	}

	return out
}

func (hw *healthOutputWriter) PublishHealth(health uint16) error {
	if hw == nil || hw.plan == nil {
		return fmt.Errorf("failed to publish health output: disabled")
	}
	if hw.cli == nil {
		return fmt.Errorf("failed to publish health output for %s: missing client for endpoint %s", hw.plan.UnitName, hw.plan.Endpoint)
	}

	value := health == status.HealthOK
	if hw.hasLast && (hw.last == status.HealthOK) == value {
		return nil
	}

	if err := hw.cli.WriteBits(hw.plan.Area, hw.plan.UnitID, hw.plan.Address, []bool{value}); err != nil {
		return fmt.Errorf("failed to publish health output for %s: %w", hw.plan.UnitName, err)
	}

	log.Printf(
		"%s health output changed %s -> %s, publishing value=%d",
		hw.plan.UnitName,
		healthLabel(hw.last, hw.hasLast),
		healthLabel(health, true),
		boolToUint(value),
	)

	hw.last = health
	hw.hasLast = true
	return nil
}

func healthLabel(health uint16, known bool) string {
	if !known {
		return "UNKNOWN"
	}

	switch health {
	case status.HealthOK:
		return "OK"
	case status.HealthError:
		return "ERROR"
	case status.HealthStale:
		return "STALE"
	case status.HealthDisabled:
		return "DISABLED"
	default:
		return "UNKNOWN"
	}
}

func boolToUint(v bool) uint8 {
	if v {
		return 1
	}
	return 0
}
