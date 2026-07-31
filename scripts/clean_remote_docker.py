#!/usr/bin/env python3
"""Clean reclaimable Docker build cache and unused images on the configured server.

By default the script performs a read-only inspection. Pass --execute to remove
all reclaimable BuildKit cache and images that are not used by any container.

Usage:
    python scripts/clean_remote_docker.py
    python scripts/clean_remote_docker.py --execute
"""

from __future__ import annotations

import argparse
import logging
import subprocess
import sys
from pathlib import Path
from typing import Sequence

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger(__name__)

SCRIPT_DIR = Path(__file__).parent
ENV_FILE = SCRIPT_DIR / ".env"


class RemoteDockerCleanupError(RuntimeError):
    """Raised when inspecting or cleaning the remote Docker host fails."""


def parse_args(argv: Sequence[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Inspect or clean reclaimable Docker data on the configured remote host.",
    )
    parser.add_argument(
        "--execute",
        action="store_true",
        help="Remove all BuildKit cache and images unused by any container.",
    )
    return parser.parse_args(argv)


def load_ssh_target() -> str:
    if not ENV_FILE.is_file():
        raise RemoteDockerCleanupError(f"configuration file does not exist: {ENV_FILE}")

    values: dict[str, str] = {}
    for line in ENV_FILE.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if line and not line.startswith("#") and "=" in line:
            key, value = line.split("=", 1)
            values[key.strip()] = value.strip()

    host = values.get("SSH_HOST")
    user = values.get("SSH_USER")
    if not host or not user:
        raise RemoteDockerCleanupError("scripts/.env must define SSH_HOST and SSH_USER")
    return f"{user}@{host}"


def run_remote(ssh_target: str, command: str, title: str) -> str:
    logger.info(title)
    try:
        result = subprocess.run(
            ["ssh", ssh_target, command],
            check=True,
            text=True,
            capture_output=True,
        )
    except FileNotFoundError as error:
        raise RemoteDockerCleanupError("ssh executable was not found") from error
    except subprocess.CalledProcessError as error:
        detail = error.stderr.strip() or error.stdout.strip() or "no command output"
        raise RemoteDockerCleanupError(f"{title} failed: {detail}") from error

    output = result.stdout.strip()
    if output:
        for line in output.splitlines():
            logger.info("  %s", line)
    return output


def show_summary(ssh_target: str, phase: str) -> None:
    logger.info("%s disk usage", phase)
    run_remote(ssh_target, "df -hPT /", "querying root filesystem")
    logger.info("%s Docker usage", phase)
    run_remote(ssh_target, "docker system df", "querying Docker disk usage")
    logger.info("active containers")
    run_remote(
        ssh_target,
        "docker ps --format 'table {{.Names}}\\t{{.Image}}\\t{{.Status}}'",
        "querying active containers",
    )


def main(argv: Sequence[str] | None = None) -> int:
    args = parse_args(argv)
    try:
        ssh_target = load_ssh_target()
        logger.info("remote Docker cleanup target: %s", ssh_target)
        show_summary(ssh_target, "before")

        if not args.execute:
            logger.info("dry run complete; no Docker data was removed")
            logger.info("rerun with --execute to remove all BuildKit cache and unused images")
            return 0

        logger.warning("removing all reclaimable BuildKit cache")
        run_remote(ssh_target, "docker builder prune --all --force", "cleaning BuildKit cache")
        logger.warning("removing images unused by every container")
        run_remote(ssh_target, "docker image prune --all --force", "cleaning unused images")
        show_summary(ssh_target, "after")
        logger.info("remote Docker cleanup completed")
        return 0
    except RemoteDockerCleanupError as error:
        logger.error("remote Docker cleanup failed: %s", error)
        return 1


if __name__ == "__main__":
    sys.exit(main())
