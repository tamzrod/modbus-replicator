// internal/status/constants.go
package status

// Device Status Block layout constants.
// These values define the protocol and MUST NOT be configurable.

// ---- BLOCK GEOMETRY ----

// SlotsPerDevice is the fixed number of logical slots per device.
const SlotsPerDevice = 20

// ---- SLOT INDICES ----

// SlotHealthCode holds the device health state.
const SlotHealthCode = 0

// SlotLastErrorCode holds the last raw error code.
const SlotLastErrorCode = 1

// SlotSecondsInError holds the duration (in seconds) the device has been in error.
const SlotSecondsInError = 2

// ---- RESERVED RANGE ----

// Slots 3–10 are reserved for future use.
const SlotReservedStart = 3
const SlotReservedEnd   = 10

// ---- DEVICE NAME ----

// SlotDeviceNameStart is the first slot used for the device name.
// Device name is always placed at the END of the status block.
const SlotDeviceNameStart = 11

// SlotDeviceNameSlots is the number of slots reserved for the device name.
const SlotDeviceNameSlots = 8

// SlotDeviceNameEnd is the last slot used for the device name (inclusive).
const SlotDeviceNameEnd = SlotDeviceNameStart + SlotDeviceNameSlots - 1

// ---- LIMITS ----

// DeviceNameMaxChars is the maximum number of ASCII characters stored for device name.
const DeviceNameMaxChars = 16

// ---- HEALTH CODES ----

// HealthUnknown represents an unknown or boot state.
const HealthUnknown uint16 = 0

// HealthOK represents a healthy device.
const HealthOK uint16 = 1

// HealthError represents a device error state.
const HealthError uint16 = 2

// HealthStale represents a stale data state.
const HealthStale uint16 = 3

// HealthDisabled represents a disabled device state.
const HealthDisabled uint16 = 4
