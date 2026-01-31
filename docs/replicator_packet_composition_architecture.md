# Replicator Packet Composition Architecture

## Status

**Authoritative — REQUIRED before further replicator changes**

This document defines the **replicator-side packet composition architecture** used to generate packets consumed by **MMA2.0 Raw Ingest v1**.

It intentionally does **not** modify, redefine, or extend MMA2.0.

---

## Purpose

The Modbus Replicator is responsible for:

1. Acquiring raw Modbus memory from field devices
2. Composing packets that conform **exactly** to the existing MMA2.0 Raw Ingest v1 packet format
3. Transmitting those packets reliably

This document exists to **lock the internal replicator architecture** so that:

* Packet composition has a single authority
* Future packet evolution is localized
* Source data acquisition remains stable
* Silent protocol drift is prevented

---

## Explicit Non-Goals

This document does **not**:

* Redefine the MMA2.0 packet format
* Act as the source of truth for packet bytes
* Introduce new packet fields
* Change MMA2.0 behavior

The packet format is **already defined** and treated here as an **external contract**.

---

## Architectural Problem Statement

Packet composition is a **volatile concern**:

* Header layout may evolve
* Versioning may change
* Flags may be added

Source data acquisition is a **stable concern**:

* Modbus polling semantics
* Register layouts
* Error ownership

Mixing these concerns causes:

* Format drift
* Change amplification
* Silent data corruption

Therefore, packet composition **must be centralized**.

---

## Core Architectural Rule (LOCKED)

> **Packet composition must exist in exactly one logical module inside the replicator.**

No other part of the replicator may:

* Write header fields
* Calculate packet sizes
* Manipulate byte offsets
* Encode version, flags, or area values

All other components exchange **typed data**, never raw bytes.

---

## Responsibility Separation

The replicator is divided into three responsibilities:

### 1) Source Acquisition (Stable)

Owns:

* Modbus polling
* Device timeouts
* Retry logic
* Raw register / bit data
* Error surfaces from Modbus

Must never:

* Know packet layout
* Write protocol headers
* Encode byte streams

---

### 2) Packet Composition (Volatile)

Owns:

* Packet header layout
* Version constants
* Area selectors
* Payload sizing rules
* Byte-level encoding

Is the **only place** where `[]byte` packets are constructed.

---

### 3) Transport (Generic)

Owns:

* TCP connections
* Connection lifecycle
* Reconnection policy
* Timeouts

Must treat packets as **opaque byte slices**.

---

## Required Module Structure

Packet composition must be isolated under a single module, for example:

```
internal/
 └─ writer/
    └─ ingest/
       ├─ constants.go   # magic, versions, area enums
       ├─ header.go      # header layout and invariants
       ├─ sizing.go      # payload sizing rules
       ├─ encode.go      # marshal → []byte
```

Notes:

* Files may be split to satisfy the ≤300-line rule
* The **module**, not the file, is the unit of responsibility
* No other package may construct ingest packets

---

## Packet Contract Reference (Informative)

The replicator packet **must exactly match** the MMA2.0 Raw Ingest v1 header:

| Offset | Size | Field   | Type   | Description           |
| -----: | ---: | ------- | ------ | --------------------- |
|      0 |    2 | Magic   | uint16 | Packet signature      |
|      2 |    1 | Version | uint8  | Packet version        |
|      3 |    1 | Area    | uint8  | Modbus memory area    |
|      4 |    2 | UnitID  | uint16 | Modbus Unit ID        |
|      6 |    2 | Address | uint16 | 0-based start address |
|      8 |    2 | Count   | uint16 | Number of items       |

Header size: **10 bytes**

This table is **reference only**. Code remains authoritative.

---

## Evolution Rule

If the packet format changes in the future:

* Only the packet composition module may be modified
* Source acquisition code must remain untouched
* Transport code must remain untouched

This guarantees safe evolution without regression.

---

## Failure Prevention Guarantee

This architecture prevents:

* Duplicated packet logic
* Partial format updates
* Silent corruption
* Cross-module coupling

Any violation of this document is considered a **replicator architecture error**.

---

## Summary

* MMA2.0 defines the packet contract
* The replicator **implements** the contract
* Packet composition is volatile and centralized
* Source data remains stable

> **Centralizing packet composition is mandatory for replicator correctness and long-term safety.**
