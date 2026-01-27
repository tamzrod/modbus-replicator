// internal/status/snapshot.go
package status

// Snapshot represents exactly what the writer is allowed to deliver.
// It contains no logic and no memory of the past beyond current state.
type Snapshot struct {
	Health         uint16
	LastErrorCode  uint16
	SecondsInError uint16
}
