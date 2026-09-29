#!/usr/bin/env python3
"""Reset, back up, or copy data for the development database."""

from __future__ import annotations

import logging
import os
import shutil
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path
from urllib.parse import SplitResult, quote, unquote, urlsplit, urlunsplit
from zoneinfo import ZoneInfo

import click
from dotenv import dotenv_values, load_dotenv

from dbtalk.database import DatabaseClient, DatabaseOperationError, ParsedDsn, parse_dsn
from dbtalk.database.dsn import sqlite_dsn
from dbtalk.database.transfer import (
    ExportOptions,
    ImportOptions,
    TransferConnection,
    export_database,
    import_database,
    read_jsonl,
)
from dbtalk.mysql.client import docker_mapped_mysql_container
from dbtalk.mysql.dump import MysqlDumpOptions, dump_database as dump_mysql_database
from dbtalk.postgres.client import PostgresConnection, docker_mapped_postgres_container
from dbtalk.postgres.database import create_database, drop_database, list_databases
from dbtalk.postgres.dump import (
    PostgresDumpOptions,
    dump_database as dump_postgres_database,
)
from dbtalk.postgres.role import grant_profile, list_roles

PROJECT_ROOT = Path(__file__).resolve().parent.parent
_ENV_FILE = Path(__file__).resolve().parent / ".env"
_ADMIN_DSN_ENV = "DB_ADMIN_DSN"
_SYSTEM_DATABASES = frozenset({"postgres", "template0", "template1"})
SQLITE_DATABASE = PROJECT_ROOT / "data" / "db" / "pomelo-orbit.db"
POSTGRES_DSN = dotenv_values(_ENV_FILE, interpolate=False).get("POSTGRES_DSN") or ""
MYSQL_DSN = dotenv_values(_ENV_FILE, interpolate=False).get("MYSQL_DSN") or ""
TRANSFER_TIMEZONE = ZoneInfo("UTC")
OUTPUT_DIRECTORY = PROJECT_ROOT / "data" / "db" / "transfers"
BACKUP_DIRECTORY = Path(__file__).resolve().parent / "backup"
POSTGRES_CLIENT_IMAGE = "postgres:18"
MYSQL_CLIENT_IMAGE = "mysql:8.4"


def _require_env(name: str) -> str:
    value = os.getenv(name)
    if not value:
        raise RuntimeError(
            f"Missing required environment variable: {name} (check scripts/.env)"
        )
    return value


# ---------------------------------------------------------------------------
# DSN helpers
# ---------------------------------------------------------------------------


def _admin_dsn(database: str | None = None) -> ParsedDsn:
    """Return the admin ParsedDsn from DB_ADMIN_DSN, optionally switching to *database*."""
    parsed = parse_dsn(_require_env(_ADMIN_DSN_ENV))
    if database is None:
        return parsed
    return ParsedDsn(
        url=parsed.url.set(database=database),
        dialect=parsed.dialect,
        async_mode=parsed.async_mode,
    )


def _admin_url() -> SplitResult:
    parse_dsn(_require_env(_ADMIN_DSN_ENV))
    admin_url = urlsplit(_require_env(_ADMIN_DSN_ENV))
    if admin_url.scheme != "postgresql+psycopg" or not admin_url.hostname:
        raise RuntimeError(
            f"{_ADMIN_DSN_ENV} must be a PostgreSQL SQLAlchemy-style DSN"
        )
    try:
        _ = admin_url.port
    except ValueError as exc:
        raise RuntimeError(f"{_ADMIN_DSN_ENV} has an invalid port") from exc
    return admin_url


def _target_migration_dsn() -> str:
    """Build the Go migration DSN from the validated management connection."""
    admin_url = _admin_url()
    host = admin_url.hostname
    if host is None:
        raise RuntimeError(f"{_ADMIN_DSN_ENV} must include a hostname")
    if ":" in host:
        host = f"[{host}]"
    port = admin_url.port
    user = quote(_require_env("TARGET_USERNAME"), safe="")
    pwd = quote(_require_env("TARGET_PASSWORD"), safe="")
    db = quote(_require_env("TARGET_DATABASE"), safe="")
    netloc = f"{user}:{pwd}@{host}"
    if port is not None:
        netloc = f"{netloc}:{port}"
    return urlunsplit(("postgres", netloc, f"/{db}", admin_url.query, ""))


def _validate_reset_target() -> None:
    admin_url = _admin_url()
    management_database = unquote(admin_url.path.lstrip("/"))
    target_database = _require_env("TARGET_DATABASE")
    if target_database.lower() in _SYSTEM_DATABASES:
        raise RuntimeError(
            f"Refusing to reset PostgreSQL system database: {target_database}"
        )
    if target_database == management_database:
        raise RuntimeError("TARGET_DATABASE must differ from the DB_ADMIN_DSN database")


def _quote_identifier(value: str) -> str:
    return f'"{value.replace(chr(34), chr(34) * 2)}"'


def _require_executables() -> None:
    if shutil.which("go") is None:
        raise RuntimeError("Required executable not found on PATH: go")


# ---------------------------------------------------------------------------
# Step 1 – drop if exists
# ---------------------------------------------------------------------------


def drop_if_exists(admin: ParsedDsn) -> None:
    target_database = _require_env("TARGET_DATABASE")
    existing = list_databases(admin)
    if target_database not in existing:
        logging.info("Database '%s' does not exist; nothing to drop.", target_database)
        return

    logging.info("Terminating active connections to '%s'.", target_database)
    with DatabaseClient(admin) as client:
        client.execute(
            "SELECT pg_terminate_backend(pid) FROM pg_stat_activity "
            "WHERE datname = :database AND pid <> pg_backend_pid()",
            {"database": target_database},
            read_only=False,
        )

    logging.info("Dropping database '%s'.", target_database)
    drop_database(admin, target_database)


# ---------------------------------------------------------------------------
# Step 2 – create
# ---------------------------------------------------------------------------


def create(admin: ParsedDsn) -> None:
    target_database = _require_env("TARGET_DATABASE")
    target_username = _require_env("TARGET_USERNAME")
    logging.info("Creating database '%s'.", target_database)
    create_database(admin, target_database)

    logging.info("Transferring ownership to role '%s'.", target_username)
    with DatabaseClient(admin) as client:
        client.execute(
            f"ALTER DATABASE {_quote_identifier(target_database)} OWNER TO {_quote_identifier(target_username)}",
            {},
            read_only=False,
        )


# ---------------------------------------------------------------------------
# Step 3 – grant
# ---------------------------------------------------------------------------


def grant() -> None:
    target_database = _require_env("TARGET_DATABASE")
    target_username = _require_env("TARGET_USERNAME")
    logging.info("Granting migrator profile on public schema to '%s'.", target_username)
    # Re-connect the admin DSN against the newly created target database.
    grant_profile(
        _admin_dsn(target_database), target_username, ("schema", "public"), "migrator"
    )


# ---------------------------------------------------------------------------
# Step 4 – migrate (and optional E2E verification)
# ---------------------------------------------------------------------------


def migrate(migration_dsn: str) -> None:
    logging.info("Applying schema migrations.")
    subprocess.run(
        ["go", "run", "./cmd/migrate"],
        cwd=PROJECT_ROOT,
        env=os.environ
        | {
            "POMELO_ORBIT_APP__ENV": "development",
            "POMELO_ORBIT_DATABASE__DRIVER": "postgres",
            "POMELO_ORBIT_DATABASE__POSTGRES__DSN": migration_dsn,
        },
        check=True,
    )


def verify_e2e(migration_dsn: str) -> None:
    logging.info("Running PostgreSQL migration E2E test.")
    subprocess.run(
        [
            "go",
            "test",
            "./internal/test/e2e",
            "-run",
            "^TestPostgreSQLMigrationE2E$",
            "-count=1",
        ],
        cwd=PROJECT_ROOT,
        env=os.environ | {"BACKEND_GO_POSTGRES_E2E_DSN": migration_dsn},
        check=True,
    )


def _require_target_role(admin: ParsedDsn) -> None:
    target_username = _require_env("TARGET_USERNAME")
    logging.info("Checking PostgreSQL role '%s'.", target_username)
    if not any(role.name == target_username for role in list_roles(admin)):
        raise RuntimeError(f"PostgreSQL role does not exist: '{target_username}'")


def _log_dry_run(*, run_e2e: bool) -> None:
    target_database = _require_env("TARGET_DATABASE")
    target_username = _require_env("TARGET_USERNAME")
    database_identifier = _quote_identifier(target_database)
    role_identifier = _quote_identifier(target_username)
    logging.info(
        "Running: SELECT 1 FROM pg_roles WHERE rolname = '%s'", target_username
    )
    logging.info(
        "Running: SELECT datname FROM pg_database WHERE datname = '%s'", target_database
    )
    logging.info(
        "Running: SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' "
        "AND pid <> pg_backend_pid() (if the database exists)",
        target_database,
    )
    logging.info(
        "Running: DROP DATABASE %s (if the database exists)", database_identifier
    )
    logging.info("Running: CREATE DATABASE %s", database_identifier)
    logging.info(
        "Running: ALTER DATABASE %s OWNER TO %s", database_identifier, role_identifier
    )
    logging.info(
        "Running: grant migrator profile on public schema to '%s'", target_username
    )
    logging.info("Running: go run ./cmd/migrate")
    if run_e2e:
        logging.info(
            "Running: go test ./internal/test/e2e -run ^TestPostgreSQLMigrationE2E$ -count=1"
        )


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------


def reset_database(*, dry_run: bool, run_e2e: bool) -> None:
    load_dotenv(_ENV_FILE)
    _validate_reset_target()
    migration_dsn = _target_migration_dsn()
    if dry_run:
        _log_dry_run(run_e2e=run_e2e)
        return

    _require_executables()
    admin = _admin_dsn()
    _require_target_role(admin)

    drop_if_exists(admin)
    create(admin)
    grant()

    migrate(migration_dsn)
    if run_e2e:
        verify_e2e(migration_dsn)


@click.group()
def cli() -> None:
    """Database operations for the development environment."""
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(message)s",
        stream=sys.stdout,
    )
    logging.getLogger("dbtalk").setLevel(logging.WARNING)


@cli.command()
@click.option("--no-dry-run", is_flag=True, help="execute the reset plan")
@click.option(
    "--verify-e2e", is_flag=True, help="run the PostgreSQL migration E2E test"
)
def reset(no_dry_run: bool, verify_e2e: bool) -> None:
    """Recreate PostgreSQL and apply Orbit migrations; dry-run by default."""
    try:
        reset_database(dry_run=not no_dry_run, run_e2e=verify_e2e)
    except (
        DatabaseOperationError,
        OSError,
        RuntimeError,
        subprocess.CalledProcessError,
    ) as error:
        logging.exception("database reset failed")
        raise click.exceptions.Exit(1) from error


def table_counts(path: Path) -> dict[str, int]:
    with path.open(encoding="utf-8") as stream:
        _, tables = read_jsonl(stream)
    return {table.header.name: len(table.rows) for table in tables}


def _log_copy_dry_run() -> None:
    logging.info("Running: check SQLite foreign keys in %s", SQLITE_DATABASE)
    logging.info("Running: check PostgreSQL schema_migrations using scripts/.env")
    logging.info(
        "Running: export SQLite tables except schema_migrations to %s",
        OUTPUT_DIRECTORY,
    )
    logging.info("Running: upsert exported rows into PostgreSQL")
    logging.info("Running: export PostgreSQL tables and compare row counts")


def copy_from_sqlite_data() -> None:
    logging.info("database copy started source=sqlite target=postgresql")
    if not POSTGRES_DSN:
        raise RuntimeError("POSTGRES_DSN is required in scripts/.env")
    source_dsn = sqlite_dsn(SQLITE_DATABASE)
    with DatabaseClient(source_dsn) as source:
        violations = source.query("SELECT COUNT(*) FROM pragma_foreign_key_check").rows[
            0
        ][0]
    if violations:
        raise RuntimeError(f"source SQLite has {violations} foreign key violations")

    with DatabaseClient(POSTGRES_DSN) as target:
        migration = target.query("SELECT version, dirty FROM schema_migrations").rows
    if len(migration) != 1 or migration[0][1]:
        raise RuntimeError("target PostgreSQL schema is not migrated and clean")

    OUTPUT_DIRECTORY.mkdir(parents=True, exist_ok=True)
    stamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%S%fZ")
    source_file = OUTPUT_DIRECTORY / f"sqlite-{stamp}.jsonl"
    target_file = OUTPUT_DIRECTORY / f"postgres-{stamp}.jsonl"

    source_connection = TransferConnection(driver="sqlite", dsn=source_dsn)
    target_connection = TransferConnection(driver="postgresql", dsn=POSTGRES_DSN)
    source_summary = export_database(
        ExportOptions(
            connection=source_connection,
            output=source_file,
            timezone=TRANSFER_TIMEZONE,
            exclude_tables=("schema_migrations",),
        )
    )
    logging.info(
        "SQLite export completed tables=%d rows=%d path=%s",
        source_summary.table_count,
        source_summary.row_count,
        source_file,
    )
    import_summary = import_database(
        ImportOptions(
            connection=target_connection,
            input=source_file,
            mode="upsert",
            timezone=TRANSFER_TIMEZONE,
        )
    )
    logging.info(
        "PostgreSQL import completed tables=%d rows=%d",
        import_summary.table_count,
        import_summary.row_count,
    )
    export_database(
        ExportOptions(
            connection=target_connection,
            output=target_file,
            timezone=TRANSFER_TIMEZONE,
            exclude_tables=("schema_migrations",),
        )
    )

    source_counts = table_counts(source_file)
    target_counts = table_counts(target_file)
    if source_counts != target_counts:
        differences = {
            table: (source_counts.get(table), target_counts.get(table))
            for table in source_counts.keys() | target_counts.keys()
            if source_counts.get(table) != target_counts.get(table)
        }
        raise RuntimeError(f"source/target row counts differ: {differences}")
    logging.info(
        "database copy verified tables=%d rows=%d target_export=%s",
        len(source_counts),
        sum(source_counts.values()),
        target_file,
    )


@cli.command("copy-from-sqlite")
@click.option("--no-dry-run", is_flag=True, help="execute the SQLite data copy")
def copy_from_sqlite(no_dry_run: bool) -> None:
    """Copy SQLite rows into PostgreSQL; dry-run by default."""
    try:
        if no_dry_run:
            copy_from_sqlite_data()
        else:
            _log_copy_dry_run()
    except Exception as error:
        logging.exception("database copy failed")
        raise click.exceptions.Exit(1) from error


def _backup_client_image(engine: str, host: str, port: int) -> str:
    executable = "pg_dump" if engine == "postgres" else "mysqldump"
    mapped_container = (
        docker_mapped_postgres_container
        if engine == "postgres"
        else docker_mapped_mysql_container
    )
    if shutil.which(executable) is not None or mapped_container(host, port) is not None:
        return ""

    image = POSTGRES_CLIENT_IMAGE if engine == "postgres" else MYSQL_CLIENT_IMAGE
    if shutil.which("docker") is None:
        raise RuntimeError(f"{executable} is unavailable and Docker is not installed")
    inspection = subprocess.run(
        ["docker", "image", "inspect", image],
        check=False,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    if inspection.returncode != 0:
        raise RuntimeError(
            f"{executable} is unavailable and Docker image {image} is not local"
        )
    return image


def backup_database(*, engine: str) -> None:
    dsn = POSTGRES_DSN if engine == "postgres" else MYSQL_DSN
    if not dsn:
        name = "POSTGRES_DSN" if engine == "postgres" else "MYSQL_DSN"
        raise RuntimeError(f"{name} is required in scripts/.env")
    parsed = parse_dsn(dsn)
    expected_dialect = "postgresql" if engine == "postgres" else "mysql"
    if parsed.dialect != expected_dialect:
        raise RuntimeError(f"{engine} backup requires a {expected_dialect} DSN")
    if not parsed.database:
        raise RuntimeError(f"{engine} backup DSN must include a database")

    stamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%S%fZ")
    extension = "dump" if engine == "postgres" else "sql"
    output = (
        BACKUP_DIRECTORY
        / f"{engine}-{quote(parsed.database, safe='')}-{stamp}.{extension}"
    )
    BACKUP_DIRECTORY.mkdir(parents=True, exist_ok=True)
    logging.info(
        "database backup started engine=%s database=%s", engine, parsed.database
    )
    if engine == "postgres":
        connection = PostgresConnection.from_parsed_dsn(parsed)
        completed = dump_postgres_database(
            PostgresDumpOptions(
                connection=connection,
                output=output,
                client_image=_backup_client_image(
                    engine, connection.host, connection.port
                ),
            )
        )
    else:
        host = parsed.host
        user = parsed.url.username
        if not host or not user:
            raise RuntimeError("MySQL backup DSN must include host and user")
        completed = dump_mysql_database(
            MysqlDumpOptions(
                host=host,
                port=parsed.port or 3306,
                user=user,
                password=parsed.url.password or "",
                database=parsed.database,
                output=output,
                client_image=_backup_client_image(engine, host, parsed.port or 3306),
            )
        )
    if not completed.is_file() or completed.stat().st_size == 0:
        raise RuntimeError(f"database backup is empty or missing: {completed}")
    logging.info(
        "database backup completed path=%s bytes=%d",
        completed,
        completed.stat().st_size,
    )


@cli.command()
@click.option(
    "--engine",
    type=click.Choice(["postgres", "mysql"]),
    default="postgres",
    show_default=True,
)
def backup(engine: str) -> None:
    """Create a native database dump."""
    try:
        backup_database(engine=engine)
    except (
        DatabaseOperationError,
        OSError,
        RuntimeError,
        ValueError,
        click.ClickException,
    ) as error:
        logging.exception("database backup failed")
        raise click.exceptions.Exit(1) from error


if __name__ == "__main__":
    cli()
