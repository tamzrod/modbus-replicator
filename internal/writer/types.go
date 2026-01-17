// internal/writer/types.go
package writer

import "modbus-replicator/internal/poller"

// MemoryDest is one MMA memory destination inside an endpoint.
type MemoryDest struct {
	MemoryID uint16
	Offsets  map[int]uint16 // per-FC offset deltas; missing FC => 0
}

// TargetEndpoint is one target endpoint (TCP) with one or more memory destinations.
type TargetEndpoint struct {
	TargetID  uint32
	Endpoint  string
	Memories  []MemoryDest
}

// Plan is the fully-built write plan for one unit.
type Plan struct {
	UnitID  string
	Targets []TargetEndpoint
}

// Writer writes poll snapshots into targets.
type Writer interface {
	Write(res poller.PollResult) error
}
