# internal/config

Configuration loading, validation, and normalization for Modbus Replicator.

## Package Responsibilities

* **`config.go`** — Type definitions for the configuration schema.
* **`load.go`** — YAML unmarshalling from a file path into `*Config`.
* **`validate.go`** — Declarative correctness checks. Must not mutate config.
* **`normalize.go`** — Post-validation normalization. Allowed to mutate config. Must be called only after `Validate()`.

---

## Schema Overview

```yaml
replicator:
  units:
    - id: "unit-1"
      source:
        endpoint: "10.5.1.101:502"
        unit_id: 1
        timeout_ms: 2000
        device_name: "SCB01"       # optional, ASCII only, max 16 chars
        status_slot: 0             # optional, opt-in status block
      reads:
        - fc: 3
          address: 0
          quantity: 50
          interval_ms: 1000        # required: per-read poll cadence (ms, must be > 0)
      targets:
        - id: 1
          endpoint: "10.5.1.20:501"
          unit_id: 2
          status_unit_id: 35       # required when source.status_slot is set
          memories:
            - memory_id: 1
              offsets: {}
```

---

## Per-Read Poll Interval

Each read block carries its own `interval_ms` field. This is the sole mechanism for controlling poll cadence.

* `interval_ms` is **required** on every read block and must be `> 0`.
* Different read blocks within the same unit may use different intervals.
* The device-level `poll.interval_ms` field is **not supported** and will be rejected at validation.

Example — fast power telemetry, slow energy counters:

```yaml
reads:
  - fc: 3
    address: 999
    quantity: 30
    interval_ms: 250

  - fc: 3
    address: 1099
    quantity: 16
    interval_ms: 5000
```

---

## Validation Rules

### Read interval
* `reads[].interval_ms` must be `> 0` (required for every read block).
* `poll.interval_ms` present on a unit → validation error.

### Status (opt-in)
* When `source.status_slot` is set:
  * Unit must have at least one target.
  * Every target must define `status_unit_id`.
  * Duplicate `(endpoint, status_unit_id, status_slot)` across units → collision error.

### Device name
* `source.device_name` must contain ASCII characters only.
* Truncated to 16 characters by `Normalize()`.

### Memory geometry
* Destination register ranges are checked for overlap per `(endpoint, memory_id, fc)`.
* Touching ranges are allowed; overlapping ranges are rejected.
