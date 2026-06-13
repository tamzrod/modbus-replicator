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
	// key = endpoint | unit_id | area | address
	healthOutputOwner := make(map[string]string)

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

		statusEnabled := u.Source.StatusSlot != nil
		if statusEnabled && len(u.Targets) == 0 {
			return fmt.Errorf(
				"unit %q: status_slot is set but no targets are defined",
				u.ID,
			)
		}

		var slot uint16
		if statusEnabled {
			slot = *u.Source.StatusSlot
		}

		for _, t := range u.Targets {
			if statusEnabled && t.StatusUnitID == nil {
				return fmt.Errorf(
					"unit %q: status_slot is set but target %q has no status_unit_id",
					u.ID,
					t.Endpoint,
				)
			}

			if statusEnabled {
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

			if t.HealthOutput == nil || !t.HealthOutput.Enabled {
				continue
			}

			if t.HealthOutput.Area != "coil" {
				return fmt.Errorf(
					"unit %q: target %q health_output.area must be %q",
					u.ID,
					t.Endpoint,
					"coil",
				)
			}

			if t.HealthOutput.Address == nil {
				return fmt.Errorf(
					"unit %q: target %q health_output.address is required when enabled",
					u.ID,
					t.Endpoint,
				)
			}

			key := fmt.Sprintf(
				"%s|%d|%s|%d",
				t.Endpoint,
				t.UnitID,
				t.HealthOutput.Area,
				*t.HealthOutput.Address,
			)

			if prev, exists := healthOutputOwner[key]; exists {
				return fmt.Errorf(
					"health_output collision: endpoint=%s unit_id=%d area=%s address=%d used by units %q and %q",
					t.Endpoint,
					t.UnitID,
					t.HealthOutput.Area,
					*t.HealthOutput.Address,
					prev,
					u.ID,
				)
			}

			healthOutputOwner[key] = u.ID
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
					// addinvert appends an inverted copy immediately after the original block,
					// doubling the destination footprint for FC1/FC2.
					if (r.FC == 1 || r.FC == 2) && r.AddInvert {
						end = start + 2*r.Quantity - 1
					}

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
