"""Minimal stderr logging setup for interactive CLI commands."""

from __future__ import annotations

import logging


def configure_logging(level: str, verbose: bool) -> None:
    """Configure plain-text diagnostics without altering command stdout."""
    resolved_level = logging.DEBUG if verbose else level.upper()
    logging.basicConfig(
        level=resolved_level,
        format="%(levelname)s %(name)s: %(message)s",
    )
