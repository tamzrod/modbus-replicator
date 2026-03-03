# Modbus Serial (RTU) Support — Design Contract (Not Active)

Version Note: 2026-03-03 (Governance status correction)

## Status

**DESIGN LOCKED — NOT IMPLEMENTED**

Serial RTU source support is **not supported in the current runtime**.
Current active runtime paths implement Modbus TCP source behavior only.

This document is a **future design contract** and is **non-normative until activation**.
It defines implementation boundaries that must be followed if RTU is activated in a future release.

---

## Purpose

Define the future RTU integration boundary without changing current behavior.

This document exists to ensure any future RTU implementation:

- remains transport-only,
- preserves existing externally observable contracts,
- and does not introduce semantic drift.

---

## Design Boundary (Future Contract)

### Core Rule

> **RTU is transport only.**
> **It must not change memory layout, slot model, status model, packet contract, or runtime semantics.**

Transport complexity must never leak upward.

### No Semantic Leakage

A future RTU source path must not:

- redefine device truth semantics,
- alter status slot meanings,
- alter Raw Ingest packet format,
- add transport-specific control logic,
- add retries beyond existing poll cadence policy.

---

## Non-Activation Statement

The following are **not active capabilities** today:

- serial device polling,
- RTU source configuration in active runtime,
- RTU production transport path.

Any references below are design constraints for future activation only.

---

## Future Activation Contract

If RTU is implemented in a future release, it must satisfy all conditions below.

### 1) Contract Preservation

- Keep the same externally observable behavior as TCP source mode.
- Preserve current status model boundaries.
- Preserve current per-target routing model.
- Preserve current no-retry behavior model.

### 2) Transport Equivalence

- Treat RTU as an alternate source transport only.
- Keep source acquisition outputs equivalent in meaning to TCP acquisition outputs.
- Keep transport differences internal to source acquisition.

### 3) Deterministic Runtime

- Respect configured polling cadence.
- Respect RTU framing/timing requirements without changing orchestrator semantics.
- Preserve deterministic poll cycle behavior (single-cycle outcome, explicit error propagation).

### 4) No Architecture Drift

- Do not move semantic ownership from existing layers.
- Do not add hidden policy in transport adapters.
- Do not bypass existing validation and status pathways.

---

## Activation Checklist (Must Be Completed Before Declaring RTU Active)

1. Implementation exists in active runtime paths.
2. Configuration schema explicitly and unambiguously supports RTU source mode.
3. Validation rules cover RTU source configuration errors.
4. Polling behavior under RTU preserves deterministic contract boundaries.
5. Status behavior remains aligned with existing status documentation.
6. Raw Ingest packet behavior remains unchanged.
7. Documentation authority set is updated concurrently to reflect activation.
8. Alignment verification confirms no contradiction with governing docs.

Until all checklist items are complete, RTU remains **NOT IMPLEMENTED**.

---

## Determinism Requirements (Future RTU Path)

A future RTU implementation must preserve:

- explicit timeout handling,
- explicit error propagation,
- no hidden retries,
- no speculative reads,
- no transport-derived semantic overrides.

RTU timing mechanics are internal transport concerns and must not alter external contract semantics.

---

## Excluded Scope

This document does not authorize:

- adding RTU features to current runtime,
- changing current implementation behavior,
- redefining existing architecture authority,
- introducing partial or implied RTU support claims.

---

## Final Statement

RTU support is currently inactive by design.
This document defines the locked future boundary for activation work.
Transport complexity must never leak upward.

