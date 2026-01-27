// internal/writer/status_writer.go
package writer

import "modbus-replicator/internal/status"

// StatusWriter is the delivery-only contract for device status.
// It receives a snapshot and writes it verbatim.
// No logic, no state, no interpretation.
type StatusWriter interface {
	WriteStatus(s status.Snapshot) error
}
