// internal/writer/writer_test.go
package writer

import (
	"errors"
	"testing"
	"time"

	"modbus-replicator/internal/poller"
)

// ------------------------------------------------------------------
// Fake endpoint client (implements endpointClient fully)
// ------------------------------------------------------------------

type fakeEndpointClient struct {
	writeErr error

	lastBitsAddr uint16
	lastRegsAddr uint16
	lastRegs     []uint16
	lastBits     []bool

	writeBitsCnt int
	writeRegsCnt int
}

func (f *fakeEndpointClient) WriteBits(area byte, unitID uint8, addr uint16, bits []bool) error {
	f.writeBitsCnt++
	f.lastBitsAddr = addr
	f.lastBits = bits
	return f.writeErr
}

func (f *fakeEndpointClient) WriteRegisters(area byte, unitID uint8, addr uint16, regs []uint16) error {
	f.writeRegsCnt++
	f.lastRegsAddr = addr
	f.lastRegs = regs
	return f.writeErr
}

// ------------------------------------------------------------------
// Tests
// ------------------------------------------------------------------

func TestWriter_DataWrite_Success(t *testing.T) {
	plan := Plan{
		UnitID: "unit-1",
		Targets: []TargetEndpoint{
			{
				TargetID: 1,
				Endpoint: "ep1",
				Memories: []MemoryDest{
					{
						Offsets: map[int]uint16{
							3: 100,
						},
					},
				},
			},
		},
	}

	fake := &fakeEndpointClient{}
	w := New(plan, map[string]endpointClient{
		"ep1": fake,
	})

	res := poller.PollResult{
		UnitID: "unit-1",
		At:     time.Now(),
		Blocks: []poller.BlockResult{
			{
				FC:        3,
				Address:   10,
				Quantity:  2,
				Registers: []uint16{11, 22},
			},
		},
	}

	if err := w.Write(res); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fake.writeRegsCnt != 1 {
		t.Fatalf("expected 1 register write, got %d", fake.writeRegsCnt)
	}

	if fake.lastRegsAddr != 110 {
		t.Fatalf("expected addr 110, got %d", fake.lastRegsAddr)
	}
}

func TestWriter_DataWrite_Error(t *testing.T) {
	plan := Plan{
		UnitID: "unit-1",
		Targets: []TargetEndpoint{
			{
				TargetID: 1,
				Endpoint: "ep1",
				Memories: []MemoryDest{
					{Offsets: nil},
				},
			},
		},
	}

	fake := &fakeEndpointClient{
		writeErr: errors.New("fail"),
	}

	w := New(plan, map[string]endpointClient{
		"ep1": fake,
	})

	res := poller.PollResult{
		UnitID: "unit-1",
		At:     time.Now(),
		Blocks: []poller.BlockResult{
			{
				FC:        3,
				Address:   0,
				Quantity:  1,
				Registers: []uint16{1},
			},
		},
	}

	if err := w.Write(res); err == nil {
		t.Fatalf("expected writer error, got nil")
	}
}

func TestWriter_SkipDataOnPollError(t *testing.T) {
	plan := Plan{
		UnitID: "unit-1",
	}

	fake := &fakeEndpointClient{}
	w := New(plan, map[string]endpointClient{
		"ep1": fake,
	})

	res := poller.PollResult{
		UnitID: "unit-1",
		At:     time.Now(),
		Err:    errors.New("poll failed"),
	}

	// Poll error is NOT a writer error
	if err := w.Write(res); err != nil {
		t.Fatalf("unexpected writer error: %v", err)
	}

	if fake.writeBitsCnt != 0 || fake.writeRegsCnt != 0 {
		t.Fatalf("expected no writes on poll error")
	}
}

// Edge case: no targets configured
func TestWriter_NoTargets_IsNoOp(t *testing.T) {
	plan := Plan{
		UnitID:  "unit-1",
		Targets: nil,
	}

	fake := &fakeEndpointClient{}
	w := New(plan, map[string]endpointClient{
		"ep1": fake,
	})

	res := poller.PollResult{
		UnitID: "unit-1",
		At:     time.Now(),
		Blocks: []poller.BlockResult{
			{
				FC:        3,
				Address:   0,
				Quantity:  1,
				Registers: []uint16{1},
			},
		},
	}

	if err := w.Write(res); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fake.writeBitsCnt != 0 || fake.writeRegsCnt != 0 {
		t.Fatalf("expected no writes when no targets are configured")
	}
}

// New edge case: target endpoint has no client
func TestWriter_MissingClient_ReturnsError(t *testing.T) {
	plan := Plan{
		UnitID: "unit-1",
		Targets: []TargetEndpoint{
			{
				TargetID: 1,
				Endpoint: "missing-ep",
				Memories: []MemoryDest{
					{Offsets: nil},
				},
			},
		},
	}

	// Note: clients map intentionally does NOT include "missing-ep"
	w := New(plan, map[string]endpointClient{})

	res := poller.PollResult{
		UnitID: "unit-1",
		At:     time.Now(),
		Blocks: []poller.BlockResult{
			{
				FC:        3,
				Address:   0,
				Quantity:  1,
				Registers: []uint16{1},
			},
		},
	}

	if err := w.Write(res); err == nil {
		t.Fatalf("expected error when client is missing, got nil")
	}
}
