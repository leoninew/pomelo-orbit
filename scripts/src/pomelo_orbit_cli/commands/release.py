"""Build the Windows, Linux, and macOS x64 release packages."""

from __future__ import annotations

import logging
import os
import re
import shutil
import stat
import subprocess
import tempfile
import zipfile
from pathlib import Path

import click

from pomelo_orbit_cli.context import app_context

logger = logging.getLogger(__name__)
PLATFORMS = (("windows", "windows"), ("linux", "linux"), ("macos", "darwin"))


def _run(command: list[str], root: Path, env: dict[str, str] | None = None) -> None:
    subprocess.run(command, cwd=root, env=env, check=True)


def _archive(package: Path, destination: Path) -> None:
    executables = {"pomelo-orbit", "pomelo-orbit.sh"}
    with zipfile.ZipFile(destination, "w", compression=zipfile.ZIP_DEFLATED) as archive:
        for path in [package, *sorted(package.rglob("*"))]:
            name = path.relative_to(package.parent).as_posix()
            info = zipfile.ZipInfo.from_file(path, name)
            info.create_system = 3
            info.compress_type = zipfile.ZIP_DEFLATED
            if path.is_dir():
                info.external_attr = ((stat.S_IFDIR | 0o755) << 16) | 0x10
                archive.writestr(info, b"")
            else:
                mode = 0o755 if path.name in executables else 0o644
                info.external_attr = (stat.S_IFREG | mode) << 16
                with path.open("rb") as source, archive.open(info, "w") as target:
                    shutil.copyfileobj(source, target)


def build_release(root: Path) -> list[Path]:
    version = (root / "VERSION").read_text(encoding="utf-8").strip()
    if re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+", version) is None:
        raise RuntimeError("VERSION must contain a numeric major.minor.patch version")
    assets = root / "scripts" / "package"
    for path in (
        root / "configs" / "config.yaml",
        root / ".env.example",
        assets / "env.release",
        assets / "pomelo-orbit.cmd",
        assets / "pomelo-orbit.sh",
    ):
        if not path.is_file():
            raise RuntimeError(f"Release input is missing: {path}")
    yarn = shutil.which("yarn")
    go = shutil.which("go")
    if not yarn or not go:
        raise RuntimeError("Release builds require Go and Yarn on PATH")

    logger.info("Building the frontend")
    _run([yarn, "--cwd", "web", "build"], root)
    frontend = root / "web" / "dist"
    if not (frontend / "index.html").is_file():
        raise RuntimeError("The frontend build did not produce web/dist/index.html")
    dist = root / "dist"
    dist.mkdir(exist_ok=True)
    archives = []
    with tempfile.TemporaryDirectory(prefix=".release-", dir=dist) as directory:
        staging = Path(directory)
        for platform, goos in PLATFORMS:
            name = f"pomelo-orbit-v{version}-{platform}-x64"
            package = staging / name
            package.mkdir()
            binary = package / (
                "pomelo-orbit.exe" if platform == "windows" else "pomelo-orbit"
            )
            logger.info("Building %s x64", platform)
            env = {**os.environ, "CGO_ENABLED": "0", "GOOS": goos, "GOARCH": "amd64"}
            _run(
                [go, "build", "-trimpath", "-o", str(binary), "./cmd/server"], root, env
            )
            if not binary.is_file():
                raise RuntimeError(f"The Go build did not produce {binary.name}")
            shutil.copytree(frontend, package / "static")
            (package / "configs").mkdir()
            shutil.copy2(
                root / "configs" / "config.yaml", package / "configs" / "config.yaml"
            )
            for source in (root / ".env.example", root / "VERSION"):
                shutil.copy2(source, package / source.name)
            shutil.copy2(assets / "env.release", package / ".env.release")
            shutil.copy2(assets / "pomelo-orbit.sh", package / "pomelo-orbit.sh")
            if platform == "windows":
                shutil.copy2(assets / "pomelo-orbit.cmd", package / "pomelo-orbit.cmd")
            else:
                binary.chmod(0o755)
            (package / "pomelo-orbit.sh").chmod(0o755)
            _archive(package, staging / f"{name}.zip")

        # Publish only after every platform has built and archived successfully.
        package_parent = dist / "package"
        package_parent.mkdir(exist_ok=True)
        if not package_parent.resolve().is_relative_to(root.resolve()):
            raise RuntimeError("Release package directory escapes the repository")
        for platform, _ in PLATFORMS:
            name = f"pomelo-orbit-v{version}-{platform}-x64"
            target = package_parent / name
            if (
                target.is_symlink()
                or target.resolve().parent != package_parent.resolve()
            ):
                raise RuntimeError(
                    f"Release package path escapes dist/package: {target}"
                )
            if target.exists():
                shutil.rmtree(target)
            shutil.move(str(staging / name), str(target))
            archive = dist / f"{name}.zip"
            (staging / archive.name).replace(archive)
            archives.append(archive)
    return archives


@click.command("release")
@click.pass_context
def release_command(ctx: click.Context) -> None:
    """Build Windows, Linux, and macOS x64 ZIP packages in dist/."""
    try:
        archives = build_release(app_context(ctx).settings.project_root)
    except subprocess.CalledProcessError as error:
        raise click.ClickException(
            f"Build failed with status {error.returncode}: {error.cmd}"
        ) from error
    except (OSError, RuntimeError) as error:
        raise click.ClickException(str(error)) from error
    for archive in archives:
        click.echo(str(archive))
