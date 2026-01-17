# Modbus Replicator

A deterministic Modbus TCP **read → replicate → write** engine designed to safely mirror data from one or more Modbus sources into one or more Modbus memory targets (e.g. MMA), with **geometry-only configuration**, strict validation, and test-backed correctness.

---

## Core Principles

* **Dumb core, smart edges**

  * No scaling, parsing, or semantics
  * Raw Modbus values only
* **Deterministic geometry**

  * Explicit addresses, quantities, offsets
  * No implicit behavior
* **Fail fast**

  * Invalid config fails at startup
  * Overlapping memory ranges are rejected
* **Test-backed**

  * Poller, writer, and validation are covered by unit tests

---

## Architecture Overview

```
        ┌──────────────┐
        │  Modbus TCP  │   (real device or MMA)
        │   Source(s)  │
        └──────┬───────┘
               │  FC read (1/2/3/4)
               ▼
        ┌──────────────┐
        │    Poller    │
        │ (per unit)   │
        └──────┬───────┘
               │  all-or-nothing snapshot
               ▼
        ┌──────────────┐
        │    Writer    │
        │ (per unit)   │
        └──────┬───────┘
               │  FC write w/ offsets
               ▼
        ┌──────────────┐
        │  Modbus TCP  │   (typically MMA)
        │   Target(s)  │
        └──────────────┘
```

### Key Properties

* One **poller goroutine per unit**
* One **writer per unit**
* Poller emits complete snapshots only
* Writer never writes partial data
* Targets are isolated by:

  * endpoint
  * memory_id
  * function code
  * offset

---

## Configuration Model

### High-level Structure

```yaml
replicator:
  units:
    - id: "unit-1"
      source: {...}
      reads: [...]
      targets: [...]
      poll: {...}
```

Each **unit** is an independent replication pipeline.

---

### Source (Modbus Read)

```yaml
source:
  endpoint: "127.0.0.1:503"   # host:port (NO tcp:// prefix)
  unit_id: 1                  # Modbus Unit ID
  timeout_ms: 1000
```

---

### Reads

Defines what to read from the source.

```yaml
reads:
  - fc: 3
    address: 0
    quantity: 20
```

Supported FCs:

* `1` – coils
* `2` – discrete inputs
* `3` – holding registers
* `4` – input registers

---

### Targets (Modbus Write)

A unit may have **multiple targets**.

```yaml
targets:
  - id: 1
    endpoint: "127.0.0.1:503"
    memories:
      - memory_id: 1
        offsets:
          1: 0
          2: 0
          3: 200
          4: 0
```

#### Important Rules

* `id` **must be numeric** (matches packet design)
* `memory_id` maps directly to **Modbus Unit ID** on the target
* `offsets` are **per function code**
* Missing offset ⇒ default `0`

---

### Poll

```yaml
poll:
  interval_ms: 1000
```

---

## Stacking Multiple Sources into One Memory

Multiple units may write into the **same memory_id** safely by using offsets.

Example:

* Unit A → holding 0–99 → memory 1 offset 0
* Unit B → holding 0–99 → memory 1 offset 100

Validation will **reject** any overlapping destination ranges.

---

## Validation (Startup Safety)

At startup, the replicator validates:

* No overlapping destination ranges
* Overlap is detected **only if all match**:

  * endpoint
  * memory_id
  * function code
  * overlapping address ranges

Touching ranges (e.g. `0–9` and `10–19`) are allowed.

If validation fails → **process exits immediately**.

---

## How to Run

### 1. Start MMA (example)

```text
Modbus TCP :503
Raw Ingest :9000
REST       :8080
```

Ensure the memory you target is enabled in MMA.

---

### 2. Run Replicator

```powershell
go run ./cmd/replicator ./config.yaml
```

---

### 3. Verify

Using modpoll:

```bash
modpoll -m tcp -p 503 -t 4 -r 1   -c 10 127.0.0.1   # source
modpoll -m tcp -p 503 -t 4 -r 201 -c 10 127.0.0.1   # destination
```

Values should match according to configured offsets.

---

## What This Is NOT

* ❌ Not a PLC
* ❌ Not a SCADA
* ❌ No scaling or data typing
* ❌ No retries or buffering
* ❌ No semantics

Those belong **outside** this system.

---

## Project Status

* Config model: **LOCKED**
* Validation: **LOCKED**
* Poller: **LOCKED**
* Writer: **LOCKED**
* Main wiring: **LOCKED**
* Runtime tests: **PASSED (real MMA loopback)**

---

## Next Extensions (Optional)

* `--validate-only` mode
* Structured logging
* Metrics export
* Release tagging

---

**If semantics appear, the configuration model is wrong.**
