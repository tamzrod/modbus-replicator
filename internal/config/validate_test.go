// internal/config/validate_test.go
package config

import "testing"

// helper to build a unit quickly
func unit(id string, endpoint string, memoryID uint16, fc uint8, addr, qty uint16, offset uint16) UnitConfig {
	return UnitConfig{
		ID: id,
		Reads: []ReadConfig{
			{
				FC:       fc,
				Address:  addr,
				Quantity: qty,
			},
		},
		Targets: []TargetConfig{
			{
				ID:       1,
				Endpoint: endpoint,
				Memories: []MemoryConfig{
					{
						MemoryID: memoryID,
						Offsets: map[int]uint16{
							int(fc): offset,
						},
					},
				},
			},
		},
	}
}

// ---- tests ----

func TestValidate_NoOverlapDifferentEndpoints(t *testing.T) {
	cfg := &Config{
		Replicator: ReplicatorConfig{
			Units: []UnitConfig{
				unit("u1", "ep1", 0, 3, 0, 10, 0),
				unit("u2", "ep2", 0, 3, 0, 10, 0),
			},
		},
	}

	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_NoOverlapDifferentMemory(t *testing.T) {
	cfg := &Config{
		Replicator: ReplicatorConfig{
			Units: []UnitConfig{
				unit("u1", "ep1", 0, 3, 0, 10, 0),
				unit("u2", "ep1", 1, 3, 0, 10, 0),
			},
		},
	}

	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_NoOverlapDifferentFC(t *testing.T) {
	cfg := &Config{
		Replicator: ReplicatorConfig{
			Units: []UnitConfig{
				unit("u1", "ep1", 0, 3, 0, 10, 0),
				unit("u2", "ep1", 0, 4, 0, 10, 0),
			},
		},
	}

	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_TouchingRangesAllowed(t *testing.T) {
	cfg := &Config{
		Replicator: ReplicatorConfig{
			Units: []UnitConfig{
				unit("u1", "ep1", 0, 3, 0, 10, 0),   // 0–9
				unit("u2", "ep1", 0, 3, 10, 10, 0), // 10–19
			},
		},
	}

	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_OverlapDetected(t *testing.T) {
	cfg := &Config{
		Replicator: ReplicatorConfig{
			Units: []UnitConfig{
				unit("u1", "ep1", 0, 3, 0, 10, 0), // 0–9
				unit("u2", "ep1", 0, 3, 5, 10, 0), // 5–14 → overlap
			},
		},
	}

	if err := Validate(cfg); err == nil {
		t.Fatalf("expected overlap error, got nil")
	}
}

func TestValidate_OverlapViaOffsetDetected(t *testing.T) {
	cfg := &Config{
		Replicator: ReplicatorConfig{
			Units: []UnitConfig{
				unit("u1", "ep1", 0, 3, 0, 10, 0),   // 0–9
				unit("u2", "ep1", 0, 3, 0, 10, 5),  // 5–14 → overlap
			},
		},
	}

	if err := Validate(cfg); err == nil {
		t.Fatalf("expected overlap error, got nil")
	}
}
