"""Calculate and apply release versions from complete Git history."""

from __future__ import annotations

import re
import subprocess
from pathlib import Path

import click

from pomelo_orbit_cli.context import app_context

APP_VERSION_RE = re.compile(
    rb"^(app:\r?\n(?:^[ \t]+[^\r\n]*\r?\n)*?^[ \t]+version:[ \t]*)([^\s#\r\n]+)",
    re.MULTILINE,
)
PACKAGE_JSON_VERSION_RE = re.compile(
    rb'^(\s*"version"\s*:\s*")([^"]*)(")', re.MULTILINE
)
ENV_APP_VERSION_RE = re.compile(
    rb"^(#\s*POMELO_ORBIT_APP__VERSION=)([^\r\n]*)", re.MULTILINE
)


def run_git(root: Path, *args: str) -> str:
    return subprocess.run(
        ["git", *args],
        cwd=root,
        capture_output=True,
        check=True,
        encoding="utf-8",
    ).stdout


def calculate_version(root: Path) -> str:
    if run_git(root, "rev-parse", "--is-shallow-repository").strip() == "true":
        raise RuntimeError(
            "cannot calculate a version from shallow Git history; "
            "run git fetch --unshallow first"
        )
    subjects = run_git(
        root, "log", "--reverse", "--encoding=UTF-8", "--format=%s", "HEAD"
    ).splitlines()
    minor = patch = 0
    for subject in subjects:
        if subject.lstrip().lower().startswith("feat"):
            minor += 1
            patch = 0
        else:
            patch += 1
    return f"0.{minor}.{patch}"


def _prepare_replacement(
    path: Path, pattern: re.Pattern[bytes], version: str, label: str
) -> tuple[Path, bytes]:
    raw = path.read_bytes()
    match = pattern.search(raw)
    if match is None:
        raise RuntimeError(f"could not find {label} in {path}")
    updated = raw[: match.start(2)] + version.encode("utf-8") + raw[match.end(2) :]
    return path, updated


def apply_version(root: Path, version: str) -> None:
    replacements = (
        (root / "VERSION", (version + "\n").encode("utf-8")),
        _prepare_replacement(
            root / "configs" / "config.yaml", APP_VERSION_RE, version, "app.version"
        ),
        _prepare_replacement(
            root / ".env.example", ENV_APP_VERSION_RE, version, "app version example"
        ),
        _prepare_replacement(
            root / "web" / "package.json",
            PACKAGE_JSON_VERSION_RE,
            version,
            "package version",
        ),
    )
    # Validate every target before changing any of them.
    for path, updated in replacements:
        path.write_bytes(updated)


@click.command("version")
@click.option(
    "--apply", is_flag=True, help="Update VERSION and application release metadata."
)
@click.pass_context
def version_command(ctx: click.Context, apply: bool) -> None:
    """Calculate a version from complete Git history."""
    root = app_context(ctx).settings.project_root
    try:
        version = calculate_version(root)
        click.echo(f"version: {version}")
        if apply:
            apply_version(root, version)
    except subprocess.CalledProcessError as error:
        raise click.ClickException(
            f"git failed: {error.stderr.strip() or error}"
        ) from error
    except (OSError, RuntimeError, ValueError, re.error) as error:
        raise click.ClickException(str(error)) from error
