# Raw Ingest v1 – Locked Specification (MMA2)

**Status:** LOCKED / WORKING
**Validated Against:** Node-RED sender + Go replicator + MMA2 appliance
**Last verified:** 2026-01-31

---

## Purpose

Raw Ingest v1 is a **binary, write-only protocol** used to push memory snapshots directly into an MMA2 memory instance.

It bypasses Modbus *request/response semantics* while **preserving Modbus memory layout exactly**.

This document exists to:

* Prevent silent protocol drift
* Prevent “helpful” extensions
* Serve as the executable contract between sender and MMA

---

## Core Principles (Non-Negotiable)

* Stateless: **1 TCP connection = 1 packet**
* No framing, no streaming
* No payload length field
* Memory is written **exactly as bytes arrive**
* Protocol correctness > visual correctness

---

## Packet Layout (LOCKED)

**Header size: 10 bytes**

| Offset | Size | Field   | Description                       |
| -----: | ---: | ------- | --------------------------------- |
|      0 |    2 | Magic   | ASCII `RI` (`0x52 0x49`)          |
|      2 |    1 | Version | `0x01`                            |
|      3 |    1 | Area    | FC selector (see below)           |
|      4 |    2 | Unit ID | Modbus Unit ID                    |
|      6 |    2 | Address | Zero-based register / bit address |
|      8 |    2 | Count   | Number of items                   |
|     10 |    N | Payload | Raw memory bytes                  |

⚠️ **There is NO payload length field in v1**
Payload size is **derived from Area + Count only**.

---

## Area Mapping

|   Area | Meaning                 | Notes             |
| -----: | ----------------------- | ----------------- |
| `0x01` | Coils (FC1)             | Bit-packed        |
| `0x02` | Discrete Inputs (FC2)   | Bit-packed        |
| `0x03` | Holding Registers (FC3) | 16-bit big-endian |
| `0x04` | Input Registers (FC4)   | 16-bit big-endian |

The Area byte is a **direct Modbus function-code selector**, not a semantic field.

---

## Payload Encoding Rules

### Bits (Areas `0x01`, `0x02`)

* Payload length = `ceil(count / 8)` bytes
* Bit order: **LSB first**
* No padding beyond final byte

### Registers (Areas `0x03`, `0x04`)

* Payload length = `count × 2` bytes
* **Big-endian register memory order**

```text
[ HIGH BYTE ][ LOW BYTE ]
```

Payload bytes are written **verbatim** into MMA memory.

---

## Response

After each packet, MMA2 returns **exactly 1 byte**:

|  Value | Meaning  |
| -----: | -------- |
| `0x00` | OK       |
| `0x01` | Rejected |

The TCP connection is then closed.

There is:

* No retry
* No error detail
* No partial acceptance

---

## Common Failure Modes (Historical, Real)

### ❌ Payload length field present

* Header bytes are written into register 0
* Produces phantom values like `250`, `500`
* Indicates protocol violation

### ❌ Wrong register byte order

* Values appear multiplied by `256`
* Indicates little-endian packing

### ❌ Persistent connections

* MMA2 closes the socket
* Each packet **must** use a fresh TCP connection

---

## Reference Implementations (Authoritative)

* Node-RED function sender
* Go replicator: `internal/writer/ingest/client.go`

Both implementations produce **bit-for-bit identical packets**.

---

## Versioning Policy

* v1 is **frozen**
* Any extension requires **v2** with explicit negotiation
* Never add fields to v1
* Never reinterpret existing fields

---

## Final Status

✅ Verified end-to-end
✅ Source and replica memory match exactly
✅ No offsets, hacks, or post-processing

**This specification is locked.**
