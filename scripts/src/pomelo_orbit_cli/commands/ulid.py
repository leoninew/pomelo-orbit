"""Generate ULIDs for the Orbit CLI."""

import click
import ulid


@click.command("ulid")
@click.option("-n", "--count", type=click.IntRange(min=1), default=1, show_default=True)
def ulid_command(count: int) -> None:
    """Generate ULIDs, one per line."""
    for _ in range(count):
        click.echo(ulid.new())
