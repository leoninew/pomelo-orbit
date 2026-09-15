"""Focused tests for Orbit's service-closure dbtalk adapter."""

from __future__ import annotations

import contextlib
import io
import os
import sys
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

from cryptography.fernet import Fernet

sys.path.insert(0, str(Path(__file__).resolve().parent))
import database_transfer as transfer


FERNET_KEY = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
PRIVATE_KEY = "-----BEGIN OPENSSH PRIVATE KEY-----\nprivate-key-data\n-----END OPENSSH PRIVATE KEY-----\n"


def table(
    name: str,
    columns: tuple[str, ...],
    rows: tuple[tuple[object, ...], ...],
    primary_key: tuple[str, ...] = ("id",),
) -> transfer.TableBlock:
    return transfer.TableBlock(
        name=name,
        columns=tuple({"name": column, "declared_type": "TEXT"} for column in columns),
        primary_key=primary_key,
        rows=rows,
    )


def fixture_transfer(
    *, source_order: tuple[str, ...] | None = None
) -> transfer.TransferFile:
    tables: dict[str, transfer.TableBlock] = {
        "project": table("project", ("id",), (("project-target",), ("project-other",))),
        "application": table(
            "application",
            ("id", "project_id"),
            (
                ("application-target", "project-target"),
                ("application-other", "project-other"),
            ),
        ),
        "version": table(
            "version",
            ("id", "application_id", "created_from_version_id"),
            (
                ("version-base", "application-target", None),
                ("version-current", "application-target", "version-base"),
                ("version-other", "application-other", None),
            ),
        ),
        "version_component": table(
            "version_component",
            ("id", "version_id"),
            (
                ("component-base", "version-base"),
                ("component-current", "version-current"),
                ("component-other", "version-other"),
            ),
        ),
        "gateway_config": table(
            "gateway_config",
            ("id", "application_id"),
            (
                ("gateway-target", "application-target"),
                ("gateway-other", "application-other"),
            ),
        ),
        "service": table(
            "service",
            ("id", "project_id", "application_id", "version_id", "code"),
            (
                (
                    "service-target",
                    "project-target",
                    "application-target",
                    "version-current",
                    "target",
                ),
                (
                    "service-other",
                    "project-other",
                    "application-other",
                    "version-other",
                    "target",
                ),
            ),
        ),
        "service_env": table(
            "service_env",
            ("id", "service_id"),
            (("env-target", "service-target"), ("env-other", "service-other")),
        ),
        "service_component": table(
            "service_component",
            ("id", "service_id", "source_version_component_id"),
            (
                ("service-component-target", "service-target", "component-current"),
                ("service-component-other", "service-other", "component-other"),
            ),
        ),
        "route": table(
            "route",
            ("id", "project_id", "service_id"),
            (
                ("route-target", "project-target", "service-target"),
                ("route-other", "project-other", "service-other"),
            ),
        ),
    }
    for name in transfer.VERSION_COMPONENT_CHILD_TABLES:
        tables[name] = table(
            name,
            ("id", "component_id"),
            (
                (f"{name}-target", "component-current"),
                (f"{name}-other", "component-other"),
            ),
        )
    for name in transfer.SERVICE_COMPONENT_CHILD_TABLES:
        tables[name] = table(
            name,
            ("id", "service_component_id"),
            (
                (f"{name}-target", "service-component-target"),
                (f"{name}-other", "service-component-other"),
            ),
        )
    names = source_order or tuple(tables)
    ordered = tuple(tables[name] for name in names)
    return transfer.TransferFile(
        header={
            "kind": "header",
            "format": transfer.TRANSFER_FORMAT,
            "source": "sqlite",
        },
        tables=ordered,
    )


def environment_fixture(
    *,
    environment_id: str = "environment-source",
    environment_project_id: str = "project-source",
    target_type: str = "ssh",
    credential_id: str | None = "credential-source",
    credential_project_id: str = "project-source",
    credential_revision: int = 3,
    bound_credential_revision: int | None = 3,
    encrypted_private_key: str | None = None,
) -> transfer.TransferFile:
    if encrypted_private_key is None:
        encrypted_private_key = (
            Fernet(FERNET_KEY.encode("ascii"))
            .encrypt(PRIVATE_KEY.encode("utf-8"))
            .decode("ascii")
        )
    ssh = target_type == "ssh"
    if target_type == "local":
        credential_id = None
        bound_credential_revision = None
    return transfer.TransferFile(
        header={
            "kind": "header",
            "format": transfer.TRANSFER_FORMAT,
            "source": "sqlite",
        },
        tables=(
            table(
                "project",
                ("id", "code"),
                (("project-source", "source-code"),),
            ),
            table(
                "repository_credential",
                ("id", "project_id", "encrypted_data"),
                (("repository-credential", "project-source", "repository-secret"),),
            ),
            table(
                "environment",
                (
                    "id",
                    "project_id",
                    "code",
                    "state",
                    "target_type",
                    "platform",
                    "host",
                    "port",
                    "username",
                    "workspace_root",
                    "ssh_credential_id",
                    "ssh_credential_revision",
                    "host_key_fingerprint",
                    "target_revision",
                    "last_probe_revision",
                    "last_probe_status",
                    "last_probe_at",
                    "last_probe_diagnostic",
                    "gateway_application_id",
                    "created_at",
                    "updated_at",
                ),
                (
                    (
                        environment_id,
                        environment_project_id,
                        "environment-source",
                        "active",
                        target_type,
                        "linux" if ssh else None,
                        "source.example.test" if ssh else None,
                        22 if ssh else None,
                        "orbit" if ssh else None,
                        "/srv/orbit",
                        credential_id,
                        bound_credential_revision,
                        "SHA256:sourcefingerprint" if ssh else None,
                        4,
                        4,
                        "succeeded",
                        "2026-09-15T00:00:00Z",
                        "source-only diagnostic",
                        "gateway-source",
                        "2026-09-15T00:00:00Z",
                        "2026-09-15T00:00:00Z",
                    ),
                ),
            ),
            table(
                "environment_credential",
                (
                    "id",
                    "project_id",
                    "public_key",
                    "encrypted_private_key",
                    "revision",
                    "created_at",
                ),
                (
                    (
                        "credential-source",
                        credential_project_id,
                        "ssh-ed25519 public-key",
                        encrypted_private_key,
                        credential_revision,
                        "2026-09-15T00:00:00Z",
                    ),
                ),
            ),
        ),
    )


class ServiceDatabaseTransferTests(unittest.TestCase):
    def test_service_tables_follow_dbtalk_foreign_key_order(self) -> None:
        self.assertEqual(
            transfer.SERVICE_TABLES,
            (
                "project",
                "application",
                "gateway_config",
                "route",
                "version",
                "service",
                "service_env",
                "version_component",
                "service_component",
                "service_component_endpoint",
                "service_component_env",
                "service_component_mount",
                "service_component_resource",
                "version_component_dependency",
                "version_component_device",
                "version_component_endpoint",
                "version_component_env",
                "version_component_healthcheck",
                "version_component_mount",
                "version_component_resource",
                "version_component_tmpfs",
                "version_component_ulimit",
            ),
        )

    def test_selects_complete_closure_from_arbitrary_full_export_order(self) -> None:
        default = fixture_transfer()
        source = fixture_transfer(
            source_order=tuple(reversed(tuple(table.name for table in default.tables)))
        )
        selected = transfer.service_tables(source, "project-target", "target")

        self.assertEqual(
            tuple(table.name for table in selected.tables), transfer.SERVICE_TABLES
        )
        self.assertEqual(
            selected.tables[transfer.SERVICE_TABLES.index("service")].rows[0][4],
            "target",
        )
        for selected_table in selected.tables:
            self.assertTrue(
                all(
                    "other" not in str(value)
                    for row in selected_table.rows
                    for value in row
                )
            )

    def test_rejects_missing_table_and_invalid_references(self) -> None:
        source = fixture_transfer()
        missing = transfer.TransferFile(
            source.header, tuple(t for t in source.tables if t.name != "route")
        )
        with self.assertRaisesRegex(
            transfer.ServiceTransferError, "missing required tables"
        ):
            transfer.service_tables(missing, "project-target", "target")

        components = next(t for t in source.tables if t.name == "service_component")
        broken = table(
            components.name,
            components.column_names,
            (
                ("service-component-target", "service-target", "missing-component"),
                ("service-component-other", "service-other", "component-other"),
            ),
        )
        broken_transfer = transfer.TransferFile(
            source.header,
            tuple(broken if t.name == components.name else t for t in source.tables),
        )
        with self.assertRaisesRegex(
            transfer.ServiceTransferError, "missing version component"
        ):
            transfer.service_tables(broken_transfer, "project-target", "target")

    def test_rejects_lineage_cycle_and_cross_application(self) -> None:
        source = fixture_transfer()
        versions = next(t for t in source.tables if t.name == "version")
        cycle = table(
            versions.name,
            versions.column_names,
            (
                ("version-base", "application-target", "version-current"),
                ("version-current", "application-target", "version-base"),
                ("version-other", "application-other", None),
            ),
        )
        cyclic = transfer.TransferFile(
            source.header,
            tuple(cycle if t.name == "version" else t for t in source.tables),
        )
        with self.assertRaisesRegex(transfer.ServiceTransferError, "cyclic"):
            transfer.service_tables(cyclic, "project-target", "target")

        cross = table(
            versions.name,
            versions.column_names,
            (
                ("version-base", "application-other", None),
                ("version-current", "application-target", "version-base"),
                ("version-other", "application-other", None),
            ),
        )
        crossed = transfer.TransferFile(
            source.header,
            tuple(cross if t.name == "version" else t for t in source.tables),
        )
        with self.assertRaisesRegex(
            transfer.ServiceTransferError, "crosses applications"
        ):
            transfer.service_tables(crossed, "project-target", "target")

    def test_rejects_inconsistent_service_project_reference(self) -> None:
        source = fixture_transfer()
        services = next(table for table in source.tables if table.name == "service")
        inconsistent = table(
            services.name,
            services.column_names,
            (
                (
                    "service-target",
                    "project-target",
                    "application-other",
                    "version-current",
                    "target",
                ),
                (
                    "service-other",
                    "project-other",
                    "application-other",
                    "version-other",
                    "target",
                ),
            ),
        )
        with self.assertRaisesRegex(
            transfer.ServiceTransferError, "inconsistent project references"
        ):
            transfer.service_tables(
                transfer.TransferFile(
                    source.header,
                    tuple(
                        inconsistent if table.name == "service" else table
                        for table in source.tables
                    ),
                ),
                "project-target",
                "target",
            )

    def test_import_requires_exact_service_file_and_forwards_insert(self) -> None:
        selected = transfer.service_tables(
            fixture_transfer(), "project-target", "target"
        )
        with tempfile.TemporaryDirectory() as directory:
            input_path = Path(directory) / "service.jsonl"
            transfer.write_transfer(input_path, selected)

            def fake_run(command: list[str], **_: object) -> SimpleNamespace:
                if command[1] == "query":
                    return SimpleNamespace(
                        returncode=0,
                        stdout='{"rows":[{"id":"project-other"}]}',
                    )
                target_input = Path(command[command.index("--input") + 1])
                target_transfer = transfer.load_transfer(target_input)
                self.assertEqual(
                    tuple(table.name for table in target_transfer.tables),
                    transfer.SERVICE_TABLES,
                )
                project_table = next(
                    table for table in target_transfer.tables if table.name == "project"
                )
                self.assertEqual(project_table.rows, ())
                for table_name in ("application", "service", "route"):
                    table_block = next(
                        table
                        for table in target_transfer.tables
                        if table.name == table_name
                    )
                    project_index = table_block.column_names.index("project_id")
                    self.assertTrue(
                        all(
                            row[project_index] == "project-other"
                            for row in table_block.rows
                        )
                    )
                return SimpleNamespace(returncode=0, stdout="")

            with (
                patch.dict(os.environ, {}, clear=True),
                patch.object(transfer.shutil, "which", return_value="dbtalk"),
                patch.object(
                    transfer.subprocess,
                    "run",
                    side_effect=fake_run,
                ) as run,
            ):
                code = transfer.import_service(
                    SimpleNamespace(
                        target="sqlite",
                        project_id="project-other",
                        input=input_path,
                        mode="insert",
                        dsn="sqlite:///./target.db",
                        dsn_env=None,
                        tz="UTC",
                        dbtalk_command="dbtalk",
                    )
                )
        self.assertEqual(code, "target")
        self.assertEqual(run.call_count, 2)
        project_query = run.call_args_list[0].args[0]
        self.assertEqual(project_query[0:2], ["dbtalk", "query"])
        self.assertIn('project_id="project-other"', project_query)
        command = run.call_args_list[1].args[0]
        self.assertEqual(command[0:4], ["dbtalk", "import", "--target", "sqlite"])
        self.assertIn("--dsn", command)
        self.assertIn("sqlite:///./target.db", command)
        self.assertIn("--mode", command)
        self.assertIn("insert", command)
        self.assertIn("--tz", command)
        self.assertNotIn("--exclude-table", command)

    def test_import_requires_an_explicit_mode(self) -> None:
        with contextlib.redirect_stderr(io.StringIO()):
            with self.assertRaises(SystemExit):
                transfer.parse_args(
                    [
                        "import",
                        "--target",
                        "sqlite",
                        "--project-id",
                        "project-target",
                        "--input",
                        "service.jsonl",
                        "--dsn",
                        "sqlite:///./target.db",
                    ]
                )

    def test_cli_defaults_to_dbtalk_command(self) -> None:
        args = transfer.parse_args(
            [
                "import",
                "--target",
                "sqlite",
                "--project-id",
                "project-target",
                "--input",
                "service.jsonl",
                "--mode",
                "upsert",
                "--dsn",
                "sqlite:///./target.db",
            ]
        )
        self.assertEqual(args.dbtalk_command, "dbtalk")

    def test_cli_accepts_postgresql_source_and_target(self) -> None:
        import_args = transfer.parse_args(
            [
                "import",
                "--target",
                "postgresql",
                "--project-id",
                "project-target",
                "--input",
                "service.jsonl",
                "--mode",
                "upsert",
                "--dsn-env",
                "ORBIT_TEST_DSN",
            ]
        )
        export_args = transfer.parse_args(
            [
                "export",
                "--source",
                "postgresql",
                "--project-id",
                "project-target",
                "--service-code",
                "target",
                "--output",
                "service.jsonl",
                "--dsn-env",
                "ORBIT_TEST_DSN",
            ]
        )
        self.assertEqual(import_args.target, "postgresql")
        self.assertEqual(export_args.source, "postgresql")

    def test_cli_requires_project_id_for_export_and_import(self) -> None:
        with contextlib.redirect_stderr(io.StringIO()):
            with self.assertRaises(SystemExit):
                transfer.parse_args(
                    [
                        "export",
                        "--source",
                        "sqlite",
                        "--service-code",
                        "target",
                        "--output",
                        "service.jsonl",
                        "--dsn",
                        "sqlite:///./source.db",
                    ]
                )
            with self.assertRaises(SystemExit):
                transfer.parse_args(
                    [
                        "import",
                        "--target",
                        "sqlite",
                        "--input",
                        "service.jsonl",
                        "--mode",
                        "upsert",
                        "--dsn",
                        "sqlite:///./target.db",
                    ]
                )

    def test_connection_arguments_require_one_dsn_source(self) -> None:
        with self.assertRaisesRegex(transfer.ServiceTransferError, "exactly one"):
            transfer.connection_arguments(None, None)
        with self.assertRaisesRegex(transfer.ServiceTransferError, "exactly one"):
            transfer.connection_arguments("sqlite:///./target.db", "ORBIT_TEST_DSN")
        self.assertEqual(
            transfer.connection_arguments(None, "ORBIT_TEST_DSN"),
            ["--dsn-env", "ORBIT_TEST_DSN"],
        )

    def test_import_forwards_upsert_and_dsn_variable_name(self) -> None:
        selected = transfer.service_tables(
            fixture_transfer(), "project-target", "target"
        )
        with tempfile.TemporaryDirectory() as directory:
            input_path = Path(directory) / "service.jsonl"
            transfer.write_transfer(input_path, selected)
            with (
                patch.dict(
                    os.environ,
                    {"ORBIT_TEST_DSN": "mysql+pymysql://root:secret@localhost/orbit"},
                    clear=True,
                ),
                patch.object(transfer.shutil, "which", return_value="dbtalk"),
                patch.object(
                    transfer.subprocess,
                    "run",
                    side_effect=lambda command, **_: SimpleNamespace(
                        returncode=0,
                        stdout=(
                            '{"rows":[{"id":"project-target"}]}'
                            if command[1] == "query"
                            else ""
                        ),
                    ),
                ) as run,
            ):
                transfer.import_service(
                    SimpleNamespace(
                        target="mysql",
                        project_id="project-target",
                        input=input_path,
                        mode="upsert",
                        dsn=None,
                        dsn_env="ORBIT_TEST_DSN",
                        tz="Asia/Shanghai",
                        dbtalk_command="dbtalk",
                    )
                )
        command = run.call_args_list[1].args[0]
        self.assertIn("upsert", command)
        self.assertIn("--dsn-env", command)
        self.assertIn("ORBIT_TEST_DSN", command)
        self.assertNotIn("root:secret@localhost/orbit", command)
        self.assertIn("Asia/Shanghai", command)

    def test_invalid_input_is_rejected_before_dbtalk_runs(self) -> None:
        selected = transfer.service_tables(
            fixture_transfer(), "project-target", "target"
        )
        service = next(table for table in selected.tables if table.name == "service")
        multiple_services = table(
            service.name,
            service.column_names,
            service.rows
            + (
                (
                    "service-extra",
                    "project-target",
                    "application-target",
                    "version-current",
                    "extra",
                ),
            ),
        )
        invalid = transfer.TransferFile(
            selected.header,
            tuple(
                multiple_services if current.name == "service" else current
                for current in selected.tables
            ),
        )
        with tempfile.TemporaryDirectory() as directory:
            input_path = Path(directory) / "invalid.jsonl"
            transfer.write_transfer(input_path, invalid)
            with patch.object(transfer.subprocess, "run") as run:
                with self.assertRaisesRegex(
                    transfer.ServiceTransferError, "exactly one service"
                ):
                    transfer.import_service(
                        SimpleNamespace(
                            target="sqlite",
                            project_id="project-target",
                            input=input_path,
                            mode="insert",
                            dsn="sqlite:///./target.db",
                            dsn_env=None,
                            tz="UTC",
                            dbtalk_command="dbtalk",
                        )
                    )
        run.assert_not_called()

    def test_import_rejects_an_unknown_target_project(self) -> None:
        selected = transfer.service_tables(
            fixture_transfer(), "project-target", "target"
        )
        with tempfile.TemporaryDirectory() as directory:
            input_path = Path(directory) / "service.jsonl"
            transfer.write_transfer(input_path, selected)
            with (
                patch.object(transfer.shutil, "which", return_value="dbtalk"),
                patch.object(
                    transfer.subprocess,
                    "run",
                    return_value=SimpleNamespace(returncode=0, stdout='{"rows":[]}'),
                ) as run,
            ):
                with self.assertRaisesRegex(
                    transfer.ServiceTransferError,
                    "target project does not exist: project-other",
                ):
                    transfer.import_service(
                        SimpleNamespace(
                            target="sqlite",
                            project_id="project-other",
                            input=input_path,
                            mode="insert",
                            dsn="sqlite:///./target.db",
                            dsn_env=None,
                            tz="UTC",
                            dbtalk_command="dbtalk",
                        )
                    )
        self.assertEqual(run.call_count, 1)

    def test_export_filters_service_tables_and_cleans_temporary_file(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "service.jsonl"
            service_export_paths: list[Path] = []

            def fake_run(command: list[str], **_: object) -> SimpleNamespace:
                service_path = Path(command[command.index("--output") + 1])
                service_export_paths.append(service_path)
                transfer.write_transfer(service_path, fixture_transfer())
                return SimpleNamespace(returncode=0)

            with (
                patch.object(transfer.shutil, "which", return_value="dbtalk"),
                patch.object(transfer.subprocess, "run", side_effect=fake_run) as run,
            ):
                result = transfer.export_service(
                    SimpleNamespace(
                        source="sqlite",
                        project_id="project-target",
                        service_code="target",
                        output=output,
                        dsn="sqlite:///./source.db",
                        dsn_env=None,
                        tz="Asia/Shanghai",
                        dbtalk_command="dbtalk",
                    )
                )
            self.assertEqual(result, output.resolve())
            self.assertTrue(output.is_file())
            self.assertTrue(service_export_paths)
            self.assertFalse(service_export_paths[0].exists())
            export_command = run.call_args.args[0]
            self.assertEqual(
                export_command[0:5],
                ["dbtalk", "export", "--source", "sqlite", "--output"],
            )
            self.assertEqual(
                export_command.count("--include-table"), len(transfer.SERVICE_TABLES)
            )
            included_tables = [
                export_command[index + 1]
                for index, argument in enumerate(export_command)
                if argument == "--include-table"
            ]
            self.assertEqual(included_tables, list(transfer.SERVICE_TABLES))
            self.assertNotIn("--exclude-table", export_command)
            self.assertEqual(
                transfer.validate_service_transfer(transfer.load_transfer(output)),
                "target",
            )

    def test_export_environment_writes_only_selected_ssh_environment(self) -> None:
        source = environment_fixture()
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "environment.jsonl"
            dbtalk_exports: list[Path] = []

            def fake_run(command: list[str], **_: object) -> SimpleNamespace:
                export_path = Path(command[command.index("--output") + 1])
                dbtalk_exports.append(export_path)
                transfer.write_transfer(export_path, source)
                return SimpleNamespace(returncode=0)

            with (
                patch.dict(os.environ, {"ORBIT_SOURCE_SECRET": FERNET_KEY}, clear=True),
                patch.object(transfer.shutil, "which", return_value="dbtalk"),
                patch.object(transfer.subprocess, "run", side_effect=fake_run) as run,
            ):
                result = transfer.export_environment(
                    SimpleNamespace(
                        source="sqlite",
                        project_id="project-source",
                        environment_id="environment-source",
                        source_secret_env="ORBIT_SOURCE_SECRET",
                        output=output,
                        dsn="sqlite:///./source.db",
                        dsn_env=None,
                        tz="Asia/Shanghai",
                        dbtalk_command="dbtalk",
                    )
                )

            exported = transfer.load_transfer(output)
            self.assertEqual(result, output.resolve())
            self.assertEqual(exported.header["orbit_scope"], "environment")
            self.assertEqual(
                tuple(table.name for table in exported.tables),
                transfer.ENVIRONMENT_TABLES,
            )
            self.assertEqual(exported.tables[0].rows, ())
            credential = exported.tables[1].row_maps()[0]
            environment = exported.tables[2].row_maps()[0]
            self.assertEqual(credential["private_key"], PRIVATE_KEY)
            self.assertNotIn("encrypted_private_key", credential)
            self.assertIsNone(environment["gateway_application_id"])
            self.assertEqual(
                transfer.environment_transfer_scope(exported),
                transfer.EnvironmentTransferScope(
                    project_id="project-source", environment_id="environment-source"
                ),
            )
            self.assertNotIn("repository-secret", output.read_text(encoding="utf-8"))
            self.assertNotIn("gateway-source", output.read_text(encoding="utf-8"))
            export_command = run.call_args.args[0]
            self.assertEqual(
                export_command.count("--include-table"),
                len(transfer.ENVIRONMENT_TABLES),
            )
            included_tables = [
                export_command[index + 1]
                for index, argument in enumerate(export_command)
                if argument == "--include-table"
            ]
            self.assertEqual(included_tables, list(transfer.ENVIRONMENT_TABLES))
            self.assertTrue(dbtalk_exports)
            self.assertFalse(dbtalk_exports[0].exists())

    def test_export_local_environment_omits_credential(self) -> None:
        exported = transfer.select_environment_transfer(
            environment_fixture(target_type="local"),
            "project-source",
            "environment-source",
            Fernet(FERNET_KEY.encode("ascii")),
        )
        self.assertEqual(
            tuple(table.name for table in exported.tables),
            ("project", "environment"),
        )
        self.assertEqual(exported.tables[0].rows, ())
        self.assertIsNone(exported.tables[1].row_maps()[0]["gateway_application_id"])
        self.assertEqual(
            transfer.environment_transfer_scope(exported).environment_id,
            "environment-source",
        )

    def test_environment_export_rejects_invalid_bindings(self) -> None:
        cases = (
            (
                "different project",
                environment_fixture(environment_project_id="project-other"),
                "project-source",
                "does not exist",
            ),
            (
                "missing credential binding",
                environment_fixture(credential_id=None),
                "project-source",
                "binding is incomplete",
            ),
            (
                "credential from another project",
                environment_fixture(credential_project_id="project-other"),
                "project-source",
                "different project",
            ),
            (
                "credential revision mismatch",
                environment_fixture(bound_credential_revision=2),
                "project-source",
                "revision does not match",
            ),
        )
        source_fernet = Fernet(FERNET_KEY.encode("ascii"))
        for name, source, project_id, message in cases:
            with self.subTest(name=name):
                with self.assertRaisesRegex(transfer.ServiceTransferError, message):
                    transfer.select_environment_transfer(
                        source,
                        project_id,
                        "environment-source",
                        source_fernet,
                    )

    def test_environment_export_does_not_leak_keys_in_errors(self) -> None:
        ciphertext = "invalid-source-ciphertext"
        source = environment_fixture(encrypted_private_key=ciphertext)
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "environment.jsonl"

            def fake_run(command: list[str], **_: object) -> SimpleNamespace:
                transfer.write_transfer(
                    Path(command[command.index("--output") + 1]), source
                )
                return SimpleNamespace(returncode=0)

            with (
                patch.dict(os.environ, {"ORBIT_SOURCE_SECRET": FERNET_KEY}, clear=True),
                patch.object(transfer.shutil, "which", return_value="dbtalk"),
                patch.object(transfer.subprocess, "run", side_effect=fake_run),
            ):
                with self.assertRaises(transfer.ServiceTransferError) as error:
                    transfer.export_environment(
                        SimpleNamespace(
                            source="sqlite",
                            project_id="project-source",
                            environment_id="environment-source",
                            source_secret_env="ORBIT_SOURCE_SECRET",
                            output=output,
                            dsn="sqlite:///./source.db",
                            dsn_env=None,
                            tz="UTC",
                            dbtalk_command="dbtalk",
                        )
                    )
        self.assertNotIn(PRIVATE_KEY, str(error.exception))
        self.assertNotIn(ciphertext, str(error.exception))

    def test_environment_export_rejects_invalid_source_fernet_key(self) -> None:
        invalid_secret = "not-a-valid-fernet-key"
        with (
            patch.dict(
                os.environ,
                {"ORBIT_SOURCE_SECRET": invalid_secret},
                clear=True,
            ),
            patch.object(transfer.subprocess, "run") as run,
        ):
            with self.assertRaises(transfer.ServiceTransferError) as error:
                transfer.export_environment(
                    SimpleNamespace(
                        source="sqlite",
                        project_id="project-source",
                        environment_id="environment-source",
                        source_secret_env="ORBIT_SOURCE_SECRET",
                        output=Path("environment.jsonl"),
                        dsn="sqlite:///./source.db",
                        dsn_env=None,
                        tz="UTC",
                        dbtalk_command="dbtalk",
                    )
                )
        run.assert_not_called()
        self.assertNotIn(invalid_secret, str(error.exception))

    def test_environment_cli_requires_ids_and_secret_variables(self) -> None:
        export_args = transfer.parse_args(
            [
                "export-environment",
                "--source",
                "postgresql",
                "--project-id",
                "project-source",
                "--environment-id",
                "environment-source",
                "--source-secret-env",
                "ORBIT_SOURCE_SECRET",
                "--output",
                "environment.jsonl",
                "--dsn-env",
                "ORBIT_SOURCE_DSN",
            ]
        )
        import_args = transfer.parse_args(
            [
                "import-environment",
                "--target",
                "postgresql",
                "--project-id",
                "project-target",
                "--input",
                "environment.jsonl",
                "--target-secret-env",
                "ORBIT_TARGET_SECRET",
                "--dsn-env",
                "ORBIT_TARGET_DSN",
            ]
        )
        self.assertEqual(export_args.source_secret_env, "ORBIT_SOURCE_SECRET")
        self.assertEqual(import_args.target_secret_env, "ORBIT_TARGET_SECRET")
        with contextlib.redirect_stderr(io.StringIO()):
            with self.assertRaises(SystemExit):
                transfer.parse_args(
                    [
                        "export-environment",
                        "--source",
                        "sqlite",
                        "--project-id",
                        "project-source",
                        "--source-secret-env",
                        "ORBIT_SOURCE_SECRET",
                        "--output",
                        "environment.jsonl",
                        "--dsn",
                        "sqlite:///./source.db",
                    ]
                )
            with self.assertRaises(SystemExit):
                transfer.parse_args(
                    [
                        "import-environment",
                        "--target",
                        "sqlite",
                        "--project-id",
                        "project-target",
                        "--input",
                        "environment.jsonl",
                        "--dsn",
                        "sqlite:///./target.db",
                    ]
                )

    def test_import_environment_retargets_and_reencrypts_private_key(self) -> None:
        source = transfer.select_environment_transfer(
            environment_fixture(),
            "project-source",
            "environment-source",
            Fernet(FERNET_KEY.encode("ascii")),
        )
        target_key = Fernet.generate_key().decode("ascii")
        target_fernet = Fernet(target_key.encode("ascii"))
        with tempfile.TemporaryDirectory() as directory:
            input_path = Path(directory) / "environment.jsonl"
            transfer.write_transfer(input_path, source)
            target_inputs: list[Path] = []

            def fake_run(command: list[str], **_: object) -> SimpleNamespace:
                if command[1] == "query":
                    sql = command[command.index("--sql") + 1]
                    if "FROM project" in sql:
                        return SimpleNamespace(
                            returncode=0,
                            stdout='{"rows":[{"id":"project-target","code":"target-code"}]}',
                        )
                    return SimpleNamespace(returncode=0, stdout='{"rows":[]}')
                target_input = Path(command[command.index("--input") + 1])
                target_inputs.append(target_input)
                imported = transfer.load_transfer(target_input)
                self.assertEqual(
                    tuple(table.name for table in imported.tables),
                    ("environment", "project", "environment_credential"),
                )
                environment = imported.tables[0].row_maps()[0]
                self.assertEqual(imported.tables[1].rows, ())
                credential = imported.tables[2].row_maps()[0]
                self.assertEqual(credential["project_id"], "project-target")
                self.assertEqual(environment["project_id"], "project-target")
                self.assertEqual(environment["code"], "target-code")
                self.assertIsNone(environment["gateway_application_id"])
                self.assertNotIn("private_key", credential)
                self.assertEqual(
                    target_fernet.decrypt(
                        credential["encrypted_private_key"].encode("ascii")
                    ).decode("utf-8"),
                    PRIVATE_KEY,
                )
                self.assertNotIn(PRIVATE_KEY, target_input.read_text(encoding="utf-8"))
                return SimpleNamespace(returncode=0)

            with (
                patch.dict(os.environ, {"ORBIT_TARGET_SECRET": target_key}, clear=True),
                patch.object(transfer.shutil, "which", return_value="dbtalk"),
                patch.object(transfer.subprocess, "run", side_effect=fake_run) as run,
            ):
                result = transfer.import_environment(
                    SimpleNamespace(
                        target="sqlite",
                        project_id="project-target",
                        input=input_path,
                        target_secret_env="ORBIT_TARGET_SECRET",
                        dsn="sqlite:///./target.db",
                        dsn_env=None,
                        tz="Asia/Shanghai",
                        dbtalk_command="dbtalk",
                    )
                )

            self.assertEqual(result, "environment-source")
            self.assertEqual(run.call_count, 3)
            import_command = run.call_args.args[0]
            self.assertIn("insert", import_command)
            self.assertNotIn(PRIVATE_KEY, import_command)
            self.assertTrue(target_inputs)
            self.assertFalse(target_inputs[0].exists())

    def test_import_environment_rejects_target_project_with_environment(self) -> None:
        source = transfer.select_environment_transfer(
            environment_fixture(),
            "project-source",
            "environment-source",
            Fernet(FERNET_KEY.encode("ascii")),
        )
        target_key = Fernet.generate_key().decode("ascii")
        with tempfile.TemporaryDirectory() as directory:
            input_path = Path(directory) / "environment.jsonl"
            transfer.write_transfer(input_path, source)

            def fake_run(command: list[str], **_: object) -> SimpleNamespace:
                sql = command[command.index("--sql") + 1]
                if "FROM project" in sql:
                    return SimpleNamespace(
                        returncode=0,
                        stdout='{"rows":[{"id":"project-target","code":"target-code"}]}',
                    )
                return SimpleNamespace(
                    returncode=0,
                    stdout='{"rows":[{"id":"environment-target"}]}',
                )

            with (
                patch.dict(os.environ, {"ORBIT_TARGET_SECRET": target_key}, clear=True),
                patch.object(transfer.shutil, "which", return_value="dbtalk"),
                patch.object(transfer.subprocess, "run", side_effect=fake_run) as run,
            ):
                with self.assertRaisesRegex(
                    transfer.ServiceTransferError,
                    "already has an environment",
                ):
                    transfer.import_environment(
                        SimpleNamespace(
                            target="sqlite",
                            project_id="project-target",
                            input=input_path,
                            target_secret_env="ORBIT_TARGET_SECRET",
                            dsn="sqlite:///./target.db",
                            dsn_env=None,
                            tz="UTC",
                            dbtalk_command="dbtalk",
                        )
                    )
        self.assertEqual(run.call_count, 2)
        self.assertNotIn("host", run.call_args_list[1].args[0])

    def test_load_transfer_rejects_invalid_jsonl(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "broken.jsonl"
            path.write_text('{"kind":"header"}\n', encoding="utf-8")
            with self.assertRaisesRegex(transfer.ServiceTransferError, "unsupported"):
                transfer.load_transfer(path)

    def test_dbtalk_failure_does_not_echo_stderr(self) -> None:
        with (
            patch.object(transfer.shutil, "which", return_value="dbtalk"),
            patch.object(
                transfer.subprocess,
                "run",
                return_value=SimpleNamespace(returncode=1, stderr="PASSWORD=secret"),
            ),
        ):
            with self.assertRaisesRegex(
                transfer.ServiceTransferError, "exit code 1"
            ) as error:
                transfer.run_dbtalk("dbtalk", [], operation="import")
        self.assertNotIn("secret", str(error.exception))


if __name__ == "__main__":
    unittest.main()
