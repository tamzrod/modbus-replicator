# Replicator Packet Composition Architecture

## Status

**Authoritative — REQUIRED before further replicator changes**

This document defines the **replicator‑side packet composition architecture** used to generate packets consumed by **MMA2.0 raw ingest**.

It intentionally does **not** modify, redefine, or extend MMA2.0.

---

## Purpose

The Modbus Replicator is responsible for:

1. Acquiring raw Modbus memory from field devices
2. Composing packets that conform to the **existing MMA2.0 raw ingest packet format**
3. Transmitting those packets reliably

This document exists to **lock the internal replicator architecture** so that:

- packet composition has a single authority
- future packet evolution is localized
- source data acquisition remains stable
- silent protocol drift is prevented

---

## Explicit Non‑Goals

This document does **not**:

- redefine the MMA2.0 packet format
- act as the source of truth for packet bytes
- introduce new packet fields
- change MMA2.0 behavior

The packet format is **already defined** and treated here as an **external contract**.

---

## Architectural Problem Statement

Packet composition is a **volatile concern**:

- header layout may evolve
- versioning may change
- flags may be added

Source data acquisition is a **stable concern**:

- Modbus polling semantics
- register layouts
- retry logic

Mixing these concerns causes:

- format drift
- change amplification
- silent data corruption

Therefore, packet composition **must be centralized**.

---

## Core Architectural Rule (LOCKED)

> **Packet composition must exist in exactly one logical module inside the replicator.**

No other part of the replicator may:

- write header fields
- calculate packet sizes
- manipulate byte offsets
- encode version, flags, or area values

All other components exchange **data**, never bytes.

---

## Responsibility Separation

The replicator is divided into three responsibilities:

### 1) Source Acquisition (Stable)

Owns:
- Modbus polling
- device timeouts
- retry logic
- raw register / bit data

Must never:
- know packet layout
- write protocol headers

---

### 2) Packet Composition (Volatile)

Owns:
- packet header layout
- version and flags
- payload sizing rules
- byte‑level encoding

Is the **only place** where packet bytes are constructed.

---

### 3) Transport (Generic)

Owns:
- TCP connections
- reconnection logic
- buffering

Must treat packets as opaque `[]byte`.

---

## Required Module Structure

Packet composition must be isolated under a single module, for example:

```
replicator/
 └─ packet/
    ├─ constants.go   # magic, versions, area enums
    ├─ header.go      # header struct and field meaning
    ├─ sizing.go      # payload sizing rules
    ├─ encode.go      # Marshal → []byte
```

Notes:
- Files may be split to satisfy the ≤300‑line rule
- The **module**, not the file, is the unit of responsibility

---

## Packet Contract Reference

The replicator packet **must exactly match** the MMA2.0 raw ingest header:

| Offset | Size | Field | Type | Description |
|------:|-----:|------|------|-------------|
| 0 | 2 | Magic | uint16 | Packet signature |
| 2 | 1 | Version | uint8 | Packet version |
| 3 | 1 | Flags | uint8 | Behavior flags |
| 4 | 1 | Area | uint8 | Modbus memory area |
| 5 | 1 | Reserved | uint8 | Must be 0 |
| 6 | 2 | UnitID | uint16 | Modbus Unit ID |
| 8 | 2 | Port | uint16 | Target Modbus port |
| 10 | 2 | Address | uint16 | 0‑based start address |
| 12 | 2 | Count | uint16 | Number of items |

Header size: **14 bytes**

This table is a **reference only**. Code remains authoritative.

---

## Evolution Rule

If the packet format changes in the future:

- only the packet module may be modified
- source acquisition code must remain untouched
- transport code must remain untouched

This guarantees safe evolution without regression.

---

## Failure Prevention Guarantee

This architecture prevents:

- duplicated packet logic
- partial format updates
- silent corruption
- cross‑module coupling

Any violation of this document is considered a **replicator architecture error**.

---

## Summary

- MMA2.0 defines the packet contract
- the replicator **implements** the contract
- packet composition is volatile and centralized
- source data remains stable

> **Centralizing packet composition is mandatory for replicator correctness and long‑term safety.**
