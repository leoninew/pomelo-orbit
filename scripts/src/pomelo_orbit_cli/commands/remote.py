"""SSH connections, command execution, file transfers, and local tunnels."""

from __future__ import annotations

import json
import os
import shlex
import socket
import subprocess
import sys
import time
from dataclasses import asdict, dataclass
from pathlib import Path

import click
import psutil
from dotenv import dotenv_values

from pomelo_orbit_cli.context import app_context


@dataclass(frozen=True)
class RemoteSettings:
    ssh_host: str
    ssh_user: str

    @property
    def ssh_target(self) -> str:
        return f"{self.ssh_user}@{self.ssh_host}"

    @classmethod
    def load(cls, path: Path) -> RemoteSettings:
        values = dotenv_values(path) if path.is_file() else {}
        names = ("SSH_HOST", "SSH_USER")
        settings = [
            os.environ.get(name, values.get(name) or "").strip() for name in names
        ]
        missing = [name for name, value in zip(names, settings) if not value]
        if missing:
            raise click.ClickException(
                f"Set {', '.join(missing)} in {path} or the process environment."
            )
        return cls(ssh_host=settings[0], ssh_user=settings[1])


def _settings(ctx: click.Context) -> RemoteSettings:
    return RemoteSettings.load(
        app_context(ctx).settings.project_root / "scripts" / ".env"
    )


def _run(command: list[str]) -> int:
    try:
        result = subprocess.run(command, check=False)
    except OSError as error:
        raise click.ClickException(f"Could not run {command[0]}: {error}") from error
    return result.returncode if result.returncode >= 0 else 128 - result.returncode


@dataclass(frozen=True)
class TunnelRecord:
    local_port: int
    remote_port: int
    ssh_target: str
    pid: int
    started_at: float
    command: list[str]


class SSHTunnel:
    def __init__(self, state_file: Path):
        self.state_file = state_file

    def _load(self) -> list[TunnelRecord]:
        try:
            if not self.state_file.exists():
                return []
            data = json.loads(self.state_file.read_text(encoding="utf-8"))
            return [TunnelRecord(**record) for record in data["tunnels"]]
        except (OSError, ValueError, KeyError, TypeError) as error:
            raise click.ClickException(
                f"Could not read tunnel state: {error}"
            ) from error

    def _save(self, records: list[TunnelRecord]) -> None:
        temporary = self.state_file.with_suffix(".json.tmp")
        try:
            if not records:
                self.state_file.unlink(missing_ok=True)
                return
            self.state_file.parent.mkdir(parents=True, exist_ok=True)
            temporary.write_text(
                json.dumps(
                    {"tunnels": [asdict(record) for record in records]}, indent=2
                ),
                encoding="utf-8",
            )
            temporary.replace(self.state_file)
        except OSError as error:
            raise click.ClickException(
                f"Could not save tunnel state: {error}"
            ) from error
        finally:
            temporary.unlink(missing_ok=True)

    @staticmethod
    def _process(record: TunnelRecord) -> psutil.Process | None:
        try:
            process = psutil.Process(record.pid)
            if (
                process.create_time() != record.started_at
                or process.name().lower() not in {"ssh", "ssh.exe"}
                or process.cmdline()[1:] != record.command[1:]
                or not process.is_running()
            ):
                return None
            return process
        except psutil.NoSuchProcess:
            return None

    @staticmethod
    def _port_in_use(port: int) -> bool:
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as connection:
            connection.settimeout(0.1)
            return connection.connect_ex(("127.0.0.1", port)) == 0

    def start(
        self,
        settings: RemoteSettings,
        remote_port: int,
        local_port: int,
        verbose: bool = False,
    ) -> None:
        records = self._load()
        for record in records:
            if record.local_port == local_port and self._process(record) is not None:
                if (
                    record.ssh_target == settings.ssh_target
                    and record.remote_port == remote_port
                    and self._port_in_use(local_port)
                ):
                    click.echo(f"Tunnel already running on localhost:{local_port}.")
                    return
                raise click.ClickException(
                    f"A recorded tunnel already uses port {local_port}."
                )
        if self._port_in_use(local_port):
            raise click.ClickException(f"Local port {local_port} is already in use.")

        command = [
            "ssh",
            "-N",
            "-o",
            "BatchMode=yes",
            "-o",
            "ForkAfterAuthentication=no",
            "-o",
            "ControlMaster=no",
            "-o",
            "ControlPath=none",
            "-o",
            "ServerAliveInterval=60",
            "-o",
            "ExitOnForwardFailure=yes",
            "-L",
            f"127.0.0.1:{local_port}:localhost:{remote_port}",
            settings.ssh_target,
        ]
        if verbose:
            command.insert(1, "-v")
        creationflags = 0
        if sys.platform == "win32":
            creationflags = (
                subprocess.CREATE_NEW_PROCESS_GROUP | subprocess.CREATE_NO_WINDOW
            )
        try:
            child = subprocess.Popen(
                command,
                stdin=subprocess.DEVNULL,
                stdout=subprocess.DEVNULL,
                stderr=None,
                start_new_session=sys.platform != "win32",
                creationflags=creationflags,
            )
        except OSError as error:
            raise click.ClickException(f"Could not start SSH: {error}") from error

        try:
            record = TunnelRecord(
                local_port=local_port,
                remote_port=remote_port,
                ssh_target=settings.ssh_target,
                pid=child.pid,
                started_at=psutil.Process(child.pid).create_time(),
                command=command,
            )
            for _ in range(50):
                code = child.poll()
                if code is not None:
                    raise click.ClickException(f"SSH tunnel exited with status {code}.")
                if self._port_in_use(local_port):
                    records = [
                        item for item in records if item.local_port != local_port
                    ]
                    self._save([*records, record])
                    click.echo(
                        f"localhost:{local_port} -> {settings.ssh_target}:{remote_port}"
                    )
                    return
                time.sleep(0.1)
            raise click.ClickException(
                f"SSH tunnel did not listen on port {local_port}."
            )
        except BaseException:
            if child.poll() is None:
                child.terminate()
                try:
                    child.wait(timeout=3)
                except subprocess.TimeoutExpired:
                    child.kill()
                    child.wait()
            raise

    def stop(self) -> None:
        records = self._load()
        remaining = []
        errors = []
        for record in records:
            try:
                process = self._process(record)
                if process is None:
                    click.echo(
                        f"Discarded inactive tunnel record on port {record.local_port}."
                    )
                    continue
                process.terminate()
                try:
                    process.wait(timeout=3)
                except psutil.TimeoutExpired:
                    process.kill()
                    process.wait(timeout=3)
                click.echo(f"Stopped tunnel on localhost:{record.local_port}.")
            except psutil.NoSuchProcess:
                continue
            except psutil.Error as error:
                remaining.append(record)
                errors.append(f"port {record.local_port}: {error}")
        self._save(remaining)
        if errors:
            raise click.ClickException("Could not stop tunnels: " + "; ".join(errors))
        if not records:
            click.echo("No active SSH tunnels.")

    def status(self) -> None:
        records = self._load()
        active = []
        for record in records:
            if self._process(record) is None:
                continue
            active.append(record)
            status = (
                "running" if self._port_in_use(record.local_port) else "not listening"
            )
            click.echo(
                f"{status}: localhost:{record.local_port} -> "
                f"{record.ssh_target}:{record.remote_port} (pid={record.pid})"
            )
        self._save(active)
        if not active:
            click.echo("No active SSH tunnels.")


def _tunnels(ctx: click.Context) -> SSHTunnel:
    root = app_context(ctx).settings.project_root
    return SSHTunnel(root / "scripts" / ".remote-tunnels.json")


@click.group()
def remote() -> None:
    """Connect to the configured remote host using OpenSSH."""


@remote.command("ssh")
@click.pass_context
def ssh_command(ctx: click.Context) -> None:
    """Open an interactive SSH session."""
    ctx.exit(_run(["ssh", _settings(ctx).ssh_target]))


@remote.command(
    "exec",
    context_settings={"ignore_unknown_options": True, "allow_interspersed_args": False},
)
@click.option("-w", "--workdir", help="Remote working directory.")
@click.argument("command", nargs=-1, required=True, type=click.UNPROCESSED)
@click.pass_context
def exec_command(
    ctx: click.Context, workdir: str | None, command: tuple[str, ...]
) -> None:
    """Execute a command with its arguments on the remote host."""
    shell_command = shlex.join(command)
    if workdir:
        shell_command = f"cd {shlex.quote(workdir)} && {shell_command}"
    ctx.exit(_run(["ssh", _settings(ctx).ssh_target, shell_command]))


@remote.command("scp")
@click.argument("direction", type=click.Choice(["to-remote", "from-remote"]))
@click.option("-r", "--recursive", is_flag=True, help="Copy a directory recursively.")
@click.argument("source")
@click.argument("destination")
@click.pass_context
def scp_command(
    ctx: click.Context, direction: str, recursive: bool, source: str, destination: str
) -> None:
    """Copy files or directories between the local and remote host."""
    target = _settings(ctx).ssh_target
    if direction == "to-remote":
        source_path = Path(source).expanduser()
        if not source_path.exists() or (recursive and not source_path.is_dir()):
            raise click.ClickException(
                "The local source must exist; -r requires a directory."
            )
        source = str(source_path)
        destination = f"{target}:{destination}"
    else:
        source = f"{target}:{source}"
        destination = str(Path(destination).expanduser())
    command = ["scp", *(["-r"] if recursive else []), "--", source, destination]
    ctx.exit(_run(command))


class PortMapping(click.ParamType):
    name = "remote:local"

    def convert(
        self, value: str, param: click.Parameter | None, ctx: click.Context | None
    ) -> tuple[int, int]:
        try:
            remote_port, local_port = map(int, value.split(":"))
            if not all(1 <= port <= 65535 for port in (remote_port, local_port)):
                raise ValueError
            return remote_port, local_port
        except ValueError:
            self.fail("Use remote:local with ports between 1 and 65535.", param, ctx)


@remote.group()
def tunnel() -> None:
    """Manage local SSH port forwards."""


@tunnel.command("start")
@click.option("-v", "--verbose", is_flag=True, help="Show SSH diagnostics.")
@click.argument("ports", nargs=-1, required=True, type=PortMapping())
@click.pass_context
def tunnel_start(
    ctx: click.Context, verbose: bool, ports: tuple[tuple[int, int], ...]
) -> None:
    settings = _settings(ctx)
    manager = _tunnels(ctx)
    try:
        for remote_port, local_port in ports:
            manager.start(
                settings, remote_port, local_port, verbose or app_context(ctx).verbose
            )
    except psutil.Error as error:
        raise click.ClickException(
            f"Could not inspect the SSH process: {error}"
        ) from error


@tunnel.command("stop")
@click.pass_context
def tunnel_stop(ctx: click.Context) -> None:
    """Stop the SSH processes recorded by this CLI."""
    _tunnels(ctx).stop()


@tunnel.command("status")
@click.pass_context
def tunnel_status(ctx: click.Context) -> None:
    """Show the state of recorded SSH port forwards."""
    try:
        _tunnels(ctx).status()
    except psutil.Error as error:
        raise click.ClickException(
            f"Could not inspect the SSH process: {error}"
        ) from error
