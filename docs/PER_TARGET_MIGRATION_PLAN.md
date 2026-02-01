# Per-Target Status Memory Migration (Global → Per Target)

## Goal

Move **Device Status Block** writes from a single global `replicator.Status_Memory` endpoint to a **per-target** status destination.

**Meaning:** each target defines where status is written (typically same TCP endpoint as data, different UnitID).

This supports:
- fan-out (same source → multiple targets)
- different targets writing status to different places (per-target truth)

---

## HARD RULES (locked for this migration)

1. **No global status memory**
   - Remove `replicator.Status_Memory` entirely.

2. **Per-target status destination**
   - Add `status_unit_id` under each target.
   - Status writes are executed **for each target** independently.

3. **Opt-in stays**
   - A unit participates in status only if `source.status_slot` exists.
   - If any unit enables status, then each of its targets must declare `status_unit_id`.

4. **No reads / no handshakes**
   - Status is written via the same write-only path (Raw Ingest).

---

## YAML (new schema)

### Minimal example (1 unit, FC3 only, fan-out 2 targets)

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
        - id: "mma-a"
          endpoint: "127.0.0.1:1502"
          unit_id: 2
          status_unit_id: 35
          memories:
            - offsets: {}

        - id: "mma-b"
          endpoint: "127.0.0.1:1503"
          unit_id: 2
          status_unit_id: 35
          memories:
            - offsets: {}

      poll:
        interval_ms: 1000
```

---

## Code changes (what to implement)

> ⚠️ I cannot produce an exact patch against your repo **without the current file tree and structs**.
> This document is **commit-ready as a plan**, plus **drop-in file templates**.
>
> To generate the exact full-file replacements, paste the current contents of:
> - `internal/config/*.go` (where `Status_Memory` and `Target` are defined)
> - `internal/writer/*` (where status writes are performed)
>
> Then I will output full-file replacements with correct paths and package names.

### 1) Config model

#### Remove
- `replicator.Status_Memory`

#### Add
- `Target.status_unit_id` (uint16)

#### Keep
- `Source.status_slot` (uint16) — still the base of the 20-slot block.

**Validation rules**
- If a unit has `status_slot`:
  - every target must have `status_unit_id` set (non-zero)
- slot collisions:
  - **no longer global**
  - collisions are evaluated **per target** *only if* you allow multiple units to share the same target + status_unit_id.
  - simplest rule: collisions are a hard error *per target endpoint + status_unit_id*.

### 2) Writer changes

#### Before (global)
- Resolve one global status destination and write status once.

#### After (per target)
- For each target:
  - write status block to that target’s `status_unit_id`.

Status remains:
- full block write on start / after previous write failure
- incremental writes as per your directives

### 3) Status destination identity

Per-target status destination is:

```
(target.endpoint, target.status_unit_id)
```

Status slot layout stays the same (20 slots per device).

---

## Drop-in interface (template)

If your writer currently takes a single status endpoint, refactor to take a resolved destination per target.

```go
// internal/status/destination.go
package status

type Destination struct {
    Endpoint string
    UnitID   uint16
}
```

Then writer can do:

```go
for _, tgt := range unit.Targets {
    dest := status.Destination{
        Endpoint: tgt.Endpoint,
        UnitID:   tgt.StatusUnitID,
    }
    // write slots 0-19 (or incremental) to dest
}
```

---

## Quick checklist (brain + runtime)

- [ ] YAML has **no** `Status_Memory`
- [ ] Every target has `status_unit_id`
- [ ] `status_slot` is set only when you want status
- [ ] Fan-out writes status to **each** target
- [ ] Readers know where to look:
  - Data: `target.unit_id`
  - Status: `target.status_unit_id`, slots `status_slot*20 .. +19`

---

## What I need from you to output full Go files

Paste **verbatim** (no screenshots) these files:

1. The config structs + loader + validation (where `Status_Memory` exists today)
2. The status writer file(s) (where status is currently written)

Once I have those, I’ll return:
- full-file replacements (correct paths)
- small-file split (≤300 lines)
- no snippets, no “insert below”
