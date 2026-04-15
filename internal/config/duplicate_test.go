// internal/config/duplicate_test.go
package config

import (
	"testing"
)

// ---- helpers ----------------------------------------------------------------

func ptr[T any](v T) *T { return &v }

// makeUnit builds a minimal UnitConfig with sensible defaults.
func makeUnit(id string, unitID uint8, statusSlot *uint16) UnitConfig {
	u := UnitConfig{
		ID: id,
		Source: SourceConfig{
			Endpoint:   "127.0.0.1:502",
			UnitID:     unitID,
			StatusSlot: statusSlot,
			DeviceName: id,
		},
		Reads: []ReadConfig{
			{FC: 3, Address: 0, Quantity: 10},
		},
		Targets: []TargetConfig{
			{
				ID:           1,
				Endpoint:     "127.0.0.1:503",
				UnitID:       uint8(unitID + 100),
				StatusUnitID: ptr(uint8(50)),
				Memories: []MemoryConfig{
					{
						MemoryID: 1,
						Offsets:  map[int]uint16{3: 0},
					},
				},
			},
		},
		Poll: PollConfig{IntervalMs: 1000},
	}
	return u
}

func cfg1(units ...UnitConfig) *Config {
	return &Config{Replicator: ReplicatorConfig{Units: units}}
}

// ---- ID uniqueness ----------------------------------------------------------

func TestDuplicateUnit_IDSuffix_FirstDuplicate(t *testing.T) {
	c := cfg1(makeUnit("Device", 1, nil))
	dup, err := DuplicateUnit(c, "Device")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dup.ID != "Device_1" {
		t.Errorf("expected ID=Device_1, got %q", dup.ID)
	}
}

func TestDuplicateUnit_IDSuffix_CollisionResolved(t *testing.T) {
	// "Device" and "Device_1" already exist → next should be "Device_2".
	c := cfg1(
		makeUnit("Device", 1, nil),
		makeUnit("Device_1", 2, nil),
	)
	dup, err := DuplicateUnit(c, "Device")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dup.ID != "Device_2" {
		t.Errorf("expected ID=Device_2, got %q", dup.ID)
	}
}

func TestDuplicateUnit_IDSuffix_DuplicatingAlreadySuffixed(t *testing.T) {
	// Duplicating "Device_1" when "Device_2" already exists → "Device_3".
	c := cfg1(
		makeUnit("Device", 1, nil),
		makeUnit("Device_1", 2, nil),
		makeUnit("Device_2", 3, nil),
	)
	dup, err := DuplicateUnit(c, "Device_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dup.ID != "Device_3" {
		t.Errorf("expected ID=Device_3, got %q", dup.ID)
	}
}

// ---- unit_id resolution -----------------------------------------------------

func TestDuplicateUnit_UnitID_SimpleIncrement(t *testing.T) {
	c := cfg1(makeUnit("A", 5, nil))
	dup, err := DuplicateUnit(c, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dup.Source.UnitID != 6 {
		t.Errorf("expected UnitID=6, got %d", dup.Source.UnitID)
	}
}

func TestDuplicateUnit_UnitID_CollisionSkipped(t *testing.T) {
	// 5 is taken, 6 is taken → should get 7.
	c := cfg1(
		makeUnit("A", 5, nil),
		makeUnit("B", 6, nil),
	)
	dup, err := DuplicateUnit(c, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dup.Source.UnitID != 7 {
		t.Errorf("expected UnitID=7, got %d", dup.Source.UnitID)
	}
}

func TestDuplicateUnit_UnitID_ExhaustedReturnsError(t *testing.T) {
	// Build units with unit_ids 254 and 255 so that start=254 has no free slot.
	c := cfg1(
		makeUnit("A", 254, nil),
		makeUnit("B", 255, nil),
	)
	_, err := DuplicateUnit(c, "A")
	if err == nil {
		t.Fatal("expected error when all unit_ids ≥ 255 are taken, got nil")
	}
}

// ---- status_slot resolution -------------------------------------------------

func TestDuplicateUnit_StatusSlot_SimpleIncrement(t *testing.T) {
	c := cfg1(makeUnit("A", 1, ptr(uint16(0))))
	dup, err := DuplicateUnit(c, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dup.Source.StatusSlot == nil {
		t.Fatal("expected StatusSlot to be set, got nil")
	}
	if *dup.Source.StatusSlot != 1 {
		t.Errorf("expected StatusSlot=1, got %d", *dup.Source.StatusSlot)
	}
}

func TestDuplicateUnit_StatusSlot_CollisionSkipped(t *testing.T) {
	// slot 0 (src) and slot 1 exist → should get slot 2.
	c := cfg1(
		makeUnit("A", 1, ptr(uint16(0))),
		makeUnit("B", 2, ptr(uint16(1))),
	)
	dup, err := DuplicateUnit(c, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dup.Source.StatusSlot == nil {
		t.Fatal("expected StatusSlot to be set, got nil")
	}
	if *dup.Source.StatusSlot != 2 {
		t.Errorf("expected StatusSlot=2, got %d", *dup.Source.StatusSlot)
	}
}

func TestDuplicateUnit_StatusSlot_ExhaustedReturnsError(t *testing.T) {
	// Build a config where status_slot 65535 is in use, so start=65534 has
	// nowhere left to go (only candidate is 65535, which is taken).
	c := cfg1(
		makeUnit("A", 1, ptr(uint16(65534))),
		makeUnit("B", 2, ptr(uint16(65535))),
	)
	_, err := DuplicateUnit(c, "A")
	if err == nil {
		t.Fatal("expected error when all status_slots ≥ 65535 are taken, got nil")
	}
}

func TestDuplicateUnit_StatusSlot_NilPreserved(t *testing.T) {
	// Source has no status_slot → duplicate should also have none.
	c := cfg1(makeUnit("A", 1, nil))
	dup, err := DuplicateUnit(c, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dup.Source.StatusSlot != nil {
		t.Errorf("expected StatusSlot=nil, got %d", *dup.Source.StatusSlot)
	}
}

// ---- source not found -------------------------------------------------------

func TestDuplicateUnit_SourceNotFound(t *testing.T) {
	c := cfg1(makeUnit("A", 1, nil))
	_, err := DuplicateUnit(c, "unknown")
	if err == nil {
		t.Fatal("expected error for unknown sourceID, got nil")
	}
}

// ---- deep copy independence -------------------------------------------------

func TestDuplicateUnit_DeepCopy_ReadsIndependent(t *testing.T) {
	c := cfg1(makeUnit("A", 1, nil))
	dup, err := DuplicateUnit(c, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mutate the duplicate's Reads.
	dup.Reads[0].Address = 9999

	// Original must be unchanged.
	if c.Replicator.Units[0].Reads[0].Address == 9999 {
		t.Error("mutating duplicate Reads affected the original")
	}
}

func TestDuplicateUnit_DeepCopy_TargetsIndependent(t *testing.T) {
	c := cfg1(makeUnit("A", 1, nil))
	dup, err := DuplicateUnit(c, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mutate the duplicate's Target.
	dup.Targets[0].Endpoint = "mutated"

	if c.Replicator.Units[0].Targets[0].Endpoint == "mutated" {
		t.Error("mutating duplicate Targets affected the original")
	}
}

func TestDuplicateUnit_DeepCopy_OffsetsIndependent(t *testing.T) {
	c := cfg1(makeUnit("A", 1, nil))
	dup, err := DuplicateUnit(c, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mutate the duplicate's Offsets map.
	dup.Targets[0].Memories[0].Offsets[3] = 9999

	if c.Replicator.Units[0].Targets[0].Memories[0].Offsets[3] == 9999 {
		t.Error("mutating duplicate Offsets map affected the original")
	}
}

func TestDuplicateUnit_DeepCopy_StatusUnitIDIndependent(t *testing.T) {
	c := cfg1(makeUnit("A", 1, ptr(uint16(0))))
	dup, err := DuplicateUnit(c, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mutate the duplicate's StatusUnitID.
	*dup.Targets[0].StatusUnitID = 200

	if *c.Replicator.Units[0].Targets[0].StatusUnitID == 200 {
		t.Error("mutating duplicate StatusUnitID affected the original")
	}
}

func TestDuplicateUnit_DeepCopy_StatusSlotIndependent(t *testing.T) {
	c := cfg1(makeUnit("A", 1, ptr(uint16(0))))
	dup, err := DuplicateUnit(c, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mutate the duplicate's StatusSlot.
	*dup.Source.StatusSlot = 9999

	if *c.Replicator.Units[0].Source.StatusSlot == 9999 {
		t.Error("mutating duplicate StatusSlot affected the original")
	}
}

// ---- endpoint copied exactly -----------------------------------------------

func TestDuplicateUnit_EndpointCopiedExact(t *testing.T) {
	u := makeUnit("A", 1, nil)
	u.Source.Endpoint = "192.168.1.100:1502"
	c := cfg1(u)

	dup, err := DuplicateUnit(c, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dup.Source.Endpoint != "192.168.1.100:1502" {
		t.Errorf("endpoint not copied: got %q", dup.Source.Endpoint)
	}
}

// ---- reads / mappings copied ------------------------------------------------

func TestDuplicateUnit_ReadsCopied(t *testing.T) {
	u := makeUnit("A", 1, nil)
	u.Reads = []ReadConfig{
		{FC: 3, Address: 100, Quantity: 20},
		{FC: 4, Address: 200, Quantity: 30},
	}
	c := cfg1(u)

	dup, err := DuplicateUnit(c, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dup.Reads) != 2 {
		t.Fatalf("expected 2 reads, got %d", len(dup.Reads))
	}
	if dup.Reads[0].FC != 3 || dup.Reads[1].FC != 4 {
		t.Error("reads not copied correctly")
	}
}

func TestDuplicateUnit_MemoriesCopied(t *testing.T) {
	u := makeUnit("A", 1, nil)
	u.Targets[0].Memories = []MemoryConfig{
		{MemoryID: 10, Offsets: map[int]uint16{3: 100, 4: 200}},
		{MemoryID: 11, Offsets: map[int]uint16{3: 300}},
	}
	c := cfg1(u)

	dup, err := DuplicateUnit(c, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dup.Targets[0].Memories) != 2 {
		t.Fatalf("expected 2 memories, got %d", len(dup.Targets[0].Memories))
	}
	if dup.Targets[0].Memories[0].Offsets[3] != 100 {
		t.Error("memory offsets not copied correctly")
	}
}

// ---- sequential duplications build a clean chain ---------------------------

func TestDuplicateUnit_SequentialDuplicates(t *testing.T) {
	c := cfg1(makeUnit("Pump", 1, ptr(uint16(0))))

	// First duplicate.
	d1, err := DuplicateUnit(c, "Pump")
	if err != nil {
		t.Fatalf("first duplicate: %v", err)
	}
	c.Replicator.Units = append(c.Replicator.Units, d1)

	// Second duplicate.
	d2, err := DuplicateUnit(c, "Pump")
	if err != nil {
		t.Fatalf("second duplicate: %v", err)
	}
	c.Replicator.Units = append(c.Replicator.Units, d2)

	// IDs must form "Pump", "Pump_1", "Pump_2".
	ids := []string{
		c.Replicator.Units[0].ID,
		c.Replicator.Units[1].ID,
		c.Replicator.Units[2].ID,
	}
	expected := []string{"Pump", "Pump_1", "Pump_2"}
	for i, want := range expected {
		if ids[i] != want {
			t.Errorf("unit[%d]: expected ID=%q, got %q", i, want, ids[i])
		}
	}

	// unit_ids must be 1, 2, 3.
	for i, want := range []uint8{1, 2, 3} {
		got := c.Replicator.Units[i].Source.UnitID
		if got != want {
			t.Errorf("unit[%d]: expected UnitID=%d, got %d", i, want, got)
		}
	}

	// status_slots must be 0, 1, 2.
	for i, want := range []uint16{0, 1, 2} {
		got := *c.Replicator.Units[i].Source.StatusSlot
		if got != want {
			t.Errorf("unit[%d]: expected StatusSlot=%d, got %d", i, want, got)
		}
	}
}

// ---- poll config copied -----------------------------------------------------

func TestDuplicateUnit_PollCopied(t *testing.T) {
	u := makeUnit("A", 1, nil)
	u.Poll.IntervalMs = 5000
	c := cfg1(u)

	dup, err := DuplicateUnit(c, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dup.Poll.IntervalMs != 5000 {
		t.Errorf("expected Poll.IntervalMs=5000, got %d", dup.Poll.IntervalMs)
	}
}
