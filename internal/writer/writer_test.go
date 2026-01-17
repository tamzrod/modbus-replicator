// internal/writer/writer_test.go
package writer

import (
	"testing"

	"modbus-replicator/internal/poller"
)

// ---- fake endpoint client ----

type fakeEndpointClient struct {
	writes []writeCall
}

type writeCall struct {
	unitID uint8
	addr   uint16
	fc     uint8
	qty    int
}

func (f *fakeEndpointClient) WriteCoils(unitID uint8, addr uint16, bits []bool) error {
	f.writes = append(f.writes, writeCall{
		unitID: unitID,
		addr:   addr,
		fc:     1,
		qty:    len(bits),
	})
	return nil
}

func (f *fakeEndpointClient) WriteRegisters(unitID uint8, addr uint16, regs []uint16) error {
	f.writes = append(f.writes, writeCall{
		unitID: unitID,
		addr:   addr,
		fc:     3,
		qty:    len(regs),
	})
	return nil
}

// ---- tests ----

func TestWriter_OffsetMathPerFC(t *testing.T) {
	fake := &fakeEndpointClient{}

	plan := Plan{
		UnitID: "unit-1",
		Targets: []TargetEndpoint{
			{
				TargetID: 1,
				Endpoint: "ep1",
				Memories: []MemoryDest{
					{
						MemoryID: 1,
						Offsets: map[int]uint16{
							1: 10,  // coils
							3: 100, // holding registers
						},
					},
				},
			},
		},
	}

	w := &modbusWriter{
		plan: plan,
		clients: map[string]endpointClient{
			"ep1": fake,
		},
	}

	res := poller.PollResult{
		UnitID: "unit-1",
		Blocks: []poller.BlockResult{
			{FC: 1, Address: 5, Quantity: 4, Bits: []bool{true, false, true, false}},
			{FC: 3, Address: 2, Quantity: 3, Registers: []uint16{1, 2, 3}},
		},
	}

	if err := w.Write(res); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(fake.writes) != 2 {
		t.Fatalf("expected 2 writes, got %d", len(fake.writes))
	}

	if fake.writes[0].addr != 15 { // 10 + 5
		t.Fatalf("expected coils addr 15, got %d", fake.writes[0].addr)
	}

	if fake.writes[1].addr != 102 { // 100 + 2
		t.Fatalf("expected regs addr 102, got %d", fake.writes[1].addr)
	}
}

func TestWriter_DefaultOffsetZero(t *testing.T) {
	fake := &fakeEndpointClient{}

	plan := Plan{
		UnitID: "unit-1",
		Targets: []TargetEndpoint{
			{
				TargetID: 1,
				Endpoint: "ep1",
				Memories: []MemoryDest{
					{
						MemoryID: 1,
						Offsets:  nil, // default zero
					},
				},
			},
		},
	}

	w := &modbusWriter{
		plan: plan,
		clients: map[string]endpointClient{
			"ep1": fake,
		},
	}

	res := poller.PollResult{
		Blocks: []poller.BlockResult{
			{FC: 3, Address: 20, Quantity: 2, Registers: []uint16{9, 9}},
		},
	}

	if err := w.Write(res); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fake.writes[0].addr != 20 {
		t.Fatalf("expected addr 20, got %d", fake.writes[0].addr)
	}
}
