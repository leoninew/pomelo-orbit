#!/usr/bin/env python3
"""Walk Git history from the first commit and derive x.y.z.

Rules (x is fixed at 0):
  * y, z start at 0
  * a commit whose subject starts with "feat"  -> y += 1, z = 0
  * any other commit                          -> z += 1
  * print one line every time y or z changes:
        <commit-date>  <sha8>  <subject-first-50-chars>  <x>.<y>.<z>

After calculation, ``--apply`` writes the final version to:
  * VERSION                         (package version source)
  * configs/config.yaml             (app.version)
  * .env.example                    (app version example)
  * web/package.json                ("version" field)

Run from any directory inside the target git repository:

    uv --directory scripts run version-calc.py
    uv --directory scripts run version-calc.py --apply
    uv --directory scripts run version-calc.py --quiet --apply
"""

from __future__ import annotations

import argparse
import re
import subprocess
import sys
from pathlib import Path
from typing import Iterator

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
    proc = subprocess.run(
        ["git", *args],
        cwd=REPO_ROOT,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=True,
        text=True,
    )
    return proc.stdout


def is_feature(subject: str) -> bool:
    return subject.lstrip().lower().startswith("feat")


def iter_commits() -> Iterator[tuple[str, str, str]]:
    """Yield (full_hash, iso_date, subject) from oldest to newest."""
    log = run_git(
        "log",
        "--reverse",
        "--pretty=format:%H%x1f%aI%x1f%s",
    )
    for line in log.splitlines():
        parts = line.split("\x1f", 2)
        if len(parts) != 3:
            continue
        full_hash, date, subject = parts
        yield full_hash, date, subject


def calculate_version(*, print_history: bool = True) -> str:
    """Walk history and return final x.y.z. Optionally print each step."""
    x = 0
    y = 0
    z = 0
    saw_commit = False

    for full_hash, date, subject in iter_commits():
        saw_commit = True
        short = full_hash[:8]
        if is_feature(subject):
            y += 1
            z = 0
        else:
            z += 1
        if print_history:
            headline = subject.split("\n", 1)[0][:50]
            print(f"{date}  {short}  {headline}  {x}.{y}.{z}")

    if not saw_commit:
        raise RuntimeError("no commits found; cannot derive version")

    return f"{x}.{y}.{z}"


def apply_version(version: str) -> None:
    """Write the calculated version to the version metadata files."""
    replacements = (
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
    VERSION_FILE.write_text(version + "\n", encoding="utf-8")
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
    parser = argparse.ArgumentParser(description="Derive x.y.z from git history")
    parser.add_argument(
        "--apply",
        action="store_true",
        help="write the calculated version to version metadata files",
    )
    parser.add_argument(
        "--quiet",
        action="store_true",
        help="do not print per-commit history lines",
    )
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    version = calculate_version(print_history=not args.quiet)
    if not args.quiet:
        print()
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
