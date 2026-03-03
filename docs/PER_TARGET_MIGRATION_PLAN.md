# Per-Target Status Memory Migration (Global → Per Target)

Version Note: 2026-03-03 (Stage 4 documentation rectification; synchronized to implemented behavior)

COMPLETED NOTICE: This migration is complete in current code. This document is now a completion record, not a normative implementation plan.

## Goal

Move Device Status Block writes from a single global destination to per-target status destinations.

---

## Implemented Outcome

1. Global status memory removed from config model.
2. Per-target `status_unit_id` is used for status destination.
3. Status remains opt-in via `source.status_slot`.
4. Status is written through the existing write-only Raw Ingest path.

---

## Implemented Schema Pattern

```yaml
replicator:
  units:
    - id: "mvps-01"
      source:
        endpoint: "127.0.0.1:1502"
        unit_id: 1
        device_name: "MVPS-01"
        status_slot: 0
      reads:
        - fc: 3
          address: 0
          quantity: 10
      targets:
        - id: 1
          endpoint: "127.0.0.1:1502"
          unit_id: 2
          status_unit_id: 35
          memories:
            - memory_id: 1
              offsets: {}
      poll:
        interval_ms: 1000
```

---

## Implemented Validation Contract

When `source.status_slot` is set:

* Unit must have at least one target.
* Every target must include `status_unit_id`.
* Collision is rejected for duplicate `(endpoint, status_unit_id, status_slot)`.

---

## Implemented Status Destination Identity

Per-target status destination:

```
(target.endpoint, target.status_unit_id)
```

Status write addressing uses base register:

```
base_addr = source.status_slot * 30
```

---

## Migration Checklist (Final)

- [x] `replicator.Status_Memory` removed
- [x] Per-target `status_unit_id` in use
- [x] Status opt-in behavior preserved
- [x] Per-target fan-out status writes active

---

## Scope of This Document

This file is retained for migration history and audit traceability.

Normative behavior is defined by current implementation and synchronized documentation set.
