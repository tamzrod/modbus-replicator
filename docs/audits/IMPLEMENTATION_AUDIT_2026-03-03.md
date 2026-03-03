---
**Status:** ARCHIVAL  
**Purpose:** Historical audit artifact  
**Non-Normative:** This document does NOT define system behavior.  
**Authority:** Documentation in root /docs defines behavior.  
**Date Archived:** 2026-03-03  
---

# Implementation Audit — Strict Code-Only Analysis (Archived)

Date: 2026-03-03  
Method: Read-only source code inspection, no documentation referenced, no intent assumed.  
Scope: All 22 .go files, 1,629 lines total production code.

---

## PHASE 1: SYSTEM BEHAVIOR MODEL (From Code Only)

### Data Flow

```
Source Device
    ↓
Poller (internal/poller)
    ├── TCP Modbus read (internal/poller/modbus/client.go)
    ├── Counter mutation (internal/poller/poller.go)
    └── PollResult emit
        ↓
Main orchestrator (cmd/replicator/main.go, lines 61–180)
    ├── Data path: Writer (internal/writer/writer.go)
    │   └── Raw Ingest v1 packets → Endpoint targets
    │
    └── Status path: StatusWriter (internal/writer/status_writer.go)
        ├── Full block assertion (needFull flag)
        ├── Incremental slot updates
        └── Raw Ingest v1 packets → Status endpoints
```

### Control Flow

1. **Startup** (main.go:17–60):
   - Load YAML config (config.go)
   - Validate config (validate.go)
   - Normalize config (normalize.go)
   - Build poller per unit (poller/builder.go)
   - Build writers per unit (writer/builder.go)
   
2. **Per-Unit Runtime** (main.go:96–180):
   - Goroutine spawned per unit
   - Status snapshot mutated in goroutine
   - Status writes triggered by:
     - Poll result arrival (`case res := <-out`)
     - 1-second ticker (increments `SecondsInError`)
   - **CRITICAL: Status snapshot is mutable local state in goroutine** (line 106)
   
3. **Config Contract** (config.go):
   - `StatusSlot` is optional pointer (`*uint16`)
   - If set, `StatusUnitID` required on every target
   - Per-target status destination (PER_TARGET_MIGRATION_PLAN implemented)
   - **Global status memory completely removed from code**

### State Mutation Points

| Location | State | Mutated By | Frequency | Side Effects |
|----------|-------|-----------|-----------|--------------|
| `snap` (main.go:106) | status.Snapshot | ErrorCode check, counter injection, ticker | Per poll + 1Hz | Sent to all status writers |
| `p.client` (poller.go:35) | Modbus TCP Client | `maybeInvalidateClient()` | On dead-conn errors | Next poll recreates |
| `p.counters` (poller.go:36-40) | TransportCounters struct | `recordSuccess()`, `recordFailure()` | Every poll | Injected into status snapshot |
| `sw.needFull` (status_writer.go:23) | bool flag | Set true on write failure | On error | Triggers full block re-write next cycle |
| `sw.last` (status_writer.go:24) | status.Snapshot | Incremental slot updates | Per changed field | Used for RBE (receive-based-on-change) logic |
| `sw.nameRegs` (status_writer.go:25) | uint16 array | Encoded once at init | Once at New() | Only written during full block |

### Invariants (Enforced Where?)

| Invariant | Enforced At | Mechanism | Notes |
|-----------|------------|-----------|-------|
| Status block = 30 slots | constants.go | Const `SlotsPerDevice = 30` | ENCODED IN CODE, NOT MUTABLE |
| Health ∈ {0,1,2,3,4} | constants.go | Named constants | Values: HealthUnknown(0), HealthOK(1), HealthError(2), HealthStale(3), HealthDisabled(4) |
| *No* global status memory | config.go + validate.go | Removed; only per-target StatusUnitID | Config enforces all targets have status_unit_id when status_slot set |
| Slot collision detection | validate.go:40–65 | Map key: `endpoint\|status_unit_id\|slot` | Returns error if collision detected |
| Time counter saturation | status_writer.go:62 | Clamp: `if s.SecondsInError > 65535` | Saturates at max uint16 |
| Packet magic bytes | ingest/client.go:90–91 | Constants 0x52 0x49 ('RI') | Locked in code |
| Packet header size | ingest/client.go:128–145 | 10 bytes fixed | No length field in packet |
| Big-endian registers | ingest/client.go:183–189 | `packRegisters()` puts bytes high-first | Network byte order |
| All-or-nothing poll | poller.go:69–160 | Return on first block error | Single `Err` field, no partial success |
| No explicit retries | poller.go:69–160 | No retry loop inside `PollOnce()` | Client invalidated on dead conn; next tick recreates |

### Error Propagation

| Layer | Error Source | Error Type | Propagation |
|-------|-------------|-----------|-------------|
| Poller | Modbus device | `error` interface, examined for `Code()` method | Stored in `PollResult.Err`; also in `RawErrorCode` if error has `Code()` method |
| Poller | Transport (TCP) | `net.Error`, string pattern match | Dead-conn errors trigger client invalidation; all errors logged to counters |
| Writer | Ingest protocol | `error` interface | Accumulated in `errs []string`, joined and returned; **data write errors do NOT block other targets** |
| Status | Write failure | `error` interface | Sets `needFull = true` on any error; next write retries full block |

---

## PHASE 2: STRUCTURAL FINDINGS

### 1) RESPONSIBILITY VIOLATION: Status Snapshot Mutation in Main Orchestrator
- **File:** [cmd/replicator/main.go](cmd/replicator/main.go#L96-L180)
- **Type:** RESPONSIBILITY VIOLATION
- **Severity:** HIGH
- **Description:**
  - Status snapshot is mutable local state in a goroutine (line 106: `snap := status.Snapshot{...}`)
  - Mutated directly in response to poll result and 1-second ticker
  - No encapsulation; all fields readable/writable
  - Represents "status orchestration logic" that should be localized to a dedicated layer
  - Currently split between main.go and status_writer.go
- **Lines:** 106–180
- **Evidence:**
  - Lines 121–160: Health logic (hardcoded state transitions)
  - Lines 123–161: Counter injection from poller
  - Lines 162–175: SecondsInError increment on ticker
  - Lines 166–180: Conditional write dispatch

### 2) DUPLICATE LOGIC: Health State Transitions (Two Locations)
- **File 1:** [cmd/replicator/main.go](cmd/replicator/main.go#L119-L160)
- **File 2:** [internal/writer/status_writer.go](internal/writer/status_writer.go#L120-L165)
- **Type:** DUPLICATED LOGIC
- **Severity:** MEDIUM
- **Description:**
  - Health state transition logic repeated in two places:
    - main.go lines 119–160: Sets `snap.Health` and resets counters on error/recovery
    - status_writer.go lines 120–165: Slot-by-slot change detection for Health field
  - Both perform "did Health change?" logic
  - Both write only if changed (RBE pattern)
  - Creates multiple points of failure if semantics drift
- **Evidence:**
  - main.go line 130: `if snap.Health != status.HealthOK`
  - status_writer.go line 120: `if sw.last.Health != s.Health`
  - Both use same comparison pattern

### 3) HIDDEN STATE MUTATION: Transport Counters Injected into Status
- **File:** [cmd/replicator/main.go](cmd/replicator/main.go#L137-L161)
- **Type:** UNDECLARED BEHAVIOR
- **Severity:** MEDIUM
- **Description:**
  - Every poll cycle, 6 transport counters are injected into status snapshot (lines 137–161)
  - Values come from poller via `p.Counters()` call (line 137)
  - Change detection is separate from poll logic (lines 136–161)
  - Status writer sees these as regular snapshot fields needing RBE
  - This behavior is not declared as part of status layer responsibility
  - Poller counters are the ONLY data owned by poller that escapes to status
- **Lines:** 137–161 (injection); status_writer.go 114–164 (write dispatch)
- **Evidence:**
  - Line 137: `c := p.Counters()`
  - Lines 139–161: 6 separate if-blocks checking counter change

### 4) INCOMPLETE INVARIANT ENFORCEMENT: Client Invalidation Heuristic
- **File:** [internal/poller/poller.go](internal/poller/poller.go#L218-L245)
- **Type:** UNENFORCED INVARIANT
- **Severity:** MEDIUM
- **Description:**
  - Dead-connection detection is string-pattern-based matching against error message (lines 237–245)
  - Conservative heuristic (returns false on timeout, true on EOF, "broken pipe", "connection reset", etc.)
  - Patterns are hard-coded and not externally configurable
  - Not all transport errors trigger invalidation (e.g., "address in use" does not)
  - Leads to implicit behavior: timeouts do NOT invalidate client; connection errors DO
  - No clear specification of what constitutes a "dead" connection
- **Lines:** 218–245
- **Evidence:**
  - Line 239: `if strings.Contains(s, "eof")` (case-insensitive)
  - Line 234: `var ne net.Error` check explicitly excludes timeouts
  - Lines 242–245: Windows-specific patterns (wsasend, wsarecv, forcibly closed)

### 5) MIXED RESPONSIBILITY: Packet Composition in Ingest Client
- **File:** [internal/writer/ingest/client.go](internal/writer/ingest/client.go#L79-L145)
- **Type:** RESPONSIBILITY VIOLATION
- **Severity:** LOW
- **Description:**
  - Ingest client owns both:
    - Transport (TCP connection, timeout, retry on new conn)
    - Packet composition (header layout, magic bytes, encoding)
  - Packet composition should be a separate concern
  - Makes reusing packet logic for other transports harder
  - However, current usage (one packet per connection) justifies co-location
- **Lines:** Packet building: 129–145; Transport: 79–110
- **Evidence:**
  - Lines 90–91: Magic bytes and version
  - Lines 128–145: Full header and payload assembly
  - Lines 79–110: Connection, write, read, close

### 6) GOD FILE: polling orchestrator logic + main
- **File:** [cmd/replicator/main.go](cmd/replicator/main.go)
- **Type:** STRUCTURAL DRIFT (combined orchestrator concern)
- **Severity:** MEDIUM
- **Lines:** 17–179 (179 lines)
- **Description:**
  - Single main function does:
    - Config loading (delegated)
    - Error code extraction (`errorCode()` function, lines 188–201)
    - Per-unit orchestration (lines 43–95: build steps)
    - Status snapshot lifecycle (lines 96–180: goroutine)
  - Status orchestration mixes:
    - Poll result handling
    - Ticker handling for seconds-in-error
    - Health state machines
    - Counter injection
    - Conditional write dispatch
  - Should be extracted to dedicated status orchestrator or runner type
- **Evidence:**
  - Lines 96–180: All status logic in single goroutine closure
  - Line 106: Initial snapshot assignment
  - Lines 121–180: All state transitions and dispatch

### 7) SILENT FALLBACK: Error Code Extraction with Multiple Interfaces
- **File:** [cmd/replicator/main.go](cmd/replicator/main.go#L188-L201)
- **Type:** UNDECLARED BEHAVIOR
- **Severity:** LOW
- **Description:**
  - `errorCode()` function tries THREE different interfaces to extract error code:
    1. `.Code()` (poller/modbus/client.go)
    2. `.ErrorCode()`
    3. `.ModbusCode()`
  - Falls back to `return 1` if none match
  - No logging of fallback; silent behavior
  - Means any `error` that doesn't implement one of these interfaces becomes code `1`
  - Exact interface chosen is not documented
- **Lines:** 188–201
- **Evidence:**
  - Line 193: `var coderA interface{ Code() uint16 }` (three variants)
  - Lines 195–205: Serial checking
  - Line 206: `return 1` (silent fallback)

### 8) DEAD CODE: Normalize() Function
- **File:** [internal/config/normalize.go](internal/config/normalize.go)
- **Type:** DEAD CODE
- **Lines:** 7–26 (full function)
- **Description:**
  - `Normalize()` function is defined but NOT called anywhere
  - Grep shows no invocation in any file
  - Contains logic to truncate device_name to 16 chars
  - Device name is validated in `config/validate.go` and enforced in `status_writer.go`
  - Function may be vestigial from incomplete refactoring
- **Impact:** None (unused)
- **Evidence:**
  - Called in main.go: No
  - Called in builder: No
  - Called in tests: No

### 9) ORPHANED INTERFACE: endpointClient (No Unified Definition)
- **File:** [internal/writer/writer.go](internal/writer/writer.go#L15-L18) and [internal/writer/status_writer.go](internal/writer/status_writer.go) and [internal/writer/ingest/client.go](internal/writer/ingest/client.go)
- **Type:** DUPLICATED LOGIC
- **Severity:** LOW
- **Description:**
  - `endpointClient` interface is defined at line 15 of writer.go
  - Implemented by `*ingest.EndpointClient`
  - ALSO implicitly required by `*status_writer.deviceStatusWriter`
  - Both call same two methods: `WriteBits()`, `WriteRegisters()`
  - Interface is not explicitly documented as the contract
  - Type assertion / implementation contract is implicit, not declared
- **Lines:** writer.go:15–18 (definition); status_writer.go:52 (usage); ingest/client.go:53–75 (implementation)
- **Evidence:**
  - writer.go line 15: `type endpointClient interface`
  - status_writer.go line 52: `cli endpointClient` parameter
  - ingest/client.go lines 53–75: Implements both methods

### 10) PARTIAL BOUNDS CHECKING: Target Unit ID
- **File:** [internal/writer/writer.go](internal/writer/writer.go#L44-L50)
- **Type:** INCOMPLETE CONTRACT
- **Severity:** MEDIUM
- **Description:**
  - Target `ID` is stored as `uint32` in config (config.go line 36)
  - Writer checks if ID > 255 (lines 44–50)
  - Returns error if out of range
  - However, config struct accepts ANY uint32
  - Test explicitly covers this error case (writer_test.go:232–268)
  - No validation at config load/validate time; only at write time
  - Means invalid configs are accepted and rejected at runtime
- **Lines:** writer.go:44–50; config.go:36
- **Evidence:**
  - writer.go line 44: `if tgt.TargetID > 255`
  - config.go line 36: `TargetID uint32`
  - No pre-validation in validate.go

### 11) UNDECLARED STATUS WRITE BEHAVIOR: Incremental Updates with Full Fallback
- **File:** [internal/writer/status_writer.go](internal/writer/status_writer.go#L57-L166)
- **Type:** UNDECLARED BEHAVIOR
- **Severity:** MEDIUM
- **Description:**
  - Status writer performs TWO different write strategies:
    1. **Full block** (first write or after error): All 30 slots
    2. **Incremental** (subsequent writes): Only changed fields
  - Strategy is stateful: `sw.needFull` flag (line 23)
  - On ANY write error, `needFull` is set true (line 163)
  - Device name is written ONLY in full block, never incremental (line 153)
  - Transport counters are always written if changed (lines 130–147)
  - No documentation explains these two branches; logic is implicit
- **Lines:** 57–166
- **Evidence:**
  - Line 65: `if sw.needFull` (branch 1)
  - Line 73: Full block write
  - Line 75: `sw.needFull = false`
  - Line 163: `sw.needFull = true` on error
  - Line 153: Device name only in `fullBlockRegs()`

---

## PHASE 3: BEHAVIORAL TRUTH TABLE

### Status Block Layout (Implemented)

| Slot(s) | Field | Type | Storage | Write Frequency | Notes |
|---------|-------|------|---------|-----------------|-------|
| 0 | health_code | uint16 | 1 register | On change | Values: 0 (Unknown), 1 (OK), 2 (Error), 3 (Stale), 4 (Disabled) |
| 1 | last_error_code | uint16 | 1 register | On change | Raw Modbus exception or transport error code |
| 2 | seconds_in_error | uint16 | 1 register | On change + 1Hz if error | Saturates at 65535 |
| 3–10 | device_name | 8×uint16 | 8 registers | Full block only | ASCII name, 16 chars max, packed 2 bytes per register |
| 11–19 | RESERVED | N/A | 9 registers | Never written | Zero-initialized in full block |
| 20–21 | requests_total | uint32 | 2 registers (low, high) | On change | Lifetime monotonic counter |
| 22–23 | responses_valid_total | uint32 | 2 registers | On change | Lifetime monotonic counter |
| 24–25 | timeouts_total | uint32 | 2 registers | On change | Incremented when net.Error.Timeout() == true |
| 26–27 | transport_errors_total | uint32 | 2 registers | On change | All non-timeout errors |
| 28 | consecutive_fail_current | uint16 | 1 register | On change | Resets to 0 on success; incremented on failure |
| 29 | consecutive_fail_max | uint16 | 1 register | On change | High-water mark of consecutive fail |

**Total: 30 slots = 60 registers per device (if offset 0)**

### Raw Ingest v1 Packet Layout (Implemented)

| Offset | Size | Field | Value | Encoding |
|--------|------|-------|-------|----------|
| 0–1 | 2 | Magic | 0x5249 | ASCII "RI" |
| 2 | 1 | Version | 0x01 | Fixed |
| 3 | 1 | Area | 1–4 | FC selector (1=coils, 2=Discrete, 3=Holdings, 4=Inputs) |
| 4–5 | 2 | UnitID | 0–255 | Big-endian uint16 (can exceed 255 but only uint8 used) |
| 6–7 | 2 | Address | 0–65535 | Big-endian uint16 |
| 8–9 | 2 | Count | 1–65535 | Big-endian uint16 (number of bits or registers) |
| 10+ | N | Payload | Raw bytes | Bits: LSB-first, byte-padded; Registers: big-endian |

**Header: 10 bytes (fixed)**  
**No length field in packet**

### Health State Machine (Implemented in main.go:119–160)

```
All states are deterministic based on poll result:
- If res.Err == nil:
  - health = HealthOK (1)
  - lastErrorCode = 0
  - secondsInError = 0
- If res.Err != nil:
  - health = HealthError (2)
  - lastErrorCode = errorCode(res.Err)
  - secondsInError unchanged (incremented by ticker)

Stale (3) and Disabled (4) states are NOT emitted by code.
```

### Status Write Behavior (Implemented in status_writer.go:57–166)

| Condition | Action | Block Size | Includes Name | Notes |
|-----------|--------|-----------|---|---------|
| First write | Full block | 30 slots | YES | `needFull` initialized true |
| Any field changed | Incremental | 1–2 slots | NO | RBE pattern; device name never incremental |
| Write error | Set needFull=true | 0 | N/A | Retries full block next cycle |
| SecondsInError increments | Slot 2 update | 1 slot | NO | Only if health != OK and value < 65535 |

### Timeout Handling (Implemented in poller.go:170–190)

```
On poll error:
  - p.recordFailure(err) called
  - If err is net.Error AND timeout: counters.TimeoutsTotal++
  - Else: counters.TransportErrorsTotal++
  - counters.ConsecutiveFailCurr++
  - If ConsecutiveFailCurr > ConsecutiveFailMax: update max

Client invalidation (maybeInvalidateClient):
  - TRUE if error contains: EOF, broken pipe, connection reset, aborted, closed, forcibly closed, wsasend, wsarecv
  - FALSE if err is timeout or doesn't match patterns
   
Next poll:
  - If client is nil, factory() called once
  - On factory error, poll fails
  - No retry inside PollOnce()
```

### Config Validation (Implemented in validate.go:11–117)

| Check | Location | Triggers Error |
|-------|----------|---|
| device_name ASCII only | Lines 26–32 | Non-ASCII byte found |
| status_slot requires targets | Lines 49–53 | status_slot set but no targets |
| each target needs status_unit_id | Lines 54–60 | status_slot set but target missing status_unit_id |
| status slot collision detection | Lines 40–65 | Two units use same (endpoint, status_unit_id, slot) |
| memory overlap detection | Lines 74–116 | Two reads write to overlapping addresses in same target |

---

## PHASE 4: RISK ZONES

### ZONE 1 — Status Orchestration (HIGHEST RISK)
- **Location:** [cmd/replicator/main.go](cmd/replicator/main.go#L96-L180)
- **Risk Type:** Hidden state mutation + implicit concurrency
- **Concerns:**
  - Mutable snapshot in goroutine
  - Two control paths (poll result + ticker) both mutate same struct
  - No mutexes, no channels; direct field writes
  - Run/test failures may be concurrency-related but appear as data issues
  - Status snapshot is shared between main loop and all per-target writers
- **Mitigation Difficulty:** HIGH (requires refactoring to dedicated state machine)
- **Example Failure Mode:**
  - Goroutine increments `SecondsInError` while writer reads it
  - Writer fails, sets `needFull=true`
  - Poll result arrives before main loop resets snapshot
  - Stale data or memory corruption possible in long-running process

### ZONE 2 — Error Code Extraction (IMPLICIT BEHAVIOR)
- **Location:** [cmd/replicator/main.go](cmd/replicator/main.go#L188-L201)
- **Risk Type:** Silent fallback to hardcoded default
- **Concerns:**
  - Three different error type assertions tried in sequence
  - Fallback to `1` if none match
  - No logging of which path was taken
  - Poller owns only `.Code()` interface; others are undefined
  - Future error types added upstream will silently use code `1`
- **Mitigation Difficulty:** MEDIUM
- **Example Failure Mode:**
  - Library updates error type
  - Code `1` no longer represents the actual error
  - Status field contains misleading value
  - No alerting mechanism

### ZONE 3 — Client Invalidation Heuristic (IMPLICIT PATTERNS)
- **Location:** [internal/poller/poller.go](internal/poller/poller.go#L218-L245)
- **Risk Type:** String pattern matching for transport state
- **Concerns:**
  - Conservative but may miss new transport errors
  - Windows-specific patterns mixed with generic ones
  - Case-insensitive matching (ToLower)
  - No spec for what constitutes "dead"
  - Timeout errors explicitly excluded (may cause connection reuse on timeout)
- **Mitigation Difficulty:** MEDIUM
- **Example Failure Mode:**
  - New OS or library changes error message text
  - Connection becomes "dead" but not recognized
  - Next poll reuses poisoned client → cascading failures
  - OR: Connection is valid but pattern matches stale text → unnecessary reconnect

### ZONE 4 — Incremental Status Writes with Full Fallback (STATE DEPENDENT)
- **Location:** [internal/writer/status_writer.go](internal/writer/status_writer.go#L57-L166)
- **Risk Type:** Stateful write behavior with implicit failure recovery
- **Concerns:**
  - Two different write paths (full vs incremental) based on flag
  - Device name silently skipped in incremental write
  - On error, needFull flag set but device identity is NOT re-asserted immediately
  - Next successful write writes full block (including old name if no change)
  - Only 1 write attempt per snapshot; on failure, next snapshot triggers retry
- **Mitigation Difficulty:** MEDIUM
- **Example Failure Mode:**
  - Device name changes in config
  - Status writer doesn't write name until next error + full block
  - MMA sees stale device identity for up to N poll cycles
  - Clients may make decisions based on stale name

### ZONE 5 — Transport Counter Injection (HIDDEN COUPLING)
- **Location:** [cmd/replicator/main.go](cmd/replicator/main.go#L137-L161)
- **Risk Type:** Tight coupling between poller and status
- **Concerns:**
  - Poller owns transport counters
  - Main orchestrator reads counters every poll cycle
  - Status writer must know they're included in snapshot
  - If poller stops updating counters, status layer unaware
  - Three different sections duplicating change detection
- **Mitigation Difficulty:** MEDIUM
- **Example Failure Mode:**
  - Poller counter logic broken silently
  - Status block shows stale counter values
  - Clients believe polling is healthy when it's not

---

## PHASE 5: CLEAN SEPARATION ZONES

### ZONE A — Modbus TCP Client (CLEAN)
- **File:** [internal/poller/modbus/client.go](internal/poller/modbus/client.go)
- **Line Range:** 1–252
- **Assessment:** WELL-ISOLATED
- **Reasons:**
  - Single responsibility: Modbus TCP reads only
  - No side effects except connection state
  - No hidden state mutation
  - Exception type preserves raw protocol values
  - Clean interface to poller layer (`Client` interface)
  - No knowledge of status, slots, or orchestration

### ZONE B — Packet Composition (LOCALIZED)
- **File:** [internal/writer/ingest/client.go](internal/writer/ingest/client.go)
- **Packet building:** Lines 129–189
- **Assessment:** ADEQUATE (small scope)
- **Reasons:**
  - Packet assembly is concentrated in `buildPacketV1()` and helpers
  - Constants locked (magic, version)
  - No packet spec documented in code comments, but layout is clear
  - Transport (connection) and packet composition co-located (acceptable for 1-packet-per-conn model)
  - No implicit behavior; all encoding deterministic

### ZONE C — Config Validation (CLEAR SEMANTICS)
- **File:** [internal/config/validate.go](internal/config/validate.go)
- **Line Range:** 11–117
- **Assessment:** WELL-DEFINED
- **Reasons:**
  - Two independent checks: status slots + memory overlap
  - Collision detection uses explicit key scheme
  - Overlap detection is geometric (no implicit policy)
  - No silent failures; all errors explicitly returned
  - Tests cover edge cases (touching ranges, different endpoints/memory/FC)

### ZONE D — Status Block Encoding (DETERMINISTIC)
- **File:** [internal/status/encode.go](internal/status/encode.go)
- **Line Range:** 1–30
- **Assessment:** PURE (no side effects)
- **Reasons:**
  - Takes snapshot in, returns `[]uint16` out
  - No mutation, no I/O, no state
  - Slot indices hard-coded as constants
  - Encoding rules (uint32 low-high, big-endian) deterministic
  - Used only by status writer and tests

### ZONE E — Constants (LOCKED)
- **File:** [internal/status/constants.go](internal/status/constants.go)
- **Line Range:** 1–69
- **Assessment:** IMMUTABLE SOURCE OF TRUTH
- **Reasons:**
  - All slot indices defined here
  - All health codes defined here
  - Prevents magic numbers scattered across files
  - Device name size, block size constants locked
  - Encoding rules documented as constants

### ZONE F — Data Structure Definitions (PASSIVE)
- **Files:**
  - [internal/config/config.go](internal/config/config.go) (47 lines)
  - [internal/poller/types.go](internal/poller/types.go) (48 lines)
  - [internal/writer/types.go](internal/writer/types.go) (31 lines)
  - [internal/status/snapshot.go](internal/status/snapshot.go) (22 lines)
- **Assessment:** PASSIVE (struct definitions only)
- **Reasons:**
  - No logic, no side effects
  - Types clearly named and documented
  - Composition is intentional (e.g., Plan contains Targets)

---

## Summary

### High-Complexity Areas
1. **main.go orchestrator** (179 lines): Status snapshot lifecycle + orchestration
2. **poller.go** (215 lines): Client management + counter mutation + dead-conn detection
3. **status_writer.go** (208 lines): Full vs incremental write strategy + slot-level updates
4. **modbus/client.go** (212 lines): Modbus TCP protocol + request/response parsing

### Areas with Implicit Behavior
- Error code extraction (3 interface types, silent fallback)
- Client invalidation heuristic (string patterns)
- Incremental update strategy (stateful flag, silent name skip)
- Transport counter injection (every poll without documentation)
- Health state machine (hardcoded in main.go)

### Areas NOT Verifiable from Code
- Whether `Normalize()` was intentionally left unused (dead code?)
- Whether status snapshot goroutine races have been tested
- Whether all platform-specific error patterns in `isDeadConnErr()` are sufficient
- Whether the 3-interface error extraction covers all library versions
- Meaning of HealthStale and HealthDisabled (never emitted)

### No Evidence of
- Hidden retries (retry logic absent)
- Implicit fallbacks to defaults (obvious fallback in error code extraction)
- Silent data corruption (bounds checking present)
- Orphaned resources (all connections closed)
- Dead goroutines (all tied to context cancellation)
