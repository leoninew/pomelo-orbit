#!/usr/bin/env python3
"""Calculate release versions from complete Git history.

Walk commits reachable from HEAD from oldest to newest, starting at 0.0.0.
Subjects starting with "feat" (case-insensitive) increase the minor version
and reset the patch version; every other commit increases the patch version.
The major version remains 0.

By default, print the calculated version without writing files. Pass
``--apply`` to update VERSION and the app's release metadata.
Git is used only to read history, never to commit or tag changes.

    uv --directory scripts run version-calc.py
    uv --directory scripts run version-calc.py --apply
"""

from __future__ import annotations

import argparse
import re
import subprocess
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
VERSION_FILE = REPO_ROOT / "VERSION"
CONFIG_FILE = REPO_ROOT / "configs" / "config.yaml"
ENV_EXAMPLE_FILE = REPO_ROOT / ".env.example"
PACKAGE_JSON = REPO_ROOT / "web" / "package.json"

APP_VERSION_RE = re.compile(
    rb"^(app:\r?\n(?:^[ \t]+[^\r\n]*\r?\n)*?^[ \t]+version:[ \t]*)([^\s#\r\n]+)",
    re.MULTILINE,
)
PACKAGE_JSON_VERSION_RE = re.compile(
    rb'^(\s*"version"\s*:\s*")([^"]*)(")',
    re.MULTILINE,
)
ENV_APP_VERSION_RE = re.compile(
    rb"^(#\s*POMELO_ORBIT_APP__VERSION=)([^\r\n]*)",
    re.MULTILINE,
)


def run_git(*args: str) -> str:
    return subprocess.run(
        ["git", *args],
        cwd=REPO_ROOT,
        capture_output=True,
        check=True,
        encoding="utf-8",
    ).stdout


def calculate_version() -> str:
    """Derive the version from commits, refusing incomplete history."""
    if run_git("rev-parse", "--is-shallow-repository").strip() == "true":
        raise RuntimeError(
            "cannot calculate a version from shallow Git history; "
            "run git fetch --unshallow first"
        )
    subjects = run_git(
        "log", "--reverse", "--encoding=UTF-8", "--format=%s", "HEAD"
    ).splitlines()
    minor = patch = 0
    for subject in subjects:
        if subject.lstrip().lower().startswith("feat"):
            minor += 1
            patch = 0
        else:
            patch += 1
    return f"0.{minor}.{patch}"


def apply_version(version: str) -> None:
    """Write the calculated version to the release metadata files."""
    replacements = (
        (VERSION_FILE, (version + "\n").encode("utf-8")),
        _prepare_version_replacement(
            CONFIG_FILE, APP_VERSION_RE, version, "app.version"
        ),
        _prepare_version_replacement(
            ENV_EXAMPLE_FILE,
            ENV_APP_VERSION_RE,
            version,
            "app version example",
        ),
        _prepare_version_replacement(
            PACKAGE_JSON, PACKAGE_JSON_VERSION_RE, version, "package version"
        ),
    )

    # Validate every target before changing any of them.
    for path, updated in replacements:
        path.write_bytes(updated)


def _prepare_version_replacement(
    path: Path, pattern: re.Pattern[bytes], version: str, label: str
) -> tuple[Path, bytes]:
    """Return the updated bytes for one version consumer."""
    raw = path.read_bytes()
    match = pattern.search(raw)
    if match is None:
        raise RuntimeError(f"could not find {label} in {path}")
    version_bytes = version.encode("utf-8")
    updated = raw[: match.start(2)] + version_bytes + raw[match.end(2) :]
    return path, updated


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Calculate a version from Git history")
    parser.add_argument(
        "--apply",
        action="store_true",
        help="write the calculated version to VERSION and app metadata",
    )
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    version = calculate_version()
    print(f"version: {version}")
    if args.apply:
        apply_version(version)
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except subprocess.CalledProcessError as exc:
        sys.stderr.write(f"git failed: {exc.stderr.strip() or exc}\n")
        sys.exit(1)
    except (OSError, RuntimeError, ValueError, re.error) as exc:
        sys.stderr.write(f"{exc}\n")
        sys.exit(1)
    except KeyboardInterrupt:
        sys.exit(130)
