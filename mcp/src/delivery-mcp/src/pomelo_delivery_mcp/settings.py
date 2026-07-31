"""Configuration loading and local JWT cache handling."""

from __future__ import annotations

import base64
import json
import os
import tempfile
import time
from collections.abc import Mapping
from dataclasses import dataclass
from pathlib import Path
from typing import cast

from dynaconf import Dynaconf  # type: ignore[attr-defined]
from dynaconf.vendor.dotenv import dotenv_values


class SettingsError(ValueError):
    """Raised when local MCP configuration is incomplete or invalid."""


def _config_directory() -> Path:
    return Path(__file__).resolve().parents[2]


def _optional_text(value: object | None) -> str | None:
    if value is None:
        return None
    value = str(value).strip()
    return value or None


def _read_dotenv(path: Path) -> dict[str, str | None]:
    values = cast(Mapping[object, object], dotenv_values(str(path)))  # type: ignore[no-untyped-call]
    return {str(key): _optional_text(value) for key, value in values.items()}


def _positive_int(value: object | None, key: str, default: int, *, allow_zero: bool = False) -> int:
    if value is None or str(value).strip() == "":
        return default
    try:
        parsed = int(str(value))
    except ValueError as error:
        raise SettingsError(f"{key} must be an integer") from error
    if parsed < 0 or (parsed == 0 and not allow_zero):
        comparator = "zero or greater" if allow_zero else "greater than zero"
        raise SettingsError(f"{key} must be {comparator}")
    return parsed


@dataclass(frozen=True)
class Settings:
    """Validated settings for one local MCP process."""

    orbit_url: str
    username: str
    password: str
    data_root: Path
    docker_context: str | None
    wait_timeout_seconds: int
    stability_window_seconds: int
    stability_poll_seconds: int
    config_dir: Path
    jwt_from_environment: str | None = None

    @property
    def jwt_cache_path(self) -> Path:
        return self.config_dir / ".mcp-jwt.env"

    def docker_cli_environment(self) -> dict[str, str]:
        """Return the inherited Docker CLI environment with an optional context."""
        environment = os.environ.copy()
        if self.docker_context:
            environment["DOCKER_CONTEXT"] = self.docker_context
        return environment

    @classmethod
    def load(
        cls,
        config_dir: Path | None = None,
        environ: Mapping[str, str] | None = None,
    ) -> Settings:
        config_dir = (config_dir or _config_directory()).resolve()
        env_file = config_dir / ".env"
        process_values = dict(os.environ if environ is None else environ)

        # Dynaconf owns .env and process-environment loading. The cache is read
        # separately because it is intentionally written by this process only.
        dynasettings = Dynaconf(
            environments=False,
            envvar_prefix=False,
            load_dotenv=True,
            dotenv_path=str(env_file),
        )
        file_values = _read_dotenv(env_file) if env_file.exists() else {}

        def get_value(key: str) -> object | None:
            if key in process_values:
                return process_values[key]
            if environ is None:
                value = cast(object | None, dynasettings.get(key))
                if value is not None:
                    return value
            return file_values.get(key)

        orbit_url = _optional_text(get_value("POMELO_ORBIT_URL"))
        username = _optional_text(get_value("POMELO_ORBIT_USERNAME"))
        password = _optional_text(get_value("POMELO_ORBIT_PASSWORD"))
        configured_data_root = _optional_text(get_value("POMELO_ORBIT_DATA_ROOT"))
        missing = [
            key
            for key, value in {
                "POMELO_ORBIT_URL": orbit_url,
                "POMELO_ORBIT_USERNAME": username,
                "POMELO_ORBIT_PASSWORD": password,
            }.items()
            if not value
        ]
        if missing:
            raise SettingsError("missing required settings: " + ", ".join(missing))
        assert orbit_url is not None
        assert username is not None
        assert password is not None

        data_root = Path(configured_data_root).expanduser() if configured_data_root else config_dir.parents[2] / "data"
        try:
            data_root.mkdir(parents=True, exist_ok=True)
        except OSError as error:
            raise SettingsError(f"could not create data root: {data_root}") from error

        return cls(
            orbit_url=orbit_url.rstrip("/"),
            username=username,
            password=password,
            data_root=data_root.resolve(),
            docker_context=_optional_text(get_value("POMELO_ORBIT_DOCKER_CONTEXT")),
            wait_timeout_seconds=_positive_int(
                get_value("POMELO_ORBIT_WAIT_TIMEOUT_SECONDS"),
                "POMELO_ORBIT_WAIT_TIMEOUT_SECONDS",
                300,
            ),
            stability_window_seconds=_positive_int(
                get_value("POMELO_ORBIT_STABILITY_WINDOW_SECONDS"),
                "POMELO_ORBIT_STABILITY_WINDOW_SECONDS",
                60,
                allow_zero=True,
            ),
            stability_poll_seconds=_positive_int(
                get_value("POMELO_ORBIT_STABILITY_POLL_SECONDS"),
                "POMELO_ORBIT_STABILITY_POLL_SECONDS",
                2,
            ),
            config_dir=config_dir,
            jwt_from_environment=_optional_text(get_value("POMELO_ORBIT_JWT")),
        )


def jwt_expiry(token: str) -> int | None:
    """Return an unsigned JWT exp claim, or None for malformed data.

    This only decides whether to reuse a cache entry. Orbit remains responsible
    for signature validation on every authenticated request.
    """

    try:
        _header, payload, _signature = token.split(".", 2)
        payload += "=" * (-len(payload) % 4)
        claims = json.loads(base64.urlsafe_b64decode(payload.encode("ascii")))
        exp = claims.get("exp")
        return int(exp) if exp is not None else None
    except (UnicodeEncodeError, ValueError, TypeError, json.JSONDecodeError):
        return None


def is_jwt_valid(token: str | None, now: float | None = None) -> bool:
    if not token:
        return False
    exp = jwt_expiry(token)
    return exp is not None and exp > int(time.time() if now is None else now)


class JWTCache:
    """A single-key local cache that never returns authentication material."""

    def __init__(self, settings: Settings) -> None:
        self._settings = settings

    def read(self) -> str | None:
        if self._settings.jwt_from_environment:
            return self._settings.jwt_from_environment
        path = self._settings.jwt_cache_path
        if not path.exists():
            return None
        return _read_dotenv(path).get("POMELO_ORBIT_JWT")

    def write(self, token: str) -> None:
        path = self._settings.jwt_cache_path
        path.parent.mkdir(parents=True, exist_ok=True)
        descriptor, temp_name = tempfile.mkstemp(prefix=".mcp-jwt-", suffix=".tmp", dir=path.parent)
        try:
            with os.fdopen(descriptor, "w", encoding="utf-8", newline="\n") as handle:
                handle.write(f"POMELO_ORBIT_JWT={token}\n")
            os.replace(temp_name, path)
        finally:
            if os.path.exists(temp_name):
                os.unlink(temp_name)

    def clear(self) -> None:
        if self._settings.jwt_from_environment:
            return
        try:
            self._settings.jwt_cache_path.unlink()
        except FileNotFoundError:
            pass
