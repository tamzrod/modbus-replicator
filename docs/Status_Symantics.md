# Device Status Semantics

Version Note: 2026-03-03 (Stage 4 documentation rectification; synchronized to implemented behavior)

This document defines implemented status semantics and externally observable status behavior.

---

## Core Rule

> **Status reflects the latest poll outcome plus runtime-maintained status counters.**

Status includes both immediate poll truth (health/error code) and continuous runtime state (`seconds_in_error`, transport counters).

---

## Status Is Data

Status is written through the same write path as data:

* Writer sends status as Raw Ingest writes.
* MMA stores status in register memory.
* Consumers read status as standard Modbus memory.

No separate probe channel is used.

---

## Health Code

Health constants defined in status model:

| Value | Symbol     | Meaning |
| ----: | ---------- | ------- |
| 0     | UNKNOWN    | Initial/unknown state constant |
| 1     | OK         | Most recent poll succeeded |
| 2     | ERROR      | Most recent poll failed |
| 3     | STALE      | Defined constant (not emitted by current runtime flow) |
| 4     | DISABLED   | Defined constant (not emitted by current runtime flow) |

Current runtime assignment behavior:

* Poll success sets `OK`.
* Poll failure sets `ERROR`.
* Initial snapshot starts as `UNKNOWN` before first write.

---

## Poll Failure Semantics

A poll cycle is failed when `PollResult.Err != nil`.

This includes Modbus exceptions, transport failures, protocol-level failures, and other poller errors.

---

## Error Code Semantics

`last_error_code` behavior:

* `0` on successful poll
* On failed poll, code is extracted from error interfaces in this order:
	1. `Code() uint16`
	2. `ErrorCode() uint16`
	3. `ModbusCode() uint16`
* Fallback value `1` when no supported error-code interface is present

No string mapping is performed.

---

## Time Accumulation Semantics

`seconds_in_error` is a runtime-maintained saturating `uint16` counter.

Implemented rules:

* Every second while `Health != OK`, increment by `1`.
* On poll success, reset to `0`.
* Clamp at `65535` (no wrap).

---

## Transport Counter Semantics

Status includes transport lifetime counters in slots 20–29.

Source of values:

* Poller mutates counters per poll cycle.
* Orchestrator injects latest poller counters into status snapshot on each poll result.
* Status writer emits changed counter fields.

These counters are instrumentation and do not change poll logic.

---

## Non-Goals

Implemented non-goals:

* No retry loop inside poll cycle
* No vendor-specific semantic decoding
* No hidden quality scoring
* No synthetic status derived from side channels

---

## Summary

Status combines:

* poll result truth,
* raw error code extraction contract,
* 1 Hz error-duration accumulation,
* transport lifetime counters.

This is the implemented behavior exposed to external consumers.
