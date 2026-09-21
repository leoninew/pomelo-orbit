#!/usr/bin/env python3
"""Recreate the development PostgreSQL database and apply Orbit migrations.

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
import subprocess
import sys
from pathlib import Path
from urllib.parse import quote

from dotenv import load_dotenv

from dbtalk.database import DatabaseClient, DatabaseOperationError, ParsedDsn, parse_dsn
from dbtalk.postgres.database import create_database, drop_database, list_databases
from dbtalk.postgres.role import grant_profile, list_roles

PROJECT_ROOT = Path(__file__).resolve().parent.parent
_ENV_FILE = Path(__file__).resolve().parent / ".env"


def _require_env(name: str) -> str:
    value = os.getenv(name)
    if not value:
        raise RuntimeError(f"Missing required environment variable: {name} (check scripts/.env)")
    return value


# ---------------------------------------------------------------------------
# DSN helpers
# ---------------------------------------------------------------------------


def _admin_dsn(database: str | None = None) -> ParsedDsn:
    """Return the admin ParsedDsn from DB_ADMIN_DSN, optionally switching to *database*."""
    parsed = parse_dsn(_require_env("DB_ADMIN_DSN"))
    if database is None:
        return parsed
    return ParsedDsn(
        url=parsed.url.set(database=database),
        dialect=parsed.dialect,
        async_mode=parsed.async_mode,
    )


def _target_migration_dsn() -> str:
    """Build the Go migration DSN, reusing host/port/sslmode from DB_ADMIN_DSN."""
    admin_url = parse_dsn(_require_env("DB_ADMIN_DSN")).url
    host = admin_url.host or "127.0.0.1"
    port = admin_url.port or 5432
    sslmode = (admin_url.query or {}).get("sslmode", "disable")
    user = quote(_require_env("TARGET_USERNAME"), safe="")
    pwd = quote(_require_env("TARGET_PASSWORD"), safe="")
    db = quote(_require_env("TARGET_DATABASE"), safe="")
    return f"postgres://{user}:{pwd}@{host}:{port}/{db}?sslmode={sslmode}"

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
            f'ALTER DATABASE "{target_database}" OWNER TO "{target_username}"',
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
    grant_profile(_admin_dsn(target_database), target_username, ("schema", "public"), "migrator")


# ---------------------------------------------------------------------------
# Step 4 – migrate (and optional E2E verification)
# ---------------------------------------------------------------------------


def migrate(migration_dsn: str) -> None:
    logging.info("Applying schema migrations.")
    subprocess.run(
        ["go", "run", "./cmd/migrate"],
        cwd=PROJECT_ROOT,
        env=os.environ | {
            "POMELO_ORBIT_APP__ENV": "development",
            "POMELO_ORBIT_DATABASE__DRIVER": "postgres",
            "POMELO_ORBIT_DATABASE__POSTGRES__DSN": migration_dsn,
        },
        check=True,
    )


def verify_e2e(migration_dsn: str) -> None:
    logging.info("Running PostgreSQL migration E2E test.")
    subprocess.run(
        ["go", "test", "./internal/test/e2e", "-run", "^TestPostgreSQLMigrationE2E$", "-count=1"],
        cwd=PROJECT_ROOT,
        env=os.environ | {"BACKEND_GO_POSTGRES_E2E_DSN": migration_dsn},
        check=True,
    )


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------


def main(run_e2e: bool) -> None:
    load_dotenv(_ENV_FILE)

    target_username = _require_env("TARGET_USERNAME")
    admin = _admin_dsn()

    # Pre-flight: the target role must already exist.
    if not any(role.name == target_username for role in list_roles(admin)):
        raise RuntimeError(f"PostgreSQL role does not exist: '{target_username}'")

    drop_if_exists(admin)
    create(admin)
    grant()

    migration_dsn = _target_migration_dsn()
    migrate(migration_dsn)
    if run_e2e:
        verify_e2e(migration_dsn)



def _parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
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
        main(_parse_args().verify_e2e)
    except (DatabaseOperationError, OSError, RuntimeError, subprocess.CalledProcessError) as exc:
        logging.exception(exc)
        raise SystemExit(1) from exc
