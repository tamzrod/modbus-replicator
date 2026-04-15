// internal/config/duplicate.go
package config

import (
	"fmt"
	"regexp"
	"strconv"
)

// suffixRe matches a trailing _N suffix (e.g. "Device_1", "unit-3_12").
var suffixRe = regexp.MustCompile(`^(.*?)_(\d+)$`)

// baseAndCounter splits an ID into its base name and numeric suffix.
// If no numeric suffix is found, the counter is 0.
func baseAndCounter(id string) (string, int) {
	if m := suffixRe.FindStringSubmatch(id); m != nil {
		n, _ := strconv.Atoi(m[2])
		return m[1], n
	}
	return id, 0
}

// uniqueID returns the lowest-numbered "base_N" (N ≥ 1) that does not
// already appear in existing.  It stops after 65536 attempts and returns an
// error, which in practice is unreachable given the hardware limits imposed by
// unit_id (uint8) and status_slot (uint16).
func uniqueID(srcID string, existing map[string]bool) (string, error) {
	base, _ := baseAndCounter(srcID)
	for n := 1; n <= 65536; n++ {
		candidate := fmt.Sprintf("%s_%d", base, n)
		if !existing[candidate] {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no unique ID available: all %s_1 through %s_65536 are in use", base, base)
}

// nextFreeUnitID returns the first uint8 value greater than start that is
// not present in used.  It searches only up to 255 (the uint8 maximum) and
// returns an error when no free value is found in that range.
func nextFreeUnitID(start uint8, used map[uint8]bool) (uint8, error) {
	for candidate := uint16(start) + 1; candidate <= 255; candidate++ {
		c := uint8(candidate)
		if !used[c] {
			return c, nil
		}
	}
	return 0, fmt.Errorf(
		"no free unit_id available: all values %d–255 are already in use",
		uint16(start)+1,
	)
}

// nextFreeStatusSlot returns the first uint16 value greater than start that
// is not present in used.  It returns an error if all values from start+1 to
// 65535 are already occupied.
func nextFreeStatusSlot(start uint16, used map[uint16]bool) (uint16, error) {
	for candidate := uint32(start) + 1; candidate <= 65535; candidate++ {
		s := uint16(candidate)
		if !used[s] {
			return s, nil
		}
	}
	return 0, fmt.Errorf(
		"no free status_slot available: all values %d–65535 are already in use",
		uint32(start)+1,
	)
}

// deepCopyUnit returns a fully independent copy of u.
// No slice or map inside the returned value shares memory with u.
func deepCopyUnit(u UnitConfig) UnitConfig {
	dup := u

	// Deep copy Reads.
	if u.Reads != nil {
		dup.Reads = make([]ReadConfig, len(u.Reads))
		copy(dup.Reads, u.Reads)
	}

	// Deep copy Targets (and nested Memories + Offsets).
	if u.Targets != nil {
		dup.Targets = make([]TargetConfig, len(u.Targets))
		for i, t := range u.Targets {
			tc := t

			// Deep copy StatusUnitID pointer.
			if t.StatusUnitID != nil {
				v := *t.StatusUnitID
				tc.StatusUnitID = &v
			}

			// Deep copy Memories.
			if t.Memories != nil {
				tc.Memories = make([]MemoryConfig, len(t.Memories))
				for j, m := range t.Memories {
					mc := m
					// Deep copy Offsets map.
					if m.Offsets != nil {
						mc.Offsets = make(map[int]uint16, len(m.Offsets))
						for k, v := range m.Offsets {
							mc.Offsets[k] = v
						}
					}
					tc.Memories[j] = mc
				}
			}

			dup.Targets[i] = tc
		}
	}

	return dup
}

// DuplicateUnit creates a deep copy of the unit identified by sourceID and
// resolves all identity conflicts so the duplicate can be safely appended to
// cfg.Replicator.Units.
//
// The following fields are modified in the duplicate:
//
//   - ID            – unique string ID derived from the source (e.g. "Dev" →
//     "Dev_1" → "Dev_2"); collisions with existing IDs are
//     resolved automatically.
//   - Source.UnitID – incremented from the source value until a value that is
//     not used by any existing unit is found.  Returns an error
//     when all values from source+1 to 255 are occupied.
//   - Source.StatusSlot – (if set) incremented from the source value until a
//     slot that is not used by any existing unit is found.
//
// Every other field (Endpoint, Reads, Targets, Memories, Offsets, Poll, …) is
// deep-copied with no shared references.
//
// The returned UnitConfig is NOT automatically appended to cfg; the caller is
// responsible for appending it and re-running Validate to confirm correctness.
func DuplicateUnit(cfg *Config, sourceID string) (UnitConfig, error) {
	// Locate the source unit.
	var src *UnitConfig
	for i := range cfg.Replicator.Units {
		if cfg.Replicator.Units[i].ID == sourceID {
			src = &cfg.Replicator.Units[i]
			break
		}
	}
	if src == nil {
		return UnitConfig{}, fmt.Errorf("duplicate: unit %q not found", sourceID)
	}

	units := cfg.Replicator.Units

	// Index occupied IDs, unit_ids, and status_slots.
	existingIDs := make(map[string]bool, len(units))
	usedUnitIDs := make(map[uint8]bool, len(units))
	usedStatusSlots := make(map[uint16]bool, len(units))
	for _, u := range units {
		existingIDs[u.ID] = true
		usedUnitIDs[u.Source.UnitID] = true
		if u.Source.StatusSlot != nil {
			usedStatusSlots[*u.Source.StatusSlot] = true
		}
	}

	// Resolve unique unit_id.
	newUnitID, err := nextFreeUnitID(src.Source.UnitID, usedUnitIDs)
	if err != nil {
		return UnitConfig{}, fmt.Errorf("duplicate: %w", err)
	}

	// Resolve unique status_slot (only when the source opted in).
	var newStatusSlot *uint16
	if src.Source.StatusSlot != nil {
		slot, err := nextFreeStatusSlot(*src.Source.StatusSlot, usedStatusSlots)
		if err != nil {
			return UnitConfig{}, fmt.Errorf("duplicate: %w", err)
		}
		newStatusSlot = &slot
	}

	// Deep-copy and assign the new identity values.
	dup := deepCopyUnit(*src)

	newID, err := uniqueID(src.ID, existingIDs)
	if err != nil {
		return UnitConfig{}, fmt.Errorf("duplicate: %w", err)
	}
	dup.ID = newID
	dup.Source.UnitID = newUnitID
	dup.Source.StatusSlot = newStatusSlot

	return dup, nil
}
