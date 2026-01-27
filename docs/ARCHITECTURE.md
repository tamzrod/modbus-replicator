# Modbus Replicator – Architecture

## Overview

The Modbus Replicator is a **read–fan-out–write** system designed to decouple unstable field devices from consumers by inserting a deterministic, memory-centric buffer (MMA – Modbus Memory Appliance) between them.

The core architectural rule is simple:

> **The source device is the truth. The writer is the delivery mechanism. Memory is the contract.**

The replicator never invents data, never probes devices independently, and never hides failures behind metadata.

---

## High-Level Flow

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

Each stage has **strict responsibility boundaries**.

---

## Component Responsibilities

### 1. Poller

**Responsibility:** Read devices.

- Executes Modbus reads against source devices
- Produces a `PollResult`
- Owns *device truth*

What the poller knows:
- Read success or failure
- Raw register / bit data

What the poller does **not** do:
- Write memory
- Maintain device state across cycles
- Interpret semantics

If a poll fails, it is reported as `PollResult.Err`.

---

### 2. Writer

**Responsibility:** Deliver data to memory.

The writer consumes a `PollResult` and pushes **data** into MMA using the Raw Ingest protocol.

Important rule:

> **A poll failure is not a writer failure.**

Writer behavior:
- Writes data blocks **only when `PollResult.Err == nil`**
- Writes device status blocks independently
- Returns errors **only** for delivery failures (network, protocol, wiring)

The writer never retries reads and never evaluates device health beyond what the poller reported.

---

### 3. Device Status Block

Device status is treated as **data**, not metadata.

- Written through the **same writer**
- Uses the **same Raw Ingest protocol**
- Stored at a **different address space**

This avoids side-channels, probes, and implicit health logic.

#### Status is Opt-In

A device participates in status reporting only when `status_slot` is configured.

If status is not configured:
- No status writes occur
- No memory is reserved

---

### 4. Status Data Model (Current Stage)

At this stage, the writer emits:

- **Health code**
  - OK when poll succeeded
  - ERROR when poll failed
- **Last error code**
  - Minimal placeholder (future expansion planned)

The writer does **not**:
- Accumulate time
- Track history
- Perform saturation logic

Those belong to the **runner/state layer**.

---

## Memory Model

### Why Memory Is Central

MMA acts as:
- A shock absorber
- A deterministic contract
- A single source for consumers

Memory semantics:
- Writes are atomic
- Addressing is explicit
- No implicit scaling or parsing

Consumers trust memory **only because the writer is honest**.

---

## Raw Ingest Protocol

The Raw Ingest protocol is:

- Stateless
- One packet = one connection
- Locked in format

Because status uses the same protocol:
- No protocol changes were required
- No version drift occurred
- Status remains forward-compatible

---

## Error Ownership Model

| Layer   | Owns Errors About |
|--------|------------------|
| Poller | Device reachability, read failures |
| Writer | Delivery failures, protocol errors |
| MMA    | Memory integrity |

This separation prevents:
- Double reporting
- False health signals
- Hidden failure modes

---

## Design Principles (Non-Negotiable)

- **Status is data**
- **Memory is the contract**
- **Writers do not interpret truth**
- **Pollers do not mutate state**
- **No background probes**
- **No hidden retries**

Every failure must be attributable to exactly one layer.

---

## Current State

Implemented:
- Poller
- Writer
- Status block wiring
- Config validation and normalization
- Deterministic memory writes

Pending (Next Stages):
- Runner/state (seconds-in-error)
- Status saturation logic
- Identity re-assertion after failure

---

## Summary

The Modbus Replicator is intentionally boring.

Its power comes from:
- Explicit boundaries
- Honest failure reporting
- Memory-first design

Nothing is hidden.
Nothing is guessed.

That is what makes it reliable.