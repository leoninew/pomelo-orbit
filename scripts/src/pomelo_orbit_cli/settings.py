"""Load and validate the Orbit configuration used by operational commands."""

from __future__ import annotations

import logging
import os
import re
from collections.abc import Mapping
from dataclasses import dataclass, field
from pathlib import Path
from typing import Literal
from urllib.parse import quote, urlsplit

from cryptography.fernet import Fernet
from dotenv import dotenv_values
from dynaconf import Dynaconf

from pomelo_orbit_cli.errors import ConfigurationError

ENV_PREFIX = "POMELO_ORBIT"
ENV_SELECTOR = "POMELO_ORBIT_APP__ENV"
CONFIG_DIRECTORY = Path("configs")
CONFIG_FILENAME = "config.yaml"
DatabaseDriver = Literal["sqlite", "mysql", "postgresql"]
ENVIRONMENT_KEYS = frozenset(
    {
        "database__driver",
        "database__sqlite__path",
        "database__mysql__dsn",
        "database__postgres__dsn",
        "jwt__secret_key",
        "logging__level",
    }
)


@dataclass(frozen=True)
class DatabaseSettings:
    """Typed database values mapped from the shared Go configuration."""

    driver: str
    sqlite_path: str | None
    mysql_dsn: str | None = field(repr=False)
    postgres_dsn: str | None = field(repr=False)


@dataclass(frozen=True)
class DatabaseConnection:
    """Database details normalized for the pomelo-dbtalk Python API."""

    driver: DatabaseDriver
    dsn: str = field(repr=False)


@dataclass(frozen=True)
class Settings:
    """Fully validated process configuration shared through the Click context."""

    database: DatabaseSettings
    connection: DatabaseConnection = field(repr=False)
    fernet: Fernet = field(repr=False)
    logging_level: str
    environment: str
    project_root: Path
    dotenv_path: Path | None


def selected_environment() -> str:
    """Read the Go-compatible profile selector before loading dotenv values."""
    value = os.environ.get(ENV_SELECTOR, "").strip()
    if any(separator in value for separator in ("/", "\\", "\x00")):
        raise ConfigurationError(f"{ENV_SELECTOR} must not contain a path separator")
    return value


def project_root() -> Path:
    """Locate the Orbit repository from the working directory or installed source."""
    working_directory = Path.cwd().resolve()
    for candidate in (working_directory, *working_directory.parents):
        if _is_project_root(candidate):
            return candidate

    source_root = Path(__file__).resolve().parents[3]
    if _is_project_root(source_root):
        return source_root
    raise ConfigurationError("could not locate the Pomelo Orbit project root")


def load_settings(root: Path | None = None) -> Settings:
    """Load all layers once, map them to dataclasses, and validate at startup."""
    try:
        resolved_root = (root or project_root()).resolve()
        environment = selected_environment()
        config = _load_dynaconf(resolved_root, environment)
        database_values = _mapping(config.get("database", {}))
        sqlite_values = _mapping(database_values.get("sqlite"))
        mysql_values = _mapping(database_values.get("mysql"))
        postgres_values = _mapping(database_values.get("postgres"))
        jwt_values = _mapping(config.get("jwt", {}))
        logging_values = _mapping(config.get("logging", {}))
        database = DatabaseSettings(
            driver=_required_string(database_values.get("driver"), "database.driver"),
            sqlite_path=_optional_string(sqlite_values.get("path")),
            mysql_dsn=_optional_string(mysql_values.get("dsn")),
            postgres_dsn=_optional_string(postgres_values.get("dsn")),
        )
        settings = Settings(
            database=database,
            connection=_database_connection(resolved_root, database),
            fernet=_fernet(
                _required_string(jwt_values.get("secret_key"), "jwt.secret_key")
            ),
            logging_level=_required_string(
                logging_values.get("level"), "logging.level"
            ),
            environment=environment,
            project_root=resolved_root,
            dotenv_path=_existing_dotenv_path(resolved_root, environment),
        )
        validate_settings(settings)
        return settings
    except ConfigurationError:
        raise
    except (OSError, TypeError, ValueError) as error:
        raise ConfigurationError(str(error)) from error


def validate_settings(settings: Settings) -> None:
    """Validate every CLI setting before any command can execute."""
    if settings.logging_level.upper() not in logging.getLevelNamesMapping():
        raise ConfigurationError("logging.level must be a valid logging level")


def _is_project_root(candidate: Path) -> bool:
    return (
        (candidate / "go.mod").is_file()
        and (candidate / CONFIG_DIRECTORY / CONFIG_FILENAME).is_file()
        and (candidate / "scripts" / "pyproject.toml").is_file()
    )


def _load_dynaconf(root: Path, environment: str) -> Dynaconf:
    base_config = root / CONFIG_DIRECTORY / CONFIG_FILENAME
    if not base_config.is_file():
        raise ConfigurationError(f"config file not found: {base_config}")

    settings_files = [str(base_config)]
    if environment:
        profile_config = root / CONFIG_DIRECTORY / f"config.{environment}.yaml"
        if profile_config.is_file():
            settings_files.append(str(profile_config))

    config = Dynaconf(
        settings_files=settings_files,
        envvar_prefix=None,
        load_dotenv=False,
        environments=False,
        merge_enabled=True,
    )
    dotenv_path = _dotenv_path(root, environment)
    if dotenv_path.is_file():
        _apply_environment_overrides(config, dotenv_values(dotenv_path))
    _apply_environment_overrides(config, os.environ)
    return config


def _existing_dotenv_path(root: Path, environment: str) -> Path | None:
    path = _dotenv_path(root, environment)
    return path if path.is_file() else None


def _dotenv_path(root: Path, environment: str) -> Path:
    filename = f".env.{environment}" if environment else ".env"
    return root / filename


def _apply_environment_overrides(
    config: Dynaconf, values: Mapping[str, str | None]
) -> None:
    prefix = f"{ENV_PREFIX}_"
    for name, value in values.items():
        key = name.removeprefix(prefix).lower()
        if name.startswith(prefix) and key in ENVIRONMENT_KEYS and value is not None:
            config.set(key, value, tomlfy=True)


def _mapping(value: object) -> dict[str, object]:
    if value is None:
        return {}
    if not isinstance(value, Mapping):
        raise ConfigurationError("configuration sections must be mappings")
    return {str(key): item for key, item in value.items()}


def _required_string(value: object, field_name: str) -> str:
    result = _optional_string(value)
    if result is None:
        raise ConfigurationError(f"{field_name} is required")
    return result


def _optional_string(value: object) -> str | None:
    if value is None:
        return None
    if not isinstance(value, str):
        raise ConfigurationError("configuration values must be strings")
    stripped = value.strip()
    return stripped or None


def _fernet(value: str) -> Fernet:
    try:
        encoded = value.encode("ascii")
        encoded += b"=" * (-len(encoded) % 4)
        return Fernet(encoded)
    except (UnicodeEncodeError, ValueError) as error:
        raise ConfigurationError("jwt.secret_key must be a valid Fernet key") from error


def _database_connection(root: Path, database: DatabaseSettings) -> DatabaseConnection:
    if database.driver == "sqlite":
        if database.sqlite_path is None:
            raise ConfigurationError("database.sqlite.path is required")
        return DatabaseConnection("sqlite", _sqlite_dsn(root, database.sqlite_path))
    if database.driver == "mysql":
        if database.mysql_dsn is None:
            raise ConfigurationError("database.mysql.dsn is required")
        return DatabaseConnection("mysql", _mysql_dsn(database.mysql_dsn))
    if database.driver == "postgres":
        if database.postgres_dsn is None:
            raise ConfigurationError("database.postgres.dsn is required")
        return DatabaseConnection("postgresql", _postgres_dsn(database.postgres_dsn))
    raise ConfigurationError("database.driver must be sqlite, mysql or postgres")


def _sqlite_dsn(root: Path, value: str) -> str:
    path = Path(value).expanduser()
    if not path.is_absolute():
        path = root / path
    return f"sqlite:///{quote(path.resolve().as_posix(), safe='/:')}"


def _mysql_dsn(value: str) -> str:
    if value.startswith("mysql+pymysql://"):
        _validate_url(value, "mysql+pymysql", "database.mysql.dsn")
        return value
    if value.startswith("mysql://"):
        canonical = "mysql+pymysql://" + value.removeprefix("mysql://")
        _validate_url(canonical, "mysql+pymysql", "database.mysql.dsn")
        return canonical

    match = re.fullmatch(
        r"(?P<user>[^:@/]+)(?::(?P<password>.*))?@tcp\((?P<host>[^)]+)\)/(?P<database>[^?]+)(?:\?(?P<query>.*))?",
        value,
    )
    if match is None:
        raise ConfigurationError(
            "database.mysql.dsn must be a MySQL URL or a Go tcp DSN"
        )
    password = match.group("password") or ""
    return (
        "mysql+pymysql://"
        f"{quote(match.group('user'), safe='')}:{quote(password, safe='')}"
        f"@{match.group('host')}/{quote(match.group('database'), safe='')}"
    )


def _postgres_dsn(value: str) -> str:
    if value.startswith("postgresql+psycopg://"):
        _validate_url(value, "postgresql+psycopg", "database.postgres.dsn")
        return value
    if value.startswith("postgresql://"):
        canonical = "postgresql+psycopg://" + value.removeprefix("postgresql://")
        _validate_url(canonical, "postgresql+psycopg", "database.postgres.dsn")
        return canonical
    if value.startswith("postgres://"):
        canonical = "postgresql+psycopg://" + value.removeprefix("postgres://")
        _validate_url(canonical, "postgresql+psycopg", "database.postgres.dsn")
        return canonical
    raise ConfigurationError("database.postgres.dsn must be a PostgreSQL URL")


def _validate_url(value: str, scheme: str, field_name: str) -> None:
    parsed = urlsplit(value)
    if parsed.scheme != scheme or not parsed.hostname or not parsed.path.strip("/"):
        raise ConfigurationError(f"{field_name} must be an absolute database URL")
