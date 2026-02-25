// internal/writer/device_status_writer_test.go
package writer

import (
	"testing"

	"github.com/tamzrod/modbus-replicator/internal/status"
)

func TestDeviceNameWrittenOnFullAssertOnly(t *testing.T) {
	cli := &fakeEndpointClient{}

	plan := Plan{
		Status: []StatusPlan{
			{
				Endpoint:   "status-endpoint",
				UnitID:     1,
				BaseSlot:   0,
				DeviceName: "DEV-01",
			},
		},
	}

	clients := map[string]endpointClient{
		"status-endpoint": cli,
	}

	writers := NewDeviceStatusWriters(plan, clients)
	if len(writers) != 1 {
		t.Fatalf("expected 1 status writer, got %d", len(writers))
	}

	sw := writers[0]

	// ---- first write: FULL ASSERT ----
	first := status.Snapshot{
		Health:         status.HealthOK,
		LastErrorCode:  0,
		SecondsInError: 0,
	}

	if err := sw.WriteStatus(first); err != nil {
		t.Fatalf("initial full assert failed: %v", err)
	}

	if len(cli.lastRegs) != status.SlotsPerDevice {
		t.Fatalf(
			"expected full block write (%d regs), got %d",
			status.SlotsPerDevice,
			len(cli.lastRegs),
		)
	}

	expectedNameRegs := encodeDeviceNameRegs("DEV-01")

	for i := 0; i < status.SlotDeviceNameSlots; i++ {
		slot := status.SlotDeviceNameStart + i
		if cli.lastRegs[slot] != expectedNameRegs[i] {
			t.Fatalf(
				"device name slot %d mismatch: got=%d want=%d",
				slot,
				cli.lastRegs[slot],
				expectedNameRegs[i],
			)
		}
	}

	// ---- second write: INCREMENTAL ONLY ----
	second := status.Snapshot{
		Health:         status.HealthError,
		LastErrorCode:  7,
		SecondsInError: 1,
	}

	if err := sw.WriteStatus(second); err != nil {
		t.Fatalf("incremental write failed: %v", err)
	}

	if len(cli.lastRegs) == status.SlotsPerDevice {
		t.Fatalf("device name should not be rewritten on incremental update")
	}
}

func TestSecondsInErrorResetOnRecovery(t *testing.T) {
	cli := &fakeEndpointClient{}

	plan := Plan{
		Status: []StatusPlan{
			{
				Endpoint:   "status-endpoint",
				UnitID:     1,
				BaseSlot:   0,
				DeviceName: "DEV-01",
			},
		},
	}

	clients := map[string]endpointClient{
		"status-endpoint": cli,
	}

	writers := NewDeviceStatusWriters(plan, clients)
	if len(writers) != 1 {
		t.Fatalf("expected 1 status writer, got %d", len(writers))
	}

	sw := writers[0]

	// simulate ERROR
	errSnap := status.Snapshot{
		Health:         status.HealthError,
		LastErrorCode:  42,
		SecondsInError: 3,
	}

	if err := sw.WriteStatus(errSnap); err != nil {
		t.Fatalf("error snapshot write failed: %v", err)
	}

	// simulate recovery
	okSnap := status.Snapshot{
		Health:         status.HealthOK,
		LastErrorCode:  0,
		SecondsInError: 0,
	}

	if err := sw.WriteStatus(okSnap); err != nil {
		t.Fatalf("recovery snapshot write failed: %v", err)
	}

	expectedAddr := uint16(0)*status.SlotsPerDevice + status.SlotSecondsInError

	if cli.lastRegsAddr != expectedAddr {
		t.Fatalf("unexpected write addr: got=%d want=%d", cli.lastRegsAddr, expectedAddr)
	}

	if len(cli.lastRegs) != 1 {
		t.Fatalf("expected 1 register write, got %d", len(cli.lastRegs))
	}

	if cli.lastRegs[0] != 0 {
		t.Fatalf("seconds_in_error not reset: got=%d want=0", cli.lastRegs[0])
	}
}