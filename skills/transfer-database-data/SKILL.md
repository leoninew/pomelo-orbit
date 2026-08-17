---
name: transfer-database-data
description: "Use scripts/database_transfer.py to safely export, import, or convert complete SQLite/MySQL database data, or export and import one SQLite service deployment closure. Trigger for full database transfer, SQLite/MySQL conversion, replacing a target database, or moving a named service and its deployable version components to another machine."
---

# Transfer Database Data

Run `scripts/database_transfer.py` from the repository root. Treat this skill as the LLM operating guide; script behavior is authoritative if it differs.

## Commands

Use the existing full-database commands without changing their behavior:

```powershell
# Full SQLite/MySQL export
python scripts/database_transfer.py export --source sqlite --sqlite-path <source.db> --output <transfer.sqlite.sql>
python scripts/database_transfer.py export --source mysql --mysql-host <host> --mysql-port <port> --mysql-user <user> --mysql-password <password> --mysql-database <database> --output <transfer.mysql.sql>

# Full SQLite/MySQL import; use --replace only when explicitly authorized
python scripts/database_transfer.py import --target sqlite --sqlite-path <target.db> --input <transfer.sqlite.sql> [--replace]
python scripts/database_transfer.py import --target mysql --mysql-host <host> --mysql-port <port> --mysql-user <user> --mysql-password <password> --mysql-database <database> --input <transfer.mysql.sql> [--replace]

# Cross-driver conversion
python scripts/database_transfer.py convert --from sqlite --to mysql --input <transfer.sqlite.sql> --output <transfer.mysql.sql>
python scripts/database_transfer.py convert --from mysql --to sqlite --input <transfer.mysql.sql> --output <transfer.sqlite.sql>
```

Use the dedicated SQLite service commands for cross-machine deployment:

```powershell
python scripts/database_transfer.py export-service --sqlite-path <source.db> --service-code <service-code> --output <service-transfer.sqlite.sql>
python scripts/database_transfer.py import-service --sqlite-path <target.db> --input <service-transfer.sqlite.sql>
```

Do not add `--scope` or `service-config` options. Service transfer is SQLite-only.

## Service Transfer Content

`export-service` selects only the named service's minimum deployable closure:

- its project and application;
- the active version and version lineage;
- version components and env, endpoint, mount, dependency, healthcheck, resource, tmpfs, ulimit, and device data;
- service env and component overrides; and
- routes targeting that service.

It excludes records for other services. It writes no `DELETE` statements. `import-service` accepts only this exact SQLite table set and always appends; it has no `--replace` option.

## Safety And Verification

1. Resolve the source database, target database, and output path before writing. Do not overwrite an existing export unless the user explicitly requests it.
2. Treat export files as sensitive. They preserve env values, secrets, certificates, JSON, controlled-file contents, and timestamps. Report paths and row counts, not contents.
3. Ensure the target database has the current compatible schema. For a service import, use a new or otherwise non-conflicting target because IDs are preserved and import is append-only.
4. For `import --replace`, confirm the exact target and deletion impact. Never add `--replace` to `import-service`.
5. Verify service exports contain exactly one requested service, the expected service-table set, and no `DELETE FROM` statement. After any SQLite import, run `PRAGMA integrity_check` and `PRAGMA foreign_key_check`.
