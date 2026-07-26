from __future__ import annotations

import base64
import json

import pytest

from pomelo_orbit_mcp.settings import JWTCache, Settings, SettingsError, is_jwt_valid, jwt_expiry

from .conftest import make_settings


def make_jwt(exp: int) -> str:
    payload = base64.urlsafe_b64encode(json.dumps({"exp": exp}).encode()).decode().rstrip("=")
    return f"header.{payload}.signature"


def test_settings_loads_dotenv_and_process_environment_has_priority(tmp_path) -> None:
    file_data_root = tmp_path / "file-data"
    process_data_root = tmp_path / "process-data"
    (tmp_path / ".env").write_text(
        "POMELO_ORBIT_URL=http://from-file\n"
        "POMELO_ORBIT_USERNAME=file-user\n"
        "POMELO_ORBIT_PASSWORD=file-password\n"
        f"POMELO_ORBIT_DATA_ROOT={file_data_root}\n"
        "POMELO_ORBIT_WAIT_TIMEOUT_SECONDS=42\n",
        encoding="utf-8",
    )
    settings = Settings.load(
        tmp_path,
        environ={
            "POMELO_ORBIT_URL": "http://from-process",
            "POMELO_ORBIT_USERNAME": "process-user",
            "POMELO_ORBIT_PASSWORD": "process-password",
            "POMELO_ORBIT_DATA_ROOT": str(process_data_root),
        },
    )
    assert settings.orbit_url == "http://from-process"
    assert settings.username == "process-user"
    assert settings.data_root == process_data_root.resolve()
    assert settings.data_root.is_dir()
    assert settings.wait_timeout_seconds == 42


def test_settings_requires_all_connection_values(tmp_path) -> None:
    with pytest.raises(SettingsError, match="POMELO_ORBIT_PASSWORD"):
        Settings.load(
            tmp_path,
            environ={
                "POMELO_ORBIT_URL": "http://orbit.test",
                "POMELO_ORBIT_USERNAME": "tester",
            },
        )


def test_settings_defaults_to_repository_data_root_and_creates_it(tmp_path) -> None:
    config_dir = tmp_path / "mcp"
    config_dir.mkdir()

    settings = Settings.load(
        config_dir,
        environ={
            "POMELO_ORBIT_URL": "http://orbit.test",
            "POMELO_ORBIT_USERNAME": "tester",
            "POMELO_ORBIT_PASSWORD": "password",
        },
    )

    assert settings.data_root == (tmp_path / "data").resolve()
    assert settings.data_root.is_dir()


def test_jwt_expiry_and_atomic_cache_round_trip(tmp_path) -> None:
    token = make_jwt(2_000)
    assert jwt_expiry(token) == 2_000
    assert is_jwt_valid(token, now=1_999)
    assert not is_jwt_valid(token, now=2_000)
    cache = JWTCache(make_settings(tmp_path))
    cache.write(token)
    assert cache.read() == token
    cache.clear()
    assert cache.read() is None
