// internal/writer/status_writer.go
package writer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tamzrod/modbus-replicator/internal/status"
)

// StatusWriter is the delivery-only contract for device status.
// It receives a snapshot and writes it verbatim.
// No logic, no state, no interpretation.
type StatusWriter interface {
	WriteStatus(s status.Snapshot) error
}

// deviceStatusWriter is the concrete implementation used by the replicator.
type deviceStatusWriter struct {
	plan *StatusPlan
	cli  endpointClient

	needFull bool
	last     status.Snapshot
	nameRegs []uint16
}

const statusAreaHoldingRegisters byte = 3

// NewDeviceStatusWriter builds a status writer if status is enabled for the unit.
// If plan.Status is nil, status is disabled.
func NewDeviceStatusWriter(plan Plan, clients map[string]endpointClient) (*deviceStatusWriter, bool) {
	if plan.Status == nil {
		return nil, false
	}

	sp := plan.Status
	cli := clients[sp.Endpoint]

	return &deviceStatusWriter{
		plan:     sp,
		cli:      cli,
		needFull: true, // full re-assert on first successful write
		last: status.Snapshot{
			Health:         status.HealthUnknown,
			LastErrorCode:  0,
			SecondsInError: 0,
		},
		nameRegs: encodeDeviceNameRegs(sp.DeviceName),
	}, true
}

// WriteStatus delivers a device status snapshot into status memory.
// On any write failure, the next successful call will re-assert the full block.
func (sw *deviceStatusWriter) WriteStatus(s status.Snapshot) error {
	if sw == nil || sw.plan == nil {
		return errors.New("status writer: disabled")
	}
	if sw.cli == nil {
		return fmt.Errorf("status writer: missing client for endpoint %s", sw.plan.Endpoint)
	}
	if sw.plan.UnitID > 255 {
		return fmt.Errorf("status writer: unit id %d out of range", sw.plan.UnitID)
	}

	// ------------------------------------------------------------
	// HARD INVARIANT: seconds_in_error MUST NOT wrap
	// ------------------------------------------------------------
	if s.SecondsInError > 65535 {
		s.SecondsInError = 65535
	}

	baseAddr := sw.baseAddr()
	unitID := uint8(sw.plan.UnitID)

	// ------------------------------------------------------------
	// Full block write (identity re-assert)
	// ------------------------------------------------------------
	if sw.needFull {
		regs := sw.fullBlockRegs(s)

		if err := sw.cli.WriteRegisters(
			statusAreaHoldingRegisters,
			unitID,
			baseAddr,
			regs,
		); err != nil {
			sw.needFull = true
			return fmt.Errorf("status writer: full block write failed: %w", err)
		}

		sw.needFull = false
		sw.last = s
		return nil
	}

	var errs []string

	// Slot 0 — health_code
	if sw.last.Health != s.Health {
		if err := sw.cli.WriteRegisters(
			statusAreaHoldingRegisters,
			unitID,
			baseAddr+status.SlotHealthCode,
			[]uint16{s.Health},
		); err != nil {
			errs = append(errs, fmt.Sprintf("slot0 health write failed: %v", err))
		} else {
			sw.last.Health = s.Health
		}
	}

	// Slot 1 — last_error_code
	if sw.last.LastErrorCode != s.LastErrorCode {
		if err := sw.cli.WriteRegisters(
			statusAreaHoldingRegisters,
			unitID,
			baseAddr+status.SlotLastErrorCode,
			[]uint16{s.LastErrorCode},
		); err != nil {
			errs = append(errs, fmt.Sprintf("slot1 last_error write failed: %v", err))
		} else {
			sw.last.LastErrorCode = s.LastErrorCode
		}
	}

	// Slot 2 — seconds_in_error
	if sw.last.SecondsInError != s.SecondsInError {
		if err := sw.cli.WriteRegisters(
			statusAreaHoldingRegisters,
			unitID,
			baseAddr+status.SlotSecondsInError,
			[]uint16{s.SecondsInError},
		); err != nil {
			errs = append(errs, fmt.Sprintf("slot2 seconds write failed: %v", err))
		} else {
			sw.last.SecondsInError = s.SecondsInError
		}
	}

	if len(errs) > 0 {
		// Any partial failure introduces doubt — re-assert on next success.
		sw.needFull = true
		return errors.New("status writer: " + strings.Join(errs, " | "))
	}

	return nil
}

func (sw *deviceStatusWriter) baseAddr() uint16 {
	// Each device owns a fixed SlotsPerDevice block.
	return sw.plan.BaseSlot * status.SlotsPerDevice
}

func (sw *deviceStatusWriter) fullBlockRegs(s status.Snapshot) []uint16 {
	regs := make([]uint16, status.SlotsPerDevice)

	// Slots 0–2: live status
	regs[status.SlotHealthCode] = s.Health
	regs[status.SlotLastErrorCode] = s.LastErrorCode
	regs[status.SlotSecondsInError] = s.SecondsInError

	// Slots 3..(device name start - 1) are RESERVED → left as zero

	// Device name always lives at the end of the block
	for i := 0; i < status.SlotDeviceNameSlots; i++ {
		dst := status.SlotDeviceNameStart + i
		if dst >= 0 && dst < len(regs) && i < len(sw.nameRegs) {
			regs[dst] = sw.nameRegs[i]
		}
	}

	return regs
}

// encodeDeviceNameRegs packs up to 16 ASCII characters into 8 uint16 registers.
// Each register stores two ASCII bytes in big-endian order.
func encodeDeviceNameRegs(name string) []uint16 {
	out := make([]uint16, status.SlotDeviceNameSlots)

	b := []byte(name)
	if len(b) > status.DeviceNameMaxChars {
		b = b[:status.DeviceNameMaxChars]
	}

	// sanitize to printable ASCII
	for i := 0; i < len(b); i++ {
		if b[i] < 0x20 || b[i] > 0x7E {
			b[i] = '?'
		}
	}

	for i := 0; i < status.DeviceNameMaxChars; i += 2 {
		var hi, lo byte
		if i < len(b) {
			hi = b[i]
		}
		if i+1 < len(b) {
			lo = b[i+1]
		}
		out[i/2] = uint16(hi)<<8 | uint16(lo)
	}

	return out
}
