from __future__ import annotations

import stat
import subprocess
import zipfile
from pathlib import Path

import pytest
from click.testing import CliRunner

from pomelo_orbit_cli import cli as cli_module
from pomelo_orbit_cli.commands import release as release_module
from pomelo_orbit_cli.settings import Settings


@pytest.fixture
def repository(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> Path:
    root = tmp_path / "repository with spaces"
    root.mkdir()
    for relative, body in (
        ("VERSION", "0.12.3\n"),
        ("configs/config.yaml", "app:\n  version: 0.12.3\n"),
        (".env.example", "# POMELO_ORBIT_APP__VERSION=0.12.3\n"),
        ("scripts/package/env.release", "POMELO_ORBIT_APP__ENV=release\n"),
        ("scripts/package/pomelo-orbit.sh", "#!/usr/bin/env sh\nexec ./pomelo-orbit\n"),
        ("scripts/package/pomelo-orbit.cmd", "@echo off\npomelo-orbit.exe\n"),
    ):
        path = root / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(body.encode("utf-8"))
    monkeypatch.setattr(
        cli_module, "load_settings", lambda: Settings("INFO", "development", root, None)
    )
    return root


@pytest.fixture
def builds(
    repository: Path, monkeypatch: pytest.MonkeyPatch
) -> list[tuple[str, dict[str, str] | None]]:
    calls: list[tuple[str, dict[str, str] | None]] = []
    monkeypatch.setattr(release_module.shutil, "which", lambda tool: f"release-{tool}")

    def run(
        command: list[str], *, cwd: Path, env: dict[str, str] | None, check: bool
    ) -> subprocess.CompletedProcess[bytes]:
        assert cwd == repository
        assert check
        calls.append((command[0], env))
        if command[0] == "release-yarn":
            assert command[1:] == ["--cwd", "web", "build"]
            frontend = repository / "web" / "dist"
            frontend.mkdir(parents=True, exist_ok=True)
            (frontend / "index.html").write_text("<html>Orbit</html>", encoding="utf-8")
            (frontend / "assets").mkdir(exist_ok=True)
            (frontend / "assets" / "app.js").write_text("orbit();", encoding="utf-8")
        else:
            assert command[0] == "release-go"
            assert command[-1] == "./cmd/server"
            assert env is not None
            Path(command[command.index("-o") + 1]).write_bytes(
                f"binary for {env['GOOS']}".encode()
            )
        return subprocess.CompletedProcess(command, 0)

    monkeypatch.setattr(release_module.subprocess, "run", run)
    return calls


def test_release_builds_all_platforms_with_complete_packages_and_executable_modes(
    repository: Path, builds: list[tuple[str, dict[str, str] | None]]
) -> None:
    result = CliRunner().invoke(cli_module.cli, ["release"])
    assert result.exit_code == 0, result.output
    assert [tool for tool, _ in builds].count("release-yarn") == 1
    environments = [env for tool, env in builds if tool == "release-go"]
    assert [env["GOOS"] for env in environments if env is not None] == [
        "windows",
        "linux",
        "darwin",
    ]
    assert all(
        env is not None and env["GOARCH"] == "amd64" and env["CGO_ENABLED"] == "0"
        for env in environments
    )
    for platform, goos in (
        ("windows", "windows"),
        ("linux", "linux"),
        ("macos", "darwin"),
    ):
        name = f"pomelo-orbit-v0.12.3-{platform}-x64"
        package = repository / "dist" / "package" / name
        archive_path = repository / "dist" / f"{name}.zip"
        assert str(archive_path) in result.output
        binary = "pomelo-orbit.exe" if platform == "windows" else "pomelo-orbit"
        with zipfile.ZipFile(archive_path) as archive:
            assert archive.testzip() is None
            files = {
                item.filename.removeprefix(f"{name}/")
                for item in archive.infolist()
                if not item.is_dir()
            }
            expected = {
                binary,
                "VERSION",
                "configs/config.yaml",
                ".env.example",
                ".env.release",
                "pomelo-orbit.sh",
                "static/index.html",
                "static/assets/app.js",
            }
            if platform == "windows":
                expected.add("pomelo-orbit.cmd")
            assert files == expected
            for relative in files:
                assert (
                    archive.read(f"{name}/{relative}")
                    == (package / relative).read_bytes()
                )
            assert archive.read(f"{name}/VERSION") == b"0.12.3\n"
            assert (
                archive.read(f"{name}/.env.release")
                == (repository / "scripts" / "package" / "env.release").read_bytes()
            )
            assert archive.read(f"{name}/{binary}") == f"binary for {goos}".encode()
            for executable in (
                "pomelo-orbit.sh",
                *(() if platform == "windows" else (binary,)),
            ):
                info = archive.getinfo(f"{name}/{executable}")
                assert info.create_system == 3
                assert stat.S_IMODE(info.external_attr >> 16) == 0o755
    assert not list((repository / "dist").glob(".release-*"))


def test_rerun_replaces_current_packages_and_preserves_other_artifacts(
    repository: Path, builds: list[tuple[str, dict[str, str] | None]]
) -> None:
    first = CliRunner().invoke(cli_module.cli, ["release"])
    assert first.exit_code == 0, first.output
    dist = repository / "dist"
    for package in (dist / "package").iterdir():
        (package / "stale.txt").write_bytes(b"old artifact")
    retained = (
        dist / "pomelo-orbit-v0.11.0-linux-x64.zip",
        dist / "package" / "pomelo-orbit-v0.11.0-linux-x64" / "pomelo-orbit",
        dist / "pomelo-orbit" / "runtime.db",
    )
    for path in retained:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(b"retained")

    second = CliRunner().invoke(cli_module.cli, ["release"])
    assert second.exit_code == 0, second.output
    assert not list((dist / "package").glob("*/stale.txt"))
    assert all(path.read_bytes() == b"retained" for path in retained)


def test_failed_build_preserves_existing_outputs_and_cleans_staging(
    repository: Path,
    builds: list[tuple[str, dict[str, str] | None]],
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    first = CliRunner().invoke(cli_module.cli, ["release"])
    assert first.exit_code == 0, first.output
    dist = repository / "dist"
    before = {path: path.read_bytes() for path in dist.rglob("*") if path.is_file()}
    run = release_module.subprocess.run

    def failing_run(
        command: list[str], *, cwd: Path, env: dict[str, str] | None, check: bool
    ) -> subprocess.CompletedProcess[bytes]:
        if env is not None and env.get("GOOS") == "linux":
            raise subprocess.CalledProcessError(1, command)
        return run(command, cwd=cwd, env=env, check=check)

    monkeypatch.setattr(release_module.subprocess, "run", failing_run)
    result = CliRunner().invoke(cli_module.cli, ["release"])
    assert result.exit_code == 1
    assert "Build failed with status 1" in result.output
    assert {
        path: path.read_bytes() for path in dist.rglob("*") if path.is_file()
    } == before
    assert not list(dist.glob(".release-*"))
