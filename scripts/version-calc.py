#!/usr/bin/env python3
"""Print or synchronize release version metadata from ``VERSION``.

By default, print the repository version without writing files. Pass
``--no-dry-run`` to synchronize the version metadata consumed by the app.
The script never invokes Git.

    uv --directory scripts run version-calc.py
    uv --directory scripts run version-calc.py --no-dry-run
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
VERSION_FILE = REPO_ROOT / "VERSION"
CONFIG_FILE = REPO_ROOT / "configs" / "config.yaml"
ENV_EXAMPLE_FILE = REPO_ROOT / ".env.example"
PACKAGE_JSON = REPO_ROOT / "web" / "package.json"

VERSION_RE = re.compile(r"\d+\.\d+\.\d+")
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


def read_version() -> str:
    """Read the release version from the repository source of truth."""
    version = VERSION_FILE.read_text(encoding="utf-8").strip()
    if not VERSION_RE.fullmatch(version):
        raise ValueError(f"invalid version in {VERSION_FILE}: {version!r}")
    return version


def apply_version(version: str) -> None:
    """Synchronize the version metadata that derives from ``VERSION``."""
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
    parser = argparse.ArgumentParser(description="Print or synchronize release version")
    parser.add_argument(
        "--no-dry-run",
        action="store_false",
        dest="dry_run",
        help="synchronize version metadata files",
    )
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    version = read_version()
    print(f"version: {version}")
    if not args.dry_run:
        apply_version(version)
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, RuntimeError, ValueError, re.error) as exc:
        sys.stderr.write(f"{exc}\n")
        sys.exit(1)
    except KeyboardInterrupt:
        sys.exit(130)
