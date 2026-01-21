# Raw Ingest v1 – Locked Specification (MMA2)

**Status:** LOCKED / WORKING  
**Validated Against:** Node-RED sender + Go replicator + MMA2 appliance  
**Last verified:** 2026‑01‑21

---

## Purpose

Raw Ingest v1 is a **binary write-only protocol** used to push memory snapshots directly into an MMA2 memory instance.

It bypasses Modbus semantics while preserving **Modbus memory layout**.

This document exists to prevent regressions.

---

## Core Principles

- Stateless: **1 TCP connection = 1 packet**
- No framing, no streaming
- No payload length field
- Memory is written **exactly as bytes arrive**
- Protocol correctness > visual correctness

---

## Packet Layout (LOCKED)

**Header size: 10 bytes**

| Offset | Size | Field | Description |
|------:|-----:|------|-------------|
| 0 | 2 | Magic | ASCII `RI` (`0x52 0x49`) |
| 2 | 1 | Version | `0x01` |
| 3 | 1 | Area | FC selector (see below) |
| 4 | 2 | Unit ID | Modbus Unit ID |
| 6 | 2 | Address | Zero‑based register / bit address |
| 8 | 2 | Count | Number of items |
| 10 | N | Payload | Raw memory bytes |

⚠️ **There is NO payload length field in v1**

---

## Area Mapping

| Area | Meaning | Notes |
|----:|--------|------|
| `0x01` | Coils (FC1) | Bit‑packed |
| `0x02` | Discrete Inputs (FC2) | Bit‑packed |
| `0x03` | Holding Registers (FC3) | 16‑bit big‑endian |
| `0x04` | Input Registers (FC4) | 16‑bit big‑endian |

---

## Payload Encoding Rules

### Bits (Areas 1 & 2)

- Payload length = `ceil(count / 8)` bytes
- Bit order: LSB first

### Registers (Areas 3 & 4)

- Payload length = `count × 2` bytes
- **Big‑endian register memory order**

```text
[ HIGH BYTE ][ LOW BYTE ]
```

---

## Response

After each packet, MMA2 returns **1 byte**:

| Value | Meaning |
|-----:|--------|
| `0x00` | OK |
| `0x01` | Rejected |

Connection is then closed.

---

## Common Failure Modes (Historical)

### ❌ Payload length field present

- Causes header bytes to be written into register 0
- Produces phantom values like `250` or `500`

### ❌ Wrong register byte order

- Produces values multiplied by `256`
- Indicates little‑endian packing

### ❌ Persistent connections

- MMA2 closes socket
- Each packet must use a fresh TCP connection

---

## Reference Implementations

- Node‑RED function sender (authoritative)
- Go replicator `internal/writer/ingest/client.go`

Both produce identical packets.

---

## Versioning Policy

- v1 is **frozen**
- Any extension requires **v2** with explicit negotiation
- Never add fields to v1

---

## Final Status

✅ Verified end‑to‑end  
✅ Source and replica memory match exactly  
✅ No offsets, hacks, or post‑processing

**This spec is locked.**

