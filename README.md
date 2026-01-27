# Modbus Replicator

A **deterministic Modbus read → fan‑out → write engine** designed to isolate unstable field devices from consumers by writing into a memory‑centric buffer (MMA – Modbus Memory Appliance).

This project is intentionally boring.

That is its strength.

---

## What This Is

Modbus Replicator:

* Reads Modbus devices (TCP today, RTU planned)
* Produces clean, bounded snapshots
* Writes results into Modbus Memory (MMA)
* Fans out to **one or many consumers** without multiplying device load

It does **not**:

* parse semantics
* scale values
* retry reads behind your back
* probe devices independently
* hide failures with metadata

---

## Why It Exists

Traditional SCADA stacks suffer from a common failure pattern:

> When devices become unstable, **clients crash first**.

This happens because:

* every client talks directly to devices
* timeouts cascade
* quality flags lie
* retry storms amplify failure

Modbus Replicator breaks this pattern by inserting a **memory contract**:

```
Devices → Replicator → Modbus Memory → Clients
```

Devices are touched once.
Clients read safely.

---

## Core Principles

* **The device is the truth**
* **Memory is the contract**
* **Status is data**
* **Poll failures are not write failures**
* **Every error has exactly one owner**

Nothing is guessed.
Nothing is hidden.

---

## Architecture (At a Glance)

```
[ Modbus Devices ]
        ↓
      Poller
        ↓
   PollResult (snapshot)
        ↓
      Writer
        ↓
[ Modbus Memory (MMA) ]
        ↓
   SCADA / Clients
```

* **Poller** reads devices
* **Writer** delivers data
* **MMA** serves as deterministic RAM

See `docs/ARCHITECTURE.md` for full details.

---

## Device Status Block

Device status is treated as **data**, not metadata.

* Written through the same writer
* Uses the same Raw Ingest protocol
* Stored in a separate memory region

Status is **opt‑in** via configuration:

* No `status_slot` → no status writes
* Configured slot → status written deterministically

This avoids probes, heartbeats, and hidden health logic.

---

## Configuration Model

Configuration is **explicit and validated**:

* YAML schema
* Deterministic address mapping
* Overlap detection
* Optional features must be declared

See:

* `docs/CONFIG.md`
* `internal/config/`

---

## Raw Ingest Protocol

The writer uses a **locked, stateless protocol**:

* One packet = one connection
* No session state
* No retries
* No negotiation

This keeps failure modes obvious and debuggable.

---

## What This Is NOT

This is **not**:

* a SCADA
* a historian
* a parser
* a rules engine
* a retry framework

Those belong *above* or *beside* this layer.

---

## Current State

Implemented:

* Modbus TCP polling
* Deterministic writer
* Device status block wiring
* Config validation & normalization
* Clean test coverage

Planned:

* Runner/state (seconds‑in‑error)
* Modbus RTU support
* Extended status semantics

---

## Who This Is For

* OT / SCADA engineers
* Energy systems
* Industrial automation
* Anyone tired of cascading Modbus failures

If you believe:

> *"A system should fail honestly."*

You’re in the right place.

---

## License

Open source. Practical. Unromantic.

Use it, study it, improve it.

---

## Final Note

This project optimizes for:

* clarity over cleverness
* determinism over convenience
* truth over green dashboards

If that resonates with you — welcome.
