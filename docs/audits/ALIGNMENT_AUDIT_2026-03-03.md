---
**Status:** ARCHIVAL  
**Purpose:** Historical audit artifact  
**Non-Normative:** This document does NOT define system behavior.  
**Authority:** Documentation in root /docs defines behavior.  
**Date Archived:** 2026-03-03  
---

# Architecture Alignment Audit (Archived)

Date: 2026-03-03  
Method: Cross-reference documentation conflicts with implementation facts.  
Scope: Determine where code matches, contradicts, or exceeds documented contracts.

---

## 1. DIRECT CONTRADICTIONS (Highest Risk)

### Contradiction 1.1: Status Block Size Model
- **Stage 1 Reference:** [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) → `Status Data Model (Current Stage)` vs [docs/Status_Block_Layout.md](docs/Status_Block_Layout.md)
  - `ARCHITECTURE` declares 3-field model (Health, LastErrorCode, SecondsInError) as "current stage"
  - `Status_Block_Layout` declares 30-slot locked canonical model
  - Neither declares which is authoritative; conflict on **scope boundaries**
- **Stage 2 Reference:** [internal/status/constants.go](internal/status/constants.go#L4), [internal/status/encode.go](internal/status/encode.go), [cmd/replicator/main.go](cmd/replicator/main.go#L106)
  - Code implements 30-slot model **entirely**
  - Transport counters (slots 20–29) are encoded and written
  - Device name (slots 3–10) is encoded and written on full block
  - `status.Snapshot` struct mirrors all 30 slots (snapshot.go lines 10–22)
- **Alignment Finding:** 
  - **Classification:** UNRESOLVED AUTHORITY IMPACT
  - **Status:** Implementation chooses `Status_Block_Layout` interpretation (30-slot)
  - **Risk:** ARCHITECTURE.md's "current stage" language is misleading; code is past this stage
  - **Evidence:** `SlotsPerDevice = 30` is immutable constant; snapshot includes transport counters; main.go injects counter values every poll

### Contradiction 1.2: Health State Enumeration
- **Stage 1 Reference:** [docs/Status_Symantics.md](docs/Status_Symantics.md) → `Health Code` vs [docs/Status_Block_Layout.md](docs/Status_Block_Layout.md) → `Slot 0`
  - `Status_Symantics` declares binary-only: OK (success) / ERROR (failure)
  - `Status_Block_Layout` defines 5-state enum: UNKNOWN(0), OK(1), ERROR(2), STALE(3), DISABLED(4)
- **Stage 2 Reference:** [internal/status/constants.go](internal/status/constants.go#L119-L131)
  - Code defines all 5 constants (HealthUnknown through HealthDisabled)
  - [cmd/replicator/main.go](cmd/replicator/main.go#L130,#L133) **only emits HealthOK or HealthError**
  - STALE and DISABLED are never assigned by poller/orchestrator
- **Alignment Finding:**
  - **Classification:** OVER-IMPLEMENTED + CODE INCONSISTENCY
  - **Status:** Code supports 5-state model; implementation uses only 2 states
  - **Risk:** Clients reading slot 0 may expect STALE/DISABLED but will never see them
  - **Evidence:** 
    - Lines 119–131 of constants.go define all 5 values
    - Line 130 of main.go: `if snap.Health != status.HealthOK { snap.Health = status.HealthOK }`
    - Line 133: `snap.Health = status.HealthError`
    - No code path sets HealthStale or HealthDisabled

### Contradiction 1.3: Time Accumulation (Forbidden vs Required)
- **Stage 1 Reference:** [docs/Status_Symantics.md](docs/Status_Symantics.md) → `What This Layer Does NOT Do` vs [docs/Status_Rules_Executable.md](docs/Status_Rules_Executable.md) → `2. SECONDS IN ERROR`
  - `Status_Symantics` explicitly forbids time accumulation ("No time accumulation")
  - `Status_Rules_Executable` explicitly requires: "Increment by 1 every second while Raw Error Code != 0"
- **Stage 2 Reference:** [cmd/replicator/main.go](cmd/replicator/main.go#L162-L175)
  - `secTicker := time.NewTicker(time.Second)` (line 165)
  - `if snap.Health != status.HealthOK && snap.SecondsInError < 65535 { snap.SecondsInError++ }`
  - Every second while health != OK, counter increments
  - [internal/writer/status_writer.go](internal/writer/status_writer.go#L62) saturates at 65535
- **Alignment Finding:**
  - **Classification:** UNRESOLVED AUTHORITY IMPACT + CONTRADICTION
  - **Status:** Implementation chooses `Status_Rules_Executable` (time accumulation required)
  - **Risk:** `Status_Symantics` claim of "no time accumulation" is violated by implementation
  - **Evidence:** main.go lines 162–175, status_writer.go line 62 (saturation clamp)

### Contradiction 1.4: Status Memory Topology (Global vs Per-Target)
- **Stage 1 Reference:** [docs/CONFIG.md](docs/CONFIG.md) → `Status Memory (Optional but Global)` vs [docs/PER_TARGET_MIGRATION_PLAN.md](docs/PER_TARGET_MIGRATION_PLAN.md) → `HARD RULES`
  - `CONFIG.md` defines `replicator.Status_Memory` as single global endpoint
  - `PER_TARGET_MIGRATION_PLAN.md` declares "Remove global status memory entirely" and "Add `status_unit_id` under each target"
  - Migration doc explicitly claims authority via **"HARD RULES (locked for this migration)"**
- **Stage 2 Reference:** [internal/config/config.go](internal/config/config.go#L27-L39)
  - `ReplicatorConfig` struct has **NO** Status_Memory field
  - `TargetConfig` has `StatusUnitID *uint8` field (line 35)
  - [internal/config/validate.go](internal/config/validate.go#L40-L65) enforces per-target collision detection via `endpoint|status_unit_id|slot` key
  - [internal/writer/builder.go](internal/writer/builder.go#L30-L42) creates per-target `StatusPlan` entries
- **Alignment Finding:**
  - **Classification:** DOCUMENTATION DRIFT (CONFIG.md is outdated; migration is implemented)
  - **Status:** Implementation is fully per-target; global model completely removed
  - **Risk:** CONFIG.md remains as normative doc but describes architecture no longer in code
  - **Evidence:** 
    - config.go: no Status_Memory type
    - validate.go line 40: per-target collision key
    - builder.go lines 30–42: per-target status plan

---

## 2. AUTHORITY CONFLICTS RESOLVED BY CODE CHOICE

### Finding 2.1: Device Name Placement ("END" vs Fixed Slots)
- **Stage 1 Reference:** [docs/Status_Rules_Executable.md](docs/Status_Rules_Executable.md) → `4. DEVICE IDENTITY` vs [docs/Status_Block_Layout.md](docs/Status_Block_Layout.md) → `Slots 3–10`
  - Executable rules: "stored at END of the status block"
  - Layout doc: "Slots 3–10 — device_name"
  - Ambiguity: Does "END" mean slots 20–29? Or end of fixed-data section (slot 10)?
- **Stage 2 Reference:** [internal/status/constants.go](internal/status/constants.go#L32-L39), [internal/writer/status_writer.go](internal/writer/status_writer.go#L228-L243)
  - `SlotDeviceNameStart = 3`
  - `SlotDeviceNameSlots = 8`
  - `SlotDeviceNameEnd = SlotDeviceNameStart + SlotDeviceNameSlots - 1` = 10
  - `encodeDeviceNameRegs()` packs 16 chars into 8 registers at slots 3–10
- **Alignment Finding:**
  - **Classification:** ALIGNED (code chooses `Status_Block_Layout` interpretation)
  - **Status:** Implementation resolves ambiguity by placing name at slots 3–10
  - **Impact:** No risk; both interpretations are resolved
  - **Evidence:** constants.go lines 32–39 are explicit; status_writer.go line 153 only writes name in full block

### Finding 2.2: Packet Header Contract Authority
- **Stage 1 Reference:** [docs/raw_ingest_v_1_spec.md](docs/raw_ingest_v_1_spec.md) → `Purpose, Packet Layout (LOCKED)` vs [docs/replicator_packet_composition_architecture.md](docs/replicator_packet_composition_architecture.md) → `Packet Contract Reference (Informative)` + "code remains authoritative"
  - Spec doc: "This document exists... Prevent silent protocol drift... executable contract"
  - Arch doc: "This table is reference only. Code remains authoritative."
  - Conflict: Is spec or code the source of truth?
- **Stage 2 Reference:** [internal/writer/ingest/client.go](internal/writer/ingest/client.go#L129-L145)
  - Header layout: magic (0x52 0x49), version (0x01), area, unitID, address, count (10 bytes fixed)
  - Matches spec byte-for-byte
  - No deviation from spec in code
- **Alignment Finding:**
  - **Classification:** ALIGNED (both authorities agree; code follows both)
  - **Status:** Implementation matches written specification exactly
  - **Impact:** No need to defer to code authority; spec is correct
  - **Evidence:** ingest/client.go lines 90–91 (magic), line 92 (version), line 128–145 (header assembly)

### Finding 2.3: Poller Retry Logic Ownership (Scope Boundary)
- **Stage 1 Reference:** [docs/replicator_packet_composition_architecture.md](docs/replicator_packet_composition_architecture.md) → `Responsibility Separation` assigns "retry logic" to Source Acquisition (Poller) vs all other docs forbid retries
- **Stage 2 Reference:** [internal/poller/poller.go](internal/poller/poller.go#L69-L160)
  - No retry loop inside `PollOnce()`
  - One poll cycle = one pass through all reads
  - On first read error, return immediately with `res.Err`
  - Client invalidation on dead-conn error (lines 200–215) is NOT retrying; it's cleanup
- **Alignment Finding:**
  - **Classification:** ALIGNED (code forbids retries; most docs are correct)
  - **Status:** Architecture doc's "retry logic ownership" is not implemented
  - **Impact:** Architecture doc statement is scope leakage but not enforced in code
  - **Evidence:** poller.go lines 69–160 have no `for { ... if !err { break } }` retry pattern

---

## 3. CODE EXCEEDS DOCUMENTATION

### Finding 3.1: Status Slot Collision Enforcement (Not Fully Documented)
- **Stage 1 Reference:** [docs/device_status_limits.md](docs/device_status_limits.md) → `Responsibility Boundary`, [docs/CONFIG.md](docs/CONFIG.md) → `Status Memory` + `Units`
  - Limits doc forbids artificial limits but does NOT define collision rules
  - CONFIG doc does NOT mention slot collision detection
- **Stage 2 Reference:** [internal/config/validate.go](internal/config/validate.go#L40-L65)
  - Validation key scheme: `fmt.Sprintf("%s|%d|%d", t.Endpoint, *t.StatusUnitID, slot)`
  - Error: "status_slot collision: endpoint=... status_unit_id=... slot=... used by units ... and ..."
  - Code actively rejects any two units writing to same (endpoint, unit_id, slot) tuple
- **Alignment Finding:**
  - **Classification:** OVER-IMPLEMENTED (code enforces rule stronger than docs require)
  - **Status:** Code is more restrictive than documented
  - **Documentation Gap:** docs do not mention collision detection or reservation
  - **Risk:** Users may write configs expecting slot sharing; code rejects it silently
  - **Evidence:** validate.go lines 40–65; test validate_test.go does NOT test collision detection

### Finding 3.2: Transport Counter Injection (Undocumented Data Flow)
- **Stage 1 Reference:** [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md), [docs/Status_Block_Layout.md](docs/Status_Block_Layout.md) describe counter slots but do NOT explain when they are written
- **Stage 2 Reference:** [cmd/replicator/main.go](cmd/replicator/main.go#L137-L161)
  - Every poll cycle, 6 counters are injected into snapshot:
    - `RequestsTotal`, `ResponsesValidTotal`, `TimeoutsTotal`, `TransportErrorsTotal`
    - `ConsecutiveFailCurr`, `ConsecutiveFailMax`
  - Poller owns counter mutation (lines 164–190 of poller.go)
  - Main orchestrator reads counters and copies into status
  - Status writer only advances counters; does not compute them
- **Alignment Finding:**
  - **Classification:** OVER-IMPLEMENTED + UNDECLARED DATA FLOW
  - **Status:** Code implements counter propagation not described in docs
  - **Documentation Gap:** No doc explains that counters are injected into every status write
  - **Risk:** Docs describe counter slots; code documents counter behavior nowhere
  - **Evidence:** 
    - main.go lines 137–161 (injection)
    - poller.go lines 164–190 (mutation)
    - status_writer.go lines 114-147 (write dispatch)
    - No doc describes this flow

### Finding 3.3: Per-Target Status Destination (Implements Migration; CONFIG.md is Stale)
- **Stage 1 Reference:** [docs/CONFIG.md](docs/CONFIG.md) (stale) vs [docs/PER_TARGET_MIGRATION_PLAN.md](docs/PER_TARGET_MIGRATION_PLAN.md) (authoritative)
- **Stage 2 Reference:** [internal/config/config.go](internal/config/config.go#L27-L39)
  - Only TargetConfig has StatusUnitID field
  - No global Status_Memory struct exists
  - Migration is complete; code is past plan stage
- **Alignment Finding:**
  - **Classification:** DOCUMENTATION DRIFT (CONFIG.md outdated; migration complete)
  - **Status:** Implementation exceeds CONFIG.md; aligns with migration plan
  - **Documentation Gap:** CONFIG.md should be deprecated
  - **Risk:** Readers of CONFIG.md will expect global status memory not present in code
  - **Evidence:** 
    - config.go: no Status_Memory type
    - Migration plan explicitly implemented
    - Example (ge_energy_meter/readme.md) still references global "Unit 100"

---

## 4. DOCUMENTATION EXCEEDS CODE (Promises Not Kept)

### Finding 4.1: "MMA Trust Loss" Trigger (Documented; Not Implemented)
- **Stage 1 Reference:** [docs/Status_Rules_Executable.md](docs/Status_Rules_Executable.md) → `4. DEVICE IDENTITY (DEVICE NAME)`
  - "Written ONLY: On startup, On reconnect, On MMA trust loss"
  - Term "MMA trust loss" is used as explicit trigger but NEVER DEFINED anywhere
- **Stage 2 Reference:** [cmd/replicator/main.go](cmd/replicator/main.go#L98-L180), [internal/writer/status_writer.go](internal/writer/status_writer.go#L57-L166)
  - Full block write triggered by `sw.needFull` flag (set to true initially or on write error)
  - No detection of "MMA trust loss" behavior
  - Device name written only on first write and after write failure
  - No ping, heartbeat, or MMA health check
- **Alignment Finding:**
  - **Classification:** DOCUMENTATION EXCEEDS CODE
  - **Status:** Doc promises behavior not implemented
  - **Documentation Gap:** "MMA trust loss" concept exists in docs; not in code
  - **Risk:** Docs set false expectation of robustness
  - **Evidence:** 
    - Status_Rules_Executable line: "On MMA trust loss"
    - Code has no MMA health check
    - Code has no "trust" concept

### Finding 4.2: Device Name Write-Only on Change (Implied; Code Differs)
- **Stage 1 Reference:** [docs/Status_Rules_Executable.md](docs/Status_Rules_Executable.md) → `5. SILENCE RULES`
  - "No change → no write"
  - "Fastest update is no update (RBE++)"
- **Stage 2 Reference:** [internal/writer/status_writer.go](internal/writer/status_writer.go#L153)
  - Device name is written ONLY in full block (never incremental)
  - Device name does NOT use RBE; it is silent in incremental path
  - Changing device name in config does NOT trigger status write until next error+full-block cycle
- **Alignment Finding:**
  - **Classification:** PARTIAL CONTRADICTION
  - **Status:** Code is STRICTER than docs (more silent)
  - **Documentation Gap:** Device name write behavior not explicitly documented
  - **Risk:** Docs claim RBE; code skips RBE for name entirely
  - **Evidence:** status_writer.go line 153 only writes name in `fullBlockRegs()`

### Finding 4.3: Undefined Terms in Executable Spec
- **Stage 1 Reference:** [docs/Status_Rules_Executable.md](docs/Status_Rules_Executable.md)
  - "RBE++" used but never defined
  - "MMA trust loss" used but never defined
- **Stage 2 Reference:** Code does not reference either term
  - Implements RBE pattern but doesn't use the term
  - No "MMA trust loss" detection
- **Alignment Finding:**
  - **Classification:** DOCUMENTATION EXCEEDS CODE (docs make promises code doesn't keep)
  - **Status:** Docs use undefined concepts
  - **Risk:** Unclear what "RBE++" or "MMA trust loss" should do

---

## 5. CLEAN ALIGNMENT ZONES

### Zone 5.1: Packet Format Lock
- **Stage 1:** [docs/raw_ingest_v_1_spec.md](docs/raw_ingest_v_1_spec.md) → `Header size: 10 bytes`, magic bytes, endianness rules
- **Stage 2:** [internal/writer/ingest/client.go](internal/writer/ingest/client.go#L90-L145)
- **Alignment:** Perfect match
- **Evidence:** Header constants (lines 90–91), `buildPacketV1()` (lines 129–145), `packRegisters()` (lines 183–189) all match spec

### Zone 5.2: Health State Transitions (Binary Only)
- **Stage 1:** [docs/Status_Symantics.md](docs/Status_Symantics.md) → "Health answers one question: Did the last poll succeed?"
- **Stage 2:** [cmd/replicator/main.go](cmd/replicator/main.go#L119-L160)
- **Alignment:** Code implements binary rule (OK on success, ERROR on failure)
- **Evidence:** main.go lines 130, 133 only set HealthOK or HealthError; never STALE/DISABLED
- **Note:** Extends 2-state implementation; doesn't contradict documented semantics

### Zone 5.3: Configuration Validation Enforcement
- **Stage 1:** [docs/CONFIG.md](docs/CONFIG.md), [docs/device_status_limits.md](docs/device_status_limits.md)
- **Stage 2:** [internal/config/validate.go](internal/config/validate.go#L11-L117)
- **Alignment:** Memory overlap detection matches documented intent
- **Evidence:** tests in validate_test.go (lines 36–125) verify all declared rules

### Zone 5.4: No Retries (Explicitly Enforced)
- **Stage 1:** README.md, ARCHITECTURE.md, all status docs forbid retries
- **Stage 2:** [internal/poller/poller.go](internal/poller/poller.go#L69-L160) has no retry loop
- **Alignment:** Complete agreement
- **Evidence:** PollOnce() returns immediately on error; no for-loop

### Zone 5.5: Status Block Size (30 Slots)
- **Stage 1:** [docs/Status_Block_Layout.md](docs/Status_Block_Layout.md) defines 30-slot model
- **Stage 2:** [internal/status/constants.go](internal/status/constants.go#L4) locks `SlotsPerDevice = 30`
- **Alignment:** Complete agreement
- **Evidence:** Constants define all 30 slots; encode and write all 30

---

## 6. DOCUMENTATION INSTABILITY IMPACTS

### Impact 6.1: Status Block Scope Boundary (Current vs Canonical)
- **Conflicting Docs:** [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) (3-field "current stage") vs [docs/Status_Block_Layout.md](docs/Status_Block_Layout.md) (30-slot "locked canonical")
- **Implementation Choice:** Code implements 30-slot model
- **Consequence:** 
  - ARCHITECTURE.md is misleading (says "current stage" but code is past it)
  - Readers may plan next stage expecting only 3 fields; code contradicts this
  - Maintenance risk: Future changes may assume only 3 fields are "current"

### Impact 6.2: Health Enum Under-utilization
- **Conflicting Docs:** [docs/Status_Block_Layout.md](docs/Status_Block_Layout.md) (5-state enum defined) vs [docs/Status_Symantics.md](docs/Status_Symantics.md) (binary only)
- **Implementation Choice:** Code defines 5 states; emits only 2
- **Consequence:**
  - STALE and DISABLED states are dead code (never reachable)
  - Clients reading state 3 or 4 will hang (never see them)
  - Docs suggest future capability; code does not provide
  - Confusing for extension: Code has constants but no way to trigger them

### Impact 6.3: Time Accumulation (Forbidden vs Required)
- **Conflicting Docs:** [docs/Status_Symantics.md](docs/Status_Symantics.md) (forbids) vs [docs/Status_Rules_Executable.md](docs/Status_Rules_Executable.md) (requires)
- **Implementation Choice:** Code implements time accumulation
- **Consequence:**
  - Designs based on Status_Symantics will be surprised by incremented counter
  - Status_Rules_Executable is authoritative in code, but no doc declares this
  - Future maintainers may try to remove accumulation per Symantics

### Impact 6.4: Global vs Per-Target Status (Outdated CONFIG.md)
- **Conflicting Docs:** [docs/CONFIG.md](docs/CONFIG.md) (global) vs [docs/PER_TARGET_MIGRATION_PLAN.md](docs/PER_TARGET_MIGRATION_PLAN.md) (per-target)
- **Implementation Choice:** Code is fully per-target
- **Consequence:**
  - Users reading CONFIG.md will write invalid configs expecting Status_Memory
  - Migration plan is complete but not marked as such
  - CONFIG.md remains as normative doc even though it describes removed feature
  - Example (ge_energy_meter/readme.md) still assumes global model

### Impact 6.5: Device Name Behavior (Underspecified)
- **Conflicting Docs:** [docs/Status_Rules_Executable.md](docs/Status_Rules_Executable.md) (says full and incremental possible) vs actual code (name only in full blocks)
- **Implementation Choice:** Code never writes name incrementally
- **Consequence:**
  - Docs suggest RBE applies to device name
  - Code silently skips name in incremental writes
  - Documentation gap: User cannot know when name is re-sent

### Impact 6.6: Transport Counters (Completely Undocumented Data Flow)
- **Missing Documentation:** How/when counters are injected into status
- **Implementation Reality:** Every poll cycle, main.go reads poller counters and injects into snapshot
- **Consequence:**
  - Docs describe slot layout; do not describe data provenance
  - Clients cannot reason about counter staleness (synced every poll? Delayed? Batch?)
  - Future changes to counter behavior will have no documented spec to violate

---

## 7. SUMMARY TABLE: Stage 1 Conflicts vs Stage 2 Implementation

| Stage 1 Conflict | Stage 2 Code Behavior | Classification | Risk |
|---|---|---|---|
| 3-field vs 30-slot status block | Implements 30-slot | UNRESOLVED AUTHORITY IMPACT | HIGH: ARCHITECTURE.md misleading |
| Binary vs 5-state health | Defines 5; emits 2 | OVER-IMPLEMENTED | MEDIUM: Dead code, confusing capability |
| No accumulation vs accumulation required | Accumulates seconds_in_error | UNRESOLVED AUTHORITY IMPACT | HIGH: Contradicts Status_Symantics |
| Global vs per-target status memory | Per-target only | DOCUMENTATION DRIFT | HIGH: CONFIG.md outdated |
| Stale example topology | Example unchanged | DOCUMENTATION DRIFT | MEDIUM: User confusion |
| Packet contract authority (spec vs code) | Both align | ALIGNED | NONE |
| RTU failure code handling ("fixed e.g. 255") | NOT IMPLEMENTED | NOT VERIFIABLE | N/A (RTU not done) |
| Device name "END" vs slots 3–10 | Slots 3–10 | ALIGNED | NONE |
| "MMA trust loss" trigger | Not implemented | DOCUMENTATION EXCEEDS CODE | MEDIUM: False promise |
| "RBE++" terminology | Code does RBE; not "RBE++" | DOCUMENTATION EXCEEDS CODE | LOW: Terminology mismatch |
| Retry ownership (poller) | No retries anywhere | ALIGNED | NONE |
| Slot collision enforcement | Enforced per-target | OVER-IMPLEMENTED | MEDIUM: Underdocumented |
| Device name incremental write | Never incremental | DOCUMENTATION EXCEEDS CODE | LOW: Docs suggest RBE |
| Transport counter flow | Injected every poll | OVER-IMPLEMENTED | HIGH: Undocumented |
| Status write strategy (incremental vs full) | Both; stateful | UNDECLARED BEHAVIOR | MEDIUM: Implicit state machine |

---

## 8. CRITICAL ALIGNMENT RISKS (Ordered by Impact)

### Risk 1: TIME ACCUMULATION (CONTRADICTION)
- **Authority Conflict:** Status_Symantics forbids; Status_Rules_Executable requires
- **Code Implements:** Time accumulation (main.go lines 162–175)
- **Impact:** System behavior contradicts at least one normative doc
- **Mitigation:** Status_Rules_Executable is currently authoritative; Status_Symantics should be deprecated if this is intentional

### Risk 2: CONFIG.MD IS OUTDATED (DOCUMENTATION DRIFT)
- **Normative Doc:** CONFIG.md describes global Status_Memory (removed)
- **Actual Code:** Per-target StatusUnitID only
- **Impact:** Users writing configs based on CONFIG.md will fail
- **Mitigation:** CONFIG.md must be updated to reflect per-target model

### Risk 3: STATUS BLOCK SCOPE BOUNDARY (UNRESOLVED AUTHORITY)
- **Authority Conflict:** ARCHITECTURE.md says "current stage" (3 fields); Status_Block_Layout says "locked canonical" (30 slots)
- **Code Implements:** 30-slot model completely
- **Impact:** Future architecture decisions may assume wrong baseline
- **Mitigation:** ARCHITECTURE.md must clarify that 30-slot is current, not 3-field

### Risk 4: TRANSPORT COUNTER INJECTION (UNDOCUMENTED)
- **Missing Spec:** How/when counters flow from poller → status → MMA
- **Code Does:** Inject every poll cycle
- **Impact:** Clients cannot predict counter staleness; future changes have no spec to violate
- **Mitigation:** Document the counter data flow explicitly

### Risk 5: HEALTH STATES OVER-DEFINED (DEAD CODE)
- **Documentation:** 5-state enum defined (STALE, DISABLED)
- **Code:** Only emits OK/ERROR
- **Impact:** Confusing API; suggests capability not present
- **Mitigation:** Either implement STALE/DISABLED or remove from enum

---

## 9. AUTHORITY PRECEDENCE (As Determined by Code)

The implementation establishes this de-facto precedence:

1. **Highest:** Status_Block_Layout (30-slot model enforced)
2. **Highest:** Status_Rules_Executable (time accumulation enforced)
3. **High:** PER_TARGET_MIGRATION_PLAN (per-target model enforced)
4. **High:** raw_ingest_v_1_spec (packet format enforced)
5. **Medium/Obsolete:** CONFIG.md (global model; superseded)
6. **Medium/Partial:** Status_Symantics (binary rules followed; time rule violated)
7. **Low/Ignored:** ARCHITECTURE.md stage description (code past this stage)
8. **Undefined:** MMA trust loss, RBE++ (terms used; not implemented)

**No Document Declares This Precedence.** Implementation had to make the choice.
