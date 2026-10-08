from __future__ import annotations

from pathlib import Path

import click
import pytest
from click.testing import CliRunner

from pomelo_orbit_cli import __version__
from pomelo_orbit_cli import cli as cli_module
from pomelo_orbit_cli.context import app_context
from pomelo_orbit_cli.settings import Settings


@pytest.mark.parametrize(
    ("arguments", "expected"),
    [
        ([], "Usage:"),
        (["--help"], "Usage:"),
        (["-h"], "Usage:"),
        (["--version"], f"pomelo-orbit-cli, version {__version__}"),
        (["cert", "--help"], "Usage:"),
        (["cert", "new", "--help"], "--cert-dir"),
        (["cert", "check", "--help"], "--cert-dir"),
        (["ulid", "--help"], "--count"),
        (["version", "--help"], "--apply"),
        (["release", "--help"], "ZIP packages"),
        (["remote", "--help"], "tunnel"),
        (["remote", "ssh", "--help"], "SSH session"),
        (["remote", "exec", "--help"], "--workdir"),
        (["remote", "scp", "--help"], "--recursive"),
        (["remote", "tunnel", "--help"], "status"),
        (["remote", "tunnel", "start", "--help"], "--verbose"),
        (["remote", "tunnel", "stop", "--help"], "recorded"),
        (["remote", "tunnel", "status", "--help"], "port forwards"),
    ],
)
def test_cli_help_and_version_without_runtime_configuration(
    monkeypatch: pytest.MonkeyPatch, arguments: list[str], expected: str
) -> None:
    def load_settings() -> Settings:
        pytest.fail("help and version must not load runtime configuration")

    monkeypatch.setattr(cli_module, "load_settings", load_settings)

    result = CliRunner().invoke(cli_module.cli, arguments)

    assert result.exit_code == 0, result.output
    assert expected in result.output


def test_cli_initializes_command_context_and_logging(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    settings = Settings(
        logging_level="INFO",
        environment="development",
        project_root=tmp_path,
        dotenv_path=None,
    )
    logging_calls: list[tuple[str, bool]] = []
    monkeypatch.setattr(cli_module, "load_settings", lambda: settings)
    monkeypatch.setattr(
        cli_module,
        "configure_logging",
        lambda level, verbose: logging_calls.append((level, verbose)),
    )

    @click.command("inspect-context")
    @click.pass_context
    def inspect_context(ctx: click.Context) -> None:
        context = app_context(ctx)
        assert context.settings is settings
        assert context.verbose
        click.echo(context.settings.environment)

    monkeypatch.setitem(cli_module.cli.commands, "inspect-context", inspect_context)

    result = CliRunner().invoke(cli_module.cli, ["--verbose", "inspect-context"])

    assert result.exit_code == 0, result.output
    assert result.output.strip() == "development"
    assert logging_calls == [("INFO", True)]
