---
**Status:** ARCHIVAL
**Purpose:** Historical behavioral reference audit (pre-refactor, per-device interval architecture)
**Scope:** internal/poller — snapshot taken against 2026-03-09 codebase
**Date:** 2026-03-09
**Archived:** 2026-03-11 — superseded by per-read interval refactor
---

> **ARCHIVAL NOTICE:** This audit was written against the pre-refactor architecture in which polling was
> scheduled at the **device level** via a single `poll.interval_ms` field, all read blocks shared one
> interval, and `PollOnce()` executed all reads as a single all-or-nothing cycle.
>
> The architecture has since changed. The current implementation uses **per-read scheduling**:
> each read block carries its own `interval_ms`, is scheduled independently, and produces its own
> `PollResult` via `executeSingleRead()`. The device-level `poll.interval_ms` field is rejected at
> validation. See `docs/ARCHITECTURE.md` and `docs/CONFIG.md` for the normative current model.
>
> Invariants **I-2**, **I-3**, **I-4**, **I-6**, **I-7**, and **I-8** remain valid.
> Invariant **I-5** ("Polling interval is per-device") is superseded — interval is now per-read-block.
> Code line references throughout this document point to the pre-refactor codebase and may be stale.

# Replicator Engine Behavior Audit

Date: 2026-03-09
Method: Read-only source code inspection.  No documentation assumed.  All claims cite specific files and line numbers.
Scope: `internal/poller` (poller.go, runner.go, types.go, builder.go, modbus/client.go) and `cmd/replicator/main.go` orchestration layer.

---

## PHASE 1 — Core Engine Components

### File Inventory

| File | Role |
|------|------|
| `internal/poller/poller.go` | Core poll execution logic: connection management, read dispatch, counter mutation, client invalidation |
| `internal/poller/runner.go` | Poll loop scheduler: wraps `time.Ticker`, drives `PollOnce()` on every tick |
| `internal/poller/types.go` | Data types: `ReadBlock`, `BlockResult`, `PollResult`, `TransportCounters` |
| `internal/poller/builder.go` | Factory: constructs a `Poller` from `UnitConfig`; wires the Modbus TCP factory |
| `internal/poller/modbus/client.go` | Modbus TCP adapter: TCP dial, ADU construction, response decoding, exception handling |
| `cmd/replicator/main.go` | Orchestrator: per-unit goroutines, `out` channel, status mutation, writer dispatch |
| `internal/writer/writer.go` | Data writer: maps `PollResult` blocks to Raw Ingest write calls |
| `internal/writer/ingest/client.go` | Raw Ingest v1 client: stateless 1-connection-per-write TCP sender |

### Component Roles

**`Poller` struct** (`internal/poller/poller.go:31`)
Central execution unit per device.  Holds the current `Client`, the factory, immutable config, and passive counters.  The only exported method that performs work is `PollOnce()`.

**`Runner`** (`internal/poller/runner.go:10`)
The `Run()` method attaches a `time.Ticker` to a `Poller` and pumps results into an output channel.  This is the entry point for the scheduler.

**`modbus.Client`** (`internal/poller/modbus/client.go:37`)
A stateful, connection-oriented adapter.  Opened by the factory.  Discarded on dead-connection errors.  Not goroutine-safe by design.

---

## PHASE 2 — Polling Model

### Scheduling Granularity

**Polling is scheduled per device (per unit), not per read block.**

The `Config` struct carries a single `Interval` field and a flat list of `ReadBlock` items:

```go
// internal/poller/types.go — ReadBlock
type ReadBlock struct {
    FC       uint8
    Address  uint16
    Quantity uint16
}

// internal/poller/poller.go — Config
type Config struct {
    UnitID   string
    Interval time.Duration
    Reads    []ReadBlock    // all reads share this interval
}
```

There is no per-read-block interval.  All reads belonging to a unit execute as a single atomic cycle on every tick.

### Configuration Structure

```
unit
 └── poll.interval_ms          (single interval for the whole unit)
      └── reads[]              (all read blocks, executed in order within one cycle)
           ├── FC, Address, Quantity
           └── ...
```

Derived from `internal/config/config.go` (`PollConfig.IntervalMs`) and `internal/poller/builder.go:35`:

```go
Interval: time.Duration(u.Poll.IntervalMs) * time.Millisecond,
```

### Next Execution Time

Scheduling is managed entirely by `time.NewTicker` (`internal/poller/runner.go:13`).

```go
ticker := time.NewTicker(p.cfg.Interval)
...
case <-ticker.C:
    res := p.PollOnce()
```

Go's `time.Ticker` fires at fixed wall-clock intervals, measured from the moment `Run()` is called.  There is **no drift compensation**, **no dynamic scheduling**, and **no `nextExec`/`lastPoll` field**.  If a poll cycle takes longer than the interval, the next tick fires as soon as the `select` unblocks — ticks are not queued.

---

## PHASE 3 — Concurrency Model

### One Goroutine Per Unit

In `cmd/replicator/main.go`, for each unit the following goroutines are spawned:

```go
go p.Run(ctx, out)        // poller goroutine (line 179)
go func(...) { ... }(...)  // orchestrator goroutine (line 69)
```

There is **exactly one poller goroutine per unit**.  No worker pool.  No dispatcher.

### Sequential Reads Within a Cycle

Inside `PollOnce()` (`internal/poller/poller.go:69`), all read blocks are iterated sequentially in a single `for` loop.  Reads are issued one at a time over the same TCP connection.  The loop aborts on the first error.

```go
for _, rb := range p.cfg.Reads {
    switch rb.FC {
    case 1:
        bits, err := p.client.ReadCoils(rb.Address, rb.Quantity)
        if err != nil { ...; return res }   // abort on first failure
    ...
    }
}
```

### Channel Between Poller and Orchestrator

```go
out := make(chan poller.PollResult)   // unbuffered (main.go:66)
out <- res                            // runner.go:30 — poller blocks until orchestrator receives
```

The channel is **unbuffered**.  The poller goroutine blocks after each poll until the orchestrator goroutine consumes the result.  This is a natural back-pressure mechanism: the poller cannot outrun the writer.

### Summary

| Question | Answer |
|----------|--------|
| Requests sequential per device? | **Yes** — single goroutine, sequential loop |
| Multiple concurrent requests for same device? | **No** — impossible by design |
| Queue or dispatcher? | **No** — unbuffered channel provides implicit flow control |
| Burst protection? | **Yes** — see Phase 5 |

---

## PHASE 4 — Request Execution Path

### Full Lifecycle

```
time.Ticker fires
    ↓
runner.go:23  case <-ticker.C
    ↓
poller.go:69  PollOnce()
    ├── client nil? → factory() creates new TCP connection
    ├── for each ReadBlock (sequential):
    │       ↓
    │   modbus/client.go  ReadCoils / ReadDiscreteInputs /
    │                     ReadHoldingRegisters / ReadInputRegisters
    │       ↓
    │   client.go:146  roundTripRead(fc, addr, qty)
    │       ├── buildReadRequest() — constructs 12-byte ADU
    │       ├── tr.Send(req) — writes to TCP conn, reads raw response
    │       ├── protocol.DecodeTCP(raw) — parses MBAP header + PDU
    │       ├── TID / ProtocolID / UnitID / Function validation
    │       └── returns []byte payload
    │       ↓
    │   doReadBits / doReadRegisters — validates byte count, trims payload
    │       ↓
    │   unpackBits / unpackRegisters — decodes to []bool / []uint16
    │       ↓
    │   BlockResult appended to slice
    │
    ├── All reads succeeded → res.Blocks = blocks; recordSuccess()
    └── Any read failed    → recordFailure(err); return res (no further reads)
    ↓
runner.go:30  out <- res  (blocks until orchestrator receives)
    ↓
main.go:89    case res := <-out
    ├── dataWriter.Write(res)   — forwards data to Raw Ingest targets
    └── statusWriters.WriteStatus(snap)  — forwards health/counters
```

### Function Responsible for Modbus Read

`roundTripRead` in `internal/poller/modbus/client.go:146`.

This function builds the request ADU, performs the synchronous TCP send/receive, decodes the response, validates MBAP fields, and returns the raw PDU payload.

---

## PHASE 5 — Burst Protection

### Mechanisms

**1. Single goroutine per device**
`p.Run()` runs in one goroutine.  No parallelism at the device level is possible.

**2. Sequential block iteration**
`PollOnce()` reads all blocks in a loop.  Each read must complete (or fail) before the next begins.

**3. Abort on first failure**
If any read fails, `PollOnce()` returns immediately.  No partial results, no retried sub-reads in the same cycle.

**4. Unbuffered output channel**
`out <- res` blocks the poller until the orchestrator processes the result.  The next tick cannot be acted upon until the current result is consumed.

**5. Single TCP connection per device**
The Modbus client uses a single `net.Conn`.  All reads for a cycle share this connection.  There is no connection pool.

### Invariant

> **The engine guarantees at most one in-flight Modbus request to a device at any given time.**

This is a structural guarantee, not a runtime lock.  The single-goroutine model makes concurrent requests to the same device impossible.

---

## PHASE 6 — Error Handling and Backoff

### Connection Error Handling

`maybeInvalidateClient` (`internal/poller/poller.go:200`) is called on every read error.  It discards the current client when the error is classified as a "dead connection":

```go
func isDeadConnErr(err error) bool {
    // Timeouts do NOT invalidate the client
    // Dead patterns: EOF, broken pipe, connection reset, connection aborted,
    //                use of closed network connection, forcibly closed,
    //                wsasend / wsarecv (Windows)
}
```

Timeouts do **not** invalidate the client; the same TCP connection is reused on the next tick.

### Client Re-creation

If the client is nil at the start of a poll cycle (either at startup or after invalidation), the factory is called once:

```go
if p.client == nil {
    c, err := p.factory()
    if err != nil { ...; return res }   // factory failure = missed tick, not a loop
    p.client = c
}
```

This is **not a retry loop**.  A factory failure simply produces a failed `PollResult` and the cycle ends.  The next attempt occurs at the next ticker tick.

### No Backoff

There is **no exponential backoff**, **no cooldown period**, and **no device disable logic**.  A device that fails every cycle will attempt reconnection on every tick.  The minimum reconnect interval equals the configured `poll.interval_ms`.

### Counter-Only Failure Tracking

`TransportCounters` tracks failures passively:

| Counter | What it tracks |
|---------|----------------|
| `RequestsTotal` | Every `PollOnce()` call |
| `ResponsesValidTotal` | Successful cycles |
| `TimeoutsTotal` | `net.Error` with `Timeout() == true` |
| `TransportErrorsTotal` | All other errors |
| `ConsecutiveFailCurr` | Current run of consecutive failures |
| `ConsecutiveFailMax` | Peak consecutive failure run |

These counters influence **nothing**.  They do not trigger retries.  They do not alter scheduling.  They are observable state only (`internal/poller/types.go:48–56`).

### Rapid Retry Risk

Because there is no backoff:
- A device that is permanently unreachable will generate one TCP dial attempt per interval.
- At a 500 ms interval this is 2 dials/second to a dead host.
- No circuit breaker prevents this.

---

## PHASE 7 — Data Output

### Data Path

Successful `PollResult` data flows to **Raw Ingest v1** endpoints via `internal/writer/ingest/client.go`.

```
PollResult.Blocks[]
    ↓
writer.Write(res)                     (internal/writer/writer.go:31)
    │
    ├── for each TargetEndpoint
    │     └── for each MemoryDest
    │           └── for each BlockResult
    │                 ├── FC 1,2 → cli.WriteBits(area, unitID, dstAddr, bits)
    │                 └── FC 3,4 → cli.WriteRegisters(area, unitID, dstAddr, regs)
    │
    └── ingest.EndpointClient.send(...)    (internal/writer/ingest/client.go:79)
          ├── buildPacketV1(...)            10-byte header + payload
          ├── net.DialTimeout("tcp", ...)   new connection per write
          ├── writeAll(conn, pkt)
          └── read 1-byte ACK (0x00=OK, 0x01=Rejected)
```

### Raw Ingest v1 Packet Layout

```
Offset  Size  Field
0–1     2     Magic "RI" (0x52 0x49)
2       1     Version (0x01)
3       1     Area (= FC number)
4–5     2     UnitID (big-endian)
6–7     2     Address (big-endian)
8–9     2     Count (big-endian)
10+     n     Payload (packed bits or big-endian uint16 registers)
```

### Status Path

In parallel to data writes, the orchestrator goroutine maintains a `status.Snapshot` and forwards it to `deviceStatusWriter` instances:

```
status.Snapshot
    ↓
deviceStatusWriter.WriteStatus(snap)    (internal/writer/status_writer.go:57)
    ├── Full block re-assert on first write or after any write failure (needFull flag)
    ├── Incremental per-slot writes on subsequent updates
    └── ingest.EndpointClient.WriteRegisters(...)
          → Raw Ingest v1 → status holding-register block
```

Status is written to a per-target `status_unit_id` holding-register area, at base address `status_slot * 30` (30 slots per device).

---

## PHASE 8 — Behavior Summary

### 1. Poll Scheduling Model

- **Model:** Fixed-interval ticker per device (unit).
- **Granularity:** Device-level; all read blocks share one interval.
- **Interval source:** `poll.interval_ms` in YAML config → `time.Duration` → `time.NewTicker`.
- **Next-tick determination:** Standard Go ticker; no drift compensation.
- **Missed ticks:** If a cycle takes longer than the interval, the next tick fires immediately after the channel is unblocked (Go ticker behavior; ticks are not queued).

### 2. Concurrency Model

- **One goroutine per unit** for polling.
- **One goroutine per unit** for orchestration (status + write dispatch).
- **No worker pool**, no shared executor.
- **Unbuffered channel** (`out chan PollResult`) between poller and orchestrator provides flow control.

### 3. Device Protection Mechanisms

- Single goroutine ownership of the `Poller` object — no concurrent access.
- Single TCP connection per device — no connection pool.
- Sequential read dispatch within each cycle — no pipelining.

### 4. Burst Protection Behavior

> **Guarantee: No concurrent requests to the same device.**

Structural enforcement:
- One goroutine → one active `PollOnce()` at a time.
- Unbuffered channel → poller blocks after each cycle until results are consumed.
- First-failure abort → partial cycles are impossible.

### 5. Error Handling Strategy

| Scenario | Behavior |
|----------|----------|
| Read timeout | Increment `TimeoutsTotal`; **retain client**; return failed `PollResult` |
| Dead connection (EOF / reset / etc.) | Invalidate client (set to nil); next tick recreates via factory |
| Factory failure (dial error) | Return failed `PollResult`; retry next tick |
| Modbus exception | `ModbusException{Function, Exception}` returned; raw code preserved via `Code()` |
| No backoff | Retries at every tick regardless of consecutive failure count |
| No circuit breaker | No disable, no cooldown, no adaptive interval |

### 6. Data Forwarding Path

```
Modbus device
    → PollOnce() → PollResult (in-memory, single allocation)
    → unbuffered channel (flow control)
    → orchestrator goroutine
        ├── dataWriter.Write()
        │       → Raw Ingest v1 TCP (new connection per register block)
        └── statusWriter.WriteStatus()
                → Raw Ingest v1 TCP (new connection per status update)
```

---

## Invariants for Aegis Puller Core Integration

The following invariants are enforced structurally by this engine and **must be preserved** in any reimplementation:

| # | Invariant | Source |
|---|-----------|--------|
| I-1 | **One active poll cycle per device at a time.** Concurrent cycles for the same device must never occur. | `runner.go` single goroutine + `PollOnce()` synchronous execution |
| I-2 | **All-or-nothing cycle semantics.** A partial result (some reads succeeded, some failed) must never be forwarded as data. | `poller.go:155` — `res.Blocks` only set when all reads succeed |
| I-3 | **Client reuse across ticks unless the connection is dead.** Reconnection must not occur on timeout or Modbus exception. | `maybeInvalidateClient` — timeout does not invalidate |
| I-4 | **No in-cycle retries.** The scheduler tick is the only retry mechanism. | `PollOnce()` returns on first failure with no internal loop |
| I-5 | **Polling interval is per-device, not per-read-block.** All blocks for a device share one interval. | `Config.Interval` single field; no per-block interval |
| I-6 | **Counters are passive.** Transport counters must never influence scheduling, retries, or connection policy. | `types.go:48–56` comments; no counter is read in any control path |
| I-7 | **Factory is called at most once per tick.** If the factory fails, the tick is skipped; no retry within the same tick. | `poller.go:86–92` |
| I-8 | **Result channel is consumed before the next cycle begins.** Backpressure must be preserved. | Unbuffered `out` channel; `out <- res` blocks |
