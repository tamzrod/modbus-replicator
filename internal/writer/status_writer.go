// internal/writer/status_writer.go
package writer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tamzrod/modbus-replicator/internal/status"
)

// StatusWriter is the delivery-only contract for device status.
// No logic, no state, no interpretation.
type StatusWriter interface {
	WriteStatus(s status.Snapshot) error
}

// deviceStatusWriter writes status for ONE target (replica).
type deviceStatusWriter struct {
	plan *StatusPlan
	cli  endpointClient

	needFull bool
	last     status.Snapshot
	nameRegs []uint16
}

const statusAreaHoldingRegisters byte = 3

// NewDeviceStatusWriters builds per-target status writers.
// Returns empty slice if status is disabled.
func NewDeviceStatusWriters(plan Plan, clients map[string]endpointClient) []StatusWriter {
	var out []StatusWriter

	for _, sp := range plan.Status {
		cli := clients[sp.Endpoint]
		if cli == nil {
			continue
		}

		out = append(out, &deviceStatusWriter{
			plan:     &sp,
			cli:      cli,
			needFull: true,
			last: status.Snapshot{
				Health:         status.HealthUnknown,
				LastErrorCode:  0,
				SecondsInError: 0,
			},
			nameRegs: encodeDeviceNameRegs(sp.DeviceName),
		})
	}

	return out
}

func (sw *deviceStatusWriter) WriteStatus(s status.Snapshot) error {
	if sw == nil || sw.plan == nil {
		return errors.New("status writer: disabled")
	}
	if sw.cli == nil {
		return fmt.Errorf("status writer: missing client for endpoint %s", sw.plan.Endpoint)
	}

	if s.SecondsInError > 65535 {
		s.SecondsInError = 65535
	}

	baseAddr := sw.baseAddr()
	unitID := uint8(sw.plan.UnitID)

	// ------------------------------------------------------------
	// FULL BLOCK RE-ASSERT
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

	// --- SLOT 0 ---
	if sw.last.Health != s.Health {
		if err := sw.writeOne(baseAddr+status.SlotHealthCode, unitID, s.Health); err != nil {
			errs = append(errs, err.Error())
		} else {
			sw.last.Health = s.Health
		}
	}

	// --- SLOT 1 ---
	if sw.last.LastErrorCode != s.LastErrorCode {
		if err := sw.writeOne(baseAddr+status.SlotLastErrorCode, unitID, s.LastErrorCode); err != nil {
			errs = append(errs, err.Error())
		} else {
			sw.last.LastErrorCode = s.LastErrorCode
		}
	}

	// --- SLOT 2 ---
	if sw.last.SecondsInError != s.SecondsInError {
		if err := sw.writeOne(baseAddr+status.SlotSecondsInError, unitID, s.SecondsInError); err != nil {
			errs = append(errs, err.Error())
		} else {
			sw.last.SecondsInError = s.SecondsInError
		}
	}

	// --- TRANSPORT COUNTERS (20–29) ---
	sw.writeUint32(&errs, baseAddr+status.SlotRequestsTotalLow, unitID,
		sw.last.RequestsTotal, s.RequestsTotal,
		func(v uint32) { sw.last.RequestsTotal = v },
	)

	sw.writeUint32(&errs, baseAddr+status.SlotResponsesValidTotalLow, unitID,
		sw.last.ResponsesValidTotal, s.ResponsesValidTotal,
		func(v uint32) { sw.last.ResponsesValidTotal = v },
	)

	sw.writeUint32(&errs, baseAddr+status.SlotTimeoutsTotalLow, unitID,
		sw.last.TimeoutsTotal, s.TimeoutsTotal,
		func(v uint32) { sw.last.TimeoutsTotal = v },
	)

	sw.writeUint32(&errs, baseAddr+status.SlotTransportErrorsTotalLow, unitID,
		sw.last.TransportErrorsTotal, s.TransportErrorsTotal,
		func(v uint32) { sw.last.TransportErrorsTotal = v },
	)

	if sw.last.ConsecutiveFailCurr != s.ConsecutiveFailCurr {
		if err := sw.writeOne(baseAddr+status.SlotConsecutiveFailCurr, unitID, s.ConsecutiveFailCurr); err != nil {
			errs = append(errs, err.Error())
		} else {
			sw.last.ConsecutiveFailCurr = s.ConsecutiveFailCurr
		}
	}

	if sw.last.ConsecutiveFailMax != s.ConsecutiveFailMax {
		if err := sw.writeOne(baseAddr+status.SlotConsecutiveFailMax, unitID, s.ConsecutiveFailMax); err != nil {
			errs = append(errs, err.Error())
		} else {
			sw.last.ConsecutiveFailMax = s.ConsecutiveFailMax
		}
	}

	if len(errs) > 0 {
		sw.needFull = true
		return errors.New(strings.Join(errs, " | "))
	}

	return nil
}

func (sw *deviceStatusWriter) baseAddr() uint16 {
	return sw.plan.BaseSlot * status.SlotsPerDevice
}

func (sw *deviceStatusWriter) fullBlockRegs(s status.Snapshot) []uint16 {
	regs := make([]uint16, status.SlotsPerDevice)

	regs[status.SlotHealthCode] = s.Health
	regs[status.SlotLastErrorCode] = s.LastErrorCode
	regs[status.SlotSecondsInError] = s.SecondsInError

	for i := 0; i < status.SlotDeviceNameSlots; i++ {
		dst := status.SlotDeviceNameStart + i
		if i < len(sw.nameRegs) {
			regs[dst] = sw.nameRegs[i]
		}
	}

	encodeUint32(regs, status.SlotRequestsTotalLow, s.RequestsTotal)
	encodeUint32(regs, status.SlotResponsesValidTotalLow, s.ResponsesValidTotal)
	encodeUint32(regs, status.SlotTimeoutsTotalLow, s.TimeoutsTotal)
	encodeUint32(regs, status.SlotTransportErrorsTotalLow, s.TransportErrorsTotal)

	regs[status.SlotConsecutiveFailCurr] = s.ConsecutiveFailCurr
	regs[status.SlotConsecutiveFailMax] = s.ConsecutiveFailMax

	return regs
}

func encodeUint32(regs []uint16, start int, v uint32) {
	regs[start] = uint16(v & 0xFFFF)
	regs[start+1] = uint16((v >> 16) & 0xFFFF)
}

func (sw *deviceStatusWriter) writeUint32(
	errs *[]string,
	addr uint16,
	unitID uint8,
	prev uint32,
	curr uint32,
	update func(uint32),
) {
	if prev == curr {
		return
	}

	lo := uint16(curr & 0xFFFF)
	hi := uint16((curr >> 16) & 0xFFFF)

	if err := sw.cli.WriteRegisters(statusAreaHoldingRegisters, unitID, addr, []uint16{lo, hi}); err != nil {
		*errs = append(*errs, err.Error())
		return
	}

	update(curr)
}

func (sw *deviceStatusWriter) writeOne(addr uint16, unitID uint8, v uint16) error {
	return sw.cli.WriteRegisters(statusAreaHoldingRegisters, unitID, addr, []uint16{v})
}

func encodeDeviceNameRegs(name string) []uint16 {
	out := make([]uint16, status.SlotDeviceNameSlots)

	b := []byte(name)
	if len(b) > status.DeviceNameMaxChars {
		b = b[:status.DeviceNameMaxChars]
	}

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