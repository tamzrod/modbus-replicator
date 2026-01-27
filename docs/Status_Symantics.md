# Device Status Semantics

This document defines the **authoritative meaning** of device status in Modbus Replicator.

It intentionally covers **semantics only**.
No implementation details.
No optimizations.
No future features.

If it is not written here, it does not exist.

---

## Core Rule

> **Status reflects the outcome of the most recent poll cycle.**

Status is **not** a prediction, a trend, or an interpretation.
It is a factual statement about the last attempt to read the device.

---

## Status Is Data

Status is treated exactly like any other data block:

* Written by the writer
* Delivered via the Raw Ingest protocol
* Stored in Modbus Memory (MMA)
* Read by clients like any other register

There are:

* no side channels
* no heartbeats
* no probes
* no hidden state

---

## Health Code

The health code is a **binary state**.

| Value | Meaning                                     |
| ----: | ------------------------------------------- |
|    OK | The most recent poll completed successfully |
| ERROR | The most recent poll failed                 |

There are no intermediate states.

---

## What Constitutes a Poll Failure

A poll is considered **failed** if **any** of the following occur:

* Modbus exception response
* TCP connection failure
* Timeout
* Protocol error
* Any non-nil `PollResult.Err`

No distinction is made at this layer.

Failure is failure.

---

## Error Code

The error code field represents the **raw error condition** observed during the poll.

Rules:

* The error code is written **as-is**
* No parsing
* No classification
* No mapping
* No normalization

At this stage, the value is:

* `0` → no error
* non-zero → error occurred

The exact meaning of non-zero values is **intentionally undefined** here.

---

## Writer Responsibilities

The writer:

* writes **OK** when `PollResult.Err == nil`
* writes **ERROR** when `PollResult.Err != nil`
* writes the associated error code
* does **not** retry reads
* does **not** interpret errors

A poll failure is **not** a writer failure.

---

## What This Layer Does NOT Do

Explicit non-goals:

* No retry logic
* No backoff
* No time accumulation
* No error counting
* No severity levels
* No vendor-specific decoding

All of the above belong to higher layers.

---

## Relationship to Future State Logic

Future runner/state logic may:

* accumulate seconds-in-error
* saturate counters
* derive quality flags

Those features:

* must consume this status data
* must not redefine it

This document remains the source of truth.

---

## Design Rationale

This model was chosen to ensure:

* deterministic behavior
* honest failure reporting
* zero ambiguity
* zero hidden logic

The system must be boring before it can be smart.

---

## Summary

* Status answers one question only:

> **Did the last poll succeed?**

Nothing more.
Nothing less.

Everything else builds on top of this.
