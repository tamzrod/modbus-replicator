# GE Energy Meter Example

Topology:
- Real device: 10.5.1.xx
- Replicator pulls power + energy blocks
- MMA stores in Unit 1,2,3
- Shared status in Unit 100

How to run:

1) Start MMA:
   mma2.exe mma.yaml

2) Start Replicator:
   replicator.exe replicator.yaml