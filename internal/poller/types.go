// internal/poller/types.go
package poller

import "time"

// ReadBlock describes one Modbus read geometry.
// Geometry only: no semantics.
type ReadBlock struct {
	FC       uint8
	Address  uint16
	Quantity uint16
}

// BlockResult is the raw result of a single read.
type BlockResult struct {
	FC       uint8
	Address  uint16
	Quantity uint16

	// Exactly one of these is used depending on FC.
	Bits      []bool   // FC 1,2
	Registers []uint16 // FC 3,4
}

// PollResult is a snapshot produced by one poll cycle.
type PollResult struct {
	UnitID string
	At     time.Time

	// RawErrorCode is copied verbatim from the device.
	// 0 means success; non-zero is opaque and device-defined.
	RawErrorCode uint16

	Blocks []BlockResult
	Err    error // non-nil means the poll cycle failed
}
