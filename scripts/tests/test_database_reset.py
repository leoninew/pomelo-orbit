from __future__ import annotations

import importlib.util
import sys
from pathlib import Path
from types import SimpleNamespace

import pytest

SCRIPTS_ROOT = Path(__file__).resolve().parents[1]
SCRIPT_PATH = SCRIPTS_ROOT / "database_reset.py"
SPEC = importlib.util.spec_from_file_location("database_reset", SCRIPT_PATH)
assert SPEC is not None
assert SPEC.loader is not None
database_reset = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = database_reset
SPEC.loader.exec_module(database_reset)


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

    assert database_reset._target_migration_dsn() == (
        "postgres://app%2Fuser:password%2Fwith%20space@[::1]:5432/pomelo_orbit?sslmode=require&connect_timeout=3"
    )


def test_main_dry_run_logs_operations_without_process_side_effects(
    monkeypatch: pytest.MonkeyPatch,
    caplog: pytest.LogCaptureFixture,
) -> None:
    configure_environment(monkeypatch)
    caplog.set_level("INFO")
    monkeypatch.setattr(database_reset, "load_dotenv", lambda _: None)
    monkeypatch.setattr(
        database_reset,
        "_require_executables",
        lambda: pytest.fail("dry run must not check executables"),
    )
    monkeypatch.setattr(
        database_reset.subprocess,
        "run",
        lambda *_args, **_kwargs: pytest.fail("dry run must not run processes"),
    )

    database_reset.main(dry_run=True, run_e2e=False)

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

    monkeypatch.setattr(database_reset, "load_dotenv", lambda _: None)
    monkeypatch.setattr(
        database_reset, "_require_executables", lambda: actions.append("executables")
    )
    monkeypatch.setattr(database_reset, "list_roles", list_target_role)
    monkeypatch.setattr(
        database_reset, "drop_if_exists", lambda _: actions.append("drop")
    )
    monkeypatch.setattr(database_reset, "create", lambda _: actions.append("create"))
    monkeypatch.setattr(database_reset, "grant", lambda: actions.append("grant"))
    monkeypatch.setattr(database_reset, "migrate", lambda _: actions.append("migrate"))
    monkeypatch.setattr(
        database_reset, "verify_e2e", lambda _: actions.append("verify")
    )

    database_reset.main(dry_run=False, run_e2e=True)

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
    monkeypatch.setattr(database_reset, "load_dotenv", lambda _: None)
    monkeypatch.setattr(database_reset, "_require_executables", lambda: None)
    monkeypatch.setattr(database_reset, "list_roles", lambda _: ())
    monkeypatch.setattr(
        database_reset, "drop_if_exists", lambda _: pytest.fail("must not reset")
    )

    with pytest.raises(RuntimeError, match="role does not exist"):
        database_reset.main(dry_run=False, run_e2e=False)


def test_quote_identifier_escapes_double_quotes() -> None:
    assert database_reset._quote_identifier('app"user') == '"app""user"'


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
    monkeypatch.setattr(database_reset, "load_dotenv", lambda _: None)
    monkeypatch.setenv("DB_ADMIN_DSN", admin_dsn)
    monkeypatch.setenv("TARGET_DATABASE", target_database)

    with pytest.raises(RuntimeError, match=message):
        database_reset.main(dry_run=True, run_e2e=False)


def test_main_allows_non_loopback_management_dsn(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    configure_environment(monkeypatch)
    monkeypatch.setattr(database_reset, "load_dotenv", lambda _: None)
    monkeypatch.setenv(
        "DB_ADMIN_DSN",
        "postgresql+psycopg://admin:password@db.example.test:5432/postgres",
    )

    database_reset.main(dry_run=True, run_e2e=False)


def test_parse_args_defaults_to_dry_run(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(sys, "argv", [str(SCRIPT_PATH)])
    arguments = database_reset._parse_args()
    assert arguments.dry_run is True
    assert arguments.verify_e2e is False

    monkeypatch.setattr(sys, "argv", [str(SCRIPT_PATH), "--no-dry-run", "--verify-e2e"])
    arguments = database_reset._parse_args()
    assert arguments.dry_run is False
    assert arguments.verify_e2e is True
