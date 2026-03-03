# GE Energy Meter Example

Version Note: 2026-03-03 (Stage 4 documentation rectification; synchronized to implemented behavior)

Topology:
- Real device: 10.5.1.xx
- Replicator pulls power + energy blocks
- MMA stores in Unit 1,2,3
- Status destination is per target via `status_unit_id` (no global shared status endpoint)

How to run:

1) Start MMA:
   mma2.exe mma.yaml

2) Start Replicator:
   replicator.exe replicator.yaml