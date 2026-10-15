"""Load and validate the Orbit configuration used by operational commands."""

from __future__ import annotations

import logging
import os
import base64
import re
from collections.abc import Mapping
from dataclasses import dataclass
from pathlib import Path
from urllib.parse import urlsplit

from dotenv import dotenv_values
from dynaconf import Dynaconf

from pomelo_orbit_cli.errors import ConfigurationError

ENV_PREFIX = "POMELO_ORBIT"
ENV_SELECTOR = "POMELO_ORBIT_APP__ENV"
CONFIG_DIRECTORY = Path("configs")
CONFIG_FILENAME = "config.yaml"
ENVIRONMENT_KEYS = frozenset({"logging__level", "cli__base_url", "mcp__access_token"})


@dataclass(frozen=True)
class ClientSettings:
    base_url: str | None = None
    access_token: str | None = None

    def validated(self) -> ClientSettings:
        if not self.base_url or not self.access_token:
            raise ConfigurationError(
                "Set POMELO_ORBIT_CLI__BASE_URL and POMELO_ORBIT_MCP__ACCESS_TOKEN"
            )
        try:
            url = urlsplit(self.base_url)
            port = url.port
        except ValueError as error:
            raise ConfigurationError("Invalid cli.base_url") from error
        if (
            url.scheme not in {"http", "https"}
            or not url.hostname
            or url.username is not None
            or url.password is not None
            or url.query
            or url.fragment
            or any(c.isspace() or ord(c) < 32 for c in self.base_url)
            or (port is not None and port < 1)
        ):
            raise ConfigurationError("cli.base_url must be an HTTP(S) service root")
        match = re.fullmatch(r"orbit_mcp_pat_([A-Za-z0-9_-]{43})", self.access_token)
        if not match or len(base64.urlsafe_b64decode(match[1] + "=")) != 32:
            raise ConfigurationError("mcp.access_token must be an Orbit PAT")
        return self


@dataclass(frozen=True)
class Settings:
    """Fully validated process configuration shared through the Click context."""

    logging_level: str
    environment: str
    project_root: Path
    dotenv_path: Path | None
    client: ClientSettings = ClientSettings()


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
        logging_values = _mapping(config.get("logging", {}))
        settings = Settings(
            logging_level=_required_string(
                logging_values.get("level"), "logging.level"
            ),
            environment=environment,
            project_root=resolved_root,
            dotenv_path=_existing_dotenv_path(resolved_root, environment),
            client=ClientSettings(
                base_url=_optional_string(
                    _mapping(config.get("cli", {})).get("base_url")
                ),
                access_token=_optional_string(
                    _mapping(config.get("mcp", {})).get("access_token")
                ),
            ),
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
