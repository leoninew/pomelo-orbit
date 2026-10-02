from __future__ import annotations

import importlib.util
import subprocess
import sys
from pathlib import Path

import pytest

SCRIPT_PATH = Path(__file__).resolve().parents[1] / "version-calc.py"
SPEC = importlib.util.spec_from_file_location("version_calc", SCRIPT_PATH)
assert SPEC is not None
assert SPEC.loader is not None
version_calc = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = version_calc
SPEC.loader.exec_module(version_calc)


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
    monkeypatch.setattr(version_calc, "REPO_ROOT", root)
    for name, relative_path, body in (
        ("VERSION_FILE", "VERSION", b"0.271.0\n"),
        (
            "CONFIG_FILE",
            "configs/config.yaml",
            b"app:\r\n  name: Orbit\r\n  version: 0.271.0 # release\r\n",
        ),
        (
            "ENV_EXAMPLE_FILE",
            ".env.example",
            b"# App version\r\n# POMELO_ORBIT_APP__VERSION=0.271.0\r\n",
        ),
        (
            "PACKAGE_JSON",
            "web/package.json",
            b'{\r\n  "name": "orbit",\r\n  "version": "0.271.0"\r\n}\r\n',
        ),
    ):
        path = root / relative_path
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(body)
        monkeypatch.setattr(version_calc, name, path)
    return root


def metadata() -> dict[Path, bytes]:
    return {
        path: path.read_bytes()
        for path in (
            version_calc.VERSION_FILE,
            version_calc.CONFIG_FILE,
            version_calc.ENV_EXAMPLE_FILE,
            version_calc.PACKAGE_JSON,
        )
    }


def test_version_advances_with_commits_and_resets_patch_for_features(
    repository: Path,
) -> None:
    git(repository, "commit", "--allow-empty", "-m", "chore: initialize")
    assert version_calc.calculate_version() == "0.0.1"
    git(repository, "commit", "--allow-empty", "-m", "feat(route): add routing")
    assert version_calc.calculate_version() == "0.1.0"
    git(repository, "commit", "--allow-empty", "-m", "fix(route): restore routes")
    assert version_calc.calculate_version() == "0.1.1"
    git(repository, "commit", "--allow-empty", "-m", "refactor: organize settings")
    git(repository, "commit", "--allow-empty", "-m", "docs: record verification")
    assert version_calc.calculate_version() == "0.1.3"
    git(repository, "commit", "--allow-empty", "-m", "feat: add gateway")
    assert version_calc.calculate_version() == "0.2.0"


def test_preview_is_read_only_and_apply_updates_all_metadata_without_git_changes(
    repository: Path, capsys: pytest.CaptureFixture[str]
) -> None:
    git(repository, "commit", "--allow-empty", "-m", "feat: initialize")
    git(repository, "commit", "--allow-empty", "-m", "fix: update")
    head = git(repository, "rev-parse", "HEAD")
    before = metadata()

    assert version_calc.main([]) == 0
    assert capsys.readouterr().out == "version: 0.1.1\n"
    assert metadata() == before

    assert version_calc.main(["--no-dry-run"]) == 0
    assert capsys.readouterr().out == "version: 0.1.1\n"
    assert metadata() == {
        path: body.replace(b"0.271.0", b"0.1.1") for path, body in before.items()
    }
    assert git(repository, "rev-parse", "HEAD") == head
    assert git(repository, "diff", "--cached", "--name-only") == ""
    assert git(repository, "tag", "--list") == ""


def test_apply_validates_all_metadata_before_writing(repository: Path) -> None:
    git(repository, "commit", "--allow-empty", "-m", "feat: initialize")
    version_calc.PACKAGE_JSON.write_bytes(b'{"name": "orbit"}\n')
    before = metadata()

    with pytest.raises(RuntimeError, match="could not find package version"):
        version_calc.main(["--no-dry-run"])

    assert metadata() == before


def test_shallow_history_is_rejected(
    repository: Path, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    git(repository, "commit", "--allow-empty", "-m", "feat: initialize")
    git(repository, "commit", "--allow-empty", "-m", "fix: update")
    shallow = tmp_path / "shallow"
    git(repository, "clone", "--depth=1", repository.as_uri(), str(shallow))
    monkeypatch.setattr(version_calc, "REPO_ROOT", shallow)

    with pytest.raises(RuntimeError, match="shallow Git history"):
        version_calc.calculate_version()
