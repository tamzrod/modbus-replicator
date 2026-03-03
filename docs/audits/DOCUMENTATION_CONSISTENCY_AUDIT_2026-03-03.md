---
**Status:** ARCHIVAL  
**Purpose:** Historical audit artifact  
**Non-Normative:** This document does NOT define system behavior.  
**Authority:** Documentation in root /docs defines behavior.  
**Date Archived:** 2026-03-03  
---

# Documentation Consistency Audit (Archived)

Date: 2026-03-03  
Scope: All Markdown files in repository (docs-only audit; no source code inspection)

## Findings

### 1) CONTRADICTION + UNRESOLVED AUTHORITY CONFLICT
- **Files/Sections:**
  - `docs/ARCHITECTURE.md` → `Status Data Model (Current Stage)`
  - `docs/Status_Block_Layout.md` → `1. Overview`, `2. Slots 0--19`, `3. Slots 20--29`
- **Issue:**
  - `ARCHITECTURE` says current stage emits exactly 3 fields (`Health`, `Last Error Code`, `Seconds-In-Error`), while `Status_Block_Layout` defines a locked 30-slot canonical model including device name and transport counters.
  - Both present as authoritative/locked, but no precedence is declared.

### 2) CONTRADICTION + UNRESOLVED AUTHORITY CONFLICT
- **Files/Sections:**
  - `docs/Status_Symantics.md` → `Health Code`
  - `docs/Status_Block_Layout.md` → `Slot 0 --- health_code`
- **Issue:**
  - `Status_Symantics` defines binary health only (`OK`/`ERROR`), while `Status_Block_Layout` defines 5 states (`UNKNOWN/BOOT`, `OK`, `ERROR`, `STALE`, `DISABLED`).
  - No precedence rule is provided.

### 3) CONTRADICTION + UNRESOLVED AUTHORITY CONFLICT
- **Files/Sections:**
  - `docs/Status_Symantics.md` → `What This Layer Does NOT Do`
  - `docs/Status_Rules_Executable.md` → `2. SECONDS IN ERROR`
- **Issue:**
  - `Status_Symantics` says no time accumulation; `Status_Rules_Executable` requires `seconds-in-error` increment each second.
  - These conflict without explicit layer/phase precedence.

### 4) CONTRADICTION + UNRESOLVED AUTHORITY CONFLICT
- **Files/Sections:**
  - `docs/CONFIG.md` → `Status Memory (Optional but Global)`
  - `docs/PER_TARGET_MIGRATION_PLAN.md` → `HARD RULES (locked for this migration)`
- **Issue:**
  - `CONFIG` defines global `replicator.Status_Memory`; migration doc requires removing global status memory and using per-target `status_unit_id`.
  - Both read as normative and active; precedence/timeline not declared.

### 5) STALE SECTION
- **File/Section:**
  - `Examples/ge_energy_meter/readme.md` → `Topology`
- **Issue:**
  - States “Shared status in Unit 100” (global/shared status model), which appears superseded by per-target migration rules.
  - No legacy/deprecated marker is present.

### 6) DUPLICATE AUTHORITY
- **Files/Sections:**
  - `docs/raw_ingest_v_1_spec.md` → `Packet Layout (LOCKED)`
  - `docs/replicator_packet_composition_architecture.md` → `Packet Contract Reference (Informative)`
- **Issue:**
  - Same header contract appears in multiple places with differing authority language.

### 7) CONTRADICTION (Authority Wording)
- **Files/Sections:**
  - `docs/raw_ingest_v_1_spec.md` → `Purpose`, `Packet Layout (LOCKED)`
  - `docs/replicator_packet_composition_architecture.md` → `Packet Contract Reference (Informative)`
- **Issue:**
  - Raw ingest spec presents document as executable contract; packet composition doc says table is informative and “code remains authoritative.”
  - Document-authority vs code-authority is unresolved.

### 8) AMBIGUITY
- **File/Section:**
  - `docs/modbus_serial_rtu_support.md` → `Reply Code Handling (Unchanged)`
- **Issue:**
  - Uses “fixed failure code (e.g. 255)” while other docs require raw pass-through/no mapping.
  - “e.g.” leaves multiple interpretations.

### 9) AMBIGUITY
- **Files/Sections:**
  - `docs/Status_Rules_Executable.md` → `4. DEVICE IDENTITY (DEVICE NAME)`
  - `docs/Status_Block_Layout.md` → `Slots 3--10 --- device_name`
- **Issue:**
  - Executable rules say device name is “stored at the END of the status block,” while layout defines fixed location (3–10).

### 10) UNDEFINED TERM
- **File/Section:**
  - `docs/Status_Rules_Executable.md` → `4. DEVICE IDENTITY (DEVICE NAME)`
- **Issue:**
  - “MMA trust loss” is used as write trigger but is not defined in the documentation set.

### 11) UNDEFINED TERM
- **File/Section:**
  - `docs/Status_Rules_Executable.md` → `5. SILENCE RULES`
- **Issue:**
  - “RBE++” is referenced but never defined.

### 12) UNDEFINED TERM
- **File/Section:**
  - `docs/PER_TARGET_MIGRATION_PLAN.md` → `Code changes (what to implement)`
- **Issue:**
  - References “incremental writes as per your directives,” but “directives” source is not specified.

### 13) SCOPE LEAKAGE
- **File/Section:**
  - `docs/replicator_packet_composition_architecture.md` → `Responsibility Separation` (`Source Acquisition` owns retry logic)
- **Issue:**
  - Other docs repeatedly declare no retries as non-goal/non-negotiable behavior.
  - Assigning retry ownership in this architecture doc leaks policy beyond established scope.

### 14) INCOMPLETE CONTRACT
- **File/Section:**
  - `docs/CONFIG.md` → `Status Memory (Optional but Global)`, `Source`
- **Issue:**
  - Describes status opt-in but omits required constraints (e.g., slot uniqueness/collision rules, bounds).

### 15) INCOMPLETE CONTRACT
- **File/Section:**
  - `docs/modbus_serial_rtu_support.md` → `Serial Configuration (YAML)`, `Slot & Memory Discipline`
- **Issue:**
  - States slot index comes from YAML but does not define exact field path/name or validation constraints.

### 16) CIRCULAR REFERENCE (Weak / De-facto)
- **Files/Sections:**
  - `docs/raw_ingest_v_1_spec.md` → `Reference Implementations (Authoritative)`
  - `docs/replicator_packet_composition_architecture.md` → `Packet Contract Reference (Informative)` + “code remains authoritative”
- **Issue:**
  - Authority loops between spec text and implementation references without a clean root precedence model.

---

## High-Risk Conflicts (Could Cause Implementation Divergence)
- Status model size/content conflict: 3-field “current stage” vs locked 30-slot block.
- Health model conflict: binary (`OK/ERROR`) vs 5-state enum.
- Global status memory vs per-target status migration hard rules.
- Time accumulation forbidden in semantics doc vs required in executable status rules.

## Structural Redundancies
- Packet contract repeated in multiple docs with different authority wording.
- Status behavior duplicated across semantics/layout/executable docs with drift.
- Core principles/non-goals repeated across root and architecture docs with partial divergence.

## Authority Overlaps
Competing normative claims appear across:
- `docs/raw_ingest_v_1_spec.md`
- `docs/replicator_packet_composition_architecture.md`
- `docs/Status_Block_Layout.md`
- `docs/Status_Symantics.md`
- `docs/Status_Rules_Executable.md`
- `docs/PER_TARGET_MIGRATION_PLAN.md`

No explicit precedence chain is defined.

## Ambiguity Zones
- RTU failure code handling (“raw pass-through” vs “fixed code e.g. 255”).
- Device name placement (“END of status block” vs fixed slot range 3–10).
- Migration applicability timing relative to current `CONFIG` and example topology docs.
- Meaning of “current stage” across architecture and status documents.

## Documentation That Appears Fully Coherent
- `docs/device_status_limits.md` (clear boundary and capacity ownership model).
- `README.md` and `docs/ARCHITECTURE.md` are largely aligned on top-level principles (memory contract, no hidden retries/probes), excluding the status-model conflicts listed above.
- Empty internal package READMEs introduce no direct contradictions but provide no authority.

---

## Unresolved Authority Conflicts (Explicit)
The following disagreements are unresolved because neither side declares precedence:
1. `docs/ARCHITECTURE.md` vs `docs/Status_Block_Layout.md` (3 fields vs 30-slot canonical block)
2. `docs/Status_Symantics.md` vs `docs/Status_Block_Layout.md` (binary vs 5-state health)
3. `docs/Status_Symantics.md` vs `docs/Status_Rules_Executable.md` (no accumulation vs required seconds accumulation)
4. `docs/CONFIG.md` vs `docs/PER_TARGET_MIGRATION_PLAN.md` (global status memory vs per-target status destination)
