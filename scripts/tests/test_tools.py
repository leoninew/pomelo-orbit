from __future__ import annotations

import subprocess
from pathlib import Path

import pytest
import ulid
from click.testing import CliRunner

from pomelo_orbit_cli import cli as cli_module
from pomelo_orbit_cli.commands import cert as cert_module
from pomelo_orbit_cli.settings import Settings


@pytest.fixture(autouse=True)
def configure_cli(monkeypatch: pytest.MonkeyPatch, tmp_path: Path) -> None:
    settings = Settings(
        logging_level="INFO",
        environment="development",
        project_root=tmp_path,
        dotenv_path=None,
    )
    monkeypatch.setattr(cli_module, "load_settings", lambda: settings)


@pytest.mark.parametrize("count", [1, 3])
def test_ulid_outputs_requested_number_of_valid_identifiers(count: int) -> None:
    arguments = ["ulid"] if count == 1 else ["ulid", "-n", str(count)]

    result = CliRunner().invoke(cli_module.cli, arguments)

    assert result.exit_code == 0, result.output
    identifiers = result.stdout.splitlines()
    assert len(identifiers) == count
    assert len(set(identifiers)) == count
    assert all(str(ulid.from_str(value)) == value for value in identifiers)


def test_cert_new_combines_mkcert_output(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    certificate = (
        b"-----BEGIN CERTIFICATE-----\ncertificate\n-----END CERTIFICATE-----\n"
    )
    private_key = (
        b"-----BEGIN PRIVATE KEY-----\nprivate-key\n-----END PRIVATE KEY-----\n"
    )
    output_directory = tmp_path / "certificates"

    def generate(command: list[str]) -> subprocess.CompletedProcess[str]:
        assert command[0] == "mkcert"
        assert command[-1] == "app.localhost"
        Path(command[command.index("-cert-file") + 1]).write_bytes(certificate)
        Path(command[command.index("-key-file") + 1]).write_bytes(private_key)
        return subprocess.CompletedProcess(command, 0, "", "")

    monkeypatch.setattr(cert_module, "run", generate)

    result = CliRunner().invoke(
        cli_module.cli,
        ["cert", "new", "-n", "app.localhost", "--cert-dir", str(output_directory)],
    )

    assert result.exit_code == 0, result.output
    output = output_directory / "app.localhost.pem"
    assert output.read_bytes() == certificate + private_key
    assert list(output_directory.iterdir()) == [output]


def test_cert_new_reports_mkcert_failure_and_preserves_existing_certificate(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    output = tmp_path / "app.localhost.pem"
    output.write_bytes(b"existing certificate")
    monkeypatch.setattr(
        cert_module,
        "run",
        lambda command: subprocess.CompletedProcess(
            command, 1, "", "mkcert could not generate the certificate"
        ),
    )

    result = CliRunner().invoke(
        cli_module.cli,
        ["cert", "new", "-n", "app.localhost", "--cert-dir", str(tmp_path)],
    )

    assert result.exit_code == 1
    assert "mkcert could not generate the certificate" in result.output
    assert output.read_bytes() == b"existing certificate"


def test_cert_check_receives_resolved_directory(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    calls: list[tuple[str, Path]] = []
    monkeypatch.setattr(
        cert_module,
        "cmd_check",
        lambda domain, directory: calls.append((domain, directory)),
    )

    result = CliRunner().invoke(
        cli_module.cli,
        ["cert", "check", "--domain", "app.localhost", "--cert-dir", "certificates"],
    )

    assert result.exit_code == 0, result.output
    assert calls == [("app.localhost", Path("certificates").resolve())]
