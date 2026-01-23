# Device Status — TODO List (Locked Scope)

> **Purpose**  
> Expose the *minimum truthful device state* via **Input Registers** using **fixed slots**.  
> No metadata. No semantics. No frontend thinking.

---

## 1. Slot Model (FOUNDATION)

- [ ] One slot = one device
- [ ] Slot index is **0-based**
- [ ] Slot size is **16 input registers** (fixed)
- [ ] Slot index configured in YAML (integer)
- [ ] Reject dynamic slot sizing
- [ ] Reject per-device custom layouts

---

## 2. Slot Layout (FINAL)

**Registers used:** Input Registers (FC4) only  
**Writers:** Replicator only  
**Readers:** PLC / SCADA / UI / Analytics

| Offset | Field | Description |
|------:|------|-------------|
| +0 | modbus_reply_code | Raw Modbus reply / exception code (`0 = OK`, non-zero = failure) |
| +1 | error_seconds_lo | Error duration (LOW word) |
| +2 | error_seconds_hi | Error duration (HIGH word) |
| +3 | online_flag | `1 = online`, `0 = offline` |
| +4…+15 | device_name | ASCII device name (max **24 chars**) |

Rules:
- Offsets are **locked**
- No future expansion inside the slot
- No reordering

---

## 3. Device Name Rules

- Source: `unit.id`
- ASCII only
- Max **24 characters**
- Excess characters **silently truncated**
- Null-padded
- No UTF-8
- No validation errors

---

## 4. YAML Configuration

- Add slot index to each unit
- Recommended key: `slot`
- Type: integer
- Base: 0

Example:

```yaml
replicator:
  units:
    - id: "SCB01MVPS01"
      slot: 0
```

Rules:
- YAML **must not** contain Modbus register addresses for status
- Register base is derived internally:

```
slot_base = slot × 16
```

---

## 5. Reply Code Handling

- Use **raw Modbus reply / exception code**
- Success = `0`
- Timeout / transport failure → single fixed code (e.g. `255`)
- No classification
- No enums
- No mapping tables

---

## 6. Error Duration Counter

- Maintain `error_seconds` as uint32 (hi / lo)
- Increment while `modbus_reply_code != 0`
- Reset to `0` when `modbus_reply_code == 0`
- No wall clock
- No timestamps
- No drift correction

Decode rule:

```
error_seconds = (error_seconds_hi << 16) | error_seconds_lo
```

---

## 7. Online Flag Logic

- `online_flag = 1` when `modbus_reply_code == 0`
- `online_flag = 0` when `modbus_reply_code != 0`
- No hysteresis
- No debounce
- No retry semantics

---

## 8. Write Discipline

- Replicator is the **only writer**
- One write per poll cycle
- Atomic write per slot
- No partial updates
- No clearing memory on silence

---

## 9. Slot Collision Rules

- Detect duplicate slot indices at startup
- Fail fast on collision
- No auto-reassignment
- No silent overrides

---

## 10. Explicit Non-Goals (DO NOT ADD)

The following are **explicitly forbidden**:

- Tags
- Vendor / model metadata
- Error types
- Status enums
- Retry counters
- Health scores
- Flapping detection
- Alarms
- Frontend hints

All of the above belong **outside** the replicator.

---

## Final Locked Statement

> **Device status is a flat, fixed, slot-indexed truth table.**  
> **It reports what happened, not what it means.**  
> **Garbage stays at the frontend.**

