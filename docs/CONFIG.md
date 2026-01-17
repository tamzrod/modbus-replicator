# Modbus Replicator — Configuration Model

## Purpose

Configuration defines **what replication units exist at startup**.

Each unit is:
- Independent
- One-way
- Deterministic
- Immutable after load

Configuration is read **once** at process start.
Any change requires a restart.

---

## Core Principles

- No runtime mutation
- No semantics
- No control logic
- No routing rules
- No retries

Configuration describes **topology only**.

---

## High-Level Structure

```yaml
replicator:
  units:
    - id: "replicator-1"

      source:
        endpoint: "tcp://10.0.0.5:502"
        unit_id: 1
        timeout_ms: 2000

      targets:
        - id: "mma-main"
          endpoint: "tcp://127.0.0.1:1502"
          memories:
            - memory_id: 0
            - memory_id: 1

        - id: "mma-backup"
          endpoint: "tcp://127.0.0.1:1602"
          memories:
            - memory_id: 0

      poll:
        interval_ms: 1000

    - id: "replicator-2"

      source:
        endpoint: "tcp://10.0.0.6:502"
        unit_id: 2
        timeout_ms: 1500

      targets:
        - id: "mma-main"
          endpoint: "tcp://127.0.0.1:1502"
          memories:
            - memory_id: 2

      poll:
        interval_ms: 500
