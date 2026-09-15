---
name: transfer-database-data
description: "Export or import an Orbit service closure, or back up and restore one Environment with its SSH key through dbtalk JSONL."
---

# Transfer Orbit Data

Use `scripts/database_transfer.py` from the Orbit repository root. This skill
moves Orbit records through dbtalk JSONL. It is not a schema migration tool or
a remote filesystem backup.

## Preconditions

- Install dbtalk so `dbtalk export` and `dbtalk import` are available on
  `PATH`.
- Install uv so the Orbit scripts run with the locked `scripts/` environment.
- Initialize the target Orbit schema with the normal Orbit migration command
  before importing. Transfer files contain data only, never DDL, indexes,
  triggers, routines, or permissions.
- Use dbtalk's canonical `--dsn` or `--dsn-env` connection option. Keep
  database credentials in an environment variable and pass its name with
  `--dsn-env`; do not put a password in a command line or transfer file.

## Service Transfer

Use `export` and `import` to move one Service deployment closure for the
current Project:

```bash
uv run --project scripts python scripts/database_transfer.py export \
  --source sqlite --dsn sqlite:///./data/db/pomelo-orbit.db \
  --project-id <source-project-id> --service-code <service-code> \
  --output data/<service-code>-<timestamp>.jsonl --tz UTC

uv run --project scripts python scripts/database_transfer.py import \
  --target sqlite --dsn sqlite:///./data/db/pomelo-orbit.db \
  --project-id <target-project-id> --input data/<service-code>-<timestamp>.jsonl \
  --mode upsert --tz UTC
```

The service file contains its deployment closure only. Its Environment remains
target-local and is not changed by the service import.

## Environment Backup And Restore

Use the dedicated environment commands to back up one selected Environment and
restore it to a specified target Project:

```bash
uv run --project scripts python scripts/database_transfer.py export-environment \
  --source sqlite --dsn sqlite:///./data/db/pomelo-orbit.db \
  --project-id <source-project-id> --environment-id <environment-id> \
  --source-secret-env <SOURCE_JWT_SECRET_ENV> \
  --output data/<environment-id>-<timestamp>.jsonl --tz UTC

uv run --project scripts python scripts/database_transfer.py import-environment \
  --target sqlite --dsn sqlite:///./data/db/pomelo-orbit.db \
  --project-id <target-project-id> --input data/<environment-id>-<timestamp>.jsonl \
  --target-secret-env <TARGET_JWT_SECRET_ENV> --tz UTC
```

For MySQL or PostgreSQL, use `--source mysql|postgresql` or
`--target mysql|postgresql` with `--dsn-env <ENV_NAME>`. PostgreSQL DSNs use
`postgresql+psycopg://user:password@host:5432/database`.

`--source-secret-env` and `--target-secret-env` name environment variables
that contain the corresponding Orbit instance's `jwt.secret_key` Fernet key.
Never place either key on the command line. The source key decrypts the stored
SSH private key for the transfer file; the target key encrypts it before the
target database is written.

The export selects exactly one Environment matching both `--project-id` and
`--environment-id`. SSH exports include its matching `environment_credential`;
the file uses a `private_key` plaintext field rather than the source instance's
`encrypted_private_key`. Local Environment exports contain no credential block.
Every environment file is marked with the Orbit `environment` scope.

Environment import verifies that the target Project exists and has no
Environment, then inserts the restored records for that target Project. It has
no `--mode`: an existing target Environment is always rejected rather than
overwritten. The imported Environment uses the target Project's code and has no
Gateway application binding.

The environment file has an empty `project` table block only because dbtalk
needs the foreign-key reference; it contains no Project rows, repository
credentials, Gateway records, workspace files, or containers. It preserves the
Environment configuration and SSH key association; the referenced host must
still be reachable and its workspace/runtime must already exist before
deployments can run.

The environment JSONL contains a plaintext SSH private key. Restrict its file
permissions and storage, transfer it only through an approved secure channel,
and delete it once restoration succeeds. Do not print the file, private key,
Fernet key, or encrypted token in logs, issue comments, or chat.

## Boundary

For general database transfer, JSONL format details, database-specific
conversion, and import semantics, use dbtalk's `dbtalk-database` skill. Do not
restore the removed Orbit-wide SQL or SQLite/MySQL implementation.
