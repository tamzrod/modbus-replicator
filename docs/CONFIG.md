# Modbus Replicator — Configuration Model

## Purpose

Configuration defines **replication topology at startup**.

It answers only:

* What devices exist
* Where they are read from
* Where data is written
* Where (optionally) status is written

> **Configuration does not define behavior. It defines wiring.**

The replicator is configured **once** at process start.
Any change requires a restart.

---

## Core Principles (Still Non‑Negotiable)

* No runtime mutation
* No semantics
* No control logic
* No routing decisions
* No retries

Configuration describes **topology only**.

---

## High‑Level Structure

```yaml
replicator:
  Status_Memory:
    endpoint: "10.5.1.20:501"
    # unit_id is supplied per unit via status_slot

  units:
    - id: "unit-1"
      source:
        endpoint: "10.5.1.101:502"
        unit_id: 1
        device_name: "SCB01"
        status_slot: 1
        timeout_ms: 2000

      reads:
        - fc: 3
          address: 0
          quantity: 50

      targets:
        - id: 1
          endpoint: "10.5.1.20:501"
          memories:
            - offsets: {}

      poll:
        interval_ms: 1000
```

---

## Replicator Root

```yaml
replicator:
```

Top‑level container for all replication wiring.

---

## Status Memory (Optional but Global)

```yaml
Status_Memory:
  endpoint: "10.5.1.20:501"
```

Defines **where device status blocks are written**.

Important rules:

* Status memory is **shared** across all units
* Unit ID for status is **not global**
* Each device selects its status block via `status_slot`

If `Status_Memory` is omitted:

* No status writers are created
* Status logic is completely disabled

---

## Units

Each entry under `units` defines **one replication pipeline**.

Units are:

* Independent
* One‑way
* Deterministic

There is **no fan‑in** and no cross‑unit interaction.

```yaml
units:
  - id: "unit-1"
```

The `id` is:

* Human‑readable
* Used only for logging and diagnostics
* Never written to memory

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

Defines the **truth device**.

Fields:

* `endpoint` — Modbus TCP endpoint of the source device
* `unit_id` — Modbus Unit ID
* `timeout_ms` — Transport timeout
* `device_name` — Optional, written verbatim into status memory
* `status_slot` — Optional; enables status reporting

If `status_slot` is omitted:

* Device has **no status block**
* No status writes occur

---

## Reads (Geometry Only)

```yaml
reads:
  - fc: 3
    address: 0
    quantity: 50
```

Defines **what is read** from the source device.

Rules:

* Geometry only
* No scaling
* No interpretation
* No transformation

Reads are executed sequentially per poll cycle.

---

## Targets

```yaml
targets:
  - id: 1
    endpoint: "10.5.1.20:501"
    memories:
      - offsets: {}
```

Defines **where data is written**.

Rules:

* Targets are write‑only
* Writes occur **only when poll succeeds**
* One poll result may fan‑out to multiple targets

### Memory Mapping

```yaml
offsets: {}
```

An empty map means:

* Identity mapping
* Register addresses preserved

Offsets exist only to shift address space, not to transform values.

---

## Poll

```yaml
poll:
  interval_ms: 1000
```

Defines poll cadence.

Rules:

* Fixed interval
* No jitter logic
* No adaptive timing

Timing behavior belongs to the runner, not the config.

---

## What Configuration Does *Not* Do

Configuration does **not**:

* Define health semantics
* Encode error meaning
* Specify retry logic
* Describe failover
* Contain business rules

Those belong to **runtime state layers**.

---

## Summary

Configuration is intentionally boring.

It describes:

* Devices
* Wiring
* Address space

Nothing more.
Nothing less.

That restraint is what makes large‑scale replication possible without human error.
