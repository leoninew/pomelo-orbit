"""Load and validate the Orbit configuration used by operational commands."""

from __future__ import annotations

import base64
import logging
import os
import re
from dataclasses import dataclass
from pathlib import Path
from urllib.parse import urlsplit

from dotenv import dotenv_values

from pomelo_orbit_cli.errors import ConfigurationError

ENV_SELECTOR = "POMELO_ORBIT_APP__ENV"


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
class RemoteSettings:
    ssh_host: str | None = None
    ssh_user: str | None = None

    @property
    def ssh_target(self) -> str:
        return f"{self.ssh_user}@{self.ssh_host}"

    def validated(self) -> RemoteSettings:
        missing = [
            name
            for name, value in (
                ("POMELO_ORBIT_SSH__HOST", self.ssh_host),
                ("POMELO_ORBIT_SSH__USER", self.ssh_user),
            )
            if not value
        ]
        if missing:
            raise ConfigurationError(f"Set {', '.join(missing)}")
        return self


@dataclass(frozen=True)
class Settings:
    """Fully validated process configuration shared through the Click context."""

    logging_level: str
    environment: str
    project_root: Path
    dotenv_path: Path | None
    client: ClientSettings = ClientSettings()
    remote: RemoteSettings = RemoteSettings()


def selected_environment() -> str:
    """Read the process profile selector before loading scripts dotenv values."""
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
        filename = f".env.{environment}" if environment else ".env"
        dotenv_path = resolved_root / "scripts" / filename
        values = dotenv_values(dotenv_path) if dotenv_path.is_file() else {}
        values.update(os.environ)
        settings = Settings(
            logging_level=_optional_string(values.get("POMELO_ORBIT_LOGGING__LEVEL"))
            or "INFO",
            environment=environment,
            project_root=resolved_root,
            dotenv_path=dotenv_path if dotenv_path.is_file() else None,
            client=ClientSettings(
                base_url=_optional_string(values.get("POMELO_ORBIT_CLI__BASE_URL")),
                access_token=_optional_string(
                    values.get("POMELO_ORBIT_MCP__ACCESS_TOKEN")
                ),
            ),
            remote=RemoteSettings(
                ssh_host=_optional_string(values.get("POMELO_ORBIT_SSH__HOST")),
                ssh_user=_optional_string(values.get("POMELO_ORBIT_SSH__USER")),
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
    return (candidate / "go.mod").is_file() and (
        candidate / "scripts" / "pyproject.toml"
    ).is_file()


def _optional_string(value: object) -> str | None:
    if value is None:
        return None
    if not isinstance(value, str):
        raise ConfigurationError("configuration values must be strings")
    stripped = value.strip()
    return stripped or None
