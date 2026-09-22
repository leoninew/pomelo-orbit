#!/usr/bin/env python3
"""Recreate the development PostgreSQL database and apply Orbit migrations.

By default the script prints the database reset plan without executing it.
Pass ``--no-dry-run`` to terminate target-database sessions, recreate the
database, grant the migration profile, and apply Orbit schema migrations.

Steps
-----
1. Drop the target database if it exists (terminating active sessions first).
2. Create the target database.
3. Grant the target role ownership + migrator profile on the public schema.
4. Run schema migrations (and optionally the PostgreSQL E2E test).
"""

from __future__ import annotations

import argparse
import logging
import os
import shutil
import subprocess
import sys
from pathlib import Path
from urllib.parse import SplitResult, quote, unquote, urlsplit, urlunsplit

from dotenv import load_dotenv

from dbtalk.database import DatabaseClient, DatabaseOperationError, ParsedDsn, parse_dsn
from dbtalk.postgres.database import create_database, drop_database, list_databases
from dbtalk.postgres.role import grant_profile, list_roles

PROJECT_ROOT = Path(__file__).resolve().parent.parent
_ENV_FILE = Path(__file__).resolve().parent / ".env"
_ADMIN_DSN_ENV = "DB_ADMIN_DSN"
_SYSTEM_DATABASES = frozenset({"postgres", "template0", "template1"})


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


def main(*, dry_run: bool, run_e2e: bool) -> None:
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


def _parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--no-dry-run",
        action="store_false",
        dest="dry_run",
        default=True,
        help="execute the database reset instead of printing its plan",
    )
    parser.add_argument(
        "--verify-e2e",
        action="store_true",
        help="run the PostgreSQL migration E2E test after migration",
    )
    return parser.parse_args()


if __name__ == "__main__":
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(message)s",
        stream=sys.stdout,
    )
    try:
        arguments = _parse_args()
        main(dry_run=arguments.dry_run, run_e2e=arguments.verify_e2e)
    except (
        DatabaseOperationError,
        OSError,
        RuntimeError,
        subprocess.CalledProcessError,
    ) as exc:
        logging.exception(exc)
        raise SystemExit(1) from exc
