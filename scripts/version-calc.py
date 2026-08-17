#!/usr/bin/env python3
"""Derive a semantic version from Git history and optionally apply it."""

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
    result = subprocess.run(
        ["git", *args],
        cwd=REPO_ROOT,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=True,
        text=True,
    )
    return result.stdout


def iter_commits() -> Iterator[tuple[str, str, str]]:
    """Yield (full_hash, ISO date, subject) from oldest to newest."""
    log = run_git("log", "--reverse", "--pretty=format:%H%x1f%aI%x1f%s")
    for line in log.splitlines():
        parts = line.split("\x1f", 2)
        if len(parts) == 3:
            yield parts[0], parts[1], parts[2]


def calculate_version(*, print_history: bool = True) -> str:
    major = 0
    minor = 0
    patch = 0
    found_commit = False

    for full_hash, date, subject in iter_commits():
        found_commit = True
        if subject.lstrip().lower().startswith("feat"):
            minor += 1
            patch = 0
        else:
            patch += 1
        if print_history:
            print(f"{date}  {full_hash[:8]}  {subject[:50]}  {major}.{minor}.{patch}")

    if not found_commit:
        raise RuntimeError("no commits found; cannot derive version")
    return f"{major}.{minor}.{patch}"


def replace_version(path: Path, pattern: re.Pattern[bytes], version: str, label: str) -> None:
    raw = path.read_bytes()
    match = pattern.search(raw)
    if match is None:
        raise RuntimeError(f"could not find {label} in {path}")
    previous = match.group(2).decode("utf-8")
    updated = raw[: match.start(2)] + version.encode("utf-8") + raw[match.end(2) :]
    path.write_bytes(updated)
    print(f"updated {path.relative_to(REPO_ROOT)}: {previous} -> {version}")


def apply_version(version: str) -> None:
    previous = VERSION_FILE.read_text(encoding="utf-8").strip() if VERSION_FILE.exists() else ""
    VERSION_FILE.write_text(version + "\n", encoding="utf-8")
    print(f"updated {VERSION_FILE.relative_to(REPO_ROOT)}: {previous} -> {version}")
    replace_version(CONFIG_FILE, APP_VERSION_RE, version, "app.version")
    replace_version(ENV_EXAMPLE_FILE, ENV_APP_VERSION_RE, version, "app version example")
    replace_version(PACKAGE_JSON, PACKAGE_JSON_VERSION_RE, version, "package version")


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Derive 0.y.z from Git history")
    parser.add_argument("--apply", action="store_true", help="update release version metadata")
    parser.add_argument("--quiet", action="store_true", help="do not print per-commit history")
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
        raise SystemExit(main())
    except subprocess.CalledProcessError as exc:
        sys.stderr.write(f"git failed: {exc.stderr.strip() or exc}\n")
        raise SystemExit(1)
    except (OSError, RuntimeError, UnicodeError, re.error) as exc:
        sys.stderr.write(f"{exc}\n")
        raise SystemExit(1)
    except KeyboardInterrupt:
        raise SystemExit(130)
