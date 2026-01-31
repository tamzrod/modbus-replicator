// internal/config/config.go
package config

type Config struct {
	Replicator ReplicatorConfig `yaml:"replicator"`
}

type ReplicatorConfig struct {
	StatusMemory *StatusMemoryConfig `yaml:"Status_Memory"`
	Units        []UnitConfig        `yaml:"units"`
}

type StatusMemoryConfig struct {
	Endpoint string `yaml:"endpoint"`
	UnitID   uint8  `yaml:"unit_id"`
}

// ---- UNIT ----

type UnitConfig struct {
	ID      string         `yaml:"id"`
	Source  SourceConfig   `yaml:"source"`
	Reads   []ReadConfig   `yaml:"reads"`
	Targets []TargetConfig `yaml:"targets"`
	Poll    PollConfig     `yaml:"poll"`
}

// ---- SOURCE ----

type SourceConfig struct {
	Endpoint  string `yaml:"endpoint"`
	UnitID    uint8  `yaml:"unit_id"`
	TimeoutMs int    `yaml:"timeout_ms"`

	// Device status block (optional)
	StatusSlot *uint16 `yaml:"status_slot"`
	DeviceName string  `yaml:"device_name"`
}

// ---- READ GEOMETRY ----

type ReadConfig struct {
	FC       uint8  `yaml:"fc"`
	Address  uint16 `yaml:"address"`
	Quantity uint16 `yaml:"quantity"`
}

// ---- TARGET ----

type TargetConfig struct {
	ID       uint32         `yaml:"id"`
	Endpoint string         `yaml:"endpoint"`
	Memories []MemoryConfig `yaml:"memories"`
}

type MemoryConfig struct {
	MemoryID uint16         `yaml:"memory_id"`
	Offsets  map[int]uint16 `yaml:"offsets"` // delta map; missing FC => 0
}

// ---- POLL ----

type PollConfig struct {
	IntervalMs int `yaml:"interval_ms"`
}
