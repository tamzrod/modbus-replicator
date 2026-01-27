// internal/config/validate.go
package config

import (
	"fmt"
)

// Validate checks configuration correctness.
// It performs declarative validation only.
// It MUST NOT mutate configuration.
func Validate(cfg *Config) error {
	type span struct {
		start uint16
		end   uint16
		unit  string
	}

	// ------------------------------------------------------------
	// DEVICE STATUS BLOCK VALIDATION (OPT-IN)
	// ------------------------------------------------------------

	statusUsed := false
	slotOwner := make(map[uint16]string)

	for _, u := range cfg.Replicator.Units {
		// device_name sanity (ASCII only)
		if u.Source.DeviceName != "" {
			for i := 0; i < len(u.Source.DeviceName); i++ {
				if u.Source.DeviceName[i] > 0x7F {
					return fmt.Errorf(
						"unit %q: device_name must contain ASCII characters only",
						u.ID,
					)
				}
			}
		}

		// status is opt-in; only validate if status_slot is provided
		if u.Source.StatusSlot == nil {
			continue
		}

		statusUsed = true
		slot := *u.Source.StatusSlot

		// status_slot uniqueness
		if prev, exists := slotOwner[slot]; exists {
			return fmt.Errorf(
				"status_slot collision: slot=%d used by units %q and %q",
				slot,
				prev,
				u.ID,
			)
		}
		slotOwner[slot] = u.ID
	}

	// Status_Memory is required only if status is used
	if statusUsed {
		if cfg.Replicator.StatusMemory == nil || cfg.Replicator.StatusMemory.Endpoint == "" {
			return fmt.Errorf(
				"Status_Memory.endpoint must be defined when any unit uses status_slot",
			)
		}
	}

	// ------------------------------------------------------------
	// DESTINATION MEMORY GEOMETRY VALIDATION
	// ------------------------------------------------------------

	// key = endpoint | memory_id | fc
	spans := make(map[string][]span)

	for _, u := range cfg.Replicator.Units {
		for _, t := range u.Targets {
			for _, m := range t.Memories {
				for _, r := range u.Reads {
					offset := uint16(0)
					if m.Offsets != nil {
						if v, ok := m.Offsets[int(r.FC)]; ok {
							offset = v
						}
					}

					start := offset + r.Address
					end := start + r.Quantity - 1

					key := fmt.Sprintf("%s|%d|%d", t.Endpoint, m.MemoryID, r.FC)

					existing := spans[key]
					for _, s := range existing {
						// overlap check (inclusive)
						if !(end < s.start || start > s.end) {
							return fmt.Errorf(
								"memory overlap: endpoint=%s memory_id=%d fc=%d range=%d-%d overlaps with unit=%s range=%d-%d",
								t.Endpoint,
								m.MemoryID,
								r.FC,
								start,
								end,
								s.unit,
								s.start,
								s.end,
							)
						}
					}

					spans[key] = append(spans[key], span{
						start: start,
						end:   end,
						unit:  u.ID,
					})
				}
			}
		}
	}

	return nil
}
