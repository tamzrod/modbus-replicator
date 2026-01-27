// internal/config/normalize.go
package config

// Normalize applies post-validation normalization.
// It is allowed to mutate configuration.
// It MUST be called only after Validate().
func Normalize(cfg *Config) {
	if cfg == nil {
		return
	}

	for ui := range cfg.Replicator.Units {
		u := &cfg.Replicator.Units[ui]

		// ------------------------------------------------------------
		// DEVICE STATUS BLOCK NORMALIZATION (OPT-IN)
		// ------------------------------------------------------------

		// Skip units that did not opt in
		if u.Source.StatusSlot == nil {
			continue
		}

		// Normalize device_name:
		// - ASCII already validated
		// - Truncate to max 16 characters
		if len(u.Source.DeviceName) > 16 {
			u.Source.DeviceName = u.Source.DeviceName[:16]
		}

		// No other normalization is performed here.
		// Slot math, packing, and runtime writes belong to later stages.
	}
}
