# Authority Hierarchy – Governance Model

**Effective Date:** 2026-03-03

**Status:** ACTIVE

---

## 1. Fundamental Principle

Documentation defines externally observable contracts.

The documented behavior represents the binding specification. Implementation must conform to this specification. External consumers (SCADA systems, MMA appliances, monitoring tools) rely upon documented behavior as the source of truth regarding system capabilities and semantics.

---

## 2. Authority Hierarchy

The following hierarchy establishes which source takes precedence in case of conflict:

### **Level 1: Authoritative Documentation**

**Sources:**
- [Status_Block_Layout.md](./Status_Block_Layout.md) — Device status block structure and emission behavior
- [Status_Rules_Executable.md](./Status_Rules_Executable.md) — Executable specification of status processing rules
- [Status_Symantics.md](./Status_Symantics.md) — Status field semantics and observable behavior
- [raw_ingest_v_1_spec.md](./raw_ingest_v_1_spec.md) — Raw Ingest v1 packet format (locked wire contract)
- [CONFIG.md](./CONFIG.md) — Configuration schema and per-target routing model
- [ARCHITECTURE.md](./ARCHITECTURE.md) — High-level system architecture and component responsibilities

**Authority:** These documents are the primary specification. External consumers depend on these documents.

### **Level 2: Executable Validation Rules**

**Sources:**
- [validate.go](../internal/config/validate.go) — Configuration collision detection and constraint enforcement
- Test suites ([validate_test.go](../internal/config/validate_test.go), [device_status_writer_test.go](../internal/writer/device_status_writer_test.go), [poller_test.go](../internal/poller/poller_test.go))

**Authority:** Validation rules are enforceable checks that must be satisfied during runtime or configuration load.

### **Level 3: Implementation Code**

**Sources:**
- Go source files in `cmd/`, `internal/`
- Runtime behavior of all components

**Authority:** Implementation must conform to Levels 1-2. Implementation is a vehicle for delivering documented contracts, not a source of new behavioral authority.

### **Level 4: Migration Plans**

**Sources:**
- [PER_TARGET_MIGRATION_PLAN.md](./PER_TARGET_MIGRATION_PLAN.md) — Active migration from legacy topology to current model

**Authority:** Explains state transitions during a bounded time window. Migration steps cease to be authoritative once marked COMPLETED.

### **Level 5: Examples and Legacy Documentation**

**Sources:**
- [Examples/](../Examples/) — Sample configurations
- Deprecated documentation with explicit legacy notices

**Authority:** Illustrative only. Not binding on system behavior.

---

## 3. Change Control

### Rule: Behavioral Change Requires Concurrent Documentation Update

When implementation changes must alter system behavior:

1. **Before implementation commit:**
   - Update all applicable authoritative documentation (Level 1)
   - Update validate.go constraints if applicable
   - Tag documentation with version date and change context

2. **Implementation follows documentation:**
   - Code changes implement the documented behavior exactly
   - No undocumented behavior extensions

3. **After merge:**
   - Validation rule updates (Level 2) take effect immediately
   - External consumers must be notified if observable contract changes

### Example:

If seconds_in_error accumulation rules change, **both**:
- Status_Rules_Executable.md must be updated, AND
- status_writer.go must implement the new rule

Updating only one source creates drift and breaks the contract.

---

## 4. Drift Prevention

### Rule: Undocumented Behavior Is Invalid

- If a feature is implemented but not documented in Level 1 sources, that feature must be removed or documented.
- If documentation specifies behavior but implementation does not provide it, the implementation is incomplete and must be corrected.
- Periodic audits verify alignment (see Section 6).

### Example:

If [ARCHITECTURE.md](./ARCHITECTURE.md) states "Poller maintains consecutive failure tracking" but no such code exists, then:
- Either implement the feature, OR
- Remove the statement from documentation

No orphaned specifications or hidden behavior.

---

## 5. Migration Governance

### ACTIVE Migrations

Migrations with status **ACTIVE** are in-progress transitions that override normal authority rules **within their bounded scope**.

**Current ACTIVE Migrations:** None

When an ACTIVE migration exists, its rules take temporary precedence for the affected components. See [PER_TARGET_MIGRATION_PLAN.md](./PER_TARGET_MIGRATION_PLAN.md) for context.

### COMPLETED Migrations

Once a migration is marked **COMPLETED**, it ceases to be authoritative. Its previous rules are retired, and normal authority hierarchy resumes.

**Example:** The global Status_Memory model (pre-2026-03-03) is marked COMPLETED and superseded by per-target routing. Legacy documentation is retained for historical context only and does not govern behavior.

---

## 6. Audit Lifecycle

### Audit Types

#### 6.1 Consistency Audit
- **Purpose:** Identify contradictions within documentation itself
- **Scope:** All Level 1 and Level 4 documents
- **Frequency:** Quarterly or on major release
- **Output:** List of contradictions marked with version date
- **Example:** [DOCUMENTATION_CONSISTENCY_AUDIT_2026-03-03.md](./DOCUMENTATION_CONSISTENCY_AUDIT_2026-03-03.md)

#### 6.2 Implementation Audit
- **Purpose:** Verify implementation matches stated capabilities
- **Scope:** Code files vs. Level 1-2 specifications
- **Frequency:** Before release or after major component changes
- **Output:** List of mismatches (missing features, extra behavior, wrong semantics)
- **Example:** [IMPLEMENTATION_AUDIT_2026-03-03.md](./IMPLEMENTATION_AUDIT_2026-03-03.md)

#### 6.3 Alignment Audit
- **Purpose:** Cross-reference documentation and implementation to confirm conformance
- **Scope:** Verification that drift-prevention rule is satisfied
- **Frequency:** After applying patches to documentation or implementation
- **Output:** Domain-by-domain verdict (ALIGNED / CONTRADICTION / UNDER-SPECIFIED / OVER-SPECIFIED / NOT VERIFIABLE)
- **Example:** [ALIGNMENT_AUDIT_2026-03-03.md](./ALIGNMENT_AUDIT_2026-03-03.md)

### Audit Resolution

When an audit identifies a gap:

1. **Consistency gaps** → Update documentation to remove contradictions
2. **Implementation gaps** → Update code to match documentation OR update documentation to match code (per authority hierarchy)
3. **Alignment gaps** → Apply both fixes above until ALIGNED verdict is reached

Audits are versioned by date and retained for historical traceability.

---

## 7. Authority Scope Boundaries

### In Scope

Authority applies to:
- Device status block structure and semantics
- Status write behavior and collision handling
- Packet format specification (Raw Ingest v1)
- Configuration schema and validation rules
- Per-target routing and destination selection
- Transport counter lifecycle and injection timing
- Poller behavior (no retries, error propagation, counter maintenance)

### Out of Scope

Authority does not govern:
- Internal implementation choice details (e.g., which Go packages, internal function structure)
- Test framework choices
- Performance optimization details
- Logging verbosity
- Deployment topology outside documented YAML schema

---

## 8. Restoration Completion

This document supersedes all temporary authority declarations in prior documentation versions.

**Effective March 3, 2026:**
- Code authority phase is concluded.
- Documentation authority is permanently restored.
- All authoritative documents are marked with version date 2026-03-03 and synchronized.
- Alignment verification confirmed zero contradictions across all domains.

Future changes are governed exclusively by Section 3 (Change Control).

---

## 9. External Communication

### For SCADA/Monitoring Systems

"The Modbus Replicator is governed by its documentation. Behavior is guaranteed to match documented specifications. If observed behavior differs, report as a bug."

### For Contributors

"Update documentation first when changing behavior. Code implements documented contracts. Undocumented behavior will be removed or documented."

### For Operators

"Documented configuration topology is the binding contract. Undocumented features are unsupported."
