# Modbus Serial (RTU) Support — Scope & Rules

> **Purpose**  
> Define how **Modbus RTU (serial)** is supported by the Replicator without violating core architecture principles.

This document is **normative**.

---

## Core Statement (Locked)

> **Modbus RTU is only a transport.**  
> **It must not change memory layout, slot model, or system behavior.**

If RTU introduces semantics, policy, or intelligence, the design is wrong.

---

## Responsibility Boundary

### Replicator

The Replicator:

- Supports Modbus RTU as a **source transport only**
- Treats RTU exactly the same as Modbus TCP
- Produces the same raw outcomes:
  - reply code
  - error duration
  - online flag
  - device name

The Replicator does **not**:

- Perform protocol translation
- Implement control logic
- Add retries beyond poll cadence
- Infer line quality or health
- Discover devices

---

### MMA (Modbus Memory Appliance)

The MMA:

- Is unaware of transport type
- Receives the same writes regardless of TCP or RTU
- Enforces bounds and memory allocation

Transport choice must be invisible to MMA.

---

## Supported Serial Mode

- Modbus RTU (binary)
- RS-485 / RS-232

Explicitly **not supported**:

- Modbus ASCII
- Vendor-specific serial protocols
- Multi-master arbitration

---

## Serial Configuration (YAML)

RTU configuration is mutually exclusive with TCP configuration.

```yaml
source:
  serial:
    device: "/dev/ttyUSB0"
    baudrate: 9600
    databits: 8
    parity: "N"
    stopbits: 1
    timeout_ms: 2000
  unit_id: 1
```

Rules:

- Exactly one source type per unit
- No auto-detection
- No guessed defaults
- Invalid config fails fast at startup

---

## Timing & Determinism

- Poll interval remains authoritative
- RTU silent interval must be respected
- One serial port = one poll loop
- No parallel access to the same serial device

The Replicator must not:

- Flood the bus
- Guess frame boundaries
- Hide timing violations

---

## Reply Code Handling (Unchanged)

RTU uses the **same reply code contract** as TCP:

- `0` → valid response
- Modbus exception → raw exception code
- Transport / timeout failure → fixed failure code (e.g. `255`)

No RTU-specific interpretation is allowed.

---

## Error Duration & Online Flag

- `error_seconds` increments while reply code != 0
- Resets when reply code == 0
- `online_flag = 1` when reply code == 0
- `online_flag = 0` when reply code != 0

Identical behavior across all transports.

---

## Slot & Memory Discipline

- RTU devices map to **device slots** exactly like TCP devices
- Slot index comes from YAML
- Slot size is fixed
- Input Registers only

Transport must never affect slot layout.

---

## Deployment Models

### Bare Metal / VM

- Replicator opens serial device directly
- Only requirement is OS-level permissions
- One process owns the serial port

### Docker

- Serial device must be passed explicitly:

```bash
docker run --device=/dev/ttyUSB0 rodtamin/modbus-replicator
```

- No `--privileged`
- No mounting `/dev`

---

## Failure Behavior

- On serial open failure: fail fast
- On poll failure: update device slot on next cycle
- No reconnect storms
- Silence is a valid state

---

## Explicit Non-Goals (DO NOT ADD)

- Auto baud detection
- Line quality metrics
- Signal strength
- Address scanning
- Device discovery
- Multi-drop orchestration
- RTU metadata

All of the above belong outside the Replicator.

---

## Final Locked Statement

> **Modbus RTU is just another wire.**  
> **Truth, slots, and memory do not care how bytes arrived.**  
> **Transport complexity must never leak upward.**

