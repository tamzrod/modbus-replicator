// internal/poller/scheduler.go
package poller

import "time"

// readSchedule tracks the per-read execution state for one ReadBlock.
type readSchedule struct {
	cfg      ReadBlock
	nextExec time.Time
}
