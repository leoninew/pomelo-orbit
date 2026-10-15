from __future__ import annotations

import base64
import io
import json
from email.message import Message
from pathlib import Path
from typing import Any
from urllib.error import HTTPError

import pytest
from click.testing import CliRunner

from pomelo_orbit_cli import cli as cli_module
from pomelo_orbit_cli.commands import environment as commands
from pomelo_orbit_cli.environment_client import (
    EnvironmentClient,
    EnvironmentError,
    NoRedirect,
)
from pomelo_orbit_cli.errors import ConfigurationError
from pomelo_orbit_cli.settings import ClientSettings, Settings, load_settings

PAT = "orbit_mcp_pat_" + base64.urlsafe_b64encode(bytes(32)).decode().rstrip("=")


@pytest.mark.parametrize(
    "args",
    [
        ["environment", "--help"],
        ["environment", "list", "--help"],
        ["environment", "ssh", "--help"],
        ["environment", "exec", "--help"],
    ],
)
def test_help_needs_no_config(monkeypatch: pytest.MonkeyPatch, args: list[str]) -> None:
    def no_settings() -> Settings:
        pytest.fail("help initialized configuration")

    monkeypatch.setattr(cli_module, "load_settings", no_settings)
    assert CliRunner().invoke(cli_module.cli, args).exit_code == 0


def test_client_configuration_layers_and_environment_only_validation(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    (tmp_path / "configs").mkdir()
    (tmp_path / "configs/config.yaml").write_text(
        "logging:\n  level: INFO\ncli:\n  base_url: https://yaml.example\n"
    )
    (tmp_path / ".env").write_text(
        f"POMELO_ORBIT_CLI__BASE_URL=https://dotenv.example/prefix\nPOMELO_ORBIT_MCP__ACCESS_TOKEN={PAT}\n"
    )
    monkeypatch.delenv("POMELO_ORBIT_APP__ENV", raising=False)
    monkeypatch.delenv("POMELO_ORBIT_MCP__ACCESS_TOKEN", raising=False)
    monkeypatch.setenv("POMELO_ORBIT_CLI__BASE_URL", "https://os.example/orbit")
    settings = load_settings(tmp_path)
    assert settings.client.validated() == ClientSettings(
        "https://os.example/orbit", PAT
    )
    assert (
        EnvironmentClient(settings.client).url("project")
        == "https://os.example/orbit/api/cli/project"
    )
    monkeypatch.setattr(
        cli_module, "load_settings", lambda: Settings("INFO", "", tmp_path, None)
    )
    result = CliRunner().invoke(cli_module.cli, ["environment", "list"])
    assert result.exit_code == 2
    assert "POMELO_ORBIT_MCP__ACCESS_TOKEN" in result.output


@pytest.mark.parametrize(
    "url",
    [
        "ftp://host",
        "https://user:secret@host",
        "https://host?x=1",
        "https://host/#fragment",
    ],
)
def test_client_rejects_invalid_service_urls(url: str) -> None:
    with pytest.raises(ConfigurationError):
        ClientSettings(url, PAT).validated()


class FakeClient:
    def __init__(self, result: dict[str, Any]):
        self.result = result
        self.payload: dict[str, Any] = {}

    def execute(self, payload: dict[str, Any], timeout: int) -> dict[str, Any]:
        self.payload = payload
        return self.result.copy()


def configure_command(
    monkeypatch: pytest.MonkeyPatch, client: FakeClient, tmp_path: Path
) -> None:
    monkeypatch.setattr(
        cli_module, "load_settings", lambda: Settings("INFO", "", tmp_path, None)
    )
    monkeypatch.setattr(commands, "_client", lambda ctx: client)


def test_exec_payload_help_literal_argv_stdin_and_separate_streams(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    client = FakeClient(
        {
            "state": "exited",
            "exit_code": 7,
            "stdout": b"out\x00\xff",
            "stderr": b"error",
            "requestId": "request",
        }
    )
    configure_command(monkeypatch, client, tmp_path)
    result = CliRunner().invoke(
        cli_module.cli,
        [
            "environment",
            "exec",
            "--project",
            "production",
            "--stdin",
            "--",
            "tool",
            "--help",
            "$HOME",
            "a b",
        ],
        input=b"input\x00",
    )
    assert result.exit_code == 7, result.exception
    assert result.stdout_bytes == b"out\x00\xff"
    assert result.stderr_bytes == b"error"
    assert client.payload["argv"] == ["tool", "--help", "$HOME", "a b"]
    assert base64.b64decode(client.payload["stdin"]) == b"input\x00"


@pytest.mark.parametrize(
    ("state", "exit_code", "expected"),
    [
        ("exited", 0, 0),
        ("exited", 999, 255),
        ("timed_out", None, 124),
        ("output_limit", None, 125),
        ("access_or_target_changed", None, 125),
        ("transport_lost", None, 255),
    ],
)
def test_exec_json_one_object_and_exit_status(
    monkeypatch: pytest.MonkeyPatch,
    tmp_path: Path,
    state: str,
    exit_code: int | None,
    expected: int,
) -> None:
    client = FakeClient(
        {
            "state": state,
            "exit_code": exit_code,
            "stdout": b"\xff",
            "stderr": b"remote error",
            "requestId": "request",
        }
    )
    configure_command(monkeypatch, client, tmp_path)
    result = CliRunner().invoke(
        cli_module.cli,
        ["environment", "exec", "--project", "production", "--json", "--", "tool"],
        input="unused",
    )
    assert result.exit_code == expected, result.exception
    value = json.loads(result.stdout)
    assert value["state"] == state and value["output_decoding_replaced"]
    assert value["stderr"] == "remote error" and not result.stderr
    assert "stdin" not in client.payload


def test_exec_requires_project_command_and_bounds_stdin(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    client = FakeClient({})
    configure_command(monkeypatch, client, tmp_path)
    for args in (
        ["exec", "--", "tool"],
        ["exec", "--project", "p"],
        ["exec", "--project", "p", "--stdin", "--", "tool"],
    ):
        result = CliRunner().invoke(
            cli_module.cli, ["environment", *args], input=b"a" * (65536 + 1)
        )
        assert result.exit_code == 2
    assert not client.payload


class ErrorOpener:
    def __init__(self, body: bytes):
        self.body = body

    def open(self, req: Any, timeout: int) -> Any:
        raise HTTPError(req.full_url, 403, "Forbidden", {}, io.BytesIO(self.body))  # type: ignore[arg-type]


def test_http_error_contract_hides_proxy_body_and_preserves_request_id() -> None:
    client = EnvironmentClient(ClientSettings("https://orbit.example", PAT))
    client.opener = ErrorOpener(b"<html>secret proxy body</html>")  # type: ignore[assignment]
    with pytest.raises(EnvironmentError) as invalid:
        client.projects()
    assert invalid.value.code == "contract_mismatch" and "secret" not in str(
        invalid.value
    )
    client.opener = ErrorOpener(
        b'{"code":"forbidden","error":"Not a member","requestId":"r1"}'
    )  # type: ignore[assignment]
    with pytest.raises(EnvironmentError) as valid:
        client.projects()
    assert valid.value.status == 403 and valid.value.request_id == "r1"
    NoRedirect().redirect_request(None, None, 302, "", {}, "https://other")


class JSONResponse(io.BytesIO):
    def __init__(self, value: dict[str, Any]):
        super().__init__(json.dumps(value).encode())
        self.headers = Message()
        self.headers["Content-Type"] = "application/json"
        self.headers["X-Request-Id"] = "request"


class SuccessOpener:
    def __init__(self) -> None:
        self.request: Any = None

    def open(self, req: Any, timeout: int) -> JSONResponse:
        self.request = req
        assert timeout == 50
        return JSONResponse(
            {
                "project_id": "id",
                "project_code": "production",
                "environment_id": "env",
                "target_revision": "2",
                "state": "exited",
                "exit_code": 7,
                "stdout": "AP8=",
                "stderr": "",
                "duration_ms": "10",
                "output_truncated": False,
            }
        )


def test_http_success_decodes_proto_bytes_and_metadata() -> None:
    client = EnvironmentClient(ClientSettings("https://orbit.example/prefix", PAT))
    opener = SuccessOpener()
    client.opener = opener  # type: ignore[assignment]
    result = client.execute(
        {"project_code": "production", "argv": ["tool", "--help"]}, 30
    )
    assert result["stdout"] == b"\x00\xff" and result["stderr"] == b""
    assert result["target_revision"] == 2 and result["requestId"] == "request"
    assert (
        opener.request.full_url
        == "https://orbit.example/prefix/api/cli/environment/exec"
    )
    assert opener.request.get_header("Authorization") == "Bearer " + PAT
    assert json.loads(opener.request.data)["argv"] == ["tool", "--help"]


def test_ssh_rejects_non_tty_without_network() -> None:
    result = CliRunner().invoke(
        cli_module.cli, ["environment", "ssh", "--project", "production"]
    )
    assert result.exit_code == 2 and "requires stdin and stdout TTYs" in result.output
