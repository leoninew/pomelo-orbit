---
name: transfer-database-data
description: "Export or import one Pomelo Orbit service deployment closure through the installed Housekeeper database JSONL CLI."
---

# Transfer An Orbit Service

Use `scripts/database_transfer.py` from the Orbit repository root. This
skill moves one service deployment closure; it is not a general database
backup or schema migration tool.

## Preconditions

- Install a published Housekeeper package so `housekeeper database export` and
  `housekeeper database import` are available on `PATH`.
- Initialize the target Orbit schema with the normal Orbit migration command
  before importing. The transfer file contains data only, never DDL, indexes,
  triggers, routines, or permissions.
- Keep MySQL credentials in an environment variable and pass its name with
  `--mysql-dsn-env`. Do not place a password in the command line or transfer
  file.

## Export

```bash
python scripts/database_transfer.py export \
  --source sqlite --sqlite-path data/db/pomelo-orbit.db \
  --service-code <service-code> --output service.jsonl --tz UTC
```

For MySQL, use `--source mysql --mysql-dsn-env <ENV_NAME>`. Orbit asks
Housekeeper for a temporary full JSONL export, selects the requested service's
project, application, version lineage, components, Gateway configuration,
service overrides, and routes, then writes the service JSONL file. The full
temporary export is removed automatically.

## Import

```bash
python scripts/database_transfer.py import \
  --target sqlite --sqlite-path data/db/pomelo-orbit.db \
  --input service.jsonl --mode upsert --tz UTC
```

`--mode` is required. `insert` fails on any database constraint conflict;
`upsert` updates existing primary-key rows. A schema initialized by the normal
Orbit migration already has the seed Project, so the standard service-migration
flow uses `upsert`. The adapter validates that the file describes exactly one
closed Orbit service deployment before invoking Housekeeper. Housekeeper owns
primary-key checks, date/time conversion, and table-level transaction behavior.

Orbit always excludes `schema_migrations` from the temporary full export. The
final service file does not contain that table, so it must not be excluded again
on import: Housekeeper rejects exclusions that are absent from the JSONL file.

Use `--tz <IANA name>` consistently for export and import when temporal values
are involved. Orbit does not reinterpret or render JSONL values itself.

## Boundary

For full-database transfer, JSONL format details, database-specific conversion,
and import semantics, use Housekeeper's database-transfer skill. The examples
there use `uv run housekeeper` inside the Housekeeper source checkout; Orbit
uses the installed `housekeeper` CLI. Do not restore the removed Orbit-wide SQL
or SQLite/MySQL implementation.
