# Device Status Block — Directives (AUTHORITATIVE)

> **Status:** LOCKED  
> **Scope:** Modbus Replicator + MMA  
> **This document overrides all previous status drafts, todos, and notes.**

---

## 1. Opt-In by Design

Device participation in the Device Status Block is **optional**.

- A unit does **not** participate by default
- A unit participates **only if** it explicitly provides a `status_slot`
- No defaults
- No auto-allocation
- No implicit enablement

If `status_slot` is absent, the unit is **status-disabled**.

---

## 2. Slot Ownership

Each status-enabled unit **owns exactly one status slot**.

- Ownership is declared via `status_slot`
- The slot identifies the base of a fixed **20-slot block**
- Slot ownership is exclusive

Any slot collision is a **hard configuration error**.

---

## 3. Status Memory Requirement

`Status_Memory` is **conditionally required**.

- If **no units** enable status → `Status_Memory` may be absent
- If **any unit** enables status → `Status_Memory.endpoint` **must exist**

There is no global requirement unless status is used.

---

## 4. Validation Rules

Validation **checks only**. It does not correct or infer.

Validation must:
- detect slot collisions
- detect missing `Status_Memory` when required
- detect invalid `device_name` characters

Validation must not:
- mutate configuration
- invent values
- normalize or truncate
- enable status implicitly

---

## 5. Device Status Block Layout

Each status-enabled unit owns **exactly 20 slots**.

```
Slot 0  → health_code
Slot 1  → last_error_code
Slot 2  → seconds_in_error

Slot 3–10 → device_name (ASCII, max 16 chars)
Slot 11–19 → RESERVED
```

Slots are **logical device slots**, not Modbus registers.

---

## 6. Slot Semantics

### Slot 0 — health_code

| Value | Meaning |
|-----:|---------|
| 0 | UNKNOWN / BOOT |
| 1 | OK |
| 2 | ERROR |
| 3 | STALE |
| 4 | DISABLED |

---

### Slot 1 — last_error_code

- Raw pass-through error code
- Written exactly as returned by the device or library
- `0` means OK
- No parsing
- No remapping
- No semantics

---

### Slot 2 — seconds_in_error

- Type: `uint16`
- Tick: 1 Hz
- Increments while `health_code != OK`
- Saturates at `65535`
- Never wraps
- Resets to `0` on recovery

---

### Slots 3–10 — device_name

- Optional
- ASCII only
- Max 16 characters
- Written from config
- Never used for logic

---

## 7. Authority Model

- Replicator is authoritative
- MMA memory is volatile
- Identity flows one way only:

```
Config → Replicator → MMA
```

MMA content is never trusted as source of truth.

---

## 8. Write Strategy

### Full Block Write (Slots 0–19)

Performed **only** when:
- Replicator starts, or
- A previous write failed and the next write succeeds

This re-asserts identity after uncertainty.

---

### Incremental Writes

During normal operation:
- Slot 0 → on health change
- Slot 1 → on failure / clear on success
- Slot 2 → once per second while in error

The full block must **not** be written repeatedly.

---

## 9. Failure Model

- Writes normally succeed
- A write failure introduces doubt
- The next successful write re-asserts identity

There is:
- No restart detection
- No reads
- No probing
- No handshakes

Only successful writes are trusted.

---

## 10. Data Validity Model

> **MMA stores facts.  
> The Device Status Block grants permission to believe them.**

- MMA data is never mutated on failure
- No zeroing
- No sentinels
- No per-register quality flags

Consumers decide whether data may be trusted based on status.

---

## 11. Explicit Non-Goals

The status block must not contain:
- analytics
- lifetime counters
- aggregates
- trends
- history
- per-tag quality

Those belong to databases and monitoring systems.

---

## Final Rule (DO NOT EDIT)

> **Status is opt-in, slot-owned, device-level truth.  
> Data remains untouched; status grants permission to believe it.**

---

**End of Document**
