// internal/status/encode.go
package status

// Encode converts a Snapshot into a full device status block.
// Layout is protocol-locked.
// No IO. No side effects.
func Encode(s Snapshot) []uint16 {
	regs := make([]uint16, SlotsPerDevice)

	regs[SlotHealthCode] = s.Health
	regs[SlotLastErrorCode] = s.LastErrorCode
	regs[SlotSecondsInError] = s.SecondsInError

	return regs
}
