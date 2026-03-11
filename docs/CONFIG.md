# Modbus Replicator — Configuration Model

Version Note: 2026-03-11 (Per-read interval refactor; synchronized to implemented behavior)

LEGACY NOTICE: This document previously defined a global `replicator.Status_Memory` model. That topology is removed from implementation and is retained here only as historical context. The normative model below reflects current code.

## Purpose

Configuration defines **replication topology at startup**.

It answers only:

* What devices exist
* Where they are read from
* Where data is written
* Where (optionally) status is written
* At what cadence each read block runs

> **Configuration defines wiring. Runtime defines behavior.**

---

## Core Principles

* No runtime mutation of config
* No semantic interpretation rules in config
* No retry policy in config
* Topology only

---

## High-Level Structure (Implemented)

```yaml
replicator:
  units:
    - id: "unit-1"
      source:
        endpoint: "10.5.1.101:502"
        unit_id: 1
        timeout_ms: 2000
        device_name: "SCB01"
        status_slot: 1
      reads:
        - fc: 3
          address: 0
          quantity: 50
          interval_ms: 1000
      targets:
        - id: 1
          endpoint: "10.5.1.20:501"
          unit_id: 2
          status_unit_id: 35
          memories:
            - memory_id: 1
              offsets: {}
```

---

## Replicator Root

```yaml
replicator:
```

Top-level container for all unit pipelines.

---

## Units

Each `units[]` entry defines one pipeline.

```yaml
units:
  - id: "unit-1"
```

`id` is runtime identity for logging/diagnostics.

---

## Source

```yaml
source:
  endpoint: "10.5.1.101:502"
  unit_id: 1
  timeout_ms: 2000
  device_name: "SCB01"
  status_slot: 1
```

Fields:

* `endpoint` (`string`)
* `unit_id` (`uint8`)
* `timeout_ms` (`int`)
* `device_name` (`string`, optional, ASCII-only validation)
* `status_slot` (`*uint16`, optional, opt-in status)

If `status_slot` is omitted, no status writers are built for that unit.

---

## Reads

```yaml
reads:
  - fc: 3
    address: 0
    quantity: 50
    interval_ms: 1000
```

Each read block defines geometry and its own independent poll cadence.

Fields:

* `fc` (`uint8`) — Modbus function code: 1 (coils), 2 (discrete inputs), 3 (holding registers), 4 (input registers)
* `address` (`uint16`) — starting register or coil address
* `quantity` (`uint16`) — number of registers or coils to read
* `interval_ms` (`int`, **required**) — how often this block is polled, in milliseconds; must be > 0

Each read block runs at its own cadence. Different blocks in the same unit may use different intervals, for example:

```yaml
reads:
  - fc: 3
    address: 999
    quantity: 30
    interval_ms: 250      # fast: power telemetry

  - fc: 3
    address: 1099
    quantity: 16
    interval_ms: 5000     # slow: energy counters
```

Read blocks within a unit always execute sequentially. No two reads run concurrently.

Each read block produces one `PollResult` representing the outcome of that individual read.

---

## Targets

```yaml
targets:
  - id: 1
    endpoint: "10.5.1.20:501"
    unit_id: 2
    status_unit_id: 35
    memories:
      - memory_id: 1
        offsets: {}
```

Implemented fields:

* `id` (`uint32`)
* `endpoint` (`string`)
* `unit_id` (`uint8`) for data writes
* `status_unit_id` (`*uint8`) for status writes when source status is enabled
* `memories[]` with `memory_id` (`uint16`) and `offsets` (`map[int]uint16`)

### Per-target status destination

Status destination identity is `(target.endpoint, target.status_unit_id)`.

There is no global status memory object in current config schema.

---

## Validation Rules (Implemented)

### Read interval rules

* `reads[].interval_ms` must be > 0 for every read block (required).
* `poll.interval_ms` is **rejected** if present. The device-level poll block is not supported; set `interval_ms` on each read block instead.

### Status rules

When `source.status_slot` is set:

* Unit must have at least one target.
* Every target must define `status_unit_id`.
* Collision is rejected for duplicate `(endpoint, status_unit_id, status_slot)` across units.

### Additional checks

* `source.device_name` must be ASCII-only.
* Destination memory overlap is rejected per `(endpoint, memory_id, fc)` range.

---

## Legacy Model (Removed)

Removed from implementation:

* `replicator.Status_Memory`
* Global shared status endpoint topology
* Device-level `poll.interval_ms` block

Any configuration using `replicator.Status_Memory` or a top-level `poll:` block under a unit is not part of current code contract and will be rejected at validation.

