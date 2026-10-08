"""Typed Click context shared by command modules."""

from __future__ import annotations

from dataclasses import dataclass

import click

from pomelo_orbit_cli.settings import Settings


@dataclass(frozen=True)
class AppContext:
    """Process state initialized once by the root Click command."""

    settings: Settings
    verbose: bool


def app_context(ctx: click.Context) -> AppContext:
    """Return the root context object or fail with a clear programming error."""
    root = ctx.find_root()
    # Remote command arguments can contain --help without requesting CLI help.
    if root.obj is None and "initialize_runtime" in root.meta:
        root.meta["initialize_runtime"]()
    value = root.obj
    if not isinstance(value, AppContext):
        raise RuntimeError("CLI context was not initialized")
    return value
