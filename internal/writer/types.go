// internal/writer/types.go
package writer

import "github.com/tamzrod/modbus-replicator/internal/poller"

// MemoryDest is one write destination inside an endpoint.
// Offsets are per-FC address deltas; missing FC => 0.
type MemoryDest struct {
	Offsets map[int]uint16
}

// TargetEndpoint is one target endpoint (TCP) with one or more destinations.
type TargetEndpoint struct {
	TargetID uint32
	Endpoint string
	Memories []MemoryDest
}

// StatusPlan describes where and how device status is written for ONE target.
type StatusPlan struct {
	Endpoint   string
	UnitID     uint8
	BaseSlot   uint16
	DeviceName string
}

// Plan is the fully-built write plan for one unit.
type Plan struct {
	UnitID  string
	Targets []TargetEndpoint
	Status  []StatusPlan // per-target status (hot-standby replication)
}

// Writer writes poll snapshots into targets.
type Writer interface {
	Write(res poller.PollResult) error
}
