# Status Rules (Executable Specification)

Version Note: 2026-03-03 (Stage 4 documentation rectification; synchronized to implemented behavior)

This document defines implemented STATUS behavior in executable terms.

---

## 1. RAW ERROR CODE

### Definition

`last_error_code` is derived from poll failure error value using runtime error-code extraction.

### Rules

* On successful poll: `last_error_code = 0`
* On failed poll: extract in this order:
  1. `Code() uint16`
  2. `ErrorCode() uint16`
  3. `ModbusCode() uint16`
* If none match: `last_error_code = 1`

---

## 2. SECONDS IN ERROR

### Definition

`seconds_in_error` is a saturating `uint16` duration counter.

### Rules

* Tick interval: 1 second.
* While `health != OK`, increment by 1 each tick.
* On poll success, reset to 0.
* Saturate at `65535`.

---

## 3. HEALTH STATE

### Definition

Health values are encoded as numeric enum constants.

### Rules

* Constants defined: `UNKNOWN=0`, `OK=1`, `ERROR=2`, `STALE=3`, `DISABLED=4`.
* Runtime poll path assigns:
  * `OK` on poll success
  * `ERROR` on poll failure
* `UNKNOWN` is initial snapshot state before first status write.
* `STALE` and `DISABLED` are currently not assigned by runtime poll/orchestrator flow.

---

## 4. DEVICE IDENTITY (DEVICE NAME)

### Definition

`device_name` is identity data encoded into status slots 3–10.

### Rules

* Source value: `source.device_name`.
* Encoding: ASCII, non-printable replaced with `?`, max 16 chars, packed into 8 registers.
* Written in full-block writes only.
* Not emitted in incremental update path.

---

## 5. WRITE STRATEGY

### Definition

Status writer uses two branches: full-block re-assert and incremental updates.

### Rules

* On writer init, `needFull=true` and first write is full block (slots 0–29).
* On incremental path, only changed fields are written:
  * slot 0 (`health_code`)
  * slot 1 (`last_error_code`)
  * slot 2 (`seconds_in_error`)
  * slots 20–29 transport counters (as changed)
* If any incremental write fails, set `needFull=true` and return error.
* Next successful call with `needFull=true` reasserts full block.

---

## 6. SILENCE RULES

### Rules

* If no tracked field changes, no incremental write is emitted.
* Status silence is valid runtime behavior.
* Device name updates are silent until a full-block write condition occurs.

---

## NON-RULES (EXPLICITLY FORBIDDEN)

* No poll retry loop inside status logic
* No error semantic remapping
* No inferred device state beyond runtime assignments above
* No synthetic error codes beyond documented fallback value `1`
