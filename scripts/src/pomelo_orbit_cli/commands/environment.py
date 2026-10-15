"""Project codes act as SSH aliases backed by Orbit's stored credentials."""

from __future__ import annotations

import base64
import json
import sys
from typing import Any

import click

from pomelo_orbit_cli.context import app_context
from pomelo_orbit_cli.environment_client import EnvironmentClient, EnvironmentError
from pomelo_orbit_cli.errors import ConfigurationError


def _client(ctx: click.Context) -> EnvironmentClient:
    try:
        return EnvironmentClient(app_context(ctx).settings.client)
    except ConfigurationError as error:
        raise click.UsageError(str(error), ctx) from error


def _failure(ctx: click.Context, error: EnvironmentError, as_json: bool) -> None:
    if as_json:
        click.echo(
            json.dumps(
                {
                    "state": "platform_error",
                    "code": error.code,
                    "error": error.message,
                    "requestId": error.request_id,
                    "status": error.status,
                },
                ensure_ascii=False,
            )
        )
    else:
        click.echo(str(error), err=True)
    ctx.exit(255)


@click.group()
def environment() -> None:
    """Use Orbit project codes as SSH aliases (PAT authentication)."""


@environment.command("list")
@click.option("--json", "as_json", is_flag=True, help="Emit one JSON object.")
@click.pass_context
def list_command(ctx: click.Context, as_json: bool) -> None:
    """List member-visible projects and environment targets."""
    try:
        result = _client(ctx).projects()
    except EnvironmentError as error:
        _failure(ctx, error, as_json)
        return
    if as_json:
        click.echo(json.dumps(result, ensure_ascii=False))
    else:
        for p in result["projects"]:
            click.echo(
                f"{p['code']}\t{p['name']}\t{p['target_type']}\t{p['username']}@{p['host']}:{p['port']}"
            )


def exit_status(result: dict[str, Any]) -> int:
    state = result["state"]
    if state == "exited":
        code = result["exit_code"]
        return code if 0 <= code <= 255 else 255
    if state == "timed_out":
        return 124
    if state in {"output_limit", "access_or_target_changed"}:
        return 125
    return 255


@environment.command(
    "exec",
    context_settings={"ignore_unknown_options": True, "allow_interspersed_args": False},
)
@click.option("--project", required=True, help="Exact project code.")
@click.option(
    "--workdir", help="Absolute remote path or ~/path; default: saved workspace."
)
@click.option("--timeout", type=click.IntRange(1, 300), default=60, show_default=True)
@click.option(
    "--stdin",
    "read_stdin",
    is_flag=True,
    help="Read up to 64 KiB of stdin; otherwise send EOF.",
)
@click.option(
    "--json",
    "as_json",
    is_flag=True,
    help="Emit output text and execution metadata as one JSON object.",
)
@click.argument("command", nargs=-1, required=True, type=click.UNPROCESSED)
@click.pass_context
def exec_command(
    ctx: click.Context,
    project: str,
    workdir: str | None,
    timeout: int,
    read_stdin: bool,
    as_json: bool,
    command: tuple[str, ...],
) -> None:
    """Execute literal argv without a PTY. Use -- before the remote program."""
    client = _client(ctx)
    payload: dict[str, Any] = {
        "project_code": project,
        "argv": list(command),
        "timeout_seconds": timeout,
    }
    if workdir is not None:
        payload["workdir"] = workdir
    try:
        if read_stdin:
            data = sys.stdin.buffer.read(64 * 1024 + 1)
            if len(data) > 64 * 1024:
                raise click.UsageError("stdin exceeds 64 KiB", ctx)
            payload["stdin"] = base64.b64encode(data).decode("ascii")
        result = client.execute(payload, timeout)
    except EnvironmentError as error:
        _failure(ctx, error, as_json)
        return
    except KeyboardInterrupt:
        ctx.exit(130)
    status = exit_status(result)
    if as_json:
        replaced = False
        for key in ("stdout", "stderr"):
            try:
                result[key] = result[key].decode("utf-8")
            except UnicodeDecodeError:
                result[key] = result[key].decode("utf-8", errors="replace")
                replaced = True
        result["output_decoding_replaced"] = replaced
        click.echo(json.dumps(result, ensure_ascii=False))
    else:
        sys.stdout.buffer.write(result["stdout"])
        sys.stdout.buffer.flush()
        sys.stderr.buffer.write(result["stderr"])
        sys.stderr.buffer.flush()
        if result["state"] != "exited":
            click.echo(
                f"\nOrbit: {result['state']}; command result incomplete (requestId={result['requestId']})",
                err=True,
            )
    ctx.exit(status)


@environment.command("ssh")
@click.option("--project", help="Exact project code; omit to select interactively.")
@click.pass_context
def ssh_command(ctx: click.Context, project: str | None) -> None:
    """Open the remote default shell. Ctrl+] detaches locally."""
    if not sys.stdin.isatty() or not sys.stdout.isatty():
        raise click.UsageError("environment ssh requires stdin and stdout TTYs", ctx)
    client = _client(ctx)
    try:
        if project is None:
            targets = [
                p for p in client.projects()["projects"] if p["target_type"] == "ssh"
            ]
            if not targets:
                raise EnvironmentError("no_ssh_target", "No visible SSH environments")
            for i, p in enumerate(targets, 1):
                click.echo(
                    f"{i}. {p['code']} ({p['name']}) {p['username']}@{p['host']}:{p['port']}"
                )
            selected = click.prompt("Project", type=click.IntRange(1, len(targets)))
            project = targets[selected - 1]["code"]
        from pomelo_orbit_cli.environment_terminal import connect

        ctx.exit(connect(client, project))
    except EnvironmentError as error:
        _failure(ctx, error, False)
    except KeyboardInterrupt:
        ctx.exit(130)
