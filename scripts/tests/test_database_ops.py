from __future__ import annotations

import importlib.util
import sys
from pathlib import Path
from types import SimpleNamespace

import pytest
from click.testing import CliRunner

SCRIPTS_ROOT = Path(__file__).resolve().parents[1]
SCRIPT_PATH = SCRIPTS_ROOT / "database-ops.py"
SPEC = importlib.util.spec_from_file_location("database_ops", SCRIPT_PATH)
assert SPEC is not None
assert SPEC.loader is not None
database_ops = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = database_ops
SPEC.loader.exec_module(database_ops)


def configure_environment(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv(
        "DB_ADMIN_DSN",
        "postgresql+psycopg://admin:admin-password@[::1]:5432/postgres?sslmode=require&connect_timeout=3",
    )
    monkeypatch.setenv("TARGET_DATABASE", "pomelo_orbit")
    monkeypatch.setenv("TARGET_USERNAME", "app_user")
    monkeypatch.setenv("TARGET_PASSWORD", "app-password")


def test_target_migration_dsn_uses_target_credentials_and_preserves_connection_options(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    configure_environment(monkeypatch)
    monkeypatch.setenv("TARGET_USERNAME", "app/user")
    monkeypatch.setenv("TARGET_PASSWORD", "password/with space")

    assert database_ops._target_migration_dsn() == (
        "postgres://app%2Fuser:password%2Fwith%20space@[::1]:5432/pomelo_orbit?sslmode=require&connect_timeout=3"
    )


def test_main_dry_run_logs_operations_without_process_side_effects(
    monkeypatch: pytest.MonkeyPatch,
    caplog: pytest.LogCaptureFixture,
) -> None:
    configure_environment(monkeypatch)
    caplog.set_level("INFO")
    monkeypatch.setattr(database_ops, "load_dotenv", lambda _: None)
    monkeypatch.setattr(
        database_ops,
        "_require_executables",
        lambda: pytest.fail("dry run must not check executables"),
    )
    monkeypatch.setattr(
        database_ops.subprocess,
        "run",
        lambda *_args, **_kwargs: pytest.fail("dry run must not run processes"),
    )

    database_ops.reset_database(dry_run=True, run_e2e=False)

    assert "SELECT 1 FROM pg_roles" in caplog.text
    assert "DROP DATABASE" in caplog.text
    assert "go run ./cmd/migrate" in caplog.text


def test_main_runs_role_preflight_then_orbit_adapters(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    configure_environment(monkeypatch)
    actions: list[str] = []

    def list_target_role(_: object) -> tuple[SimpleNamespace]:
        actions.append("role")
        return (SimpleNamespace(name="app_user"),)

    monkeypatch.setattr(database_ops, "load_dotenv", lambda _: None)
    monkeypatch.setattr(
        database_ops, "_require_executables", lambda: actions.append("executables")
    )
    monkeypatch.setattr(database_ops, "list_roles", list_target_role)
    monkeypatch.setattr(
        database_ops, "drop_if_exists", lambda _: actions.append("drop")
    )
    monkeypatch.setattr(database_ops, "create", lambda _: actions.append("create"))
    monkeypatch.setattr(database_ops, "grant", lambda: actions.append("grant"))
    monkeypatch.setattr(database_ops, "migrate", lambda _: actions.append("migrate"))
    monkeypatch.setattr(database_ops, "verify_e2e", lambda _: actions.append("verify"))

    database_ops.reset_database(dry_run=False, run_e2e=True)

    assert actions == [
        "executables",
        "role",
        "drop",
        "create",
        "grant",
        "migrate",
        "verify",
    ]


def test_main_rejects_missing_target_role_before_reset(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    configure_environment(monkeypatch)
    monkeypatch.setattr(database_ops, "load_dotenv", lambda _: None)
    monkeypatch.setattr(database_ops, "_require_executables", lambda: None)
    monkeypatch.setattr(database_ops, "list_roles", lambda _: ())
    monkeypatch.setattr(
        database_ops, "drop_if_exists", lambda _: pytest.fail("must not reset")
    )

    with pytest.raises(RuntimeError, match="role does not exist"):
        database_ops.reset_database(dry_run=False, run_e2e=False)


def test_quote_identifier_escapes_double_quotes() -> None:
    assert database_ops._quote_identifier('app"user') == '"app""user"'


@pytest.mark.parametrize(
    ("admin_dsn", "target_database", "message"),
    [
        (
            "postgresql+psycopg://admin:password@127.0.0.1:5432/pomelo_orbit",
            "pomelo_orbit",
            "differ",
        ),
        (
            "postgresql+psycopg://admin:password@127.0.0.1:5432/postgres",
            "template1",
            "system database",
        ),
    ],
)
def test_main_rejects_unsafe_reset_target(
    monkeypatch: pytest.MonkeyPatch,
    admin_dsn: str,
    target_database: str,
    message: str,
) -> None:
    configure_environment(monkeypatch)
    monkeypatch.setattr(database_ops, "load_dotenv", lambda _: None)
    monkeypatch.setenv("DB_ADMIN_DSN", admin_dsn)
    monkeypatch.setenv("TARGET_DATABASE", target_database)

    with pytest.raises(RuntimeError, match=message):
        database_ops.reset_database(dry_run=True, run_e2e=False)


def test_main_allows_non_loopback_management_dsn(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    configure_environment(monkeypatch)
    monkeypatch.setattr(database_ops, "load_dotenv", lambda _: None)
    monkeypatch.setenv(
        "DB_ADMIN_DSN",
        "postgresql+psycopg://admin:password@db.example.test:5432/postgres",
    )

    database_ops.reset_database(dry_run=True, run_e2e=False)


def test_click_reset_options(monkeypatch: pytest.MonkeyPatch) -> None:
    calls: list[tuple[bool, bool]] = []

    def fake_reset(*, dry_run: bool, run_e2e: bool) -> None:
        calls.append((dry_run, run_e2e))

    monkeypatch.setattr(database_ops, "reset_database", fake_reset)
    runner = CliRunner()
    assert runner.invoke(database_ops.cli, ["reset"]).exit_code == 0
    assert (
        runner.invoke(
            database_ops.cli, ["reset", "--no-dry-run", "--verify-e2e"]
        ).exit_code
        == 0
    )
    assert calls == [(True, False), (False, True)]


def test_click_copy_from_sqlite(
    monkeypatch: pytest.MonkeyPatch,
    caplog: pytest.LogCaptureFixture,
) -> None:
    calls: list[str] = []
    monkeypatch.setattr(
        database_ops, "copy_from_sqlite_data", lambda: calls.append("copy")
    )
    caplog.set_level("INFO")
    runner = CliRunner()

    preview = runner.invoke(database_ops.cli, ["copy-from-sqlite"])
    assert preview.exit_code == 0
    assert calls == []
    assert "check SQLite foreign keys" in caplog.text
    assert "upsert exported rows into PostgreSQL" in caplog.text

    execution = runner.invoke(database_ops.cli, ["copy-from-sqlite", "--no-dry-run"])
    assert execution.exit_code == 0
    assert calls == ["copy"]


@pytest.mark.parametrize(
    ("engine", "dsn", "extension"),
    [
        (
            "postgres",
            "postgresql+psycopg://orbit:secret@127.0.0.1:5432/pomelo_orbit",
            ".dump",
        ),
        (
            "mysql",
            "mysql+pymysql://orbit:secret@127.0.0.1:3306/pomelo_orbit",
            ".sql",
        ),
    ],
)
def test_click_backup_dumps_with_selected_engine(
    monkeypatch: pytest.MonkeyPatch,
    caplog: pytest.LogCaptureFixture,
    tmp_path: Path,
    engine: str,
    dsn: str,
    extension: str,
) -> None:
    backup_directory = tmp_path / "backup"
    monkeypatch.setattr(database_ops, "BACKUP_DIRECTORY", backup_directory)
    monkeypatch.setattr(
        database_ops, "POSTGRES_DSN" if engine == "postgres" else "MYSQL_DSN", dsn
    )
    monkeypatch.setattr(database_ops, "_backup_client_image", lambda *_args: "")
    calls: list[object] = []

    def fake_dump(options: object) -> Path:
        calls.append(options)
        output: Path = getattr(options, "output")
        output.write_bytes(b"backup")
        return output

    dump_name = (
        "dump_postgres_database" if engine == "postgres" else "dump_mysql_database"
    )
    monkeypatch.setattr(database_ops, dump_name, fake_dump)
    caplog.set_level("INFO")
    runner = CliRunner()

    arguments = ["backup"] if engine == "postgres" else ["backup", "--engine", engine]
    execution = runner.invoke(database_ops.cli, arguments)
    assert execution.exit_code == 0
    assert len(calls) == 1
    files = list(backup_directory.iterdir())
    assert len(files) == 1
    assert files[0].suffix == extension
    assert files[0].read_bytes() == b"backup"
    assert "secret" not in caplog.text


def test_backup_does_not_pull_missing_client_image(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    commands: list[list[str]] = []
    monkeypatch.setattr(
        database_ops.shutil,
        "which",
        lambda name: "docker" if name == "docker" else None,
    )
    monkeypatch.setattr(
        database_ops, "docker_mapped_postgres_container", lambda *_: None
    )

    def inspect(command: list[str], **_kwargs: object) -> SimpleNamespace:
        commands.append(command)
        return SimpleNamespace(returncode=1)

    monkeypatch.setattr(database_ops.subprocess, "run", inspect)

    with pytest.raises(RuntimeError, match="is not local"):
        database_ops._backup_client_image("postgres", "db.example.test", 5432)

    assert commands == [["docker", "image", "inspect", "postgres:18"]]
