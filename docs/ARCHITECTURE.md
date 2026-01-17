# Modbus Replicator – Architecture

## Core Principle

Modbus Replicator is a one-way data replication service.

Source → Target only.

No control.
No semantics.
No interpretation.

## Design Center

The Modbus Memory Appliance (MMA) is the reference memory model.

Replicator adapts to MMA rules, not the other way around.
