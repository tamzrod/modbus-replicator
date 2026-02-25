// internal/status/snapshot.go
package status

// Snapshot represents exactly what the writer is allowed to deliver.
// It contains no logic and no memory of the past beyond current state.
//
// This structure mirrors the 30-slot Device Status Block layout.
//
// Slots 0–19  → Operational Truth
// Slots 20–29 → Transport Lifetime Counters
type Snapshot struct {
	// --- Operational Truth (Slots 0–2) ---
	Health         uint16
	LastErrorCode  uint16
	SecondsInError uint16

	// --- Transport Lifetime Counters (Slots 20–29) ---

	RequestsTotal        uint32
	ResponsesValidTotal  uint32
	TimeoutsTotal        uint32
	TransportErrorsTotal uint32

	ConsecutiveFailCurr uint16
	ConsecutiveFailMax  uint16
}