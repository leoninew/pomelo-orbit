from __future__ import annotations

import subprocess
from pathlib import Path

import pytest
from click.testing import CliRunner

from pomelo_orbit_cli import cli as cli_module
from pomelo_orbit_cli.commands import version as version_module
from pomelo_orbit_cli.settings import Settings


def git(repository: Path, *args: str) -> str:
    return subprocess.run(
        [
            "git",
            "-c",
            "user.name=Version Test",
            "-c",
            "user.email=version-test@example.test",
            "-c",
            "commit.gpgsign=false",
            "-c",
            f"core.hooksPath={repository / 'test-hooks'}",
            *args,
        ],
        cwd=repository,
        capture_output=True,
        check=True,
        encoding="utf-8",
    ).stdout.strip()


@pytest.fixture
def repository(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> Path:
    root = tmp_path / "repository"
    root.mkdir()
    git(root, "init")
    monkeypatch.setattr(
        cli_module, "load_settings", lambda: Settings("INFO", "development", root, None)
    )
    for relative_path, body in (
        ("VERSION", b"0.271.0\n"),
        (
            "configs/config.yaml",
            b"app:\r\n  name: Orbit\r\n  version: 0.271.0 # release\r\n",
        ),
        (
            ".env.example",
            b"# App version\r\n# POMELO_ORBIT_APP__VERSION=0.271.0\r\n",
        ),
        (
            "web/package.json",
            b'{\r\n  "name": "orbit",\r\n  "version": "0.271.0"\r\n}\r\n',
        ),
    ):
        path = root / relative_path
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(body)
    return root


def metadata(root: Path) -> dict[Path, bytes]:
    return {
        path: path.read_bytes()
        for path in (
            root / "VERSION",
            root / "configs" / "config.yaml",
            root / ".env.example",
            root / "web" / "package.json",
        )
    }


def test_version_advances_with_commits_and_resets_patch_for_features(
    repository: Path,
) -> None:
    git(repository, "commit", "--allow-empty", "-m", "chore: initialize")
    assert version_module.calculate_version(repository) == "0.0.1"
    git(repository, "commit", "--allow-empty", "-m", "feat(route): add routing")
    assert version_module.calculate_version(repository) == "0.1.0"
    git(repository, "commit", "--allow-empty", "-m", "fix(route): restore routes")
    assert version_module.calculate_version(repository) == "0.1.1"
    git(repository, "commit", "--allow-empty", "-m", "refactor: organize settings")
    git(repository, "commit", "--allow-empty", "-m", "docs: record verification")
    assert version_module.calculate_version(repository) == "0.1.3"
    git(repository, "commit", "--allow-empty", "-m", "feat: add gateway")
    assert version_module.calculate_version(repository) == "0.2.0"


def test_preview_is_read_only_and_apply_updates_all_metadata_without_git_changes(
    repository: Path,
) -> None:
    git(repository, "commit", "--allow-empty", "-m", "feat: initialize")
    git(repository, "commit", "--allow-empty", "-m", "fix: update")
    head = git(repository, "rev-parse", "HEAD")
    before = metadata(repository)

    preview = CliRunner().invoke(cli_module.cli, ["version"])
    assert preview.exit_code == 0, preview.output
    assert preview.output == "version: 0.1.1\n"
    assert metadata(repository) == before

    applied = CliRunner().invoke(cli_module.cli, ["version", "--apply"])
    assert applied.exit_code == 0, applied.output
    assert applied.output == "version: 0.1.1\n"
    assert metadata(repository) == {
        path: body.replace(b"0.271.0", b"0.1.1") for path, body in before.items()
    }
    assert git(repository, "rev-parse", "HEAD") == head
    assert git(repository, "diff", "--cached", "--name-only") == ""
    assert git(repository, "tag", "--list") == ""


def test_apply_validates_all_metadata_before_writing(repository: Path) -> None:
    git(repository, "commit", "--allow-empty", "-m", "feat: initialize")
    (repository / "web" / "package.json").write_bytes(b'{"name": "orbit"}\n')
    before = metadata(repository)

    result = CliRunner().invoke(cli_module.cli, ["version", "--apply"])
    assert result.exit_code == 1
    assert "could not find package version" in result.output

    assert metadata(repository) == before


def test_shallow_history_is_rejected(
    repository: Path, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    git(repository, "commit", "--allow-empty", "-m", "feat: initialize")
    git(repository, "commit", "--allow-empty", "-m", "fix: update")
    shallow = tmp_path / "shallow"
    git(repository, "clone", "--depth=1", repository.as_uri(), str(shallow))
    monkeypatch.setattr(
        cli_module,
        "load_settings",
        lambda: Settings("INFO", "development", shallow, None),
    )

    result = CliRunner().invoke(cli_module.cli, ["version"])
    assert result.exit_code == 1
    assert "shallow Git history" in result.output
