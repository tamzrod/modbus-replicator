# Device Status Block --- Layout Specification

Version Note: 2026-03-03 (Stage 4 documentation rectification; synchronized to implemented behavior)

Authority Note: This document defines the authoritative specification for externally observable behavior.

Status: LOCKED\
Scope: Modbus Replicator + MMA\
This document defines the canonical device status block layout.\
If this document changes, the memory contract changes.

------------------------------------------------------------------------

# 1. Overview

Each status-enabled unit owns exactly **30 logical slots**.

Slots are logical device fields.\
Slots are not Modbus registers.\
Registers are a storage implementation detail only.

The block is divided into:

-   Slots 0--19 → Operational Truth\
-   Slots 20--29 → Transport Lifetime Counters

------------------------------------------------------------------------

# 2. Slots 0--19 --- Operational Truth

Slot 0 → health_code\
Slot 1 → last_error_code\
Slot 2 → seconds_in_error

Slot 3--10 → device_name (ASCII, max 16 chars)\
Slot 11--19 → RESERVED

These slots represent device-level operational condition only.

------------------------------------------------------------------------

## Slot 0 --- health_code

0 → UNKNOWN / BOOT\
1 → OK\
2 → ERROR\
3 → STALE\
4 → DISABLED

Emission behavior in current runtime:

-   UNKNOWN appears on initial status snapshot assertion\
-   OK and ERROR are assigned during poll processing\
-   STALE and DISABLED are defined values but not currently assigned by runtime flow

------------------------------------------------------------------------

## Slot 1 --- last_error_code

-   Raw pass-through error code\
-   Written exactly as returned by the device or library\
-   0 means OK\
-   No parsing\
-   No remapping\
-   No semantics

------------------------------------------------------------------------

## Slot 2 --- seconds_in_error

-   Type: uint16\
-   Tick: 1 Hz\
-   Increments while health_code != OK\
-   Saturates at 65535\
-   Never wraps\
-   Resets to 0 on recovery

------------------------------------------------------------------------

## Slots 3--10 --- device_name

-   Optional\
-   ASCII only\
-   Maximum 16 characters\
-   Written from configuration\
-   Never used for logic

------------------------------------------------------------------------

# 3. Slots 20--29 --- Transport Lifetime Counters

Transport counters are lifetime, monotonic, integer-only values.

They represent communication behavior.\
They do not represent device semantics.

They must never influence:

-   health_code\
-   retry logic\
-   timeout thresholds\
-   scheduling\
-   control flow

------------------------------------------------------------------------

## Counter Layout

Slot 20--21 → requests_total (uint32)\
Slot 22--23 → responses_valid_total (uint32)\
Slot 24--25 → timeouts_total (uint32)\
Slot 26--27 → transport_errors_total (uint32)\
Slot 28 → consecutive_fail_current (uint16)\
Slot 29 → consecutive_fail_max (uint16)

------------------------------------------------------------------------

## Encoding Rules

-   uint32 values occupy two consecutive slots\
-   Lower slot = low 16 bits\
-   Upper slot = high 16 bits\
-   All values are unsigned\
-   All counters are monotonic\
-   Counters may wrap naturally\
-   No overflow handling logic is required

------------------------------------------------------------------------

# 4. Write Strategy

Full block write (Slots 0--29) occurs when `needFull` is true:

-   On replicator startup\
-   After status write failure (re-assert path)

Incremental updates:

-   Slot 0 → on health change\
-   Slot 1 → on error change\
-   Slot 2 → once per second while in error\
-   Slots 20--29 → updated when their values change

Device name (Slots 3--10) is written in full-block path only.

The full block must not be rewritten continuously.

------------------------------------------------------------------------

# 5. Non-Goals

The status block must not contain:

-   floating-point values\
-   success percentages\
-   rolling windows\
-   analytics\
-   trends\
-   aggregates\
-   history\
-   per-tag quality

Derived metrics must be computed externally.

Example:

success_rate = responses_valid_total / requests_total

This computation does not belong inside the status block.

------------------------------------------------------------------------

# 6. Final Principle

Status is device-level truth.\
Transport counters are passive lifetime instrumentation.\
Data remains untouched.\
Status grants permission to believe it.
