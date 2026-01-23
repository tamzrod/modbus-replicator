# Device Status — Limits & Responsibility Boundary

> **Purpose**  
> Clarify *where limits actually live* for device status slots, and prevent artificial limits from being introduced in the Replicator.

This document is **normative**.

---

## Core Statement (Locked)

> **There is no device-count limit in the Replicator.**  
> **All real limits are enforced by MMA memory allocation.**

If this statement is violated, the architecture is wrong.

---

## Responsibility Boundary

### Replicator (This Project)

The Replicator:

- Does **not** define a maximum number of devices
- Does **not** enforce capacity policies
- Does **not** warn about scale
- Does **not** contain configuration knobs for limits

Its responsibilities are strictly:

- Accept a `slot` index (0-based)
- Compute:

```
slot_base = slot_index × SLOT_SIZE
```

- Write device status into Input Registers
- Perform **bounds checking** against MMA memory
- Fail fast if a write exceeds allocated memory

Nothing more.

---

### MMA (Modbus Memory Appliance)

The MMA:

- Owns memory allocation
- Defines how many Input Registers exist
- Enforces address bounds
- Is the **only component that limits capacity**

If more devices are required:

- Allocate more Input Registers in MMA
- Or deploy additional MMA instances
- Or shard by topology

All choices are **explicit and physical**.

---

## Slot Model Recap

- One slot = one device
- Slot size = **20 Input Registers**
- Slot index is logical, not physical

Maximum device count is therefore:

```
max_devices = floor(total_input_registers / 20)
```

This is **derived**, never configured.

---

## What Is Explicitly Forbidden

The following must **never** be added to the Replicator:

- Hard-coded device limits
- Soft “recommended maximums"
- Scaling heuristics
- Automatic slot reassignment
- Capacity warnings
- Policy-based enforcement

All of the above are violations of responsibility boundaries.

---

## Why This Matters

Artificial limits:

- Hide real constraints
- Create false safety
- Break scale silently
- Force workarounds later

Physical limits:

- Are honest
- Are explicit
- Are measurable
- Are solvable by deployment

---

## Final Locked Statement

> **The Replicator never decides how big the system is.**  
> **It only respects the memory it is given.**

If someone asks “what is the maximum number of devices?”, the correct answer is:

> **“That depends on how much memory MMA allocates.”**

