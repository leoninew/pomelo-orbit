---
name: transfer-database-data
description: "Export or import one Pomelo Orbit service deployment closure through the installed dbtalk database JSONL CLI."
---

# Transfer An Orbit Service

Use `scripts/database_transfer.py` from the Orbit repository root. This skill
moves one service deployment closure through dbtalk; it is not a schema
migration tool or a remote filesystem backup.

## Preconditions

- Install dbtalk so `dbtalk database export` and `dbtalk database import` are
  available on `PATH`.
- Install uv so the Orbit scripts run with the locked `scripts/` environment.
- Initialize the target Orbit schema with the normal Orbit migration command
  before importing. The transfer file contains data only, never DDL, indexes,
  triggers, routines, or permissions.
- Use dbtalk's canonical `--dsn` or `--dsn-env` connection option. Keep
  credentials in an environment variable and pass its name with `--dsn-env`;
  do not place a password in the command line or transfer file.

## Export

```bash
uv run --project scripts python scripts/database_transfer.py export \
  --source sqlite --dsn sqlite:///./data/db/pomelo-orbit.db \
  --service-code <service-code> \
  --output data/<service-code>-<timestamp>.jsonl --tz UTC
```

For MySQL, use `--source mysql --dsn-env <ENV_NAME>`. Orbit asks dbtalk
for the service closure tables, selects the requested service's project,
application, version lineage, components, Gateway configuration, service
overrides, and routes, then writes the service JSONL file. The temporary
dbtalk export is removed automatically.

## Export File Names

Direct `dbtalk database export` accepts either an output file or an existing
directory. When `--output` is omitted, dbtalk writes
`data/<source>-<timestamp>.jsonl`.

This Orbit wrapper intentionally differs: its final `--output` is required,
while its internal dbtalk export is a temporary `service.jsonl` file. Give the
final service transfer an explicit, service-specific path such as
`data/<service-code>-<timestamp>.jsonl`.

## Import

```bash
uv run --project scripts python scripts/database_transfer.py import \
  --target sqlite --dsn sqlite:///./data/db/pomelo-orbit.db \
  --input service.jsonl --mode upsert --tz UTC
```

`--mode` is required. `insert` fails on any database constraint conflict;
`upsert` updates existing primary-key rows. A schema initialized by the normal
Orbit migration already has the seed Project, so the standard service-migration
flow uses `upsert`. The adapter validates that the file describes exactly one
closed Orbit service deployment before invoking dbtalk. dbtalk owns primary-key
checks, date/time conversion, and table-level transaction behavior.

The service export requests only the tables in the Orbit deployment closure.
The final service file does not contain `schema_migrations` or unrelated tables;
the import command therefore does not pass table exclusions.

Use `--tz <IANA name>` consistently for export and import when temporal values
are involved. Orbit does not reinterpret or render dbtalk JSONL values itself.

## Boundary

For general database transfer, JSONL format details, database-specific
conversion, and import semantics, use dbtalk's `dbtalk-database` skill. Do not
restore the removed Orbit-wide SQL or SQLite/MySQL implementation.
