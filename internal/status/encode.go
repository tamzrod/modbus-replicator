// internal/status/encode.go
package status

// Encode converts a Snapshot into a full device status block.
// Layout is protocol-locked.
// No IO. No side effects.
func Encode(s Snapshot) []uint16 {
	regs := make([]uint16, SlotsPerDevice)

	// --- Slots 0–2 : Operational Truth ---
	regs[SlotHealthCode] = s.Health
	regs[SlotLastErrorCode] = s.LastErrorCode
	regs[SlotSecondsInError] = s.SecondsInError

	// --- Slots 20–29 : Transport Lifetime Counters ---

	// uint32 → two uint16 (low first, then high)

	// requests_total
	regs[SlotRequestsTotalLow] = uint16(s.RequestsTotal & 0xFFFF)
	regs[SlotRequestsTotalHigh] = uint16((s.RequestsTotal >> 16) & 0xFFFF)

	// responses_valid_total
	regs[SlotResponsesValidTotalLow] = uint16(s.ResponsesValidTotal & 0xFFFF)
	regs[SlotResponsesValidTotalHigh] = uint16((s.ResponsesValidTotal >> 16) & 0xFFFF)

	// timeouts_total
	regs[SlotTimeoutsTotalLow] = uint16(s.TimeoutsTotal & 0xFFFF)
	regs[SlotTimeoutsTotalHigh] = uint16((s.TimeoutsTotal >> 16) & 0xFFFF)

	// transport_errors_total
	regs[SlotTransportErrorsTotalLow] = uint16(s.TransportErrorsTotal & 0xFFFF)
	regs[SlotTransportErrorsTotalHigh] = uint16((s.TransportErrorsTotal >> 16) & 0xFFFF)

	// consecutive_fail counters (uint16 direct)
	regs[SlotConsecutiveFailCurr] = s.ConsecutiveFailCurr
	regs[SlotConsecutiveFailMax] = s.ConsecutiveFailMax

	return regs
}