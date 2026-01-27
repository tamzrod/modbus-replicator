# Status Rules (Executable Specification)

This document defines the STATUS feature behavior in strict, executable terms.
Implementation MUST follow this order and these rules exactly.

---

## 1. RAW ERROR CODE (SOURCE OF TRUTH)

### Definition
- Raw Error Code is copied directly from the Modbus device result.
- The writer NEVER interprets, maps, or invents error codes.
- Value is opaque and device-defined.

### Rules
- On successful Modbus read:
  - Raw Error Code = 0
- On Modbus read failure:
  - Raw Error Code = device-returned error code
- Writer must only COPY this value.
- Poller is the ONLY source of this value.

---

## 2. SECONDS IN ERROR

### Definition
- Seconds-in-Error is a saturating uint16 counter.
- It measures continuous time spent in error state.

### Rules
- Increment by 1 every second while Raw Error Code != 0
- Reset to 0 immediately on first Raw Error Code == 0
- Saturate at max(uint16)
- No history, no averaging, no smoothing

---

## 3. HEALTH (DERIVED STATE)

### Definition
- Health is a derived classification based on error state.
- Health has NO independent state.

### Rules
- If Raw Error Code == 0:
  - Health = OK
- If Raw Error Code != 0:
  - Health = ERROR
- Health MUST NOT be set directly
- Health MUST NOT be inferred elsewhere

---

## 4. DEVICE IDENTITY (DEVICE NAME)

### Definition
- Device Name represents identity, not telemetry.

### Rules
- Written ONLY:
  - On startup
  - On reconnect
  - On MMA trust loss
- NEVER written during normal cycles
- Silence means identity unchanged and trusted
- Stored at the END of the status block

---

## 5. SILENCE RULES

### Definition
- Silence is a valid and meaningful state.

### Rules
- No change → no write
- Fastest update is no update (RBE++)
- Status writer must not emit unchanged data

---

## NON-RULES (EXPLICITLY FORBIDDEN)

- No retries
- No error classification
- No health heuristics
- No inferred device state
- No synthetic error codes
