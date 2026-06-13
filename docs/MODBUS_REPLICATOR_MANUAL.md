# Modbus Replicator Manual

## Purpose

Modbus Replicator is a deterministic **read -> snapshot -> fan-out -> write** service.

It polls Modbus source devices once per cycle and writes the resulting data into one or more Modbus Memory Appliance (MMA) targets so downstream clients can read memory instead of talking directly to unstable field devices.

## Current Capabilities

Implemented today:

- Modbus TCP source polling
- FC1, FC2, FC3, and FC4 reads
- Fan-out writes to one or more targets
- Optional per-target status publishing
- Optional per-target health control output (coil)
- Configuration validation before startup

Not implemented today:

- Modbus RTU source polling
- Semantic parsing, scaling, or rules processing
- Hidden retries inside a poll cycle

## How the Runtime Works

For each configured unit, the runtime:

1. Loads and validates configuration
2. Builds one poller for the source device
3. Builds writer plans for every target memory destination
4. Polls configured read blocks on a fixed interval
5. Writes successful snapshots to every configured target
6. Updates status data independently of data-write success

Core rule:

> The source device is the truth. Memory is the contract.

## Configuration Overview

The runtime starts from a YAML file with this high-level shape:

```yaml
replicator:
  units:
    - id: "unit-1"
      source: {}
      reads: []
      targets: []
      poll: {}
```

Each `units[]` entry is one independent replication pipeline.

### Source

`source` defines where the runtime reads from:

- `endpoint`: source Modbus TCP address such as `10.0.0.10:502`
- `unit_id`: Modbus unit ID of the source device
- `timeout_ms`: timeout for source reads and target writes
- `device_name`: optional ASCII name for status output
- `status_slot`: optional base slot for status publishing

If `status_slot` is omitted, status writers are not created for that unit.

### Reads

`reads` defines the Modbus blocks read every poll cycle:

- `fc`: function code (`1`, `2`, `3`, or `4`)
- `address`: starting address
- `quantity`: number of coils/inputs/registers
- `invert`: optional bit inversion for FC1 and FC2 only
- `addinvert`: optional appended inverted copy for FC1 and FC2 only

`invert` and `addinvert` are ignored for FC3 and FC4.

### Targets

Each target defines where snapshots are written:

- `id`: target identifier
- `endpoint`: target Raw Ingest endpoint
- `unit_id`: destination unit ID for data writes
- `status_unit_id`: optional destination unit ID for status writes
- `health_output`: optional coil output representing unit health
- `memories`: destination memory mappings

Each memory mapping contains:

- `memory_id`: destination MMA memory
- `offsets`: per-function-code offset map

Health control constraints when `health_output.enabled: true`:

- `health_output.area` must be `coil`
- `health_output.address` is required
- collision is rejected for duplicate `(endpoint, unit_id, area, address)` across units

### Poll

`poll.interval_ms` defines the fixed poll cadence for the unit.

## Minimal Example

```yaml
replicator:
  units:
    - id: "meter-1"
      source:
        endpoint: "10.5.1.101:502"
        unit_id: 1
        timeout_ms: 2000
        device_name: "METER01"
        status_slot: 1
      reads:
        - fc: 3
          address: 0
          quantity: 50
      targets:
        - id: 1
          endpoint: "10.5.1.20:501"
          unit_id: 2
          status_unit_id: 35
          memories:
            - memory_id: 1
              offsets: {}
      poll:
        interval_ms: 1000
```

For the full schema and validation rules, see `docs/CONFIG.md`.

## Running the Service

The command-line interface expects a config file path:

```bash
replicator /path/to/config.yaml
```

Equivalent development run:

```bash
go run ./cmd/replicator /path/to/config.yaml
```

If the config cannot be loaded or validated, the process exits immediately with a fatal error.

## Status Behavior

Status is optional and is enabled only when `source.status_slot` is configured.

When enabled:

- status is written through the same writer layer as normal data
- each target must define `status_unit_id`
- the status destination is per target
- the runtime emits a 30-slot status block

Current runtime status values include:

- health code
- last error code
- seconds in error
- device name
- transport lifetime counters

Detailed status layout is documented in:

- `docs/Status_Block_Layout.md`
- `docs/ARCHITECTURE.md`

## Health Output Behavior

`health_output` is an optional per-target feature that publishes unit health to a destination coil.

Current behavior:

- only `area: coil` is supported
- published value is mapped from runtime health:
  - `OK` -> `1`
  - `ERROR` -> `0`
  - `UNKNOWN` -> `0`
  - `STALE` -> `0`
  - `DISABLED` -> `0`
- writes are edge-triggered by the mapped output value:
  - non-OK -> non-OK does not publish again
  - OK -> OK does not publish again
  - transitions between mapped outputs (`0 <-> 1`) publish once
- if a publish fails, the last published state is not advanced; the next call retries the same mapped value
- publish failures are logged and do not stop replication

## Operational Notes

- Data writes happen only for successful poll cycles.
- Poll failures do not become fake data writes.
- Status writes are independent from data delivery success.
- The runtime does not perform hidden retries inside a poll cycle.
- A timeout on the source poll is treated as a poll failure.

## Examples and Reference Docs

- Example configurations: `Examples/`
- Configuration reference: `docs/CONFIG.md`
- Architecture reference: `docs/ARCHITECTURE.md`
- Raw ingest protocol: `docs/raw_ingest_v_1_spec.md`
- RTU design status: `docs/modbus_serial_rtu_support.md`

## Troubleshooting

Startup failure usually means one of these:

- config file path is wrong
- YAML is invalid
- validation rejected the topology

Runtime problems usually appear as:

- source poll errors in logs
- writer delivery errors in logs
- status health changing to error

The service is designed to fail honestly rather than hide transport problems.
