# Modbus Replicator — Configuration Model

Version Note: 2026-05-26 (Added `addinvert` read-block documentation)

LEGACY NOTICE: This document previously defined a global `replicator.Status_Memory` model. That topology is removed from implementation and is retained here only as historical context. The normative model below reflects current code.

## Purpose

Configuration defines **replication topology at startup**.

It answers only:

* What devices exist
* Where they are read from
* Where data is written
* Where (optionally) status is written

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
        - fc: 1
          address: 0
          quantity: 32
          invert: true
          addinvert: true
        - fc: 3
          address: 0
          quantity: 50
      targets:
        - id: 1
          endpoint: "10.5.1.20:501"
          unit_id: 2
          status_unit_id: 35
          memories:
            - memory_id: 1
              offsets: {}
      poll:
        interval_ms: 1000
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
* `timeout_ms` (`int`) — applies to both source Modbus reads and Raw Ingest writes to all targets
* `device_name` (`string`, optional, ASCII-only validation)
* `status_slot` (`*uint16`, optional, opt-in status)

If `status_slot` is omitted, no status writers are built for that unit.

---

## Reads

```yaml
reads:
  - fc: 1
    address: 0
    quantity: 32
    invert: true   # optional; only meaningful for FC1 and FC2
    addinvert: true # optional; append inverted copy after the original block

  - fc: 3
    address: 0
    quantity: 50
    # invert/addinvert silently ignored for FC3/FC4 — no overhead
```

Read definitions per poll cycle.

Fields:

* `fc` (`uint8`) — Modbus function code: 1 (Coils), 2 (Discrete Inputs), 3 (Holding Registers), 4 (Input Registers)
* `address` (`uint16`) — starting register/coil address
* `quantity` (`uint16`) — number of coils or registers to read
* `invert` (`bool`, optional, default `false`) — flip every bit after reading; **only applies to FC1 and FC2**; silently ignored for FC3 and FC4 with zero runtime overhead
* `addinvert` (`bool`, optional, default `false`) — for **FC1 and FC2 only**, write the original block first and then append an inverted copy immediately after it in MMA; ignored for FC3 and FC4 with zero runtime overhead

When `addinvert: true`:

* Original block occupies `base_offset + address` → `base_offset + address + quantity - 1`
* Appended inverted block occupies `base_offset + address + quantity` → `base_offset + address + (2 * quantity) - 1`
* `invert` and `addinvert` remain independent:
  * `invert` transforms the primary FC1/FC2 view
  * `addinvert` duplicates and appends the inverse of that primary view

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

## Poll

```yaml
poll:
  interval_ms: 1000
```

Fixed cadence for poll execution.

---

## Validation Rules (Implemented)

When `source.status_slot` is set:

* Unit must have at least one target.
* Every target must define `status_unit_id`.
* Collision is rejected for duplicate `(endpoint, status_unit_id, status_slot)` across units.

Additional implemented checks:

* `source.device_name` must be ASCII-only.
* Destination memory overlap is rejected per `(endpoint, memory_id, fc)` range.
* For FC1/FC2 reads with `addinvert: true`, overlap detection uses the expanded `2 * quantity` footprint.

---

## Legacy Model (Removed)

Removed from implementation:

* `replicator.Status_Memory`
* Global shared status endpoint topology

Any configuration using `replicator.Status_Memory` is not part of current code contract.
