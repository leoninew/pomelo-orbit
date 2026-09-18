---
name: transfer-database-data
description: "Export or import an Orbit service closure, or back up and restore one Environment with its SSH key through dbtalk JSONL."
---

# Transfer Orbit Data

Use `pomelo-orbit-cli database-transfer` from the Orbit repository root. This skill moves Orbit records through dbtalk JSONL. It is not a schema migration tool or a remote filesystem backup.

## Preconditions

- Install uv so the Orbit scripts run with the locked `scripts/` environment.
- Run from the source or target Orbit repository root. The CLI reads the shared Orbit database and JWT configuration; select a profile through the process variable `POMELO_ORBIT_APP__ENV` when needed.
- Initialize the target Orbit schema with the normal Orbit migration command before importing. Transfer files contain data only, never DDL, indexes, triggers, routines, or permissions.
- Do not provide a second `DBTALK_*` connection configuration. The CLI maps the shared Go database configuration to dbtalk's canonical connection format without exposing credentials on the command line.

## Service Transfer

Use `export` and `import` to move one Service deployment closure for the current Project:

```bash
uv --directory scripts run pomelo-orbit-cli database-transfer export \
  --project-id <source-project-id> --service-code <service-code> \
  --output data/<service-code>-<timestamp>.jsonl --tz UTC

uv --directory scripts run pomelo-orbit-cli database-transfer import \
  --project-id <target-project-id> --input data/<service-code>-<timestamp>.jsonl \
  --mode upsert --tz UTC
```

The service file contains its deployment closure only. Its Environment remains target-local and is not changed by the service import.

## Environment Backup And Restore

Use the dedicated environment commands to back up one selected Environment and restore it to a specified target Project:

```bash
uv --directory scripts run pomelo-orbit-cli database-transfer export-environment \
  --project-id <source-project-id> --environment-id <environment-id> \
  --output data/<environment-id>-<timestamp>.jsonl --tz UTC

uv --directory scripts run pomelo-orbit-cli database-transfer import-environment \
  --project-id <target-project-id> --input data/<environment-id>-<timestamp>.jsonl \
  --tz UTC
```

For MySQL or PostgreSQL, configure the matching Go `database.driver` and DSN in the source or target Orbit profile. The CLI normalizes Go MySQL TCP DSNs and PostgreSQL `postgres://` URLs for dbtalk. It uses the configured `jwt.secret_key` to decrypt source SSH keys and encrypt target SSH keys; never place a Fernet key on the command line.

The export selects exactly one Environment matching both `--project-id` and `--environment-id`. SSH exports include its matching `environment_credential`; the file uses a `private_key` plaintext field rather than the source instance's `encrypted_private_key`. Local Environment exports contain no credential block. Environment file shape is restricted to the `environment` scope.

Environment import verifies that the target Project exists and has no Environment, then inserts the restored records for that target Project. It has no `--mode`: an existing target Environment is always rejected rather than overwritten. The imported Environment uses the target Project's code and has no Gateway application binding.

The environment file has an empty `project` table block only because dbtalk needs the foreign-key reference; it contains no Project rows, repository credentials, Gateway records, workspace files, or containers. It preserves the Environment configuration and SSH key association; the referenced host must still be reachable and its workspace/runtime must already exist before deployments can run.

The environment JSONL contains a plaintext SSH private key. Restrict its file permissions and storage, transfer it only through an approved secure channel, and delete it once restoration succeeds. Do not print the file, private key, Fernet key, or encrypted token in logs, issue comments, or chat.

## Boundary

For general database transfer, JSONL format details, database-specific conversion, and import semantics, use dbtalk's `dbtalk-database` skill. Do not restore the removed Orbit-wide SQL or SQLite/MySQL implementation.
