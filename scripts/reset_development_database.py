#!/usr/bin/env python3
"""Recreate the development PostgreSQL database and apply Orbit migrations."""

from __future__ import annotations

import argparse
import logging
import os
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path
from urllib.parse import quote, urlsplit, urlunsplit

from dbtalk.database import DatabaseClient, DatabaseOperationError, ParsedDsn, parse_dsn
from dbtalk.postgres.database import (
    create_database,
    drop_database,
    list_databases,
)
from dbtalk.postgres.role import grant_profile, list_roles
from dotenv import dotenv_values

TARGET_DATABASE = "pomelo_orbit_0921"
TARGET_ROLE = "orbit"
SCRIPTS_DIRECTORY = Path(__file__).resolve().parent
PROJECT_ROOT = SCRIPTS_DIRECTORY.parent


@dataclass(frozen=True)
class ResetConfig:
    management: ParsedDsn
    administration: ParsedDsn
    migration_dsn: str


def required_value(values: dict[str, str | None], key: str, path: Path) -> str:
    value = values.get(key)
    if not isinstance(value, str) or not value.strip():
        raise RuntimeError(f"{key} is required in {path}")
    return value.strip()


def target_database_dsn(value: str) -> str:
    parsed = urlsplit(value.strip())
    if (
        parsed.scheme not in {"postgres", "postgresql"}
        or not parsed.netloc
        or not parsed.path.strip("/")
    ):
        raise RuntimeError(
            "POMELO_ORBIT_DATABASE__POSTGRES__DSN must be a PostgreSQL URL"
        )
    return urlunsplit(
        (
            parsed.scheme,
            parsed.netloc,
            f"/{quote(TARGET_DATABASE, safe='')}",
            parsed.query,
            "",
        )
    )


def with_database(parsed: ParsedDsn, database_name: str) -> ParsedDsn:
    return ParsedDsn(
        url=parsed.url.set(database=database_name),
        dialect=parsed.dialect,
        async_mode=parsed.async_mode,
    )


def execute(parsed: ParsedDsn, statement: str, parameters: dict[str, object]) -> None:
    with DatabaseClient(parsed) as client:
        client.execute(statement, parameters, read_only=False)


def load_config() -> ResetConfig:
    dbtalk_path = SCRIPTS_DIRECTORY / ".env"
    dbtalk_values = dotenv_values(dbtalk_path)
    development_path = PROJECT_ROOT / ".env.development"
    development_values = dotenv_values(development_path)
    return ResetConfig(
        management=parse_dsn(
            required_value(dbtalk_values, "DBTALK_DSN_POSTGRES_MANAGEMENT", dbtalk_path)
        ),
        administration=parse_dsn(
            required_value(dbtalk_values, "DBTALK_DSN_POSTGRES_ADMIN", dbtalk_path)
        ),
        migration_dsn=target_database_dsn(
            required_value(
                development_values,
                "POMELO_ORBIT_DATABASE__POSTGRES__DSN",
                development_path,
            )
        ),
    )


def apply_migrations(migration_dsn: str) -> None:
    environment = os.environ | {
        "POMELO_ORBIT_APP__ENV": "development",
        "POMELO_ORBIT_DATABASE__POSTGRES__DSN": migration_dsn,
    }
    subprocess.run(
        ["go", "run", "./cmd/migrate"],
        cwd=PROJECT_ROOT,
        env=environment,
        check=True,
    )


def verify_postgres_e2e(migration_dsn: str) -> None:
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


def main(verify_e2e: bool) -> None:
    config = load_config()
    if not any(role.name == TARGET_ROLE for role in list_roles(config.administration)):
        raise RuntimeError(f"PostgreSQL role does not exist: {TARGET_ROLE}")

    logging.info("Using the dbtalk PostgreSQL management connection.")
    logging.info("Listing development databases.")
    databases = list_databases(config.management)
    logging.info("Development databases: %s", databases)

    if TARGET_DATABASE in databases:
        logging.info("Terminating active development database sessions.")
        execute(
            config.management,
            "SELECT pg_terminate_backend(pid) FROM pg_stat_activity "
            "WHERE datname = :database AND pid <> pg_backend_pid()",
            {"database": TARGET_DATABASE},
        )
        logging.info("Dropping the development database.")
        drop_database(config.management, TARGET_DATABASE)
    else:
        logging.info("Development database does not exist; creating it.")

    logging.info("Creating the development database.")
    create_database(config.management, TARGET_DATABASE)
    logging.info("Assigning ownership of the development database.")
    execute(
        config.management,
        f'ALTER DATABASE "{TARGET_DATABASE}" OWNER TO "{TARGET_ROLE}"',
        {},
    )
    logging.info("Granting migrator access to the development database.")
    grant_profile(
        with_database(config.administration, TARGET_DATABASE),
        TARGET_ROLE,
        ("schema", "public"),
        "migrator",
    )

    logging.info("Applying development schema migrations.")
    apply_migrations(config.migration_dsn)
    if verify_e2e:
        logging.info("Verifying migrations with the PostgreSQL E2E test.")
        verify_postgres_e2e(config.migration_dsn)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--verify-e2e",
        action="store_true",
        help="run the PostgreSQL migration end-to-end test after migration",
    )
    return parser.parse_args()


if __name__ == "__main__":
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(message)s",
        stream=sys.stdout,
    )
    try:
        main(parse_args().verify_e2e)
    except (
        DatabaseOperationError,
        OSError,
        RuntimeError,
        subprocess.CalledProcessError,
    ) as error:
        logging.error("%s", error)
        raise SystemExit(1) from error
