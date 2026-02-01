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
	// DEVICE STATUS BLOCK VALIDATION (PER-TARGET, OPT-IN)
	// ------------------------------------------------------------

	// key = endpoint | status_unit_id | status_slot
	statusOwner := make(map[string]string)

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

		// status is opt-in
		if u.Source.StatusSlot == nil {
			continue
		}

		// status requires at least one target
		if len(u.Targets) == 0 {
			return fmt.Errorf(
				"unit %q: status_slot is set but no targets are defined",
				u.ID,
			)
		}

		slot := *u.Source.StatusSlot

		for _, t := range u.Targets {
			// each target must declare status_unit_id
			if t.StatusUnitID == nil {
				return fmt.Errorf(
					"unit %q: status_slot is set but target %q has no status_unit_id",
					u.ID,
					t.Endpoint,
				)
			}

			key := fmt.Sprintf(
				"%s|%d|%d",
				t.Endpoint,
				*t.StatusUnitID,
				slot,
			)

			if prev, exists := statusOwner[key]; exists {
				return fmt.Errorf(
					"status_slot collision: endpoint=%s status_unit_id=%d slot=%d used by units %q and %q",
					t.Endpoint,
					*t.StatusUnitID,
					slot,
					prev,
					u.ID,
				)
			}

			statusOwner[key] = u.ID
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
