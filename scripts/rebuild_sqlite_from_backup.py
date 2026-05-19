from __future__ import annotations

import argparse
import hashlib
import importlib
import json
import logging
import sqlite3
import sys
from pathlib import Path

from sqlalchemy import create_engine

REPO_ROOT = Path(__file__).resolve().parents[1]
BACKEND_ROOT = REPO_ROOT / "backend"
MIGRATIONS_DIR = BACKEND_ROOT / "migrations"
SCHEMA_FILE = MIGRATIONS_DIR / "v0.7.0__schema.sql"
DEFAULT_PROJECT_ID = "01KRRKK0K3T519ZQZES3M4QA9Z"
logger = logging.getLogger(__name__)

COPY_ORDER = [
    "user",
    "project",
    "project_member",
    "login_history",
    "login_attempt",
    "application",
    "deployment",
    "application_config_file",
    "application_route",
    "application_service",
    "route",
    "credential",
    "pipeline_template",
    "build_stage",
    "pipeline_template_stage",
    "pipeline_snapshot",
    "repository",
    "repository_webhook",
    "pipeline_run",
    "stage_run",
    "artifact",
]

PROJECT_ID_TABLES = {
    "application",
    "artifact",
    "build_stage",
    "credential",
    "deployment",
    "pipeline_run",
    "pipeline_snapshot",
    "pipeline_template",
    "repository",
    "route",
}

CURRENT_MIGRATION_FILES = [
    "v0.7.0__schema.sql",
    "v0.7.2__business_data.json",
    "v0.8.0__auth_schema.sql",
    "v0.8.1__init_data.json",
]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Rebuild Pomelo Orbit SQLite database from an old backup.")
    parser.add_argument("--backup", type=Path, required=True, help="Old SQLite backup database")
    parser.add_argument("--output", type=Path, required=True, help="Rebuilt SQLite database path")
    parser.add_argument("--json-output", type=Path, help="Optional structured insert JSON output")
    parser.add_argument("--schema", type=Path, default=SCHEMA_FILE, help="Current schema migration SQL file")
    parser.add_argument("--migrations-dir", type=Path, default=MIGRATIONS_DIR, help="Current migrations directory")
    parser.add_argument("--default-project-id", default=DEFAULT_PROJECT_ID, help="Default project id for migrated rows")
    parser.add_argument("--default-project-name", default="默认项目", help="Default project name")
    parser.add_argument("--default-project-code", default="default", help="Default project code")
    parser.add_argument("--force", action="store_true", help="Overwrite output files if they exist")
    return parser.parse_args()


def table_names(conn: sqlite3.Connection) -> set[str]:
    return {row[0] for row in conn.execute("SELECT name FROM sqlite_master WHERE type='table'")}


def table_columns(conn: sqlite3.Connection, table: str) -> list[str]:
    return [row["name"] for row in conn.execute(f'PRAGMA table_info("{table}")')]


def insert_rows(conn: sqlite3.Connection, table: str, rows: list[dict[str, object]], columns: list[str]) -> int:
    if not rows:
        return 0
    placeholders = ", ".join("?" for _ in columns)
    col_sql = ", ".join(f'"{column}"' for column in columns)
    conn.executemany(
        f'INSERT INTO "{table}" ({col_sql}) VALUES ({placeholders})',
        [[row[column] for column in columns] for row in rows],
    )
    return len(rows)


def create_history_table(conn: sqlite3.Connection) -> None:
    conn.execute(
        """
        CREATE TABLE IF NOT EXISTS __migration_history (
            id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
            filename TEXT NOT NULL UNIQUE,
            checksum VARCHAR(32) NOT NULL,
            executed_at DATETIME NOT NULL,
            execution_time_ms INTEGER NOT NULL
        )
        """
    )


def calculate_current_history(migrations_dir: Path) -> list[dict[str, object]]:
    logger.info("Calculating migration history: migrations_dir=%s", migrations_dir)
    rows = []
    for filename in CURRENT_MIGRATION_FILES:
        migration_file = migrations_dir / filename
        if not migration_file.exists():
            raise RuntimeError(f"Migration file not found: {migration_file}")
        rows.append(
            {
                "filename": filename,
                "checksum": hashlib.md5(migration_file.read_bytes()).hexdigest(),
                "executed_at": "2026-05-18T00:00:00Z",
                "execution_time_ms": 0,
            }
        )
    return rows


def choose_default_member_user_id(old: sqlite3.Connection) -> str:
    user = old.execute("SELECT id FROM user WHERE username = 'admin' ORDER BY created_at LIMIT 1").fetchone()
    if user is None:
        user = old.execute("SELECT id FROM user ORDER BY created_at LIMIT 1").fetchone()
    if user is None:
        raise RuntimeError("Backup database has no user rows; cannot create default project member")
    logger.info("Selected default project member: user_id=%s", user["id"])
    return user["id"]


def collect_rows(
    old: sqlite3.Connection,
    new: sqlite3.Connection,
    current_history: list[dict[str, object]],
    default_project_id: str,
    default_project_name: str,
    default_project_code: str,
) -> list[tuple[str, list[dict[str, object]], list[str]]]:
    old_tables = table_names(old)
    logger.info("Collecting rows from backup: tables=%s", len(old_tables))
    result = []
    default_member_user_id = choose_default_member_user_id(old)

    for table in COPY_ORDER:
        new_cols = table_columns(new, table)
        if not new_cols:
            continue

        if table == "project" and table not in old_tables:
            row = {
                "id": default_project_id,
                "name": default_project_name,
                "code": default_project_code,
                "created_at": "2024-03-16T00:00:00Z",
                "updated_at": "2024-03-16T00:00:00Z",
                "is_active": 1,
            }
            columns = [column for column in new_cols if column in row]
            result.append((table, [row], columns))
            logger.info("Prepared default project row: project_id=%s", default_project_id)
            continue

        if table == "project_member" and table not in old_tables:
            row = {
                "project_id": default_project_id,
                "user_id": default_member_user_id,
                "created_at": "2024-03-16T00:00:00Z",
            }
            columns = [column for column in new_cols if column in row]
            result.append((table, [row], columns))
            logger.info(
                "Prepared default project member row: project_id=%s, user_id=%s",
                default_project_id,
                default_member_user_id,
            )
            continue

        if table not in old_tables:
            result.append((table, [], new_cols))
            continue

        old_cols = table_columns(old, table)
        columns = [column for column in new_cols if column in old_cols]
        select_cols = ", ".join(f'"{column}"' for column in columns)
        rows = [dict(row) for row in old.execute(f'SELECT {select_cols} FROM "{table}"')]

        if table in PROJECT_ID_TABLES and "project_id" in new_cols and "project_id" not in columns:
            for row in rows:
                row["project_id"] = default_project_id
            columns.append("project_id")
            logger.info("Filled default project for table: table=%s, rows=%s", table, len(rows))

        result.append((table, rows, columns))
        logger.info("Prepared table rows: table=%s, rows=%s, columns=%s", table, len(rows), len(columns))

    result.append(
        ("__migration_history", current_history, ["filename", "checksum", "executed_at", "execution_time_ms"])
    )
    return result


def write_json_output(path: Path, rows_by_table: list[tuple[str, list[dict[str, object]], list[str]]]) -> None:
    logger.info("Writing import JSON: path=%s", path)
    operations = []
    for table, rows, columns in rows_by_table:
        if rows:
            operations.append(
                {
                    "type": "insert",
                    "table": table,
                    "data": [{column: row[column] for column in columns} for row in rows],
                    "comment": f"Import {table} from backup",
                }
            )
    path.write_text(json.dumps(operations, ensure_ascii=False, indent=2), encoding="utf-8")


def validate_database(output: Path, migrations_dir: Path) -> None:
    logger.info("Validating rebuilt database: output=%s", output)
    conn = sqlite3.connect(output)
    conn.row_factory = sqlite3.Row
    fk_errors = list(conn.execute("PRAGMA foreign_key_check"))
    if fk_errors:
        details = "; ".join(str(tuple(row)) for row in fk_errors[:20])
        raise RuntimeError(f"Foreign key check failed: {details}")
    logger.info("Foreign key validation passed")

    for table in sorted(PROJECT_ID_TABLES):
        columns = table_columns(conn, table)
        if "project_id" in columns:
            null_count = conn.execute(f'SELECT COUNT(*) FROM "{table}" WHERE project_id IS NULL').fetchone()[0]
            if null_count:
                raise RuntimeError(f"Table {table} has {null_count} rows with NULL project_id")

    history = {
        row["filename"]: row["checksum"]
        for row in conn.execute("SELECT filename, checksum FROM __migration_history ORDER BY filename")
    }
    conn.close()

    expected = {row["filename"]: row["checksum"] for row in calculate_current_history(migrations_dir)}
    if history != expected:
        raise RuntimeError(f"Migration history mismatch: expected={expected}, actual={history}")
    logger.info("Migration history validation passed: migrations=%s", len(history))

    sys.path.insert(0, str(BACKEND_ROOT / "src"))
    migrator = importlib.import_module("pomelo_orbit.infrastructure.migration.migrator")

    engine = create_engine(f"sqlite:///{output.resolve()}")
    migrator.run_migrations(engine, migrations_dir=str(migrations_dir))
    logger.info("Migration replay validation passed")


def ensure_writable(path: Path, force: bool) -> None:
    if path.exists():
        if not force:
            raise RuntimeError(f"Output already exists: {path}. Use --force to overwrite.")
        path.unlink()
        logger.info("Removed existing output: path=%s", path)
    path.parent.mkdir(parents=True, exist_ok=True)


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
    args = parse_args()
    logger.info(
        "Starting SQLite rebuild: backup=%s, output=%s, schema=%s, migrations_dir=%s",
        args.backup,
        args.output,
        args.schema,
        args.migrations_dir,
    )
    ensure_writable(args.output, args.force)
    if args.json_output:
        ensure_writable(args.json_output, args.force)

    old = sqlite3.connect(f"file:{args.backup.resolve()}?mode=ro", uri=True)
    old.row_factory = sqlite3.Row
    new = sqlite3.connect(args.output)
    new.row_factory = sqlite3.Row
    new.execute("PRAGMA foreign_keys = OFF")
    logger.info("Applying base schema: schema=%s", args.schema)
    new.executescript(args.schema.read_text(encoding="utf-8"))
    create_history_table(new)

    current_history = calculate_current_history(args.migrations_dir)
    rows_by_table = collect_rows(
        old=old,
        new=new,
        current_history=current_history,
        default_project_id=args.default_project_id,
        default_project_name=args.default_project_name,
        default_project_code=args.default_project_code,
    )

    counts = {}
    for table, rows, columns in rows_by_table:
        counts[table] = insert_rows(new, table, rows, columns)
        logger.info("Imported table rows: table=%s, rows=%s", table, counts[table])

    new.commit()
    logger.info("Rebuilt database committed: output=%s", args.output)
    new.close()
    old.close()

    if args.json_output:
        write_json_output(args.json_output, rows_by_table)

    validate_database(args.output, args.migrations_dir)
    logger.info("SQLite rebuild completed successfully: output=%s", args.output)

    print(f"Rebuilt database: {args.output}")
    if args.json_output:
        print(f"Import JSON: {args.json_output}")
    for table, count in counts.items():
        print(f"{table}: {count}")
    print("Validation: ok")


if __name__ == "__main__":
    main()
