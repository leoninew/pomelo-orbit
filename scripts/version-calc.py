#!/usr/bin/env python3
"""Walk git history from the first commit and derive x.y.z.

Rules (x is fixed at 0):
  * y, z start at 0
  * a commit whose subject starts with "feat"  -> y += 1, z = 0
  * any other commit                          -> z += 1
  * print one line every time y or z changes:
        <commit-date>  <sha8>  <subject-first-50-chars>  <x>.<y>.<z>

After calculation, optionally apply the final version to:
  * internal/shared/common/utils/version/version.go  (Version var)
  * web/package.json                                 ("version" field)

Run from any directory inside the target git repository:

    python scripts/version-calc.py
    python scripts/version-calc.py --apply
    python scripts/version-calc.py --quiet --apply
"""

from __future__ import annotations

import argparse
import re
import subprocess
import sys
from pathlib import Path
from typing import Iterator, Tuple

REPO_ROOT = Path(__file__).resolve().parent.parent
VERSION_GO = REPO_ROOT / "internal" / "shared" / "common" / "utils" / "version" / "version.go"
PACKAGE_JSON = REPO_ROOT / "web" / "package.json"

# Match only the package-level Version = "..." assignment (tab/space indent ok).
VERSION_GO_RE = re.compile(
    rb'^(\tVersion\s*=\s*")([^"]*)(")',
    re.MULTILINE,
)
PACKAGE_JSON_VERSION_RE = re.compile(
    rb'^(\s*"version"\s*:\s*")([^"]*)(")',
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


def iter_commits() -> Iterator[Tuple[str, str, str]]:
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
    """Write version into version.go and web/package.json."""
    update_version_go(version)
    update_package_json(version)


def _replace_version_bytes(
    path: Path,
    pattern: re.Pattern[bytes],
    version: str,
    label: str,
) -> str:
    """Replace the first captured version string; preserve encoding and newlines."""
    raw = path.read_bytes()
    match = pattern.search(raw)
    if match is None:
        raise RuntimeError(f"could not find {label} in {path}")
    previous = match.group(2).decode("utf-8")
    version_bytes = version.encode("utf-8")
    new_raw = raw[: match.start(2)] + version_bytes + raw[match.end(2) :]
    # Safety: only one field should change in size by the version length delta.
    if new_raw == raw and previous == version:
        print(f"unchanged {path.relative_to(REPO_ROOT)} -> {label} = {version!r}")
        return previous
    path.write_bytes(new_raw)
    print(
        f"updated {path.relative_to(REPO_ROOT)} -> "
        f"{label} {previous!r} -> {version!r}"
    )
    return previous


def update_version_go(version: str) -> None:
    _replace_version_bytes(VERSION_GO, VERSION_GO_RE, version, "Version")


def update_package_json(version: str) -> None:
    _replace_version_bytes(PACKAGE_JSON, PACKAGE_JSON_VERSION_RE, version, "version")


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Derive x.y.z from git history")
    parser.add_argument(
        "--apply",
        action="store_true",
        help="write final version to version.go and web/package.json",
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
