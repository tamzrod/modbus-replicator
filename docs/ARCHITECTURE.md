# Modbus Replicator – Architecture

Version Note: 2026-04-13 (Documentation audit; connection lifecycle added to Poller section)

## Overview

The Modbus Replicator is a **read–fan-out–write** system that reads source devices and delivers snapshots to MMA memory.

Core rule:

> **The source device is the truth. The writer is the delivery mechanism. Memory is the contract.**

Implementation does not add semantic interpretation of source telemetry and does not perform hidden retries.

---

## High-Level Flow

```
[ Modbus Devices ]
        ↓
      Poller
        ↓
   PollResult
        ↓
      Writer
        ↓
[ Modbus Memory (MMA) ]
        ↓
   SCADA / Clients
```

When status is enabled, the same runtime also maintains a per-unit status snapshot and writes it through per-target status writers.

---

## Component Responsibilities

### 1. Poller

**Responsibility:** read device blocks and expose transport counters.

Poller behavior as implemented:

* Executes configured FC1/FC2/FC3/FC4 reads in sequence per poll cycle.
* Returns success only when all read blocks succeed in that cycle.
* On error, returns immediately with `PollResult.Err`.
* Performs no retry loop inside `PollOnce()`.
* Maintains lifetime transport counters (`requests_total`, `responses_valid_total`, `timeouts_total`, `transport_errors_total`, `consecutive_fail_current`, `consecutive_fail_max`).

Connection lifecycle as implemented:

* No dial occurs at startup; the initial client is nil.
* On each poll cycle where the client is nil, one connection attempt is made via the factory.
* If the factory fails, the failure is recorded as a poll failure and the cycle returns immediately without executing reads.
* On "dead connection" errors (EOF, broken pipe, connection reset, connection aborted, use of closed network connection), the client is discarded and set to nil so the next poll tick may reconnect.
* On timeout errors, the client is **not** discarded and is reused on the next poll tick.

### 2. Writer

**Responsibility:** deliver data and status packets to configured targets.

Writer behavior as implemented:

* Data writes execute only when `PollResult.Err == nil`.
* Status writes are independent of data success/failure.
* Status destination is **per target** (`target.endpoint`, `target.status_unit_id`) when `source.status_slot` is configured.

### 3. Status Snapshot Orchestration

Runtime status snapshot is owned by the per-unit orchestrator loop.

Implemented transitions:

* On poll success: `Health=OK`, `LastErrorCode=0`, `SecondsInError=0`.
* On poll failure: `Health=ERROR`, `LastErrorCode=errorCode(PollResult.Err)`.
* Every second while `Health != OK`: increment `SecondsInError` by 1 up to 65535.
* On each poll result: inject latest transport counters from the poller into status snapshot.

---

## Status Data Model (Implemented)

Each status-enabled unit owns exactly **30 slots** per configured `status_slot` base.

* Slots 0–2: operational truth (`health_code`, `last_error_code`, `seconds_in_error`)
* Slots 3–10: `device_name` (8 registers / 16 ASCII chars max)
* Slots 11–19: reserved
* Slots 20–29: transport lifetime counters

Health constants defined in code:

* `0` Unknown
* `1` OK
* `2` Error
* `3` Stale
* `4` Disabled

Current emission behavior:

* Runtime assigns `HealthOK (1)` and `HealthError (2)` during poll operation.
* `HealthUnknown (0)` is used for initial snapshot before first status write.
* `HealthStale (3)` and `HealthDisabled (4)` are defined constants but are not assigned by current runtime flow.

---

## Raw Ingest Protocol

Raw Ingest packet format is fixed in implementation (`internal/writer/ingest/client.go`):

* Magic bytes `RI` (`0x52`, `0x49`)
* Version `0x01`
* Header size `10` bytes
* Big-endian register payload encoding
* One TCP connection per packet write

---

## Error Ownership Model

| Layer  | Owns Errors About                      |
| ------ | -------------------------------------- |
| Poller | Device reachability, Modbus exceptions |
| Writer | Delivery failures, protocol write path |
| MMA    | Memory integrity and destination bounds |

---

## Design Principles (Implemented)

* **Status is data**
* **Memory is the contract**
* **No hidden retries in poll cycle**
* **No background probes in runtime**
* **Per-target status destinations when status is enabled**

---

## Summary

The current implementation is deterministic and explicit:

* All externally observable status behavior is carried through the same write channel.
* Status memory topology is per target.
* 30-slot status model is implemented and active.
