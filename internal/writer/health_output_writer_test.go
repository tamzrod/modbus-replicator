package writer

import (
	"errors"
	"testing"

	cfg "github.com/tamzrod/modbus-replicator/internal/config"
	"github.com/tamzrod/modbus-replicator/internal/status"
)

func TestBuildPlan_HealthOutputEnabled(t *testing.T) {
	addr := uint16(99)

	plan, err := BuildPlan(cfg.UnitConfig{
		ID: "meter_1",
		Targets: []cfg.TargetConfig{
			{
				ID:       1,
				Endpoint: "mma:502",
				UnitID:   7,
				HealthOutput: &cfg.HealthOutputConfig{
					Enabled: true,
					Area:    "coil",
					Address: &addr,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildPlan returned error: %v", err)
	}

	if len(plan.HealthOutputs) != 1 {
		t.Fatalf("expected 1 health output plan, got %d", len(plan.HealthOutputs))
	}

	got := plan.HealthOutputs[0]
	if got.Endpoint != "mma:502" || got.UnitID != 7 || got.Area != 1 || got.Address != 99 || got.UnitName != "meter_1" {
		t.Fatalf("unexpected health output plan: %+v", got)
	}
}

func TestHealthOutputWriter_PublishesOnMappedEdgeOnly(t *testing.T) {
	cli := &fakeEndpointClient{}
	writers := NewHealthOutputWriters(Plan{
		HealthOutputs: []HealthOutputPlan{
			{
				Endpoint: "mma:502",
				UnitID:   7,
				Area:     1,
				Address:  99,
				UnitName: "meter_1",
			},
		},
	}, map[string]endpointClient{"mma:502": cli})

	if len(writers) != 1 {
		t.Fatalf("expected 1 health output writer, got %d", len(writers))
	}

	hw := writers[0]

	if err := hw.PublishHealth(status.HealthError); err != nil {
		t.Fatalf("PublishHealth(ERROR) failed: %v", err)
	}
	if cli.writeBitsCnt != 1 {
		t.Fatalf("expected first publish, got %d writes", cli.writeBitsCnt)
	}
	if cli.lastBitsArea != 1 || cli.lastBitsUnitID != 7 || cli.lastBitsAddr != 99 {
		t.Fatalf("unexpected write destination: area=%d unit=%d addr=%d", cli.lastBitsArea, cli.lastBitsUnitID, cli.lastBitsAddr)
	}
	if len(cli.lastBits) != 1 || cli.lastBits[0] {
		t.Fatalf("expected published coil value 0, got %v", cli.lastBits)
	}

	if err := hw.PublishHealth(status.HealthUnknown); err != nil {
		t.Fatalf("PublishHealth(UNKNOWN) failed: %v", err)
	}
	if cli.writeBitsCnt != 1 {
		t.Fatalf("expected no write for non-OK to non-OK transition, got %d writes", cli.writeBitsCnt)
	}

	if err := hw.PublishHealth(status.HealthOK); err != nil {
		t.Fatalf("PublishHealth(OK) failed: %v", err)
	}
	if cli.writeBitsCnt != 2 {
		t.Fatalf("expected second publish on recovery, got %d writes", cli.writeBitsCnt)
	}
	if len(cli.lastBits) != 1 || !cli.lastBits[0] {
		t.Fatalf("expected published coil value 1, got %v", cli.lastBits)
	}

	if err := hw.PublishHealth(status.HealthOK); err != nil {
		t.Fatalf("PublishHealth(OK repeat) failed: %v", err)
	}
	if cli.writeBitsCnt != 2 {
		t.Fatalf("expected no repeat write for steady OK, got %d writes", cli.writeBitsCnt)
	}
}

func TestHealthOutputWriter_RetriesAfterFailure(t *testing.T) {
	cli := &fakeEndpointClient{writeErr: errors.New("fail")}
	writers := NewHealthOutputWriters(Plan{
		HealthOutputs: []HealthOutputPlan{
			{
				Endpoint: "mma:502",
				UnitID:   7,
				Area:     1,
				Address:  99,
				UnitName: "meter_1",
			},
		},
	}, map[string]endpointClient{"mma:502": cli})

	hw := writers[0]

	if err := hw.PublishHealth(status.HealthError); err == nil {
		t.Fatal("expected publish failure")
	}
	if cli.writeBitsCnt != 1 {
		t.Fatalf("expected first publish attempt, got %d writes", cli.writeBitsCnt)
	}

	cli.writeErr = nil
	if err := hw.PublishHealth(status.HealthError); err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	if cli.writeBitsCnt != 2 {
		t.Fatalf("expected retry write, got %d writes", cli.writeBitsCnt)
	}
}
