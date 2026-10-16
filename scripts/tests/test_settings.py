from __future__ import annotations

from pathlib import Path

import pytest

from pomelo_orbit_cli.settings import ENV_SELECTOR, load_settings


@pytest.mark.parametrize(
    ("environment", "dotenv_level", "environment_level", "expected"),
    [
        ("", None, None, "INFO"),
        ("development", None, None, "INFO"),
        ("", "ERROR", None, "ERROR"),
        ("development", "ERROR", None, "ERROR"),
        ("development", "ERROR", "CRITICAL", "CRITICAL"),
    ],
)
def test_shared_settings_configuration_layers(
    monkeypatch: pytest.MonkeyPatch,
    tmp_path: Path,
    environment: str,
    dotenv_level: str | None,
    environment_level: str | None,
    expected: str,
) -> None:
    directory = tmp_path / "scripts"
    directory.mkdir()
    filename = f".env.{environment}" if environment else ".env"
    (tmp_path / filename).write_text(
        "POMELO_ORBIT_LOGGING__LEVEL=backend-only\n", encoding="utf-8"
    )
    if environment:
        (directory / ".env").write_text(
            "POMELO_ORBIT_LOGGING__LEVEL=DEBUG\n", encoding="utf-8"
        )
    dotenv_path = directory / filename
    if dotenv_level is not None:
        dotenv_path.write_text(
            f"POMELO_ORBIT_LOGGING__LEVEL={dotenv_level}\n", encoding="utf-8"
        )
    monkeypatch.setenv(ENV_SELECTOR, environment)
    if environment_level is None:
        monkeypatch.delenv("POMELO_ORBIT_LOGGING__LEVEL", raising=False)
    else:
        monkeypatch.setenv("POMELO_ORBIT_LOGGING__LEVEL", environment_level)

    settings = load_settings(tmp_path)

    assert settings.logging_level == expected
    assert settings.environment == environment
    assert settings.project_root == tmp_path.resolve()
    assert settings.dotenv_path == (dotenv_path if dotenv_level is not None else None)


@pytest.mark.parametrize("working_directory", [".", "scripts", "scripts/src"])
def test_scripts_dotenv_is_independent_of_working_directory(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path, working_directory: str
) -> None:
    directory = tmp_path / "scripts"
    (directory / "src").mkdir(parents=True)
    (directory / "pyproject.toml").touch()
    (tmp_path / "go.mod").touch()
    dotenv_path = directory / ".env"
    dotenv_path.write_text("POMELO_ORBIT_LOGGING__LEVEL=WARNING\n", encoding="utf-8")
    monkeypatch.delenv(ENV_SELECTOR, raising=False)
    monkeypatch.delenv("POMELO_ORBIT_LOGGING__LEVEL", raising=False)
    monkeypatch.chdir(tmp_path / working_directory)

    settings = load_settings()

    assert settings.project_root == tmp_path.resolve()
    assert settings.dotenv_path == dotenv_path
    assert settings.logging_level == "WARNING"
