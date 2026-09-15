"""Tests for the database-transfer Click command and domain adapter."""

from __future__ import annotations

import os
import sqlite3
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch
from zoneinfo import ZoneInfo

from click.testing import CliRunner
from cryptography.fernet import Fernet
from dbtalk.database.transfer import TRANSFER_FORMAT, ColumnDefinition, TransferHeader

from pomelo_orbit_cli import cli
from pomelo_orbit_cli.commands import database_transfer as transfer
from pomelo_orbit_cli.errors import ConfigurationError
from pomelo_orbit_cli.settings import (
    DatabaseConnection,
    DatabaseSettings,
    Settings,
)
from pomelo_orbit_cli import settings as settings_module

FERNET_KEY = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
PRIVATE_KEY = "-----BEGIN OPENSSH PRIVATE KEY-----\nprivate-key-data\n-----END OPENSSH PRIVATE KEY-----\n"


def table(
    name: str,
    columns: tuple[str, ...],
    rows: tuple[tuple[object, ...], ...] = (),
    primary_key: tuple[str, ...] = ("id",),
) -> transfer.TableBlock:
    return transfer.TableBlock(
        name=name,
        columns=tuple(ColumnDefinition(column, "TEXT") for column in columns),
        primary_key=primary_key,
        rows=rows,
    )


def service_fixture() -> transfer.TransferFile:
    columns = {
        "project": ("id",),
        "application": ("id", "project_id"),
        "gateway_config": ("id", "application_id"),
        "route": ("id", "service_id", "project_id"),
        "version": ("id", "application_id", "created_from_version_id"),
        "service": ("id", "project_id", "application_id", "version_id", "code"),
        "service_env": ("id", "service_id"),
        "version_component": ("id", "version_id"),
        "service_component": ("id", "service_id", "source_version_component_id"),
    }
    for name in transfer.VERSION_COMPONENT_CHILD_TABLES:
        columns[name] = ("id", "component_id")
    for name in transfer.SERVICE_COMPONENT_CHILD_TABLES:
        columns[name] = ("id", "service_component_id")

    rows = {
        "project": (("project-source",),),
        "application": (("application-source", "project-source"),),
        "gateway_config": (("gateway-source", "application-source"),),
        "route": (("route-source", "service-source", "project-source"),),
        "version": (("version-source", "application-source", None),),
        "service": (
            (
                "service-source",
                "project-source",
                "application-source",
                "version-source",
                "orders",
            ),
        ),
        "service_env": (("service-env-source", "service-source"),),
        "version_component": (("component-source", "version-source"),),
        "service_component": (
            ("service-component-source", "service-source", "component-source"),
        ),
    }
    return transfer.TransferFile(
        header=TransferHeader(format=TRANSFER_FORMAT, source="sqlite"),
        tables=tuple(
            table(name, columns[name], rows.get(name, ()))
            for name in transfer.SERVICE_TABLES
        ),
    )


def environment_fixture(target_type: str = "ssh") -> transfer.TransferFile:
    credential_id = "credential-source" if target_type == "ssh" else None
    revision = "1" if target_type == "ssh" else None
    encrypted_private_key = (
        Fernet(FERNET_KEY.encode("ascii"))
        .encrypt(PRIVATE_KEY.encode("utf-8"))
        .decode("ascii")
    )
    tables = (
        table("project", ("id",), (("project-source",),)),
        table(
            "environment_credential",
            ("id", "project_id", "revision", "encrypted_private_key"),
            (("credential-source", "project-source", "1", encrypted_private_key),),
        ),
        table(
            "environment",
            (
                "id",
                "project_id",
                "code",
                "target_type",
                "ssh_credential_id",
                "ssh_credential_revision",
                "gateway_application_id",
            ),
            (
                (
                    "environment-source",
                    "project-source",
                    "source",
                    target_type,
                    credential_id,
                    revision,
                    "gateway-source",
                ),
            ),
        ),
    )
    return transfer.TransferFile(
        header=TransferHeader(format=TRANSFER_FORMAT, source="sqlite"),
        tables=tables,
    )


def cli_settings(root: Path) -> Settings:
    database_path = root / "orbit.db"
    return Settings(
        database=DatabaseSettings(
            driver="sqlite",
            sqlite_path="orbit.db",
            mysql_dsn=None,
            postgres_dsn=None,
        ),
        connection=DatabaseConnection(
            "sqlite", f"sqlite:///{database_path.as_posix()}"
        ),
        fernet=Fernet(FERNET_KEY.encode("ascii")),
        logging_level="INFO",
        environment="test",
        project_root=root,
        dotenv_path=None,
    )


class DatabaseTransferTests(unittest.TestCase):
    def test_service_transfer_selects_and_retargets_one_closure(self) -> None:
        source = service_fixture()

        selected = transfer.service_tables(source, "project-source", "orders")
        self.assertEqual(
            tuple(table_block.name for table_block in selected.tables),
            transfer.SERVICE_TABLES,
        )
        self.assertEqual(
            transfer.service_transfer_scope(selected).service_code, "orders"
        )

        retargeted = transfer.retarget_service_transfer(selected, "project-target")
        project = next(
            table_block
            for table_block in retargeted.tables
            if table_block.name == "project"
        )
        self.assertEqual(project.rows, ())
        for name in ("application", "service", "route"):
            table_block = next(item for item in retargeted.tables if item.name == name)
            project_index = table_block.column_names.index("project_id")
            self.assertTrue(
                all(row[project_index] == "project-target" for row in table_block.rows)
            )

    def test_service_transfer_rejects_invalid_closure_reference(self) -> None:
        source = service_fixture()
        broken_component = table(
            "service_component",
            ("id", "service_id", "source_version_component_id"),
            (("service-component-source", "service-source", "missing-component"),),
        )
        broken = transfer.TransferFile(
            source.header,
            tuple(
                broken_component if item.name == "service_component" else item
                for item in source.tables
            ),
        )

        with self.assertRaisesRegex(
            transfer.ServiceTransferError, "missing version component"
        ):
            transfer.service_tables(broken, "project-source", "orders")

    def test_transfer_file_round_trip_uses_dbtalk_jsonl(self) -> None:
        expected = service_fixture()
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "service.jsonl"
            transfer.write_transfer(path, expected)
            actual = transfer.load_transfer(path)

        self.assertEqual(actual, expected)

    def test_environment_transfer_reencrypts_private_key_and_orders_import(
        self,
    ) -> None:
        exported = transfer.select_environment_transfer(
            environment_fixture(),
            "project-source",
            "environment-source",
            Fernet(FERNET_KEY.encode("ascii")),
        )
        scope = transfer.environment_transfer_scope(exported)
        self.assertEqual(scope.environment_id, "environment-source")
        credential = next(
            item for item in exported.tables if item.name == "environment_credential"
        )
        self.assertIn("private_key", credential.column_names)
        self.assertNotIn("encrypted_private_key", credential.column_names)

        target_key = Fernet.generate_key()
        imported = transfer.retarget_environment_transfer(
            exported,
            "project-target",
            "target",
            Fernet(target_key),
        )
        self.assertEqual(
            tuple(item.name for item in imported.tables),
            transfer.ENVIRONMENT_DBTALK_IMPORT_TABLES,
        )
        credential = imported.tables[2]
        encrypted_index = credential.column_names.index("encrypted_private_key")
        self.assertEqual(
            Fernet(target_key).decrypt(
                credential.rows[0][encrypted_index].encode("ascii")
            ),
            PRIVATE_KEY.encode("utf-8"),
        )

    def test_local_environment_transfer_omits_ssh_credential(self) -> None:
        exported = transfer.select_environment_transfer(
            environment_fixture("local"),
            "project-source",
            "environment-source",
            Fernet(FERNET_KEY.encode("ascii")),
        )

        self.assertEqual(
            tuple(item.name for item in exported.tables), ("project", "environment")
        )
        self.assertEqual(
            transfer.environment_transfer_scope(exported).project_id, "project-source"
        )

    def test_export_and_import_use_real_sqlite_dbtalk_api(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source_path = root / "source.db"
            target_path = root / "target.db"
            export_path = root / "transfer.jsonl"
            source = sqlite3.connect(source_path)
            try:
                source.execute("CREATE TABLE sample (id TEXT PRIMARY KEY, value TEXT)")
                source.execute("INSERT INTO sample (id, value) VALUES ('one', 'value')")
                source.commit()
            finally:
                source.close()
            target = sqlite3.connect(target_path)
            try:
                target.execute("CREATE TABLE sample (id TEXT PRIMARY KEY, value TEXT)")
                target.commit()
            finally:
                target.close()

            source_connection = DatabaseConnection(
                "sqlite", f"sqlite:///{source_path.as_posix()}"
            )
            target_connection = DatabaseConnection(
                "sqlite", f"sqlite:///{target_path.as_posix()}"
            )
            transfer.export_from_database(
                source_connection,
                export_path,
                ("sample",),
                ZoneInfo("UTC"),
                operation="export",
            )
            transfer.import_into_database(
                target_connection,
                export_path,
                "insert",
                ZoneInfo("UTC"),
                operation="import",
            )
            target = sqlite3.connect(target_path)
            try:
                row = target.execute("SELECT id, value FROM sample").fetchone()
            finally:
                target.close()

        self.assertEqual(row, ("one", "value"))

    def test_cli_routes_export_with_initialized_settings(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            settings = cli_settings(Path(directory))
            with (
                patch.object(cli, "load_settings", return_value=settings),
                patch.object(
                    transfer, "export_service", return_value=Path("out.jsonl")
                ) as export_service,
            ):
                result = CliRunner().invoke(
                    cli.cli,
                    [
                        "database-transfer",
                        "export",
                        "--project-id",
                        "project-source",
                        "--service-code",
                        "orders",
                        "--output",
                        "out.jsonl",
                    ],
                )

        self.assertEqual(result.exit_code, 0, result.output)
        self.assertEqual(result.output, "service transfer written to out.jsonl\n")
        self.assertEqual(
            export_service.call_args.kwargs["project_id"], "project-source"
        )
        self.assertEqual(export_service.call_args.kwargs["service_code"], "orders")

    def test_cli_renders_help_without_loading_runtime_configuration(self) -> None:
        with patch.object(cli, "load_settings") as load_settings:
            result = CliRunner().invoke(
                cli.cli,
                ["database-transfer", "export", "--help"],
            )

        self.assertEqual(result.exit_code, 0, result.output)
        self.assertIn("Export one service deployment closure.", result.output)
        load_settings.assert_not_called()

    def test_cli_renders_safe_transfer_errors(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            settings = cli_settings(Path(directory))
            with (
                patch.object(cli, "load_settings", return_value=settings),
                patch.object(
                    transfer,
                    "export_service",
                    side_effect=transfer.ServiceTransferError("database export failed"),
                ),
            ):
                result = CliRunner().invoke(
                    cli.cli,
                    [
                        "database-transfer",
                        "export",
                        "--project-id",
                        "project-source",
                        "--service-code",
                        "orders",
                        "--output",
                        "out.jsonl",
                    ],
                )

        self.assertEqual(result.exit_code, 1)
        self.assertIn(
            "database transfer blocked: database export failed", result.output
        )
        self.assertNotIn(PRIVATE_KEY, result.output)

    def test_settings_merges_selected_profile_dotenv_and_process_environment(
        self,
    ) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            configs = root / "configs"
            configs.mkdir()
            (configs / "config.yaml").write_text(
                """database:
  driver: sqlite
  sqlite:
    path: data/base.db
jwt:
  secret_key: AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=
logging:
  level: info
""",
                encoding="utf-8",
            )
            (configs / "config.development.yaml").write_text(
                """database:
  sqlite:
    path: data/profile.db
""",
                encoding="utf-8",
            )
            (root / ".env.development").write_text(
                "POMELO_ORBIT_DATABASE__SQLITE__PATH=data/dotenv.db\n",
                encoding="utf-8",
            )
            with patch.dict(
                os.environ,
                {
                    "POMELO_ORBIT_APP__ENV": "development",
                    "POMELO_ORBIT_DATABASE__SQLITE__PATH": "data/process.db",
                },
                clear=True,
            ):
                settings = settings_module.load_settings(root)

        self.assertEqual(settings.environment, "development")
        self.assertEqual(
            settings.connection.dsn,
            f"sqlite:///{(root / 'data' / 'process.db').resolve().as_posix()}",
        )

    def test_settings_validates_fernet_and_go_dsn_mappings_at_initialization(
        self,
    ) -> None:
        invalid_key = "not-a-key"
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            configs = root / "configs"
            configs.mkdir()
            (configs / "config.yaml").write_text(
                f"""database:
  driver: sqlite
  sqlite:
    path: data/orbit.db
jwt:
  secret_key: {invalid_key}
logging:
  level: info
""",
                encoding="utf-8",
            )
            with patch.dict(os.environ, {}, clear=True):
                with self.assertRaisesRegex(ConfigurationError, "Fernet") as error:
                    settings_module.load_settings(root)

        self.assertNotIn(invalid_key, str(error.exception))

        mysql = settings_module._mysql_dsn("orbit:secret@tcp(db.example:3306)/orbit")
        postgres = settings_module._postgres_dsn(
            "postgres://orbit:secret@db.example/orbit"
        )

        self.assertTrue(mysql.startswith("mysql+pymysql://"))
        self.assertTrue(postgres.startswith("postgresql+psycopg://"))


if __name__ == "__main__":
    unittest.main()
