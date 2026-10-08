from __future__ import annotations

import shlex
import subprocess
from pathlib import Path
from unittest.mock import Mock

import click
import psutil
import pytest
from click.testing import CliRunner
from psutil import Process

from pomelo_orbit_cli import cli as cli_module
from pomelo_orbit_cli.commands import remote as remote_module
from pomelo_orbit_cli.commands.remote import RemoteSettings, SSHTunnel, TunnelRecord
from pomelo_orbit_cli.settings import Settings


@pytest.fixture(autouse=True)
def configure_cli(monkeypatch: pytest.MonkeyPatch, tmp_path: Path) -> None:
    monkeypatch.delenv("SSH_HOST", raising=False)
    monkeypatch.delenv("SSH_USER", raising=False)
    settings = Settings("INFO", "development", tmp_path, None)
    monkeypatch.setattr(cli_module, "load_settings", lambda: settings)
    directory = tmp_path / "scripts"
    directory.mkdir()
    (directory / ".env").write_text(
        'SSH_HOST="example.test"\nSSH_USER=ubuntu\n', encoding="utf-8"
    )


def tunnel_record() -> TunnelRecord:
    return TunnelRecord(
        local_port=8888,
        remote_port=8080,
        ssh_target="ubuntu@example.test",
        pid=42,
        started_at=10.0,
        command=[
            "ssh",
            "-N",
            "-L",
            "127.0.0.1:8888:localhost:8080",
            "ubuntu@example.test",
        ],
    )


def ssh_process(record: TunnelRecord) -> Mock:
    process = Mock(spec=Process)
    process.create_time.return_value = record.started_at
    process.name.return_value = "ssh.exe"
    process.cmdline.return_value = record.command
    process.is_running.return_value = True
    return process


def test_settings_need_only_ssh_connection_values_and_allow_environment_overrides(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    env_file = tmp_path / "scripts" / ".env"
    assert RemoteSettings.load(env_file).ssh_target == "ubuntu@example.test"
    monkeypatch.setenv("SSH_USER", "operator")
    assert RemoteSettings.load(env_file).ssh_target == "operator@example.test"


@pytest.mark.parametrize(
    "arguments",
    [
        ["remote", "ssh"],
        ["remote", "exec", "false"],
        ["remote", "scp", "from-remote", "/tmp/source", "./destination"],
    ],
)
def test_remote_commands_preserve_client_exit_status(
    monkeypatch: pytest.MonkeyPatch, arguments: list[str]
) -> None:
    calls: list[list[str]] = []

    def run(command: list[str], **kwargs: object) -> subprocess.CompletedProcess[str]:
        calls.append(command)
        return subprocess.CompletedProcess(command, 17)

    monkeypatch.setattr(remote_module.subprocess, "run", run)
    result = CliRunner().invoke(cli_module.cli, arguments)

    assert result.exit_code == 17, result.output
    assert len(calls) == 1
    assert any("ubuntu@example.test" in argument for argument in calls[0])
    if arguments == ["remote", "ssh"]:
        assert calls[0] == ["ssh", "ubuntu@example.test"]


@pytest.mark.parametrize("separator", [[], ["--"]])
def test_exec_preserves_argument_boundaries_and_payload_help(
    monkeypatch: pytest.MonkeyPatch, separator: list[str]
) -> None:
    calls: list[list[str]] = []

    def run(command: list[str], **kwargs: object) -> subprocess.CompletedProcess[str]:
        calls.append(command)
        return subprocess.CompletedProcess(command, 0)

    monkeypatch.setattr(remote_module.subprocess, "run", run)
    payload = ["printf", "%s\n", "hello world", "$HOME", "--help"]
    result = CliRunner().invoke(
        cli_module.cli,
        ["remote", "exec", "--workdir", "/srv/my app", *separator, *payload],
    )

    assert result.exit_code == 0, result.output
    assert calls[0][:2] == ["ssh", "ubuntu@example.test"]
    assert shlex.split(calls[0][2]) == ["cd", "/srv/my app", "&&", *payload]


@pytest.mark.parametrize("direction", ["to-remote", "from-remote"])
@pytest.mark.parametrize("relative", [False, True])
def test_scp_passes_recursive_paths_without_a_local_shell(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path, direction: str, relative: bool
) -> None:
    monkeypatch.chdir(tmp_path)
    local = Path("local directory") if relative else tmp_path / "local directory"
    local.mkdir()
    remote = "/srv/remote directory"
    calls: list[list[str]] = []

    def run(command: list[str], **kwargs: object) -> subprocess.CompletedProcess[str]:
        calls.append(command)
        return subprocess.CompletedProcess(command, 0)

    monkeypatch.setattr(remote_module.subprocess, "run", run)
    paths = [str(local), remote] if direction == "to-remote" else [remote, str(local)]
    result = CliRunner().invoke(
        cli_module.cli, ["remote", "scp", "-r", direction, *paths]
    )

    assert result.exit_code == 0, result.output
    remote_path = f"ubuntu@example.test:{remote}"
    expected = (
        [str(local), remote_path]
        if direction == "to-remote"
        else [remote_path, str(local)]
    )
    assert calls == [["scp", "-r", "--", *expected]]


@pytest.mark.parametrize("platform", ["win32", "linux", "darwin"])
def test_tunnel_start_records_its_child_and_detaches_for_each_platform(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path, platform: str
) -> None:
    manager = SSHTunnel(tmp_path / "state.json")
    child = Mock(spec=subprocess.Popen)
    child.pid = 42
    child.poll.return_value = None
    popen = Mock(return_value=child)
    monkeypatch.setattr(remote_module.subprocess, "Popen", popen)
    monkeypatch.setattr(
        remote_module.psutil, "Process", lambda pid: ssh_process(tunnel_record())
    )
    monkeypatch.setattr(remote_module.sys, "platform", platform)
    monkeypatch.setattr(
        remote_module.subprocess, "CREATE_NEW_PROCESS_GROUP", 0x200, raising=False
    )
    monkeypatch.setattr(
        remote_module.subprocess, "CREATE_NO_WINDOW", 0x8000000, raising=False
    )
    ports = iter([False, True])
    monkeypatch.setattr(manager, "_port_in_use", lambda port: next(ports))

    manager.start(RemoteSettings("example.test", "ubuntu"), 8080, 8888, verbose=True)

    command = popen.call_args.args[0]
    options = popen.call_args.kwargs
    assert command[:3] == ["ssh", "-v", "-N"]
    assert "BatchMode=yes" in command
    assert "127.0.0.1:8888:localhost:8080" in command
    assert options["start_new_session"] == (platform != "win32")
    assert options["creationflags"] == (0x8000200 if platform == "win32" else 0)
    assert options["stdin"] == subprocess.DEVNULL
    record = manager._load()[0]
    assert record.pid == child.pid
    assert record.started_at == 10.0
    assert record.command == command
    child.terminate.assert_not_called()


def test_tunnel_start_failure_returns_nonzero(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    child = Mock(spec=subprocess.Popen)
    child.pid = 42
    child.poll.return_value = 255
    monkeypatch.setattr(remote_module.subprocess, "Popen", Mock(return_value=child))
    monkeypatch.setattr(
        remote_module.psutil, "Process", lambda pid: ssh_process(tunnel_record())
    )
    monkeypatch.setattr(SSHTunnel, "_port_in_use", lambda self, port: False)

    result = CliRunner().invoke(
        cli_module.cli, ["remote", "tunnel", "start", "8080:8888"]
    )

    assert result.exit_code == 1
    assert "status 255" in result.output


@pytest.mark.parametrize(
    "error", [click.ClickException("write failed"), KeyboardInterrupt()]
)
def test_tunnel_startup_failure_does_not_leave_an_unrecorded_process(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path, error: BaseException
) -> None:
    manager = SSHTunnel(tmp_path / "state.json")
    child = Mock(spec=subprocess.Popen)
    child.pid = 42
    child.poll.return_value = None
    monkeypatch.setattr(remote_module.subprocess, "Popen", Mock(return_value=child))
    monkeypatch.setattr(
        remote_module.psutil, "Process", lambda pid: ssh_process(tunnel_record())
    )
    ports = iter([False, True])
    monkeypatch.setattr(manager, "_port_in_use", lambda port: next(ports))
    monkeypatch.setattr(manager, "_save", Mock(side_effect=error))

    with pytest.raises(type(error)):
        manager.start(RemoteSettings("example.test", "ubuntu"), 8080, 8888)

    child.terminate.assert_called_once()
    child.wait.assert_called_once_with(timeout=3)


def test_start_existing_tunnel_does_not_spawn_another_process(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    manager = SSHTunnel(tmp_path / "state.json")
    record = tunnel_record()
    manager._save([record])
    monkeypatch.setattr(
        remote_module.psutil, "Process", lambda pid: ssh_process(record)
    )
    monkeypatch.setattr(manager, "_port_in_use", lambda port: True)
    popen = Mock()
    monkeypatch.setattr(remote_module.subprocess, "Popen", popen)

    manager.start(RemoteSettings("example.test", "ubuntu"), 8080, 8888)

    popen.assert_not_called()
    assert manager._load() == [record]


def test_start_refuses_an_occupied_local_port(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    manager = SSHTunnel(tmp_path / "state.json")
    monkeypatch.setattr(manager, "_port_in_use", lambda port: True)
    popen = Mock()
    monkeypatch.setattr(remote_module.subprocess, "Popen", popen)

    with pytest.raises(click.ClickException, match="already in use"):
        manager.start(RemoteSettings("example.test", "ubuntu"), 8080, 8888)

    popen.assert_not_called()


@pytest.mark.parametrize("mismatch", ["pid reuse", "process name", "command line"])
def test_stop_never_signals_a_process_that_does_not_match_the_record(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path, mismatch: str
) -> None:
    manager = SSHTunnel(tmp_path / "state.json")
    record = tunnel_record()
    manager._save([record])
    process = ssh_process(record)
    if mismatch == "pid reuse":
        process.create_time.return_value += 1
    elif mismatch == "process name":
        process.name.return_value = "python.exe"
    else:
        process.cmdline.return_value = ["ssh", "other-host"]
    monkeypatch.setattr(remote_module.psutil, "Process", lambda pid: process)

    manager.stop()

    process.terminate.assert_not_called()
    process.kill.assert_not_called()
    assert not manager.state_file.exists()


def test_stop_terminates_owned_process_without_connection_configuration(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    state_file = tmp_path / "scripts" / ".remote-tunnels.json"
    record = tunnel_record()
    SSHTunnel(state_file)._save([record])
    (tmp_path / "scripts" / ".env").unlink()
    process = ssh_process(record)
    monkeypatch.setattr(remote_module.psutil, "Process", lambda pid: process)

    result = CliRunner().invoke(cli_module.cli, ["remote", "tunnel", "stop"])

    assert result.exit_code == 0, result.output
    process.terminate.assert_called_once()
    process.wait.assert_called_once_with(timeout=3)
    assert not state_file.exists()


def test_stop_failure_preserves_the_record_for_retry(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    manager = SSHTunnel(tmp_path / "state.json")
    record = tunnel_record()
    manager._save([record])
    process = ssh_process(record)
    process.terminate.side_effect = psutil.AccessDenied(record.pid)
    monkeypatch.setattr(remote_module.psutil, "Process", lambda pid: process)

    with pytest.raises(click.ClickException, match="Could not stop tunnels"):
        manager.stop()

    assert manager._load() == [record]


def test_status_discards_stale_records_without_connection_configuration(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    state_file = tmp_path / "scripts" / ".remote-tunnels.json"
    record = tunnel_record()
    SSHTunnel(state_file)._save([record])
    (tmp_path / "scripts" / ".env").unlink()
    monkeypatch.setattr(
        remote_module.psutil,
        "Process",
        Mock(side_effect=psutil.NoSuchProcess(record.pid)),
    )

    result = CliRunner().invoke(cli_module.cli, ["remote", "tunnel", "status"])

    assert result.exit_code == 0, result.output
    assert "No active SSH tunnels" in result.output
    assert not state_file.exists()


def test_multiple_port_mappings_and_verbose_reach_the_tunnel_manager(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    start = Mock()
    monkeypatch.setattr(SSHTunnel, "start", start)

    result = CliRunner().invoke(
        cli_module.cli,
        ["remote", "tunnel", "start", "-v", "8080:8888", "9090:9999"],
    )

    assert result.exit_code == 0, result.output
    assert [call.args[1:] for call in start.call_args_list] == [
        (8080, 8888, True),
        (9090, 9999, True),
    ]
