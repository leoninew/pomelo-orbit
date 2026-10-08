from __future__ import annotations

from pathlib import Path

import pytest

from pomelo_orbit_cli.settings import ENV_SELECTOR, load_settings


@pytest.mark.parametrize(
    ("profile_level", "dotenv_level", "environment_level", "expected"),
    [
        (None, None, None, "INFO"),
        ("WARNING", None, None, "WARNING"),
        ("WARNING", "ERROR", None, "ERROR"),
        ("WARNING", "ERROR", "CRITICAL", "CRITICAL"),
    ],
)
def test_shared_settings_configuration_layers(
    monkeypatch: pytest.MonkeyPatch,
    tmp_path: Path,
    profile_level: str | None,
    dotenv_level: str | None,
    environment_level: str | None,
    expected: str,
) -> None:
    configuration_directory = tmp_path / "configs"
    configuration_directory.mkdir()
    (configuration_directory / "config.yaml").write_text(
        "logging:\n  level: INFO\n", encoding="utf-8"
    )
    if profile_level is not None:
        (configuration_directory / "config.development.yaml").write_text(
            f"logging:\n  level: {profile_level}\n", encoding="utf-8"
        )
    dotenv_path = tmp_path / ".env.development"
    if dotenv_level is not None:
        dotenv_path.write_text(
            f"POMELO_ORBIT_LOGGING__LEVEL={dotenv_level}\n", encoding="utf-8"
        )
    monkeypatch.setenv(ENV_SELECTOR, "development")
    if environment_level is None:
        monkeypatch.delenv("POMELO_ORBIT_LOGGING__LEVEL", raising=False)
    else:
        monkeypatch.setenv("POMELO_ORBIT_LOGGING__LEVEL", environment_level)

    settings = load_settings(tmp_path)

    assert settings.logging_level == expected
    assert settings.environment == "development"
    assert settings.project_root == tmp_path.resolve()
    assert settings.dotenv_path == (dotenv_path if dotenv_level is not None else None)
