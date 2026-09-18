"""Click root command and process lifecycle."""

from __future__ import annotations

import click

from pomelo_orbit_cli import __version__
from pomelo_orbit_cli.commands.database_transfer import database_transfer
from pomelo_orbit_cli.context import AppContext
from pomelo_orbit_cli.errors import ConfigurationError
from pomelo_orbit_cli.logging_config import configure_logging
from pomelo_orbit_cli.settings import load_settings

HELP_OPTION_NAMES = ("-h", "--help")
CONTEXT_SETTINGS = {"help_option_names": list(HELP_OPTION_NAMES)}


class CLIGroup(click.Group):
    """Preserve help rendering when runtime configuration is intentionally absent."""

    def parse_args(self, ctx: click.Context, args: list[str]) -> list[str]:
        ctx.meta["help_requested"] = any(
            argument in HELP_OPTION_NAMES for argument in args
        )
        return super().parse_args(ctx, args)


@click.group(
    cls=CLIGroup,
    context_settings=CONTEXT_SETTINGS,
    invoke_without_command=True,
)
@click.version_option(version=__version__, prog_name="pomelo-orbit-cli")
@click.option("-v", "--verbose", is_flag=True, help="Enable debug diagnostics.")
@click.pass_context
def cli(ctx: click.Context, verbose: bool) -> None:
    """Run Pomelo Orbit operational commands."""
    if ctx.invoked_subcommand is None:
        click.echo(ctx.get_help())
        return
    if ctx.meta.get("help_requested"):
        return
    try:
        settings = load_settings()
    except ConfigurationError as error:
        raise click.ClickException(str(error)) from error
    configure_logging(settings.logging_level, verbose)
    ctx.obj = AppContext(settings=settings, verbose=verbose)


cli.add_command(database_transfer)


def main() -> None:
    """Invoke the command-line application."""
    cli()
