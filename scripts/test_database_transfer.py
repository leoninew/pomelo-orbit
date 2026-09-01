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

sys.path.insert(0, str(Path(__file__).resolve().parent))
import database_transfer as transfer


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
            ("id", "application_id", "version_id", "code"),
            (
                ("service-target", "application-target", "version-current", "target"),
                ("service-other", "application-other", "version-other", "other"),
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
        selected = transfer.service_tables(source, "target")

        self.assertEqual(
            tuple(table.name for table in selected.tables), transfer.SERVICE_TABLES
        )
        self.assertEqual(
            selected.tables[transfer.SERVICE_TABLES.index("service")].rows[0][3],
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
            transfer.service_tables(missing, "target")

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
            transfer.service_tables(broken_transfer, "target")

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
            transfer.service_tables(cyclic, "target")

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
            transfer.service_tables(crossed, "target")

    def test_import_requires_exact_service_file_and_forwards_insert(self) -> None:
        selected = transfer.service_tables(fixture_transfer(), "target")
        with tempfile.TemporaryDirectory() as directory:
            input_path = Path(directory) / "service.jsonl"
            transfer.write_transfer(input_path, selected)
            with (
                patch.dict(os.environ, {}, clear=True),
                patch.object(transfer.shutil, "which", return_value="dbtalk"),
                patch.object(
                    transfer.subprocess,
                    "run",
                    return_value=SimpleNamespace(returncode=0),
                ) as run,
            ):
                code = transfer.import_service(
                    SimpleNamespace(
                        target="sqlite",
                        input=input_path,
                        mode="insert",
                        dsn="sqlite:///./target.db",
                        dsn_env=None,
                        tz="UTC",
                        dbtalk_command="dbtalk",
                    )
                )
        self.assertEqual(code, "target")
        command = run.call_args.args[0]
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

    def test_connection_arguments_require_one_available_dsn_source(self) -> None:
        with self.assertRaisesRegex(transfer.ServiceTransferError, "exactly one"):
            transfer.connection_arguments(None, None)
        with self.assertRaisesRegex(transfer.ServiceTransferError, "exactly one"):
            transfer.connection_arguments("sqlite:///./target.db", "ORBIT_TEST_DSN")
        with patch.dict(os.environ, {}, clear=True):
            with self.assertRaisesRegex(
                transfer.ServiceTransferError, "DSN environment variable is not set"
            ):
                transfer.connection_arguments(None, "ORBIT_TEST_DSN")

    def test_import_forwards_upsert_and_dsn_variable_name(self) -> None:
        selected = transfer.service_tables(fixture_transfer(), "target")
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
                    return_value=SimpleNamespace(returncode=0),
                ) as run,
            ):
                transfer.import_service(
                    SimpleNamespace(
                        target="mysql",
                        input=input_path,
                        mode="upsert",
                        dsn=None,
                        dsn_env="ORBIT_TEST_DSN",
                        tz="Asia/Shanghai",
                        dbtalk_command="dbtalk",
                    )
                )
        command = run.call_args.args[0]
        self.assertIn("upsert", command)
        self.assertIn("--dsn-env", command)
        self.assertIn("ORBIT_TEST_DSN", command)
        self.assertNotIn("root:secret@localhost/orbit", command)
        self.assertIn("Asia/Shanghai", command)

    def test_invalid_input_is_rejected_before_dbtalk_runs(self) -> None:
        selected = transfer.service_tables(fixture_transfer(), "target")
        service = next(table for table in selected.tables if table.name == "service")
        multiple_services = table(
            service.name,
            service.column_names,
            service.rows
            + (("service-extra", "application-target", "version-current", "extra"),),
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
                            input=input_path,
                            mode="insert",
                            dsn="sqlite:///./target.db",
                            dsn_env=None,
                            tz="UTC",
                            dbtalk_command="dbtalk",
                        )
                    )
        run.assert_not_called()

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
