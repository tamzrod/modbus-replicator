// internal/config/validate.go
package config

import (
	"fmt"
)

// Validate checks for destination memory overlaps across all units.
// It enforces geometry safety only.
func Validate(cfg *Config) error {
	type span struct {
		start uint16
		end   uint16
		unit  string
	}

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
