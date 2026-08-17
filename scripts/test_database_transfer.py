"""Tests for the generic SQLite and MySQL database transfer tool."""

from __future__ import annotations

import argparse
import sqlite3
import sys
import tempfile
import unittest
from pathlib import Path
from typing import Any

sys.path.insert(0, str(Path(__file__).resolve().parent))
import database_transfer as transfer


SCHEMA = """
CREATE TABLE parent (id TEXT PRIMARY KEY, name TEXT NOT NULL);
CREATE TABLE child (id TEXT PRIMARY KEY, parent_id TEXT NOT NULL REFERENCES parent(id), binary_value BLOB NOT NULL, text_value TEXT NOT NULL);
CREATE TABLE record (id TEXT PRIMARY KEY, parent_id TEXT NOT NULL REFERENCES parent(id), status TEXT NOT NULL);
"""

SERVICE_SELECTION_SCHEMA = """
CREATE TABLE project (id TEXT PRIMARY KEY);
CREATE TABLE application (id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES project(id));
CREATE TABLE version (id TEXT PRIMARY KEY, application_id TEXT NOT NULL REFERENCES application(id), created_from_version_id TEXT REFERENCES version(id));
CREATE TABLE version_component (id TEXT PRIMARY KEY, version_id TEXT NOT NULL REFERENCES version(id));
CREATE TABLE gateway_config (application_id TEXT PRIMARY KEY REFERENCES application(id), raw_value TEXT NOT NULL);
CREATE TABLE service (id TEXT PRIMARY KEY, application_id TEXT NOT NULL REFERENCES application(id), version_id TEXT NOT NULL REFERENCES version(id), code TEXT NOT NULL);
CREATE TABLE service_env (service_id TEXT NOT NULL REFERENCES service(id), raw_value TEXT NOT NULL);
CREATE TABLE service_component (id TEXT PRIMARY KEY, service_id TEXT NOT NULL REFERENCES service(id), source_version_component_id TEXT NOT NULL REFERENCES version_component(id));
CREATE TABLE route (id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES project(id), service_id TEXT REFERENCES service(id), raw_value TEXT NOT NULL);
""" + "\n".join(
    f'CREATE TABLE "{name}" (component_id TEXT NOT NULL REFERENCES version_component(id), raw_value TEXT NOT NULL);'
    for name in transfer.VERSION_COMPONENT_CHILD_TABLES
) + "\n" + "\n".join(
    f'CREATE TABLE "{name}" (service_component_id TEXT NOT NULL REFERENCES service_component(id), raw_value TEXT NOT NULL);'
    for name in transfer.SERVICE_COMPONENT_CHILD_TABLES
)


def connect(path: Path) -> sqlite3.Connection:
    connection = sqlite3.connect(path)
    connection.executescript(SCHEMA)
    connection.execute("PRAGMA foreign_keys = ON")
    return connection


def connect_service_selection(path: Path) -> sqlite3.Connection:
    connection = sqlite3.connect(path)
    connection.executescript(SERVICE_SELECTION_SCHEMA)
    connection.execute("PRAGMA foreign_keys = ON")
    return connection


class DatabaseTransferTests(unittest.TestCase):
    def populate(self, connection: sqlite3.Connection, suffix: str = "") -> None:
        parent = f"parent{suffix}"
        connection.execute("INSERT INTO parent VALUES (?, ?)", (parent, f"Parent{suffix}"))
        connection.execute(
            "INSERT INTO child VALUES (?, ?, ?, ?)",
            (f"child{suffix}", parent, b"\x00binary-value", f"plain-text{suffix};with:semicolon"),
        )
        connection.execute("INSERT INTO record VALUES (?, ?, ?)", (f"record{suffix}", parent, "active"))
        connection.commit()

    def test_sqlite_export_contains_all_tables_and_original_values(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source_path = Path(directory) / "source.db"
            source = connect(source_path)
            self.populate(source)
            source.close()

            archive = transfer.archive_from_sqlite(source_path)
            self.assertEqual([table["name"] for table in archive["tables"]], ["child", "parent", "record"])
            rendered = transfer.render_sql_file(archive, "sqlite")
            self.assertIn("PRAGMA foreign_keys = OFF;", rendered)
            self.assertIn('DELETE FROM "record";', rendered)
            self.assertIn("plain-text", rendered)
            self.assertIn("X'0062696E6172792D76616C7565'", rendered)
            self.assertIn(transfer.ARCHIVE_PREFIX, rendered)

    def test_sqlite_import_replaces_exported_tables_without_business_filtering(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source_path = root / "source.db"
            source = connect(source_path)
            self.populate(source)
            source.close()
            archive = transfer.archive_from_sqlite(source_path)

            target_path = root / "target.db"
            target = connect(target_path)
            self.populate(target, "-old")
            target.close()
            transfer.import_into_sqlite(archive, target_path, replace=True)

            restored = sqlite3.connect(target_path)
            try:
                self.assertEqual(restored.execute("SELECT id FROM parent").fetchone()[0], "parent")
                self.assertEqual(restored.execute("SELECT binary_value FROM child").fetchone()[0], b"\x00binary-value")
                self.assertEqual(restored.execute("SELECT COUNT(*) FROM record").fetchone()[0], 1)
            finally:
                restored.close()

    def test_export_and_import_service_transfer_deployment_closure_without_deletes(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source_path = Path(directory) / "source.db"
            source = connect_service_selection(source_path)
            try:
                source.executemany("INSERT INTO project VALUES (?)", (("project-target",), ("project-other",)))
                source.executemany(
                    "INSERT INTO application VALUES (?, ?)",
                    (("application-target", "project-target"), ("application-other", "project-other")),
                )
                source.executemany(
                    "INSERT INTO version VALUES (?, ?, ?)",
                    (("version-target-base", "application-target", None), ("version-target-current", "application-target", "version-target-base"), ("version-other", "application-other", None)),
                )
                source.executemany(
                    "INSERT INTO version_component VALUES (?, ?)",
                    (("component-target-base", "version-target-base"), ("component-target-current", "version-target-current"), ("component-other", "version-other")),
                )
                for table_name in transfer.VERSION_COMPONENT_CHILD_TABLES:
                    source.executemany(
                        f'INSERT INTO "{table_name}" VALUES (?, ?)',
                        (("component-target-base", "target-base"), ("component-target-current", "target-current"), ("component-other", "other")),
                    )
                source.executemany(
                    "INSERT INTO gateway_config VALUES (?, ?)",
                    (("application-target", "gateway-target"), ("application-other", "gateway-other")),
                )
                source.executemany(
                    "INSERT INTO service VALUES (?, ?, ?, ?)",
                    (("service-target", "application-target", "version-target-current", "sub2api-default"), ("service-other", "application-other", "version-other", "other-default")),
                )
                source.executemany(
                    "INSERT INTO service_env VALUES (?, ?)",
                    (("service-target", "target"), ("service-other", "other")),
                )
                source.executemany(
                    "INSERT INTO service_component VALUES (?, ?, ?)",
                    (("service-component-target", "service-target", "component-target-current"), ("service-component-other", "service-other", "component-other")),
                )
                for table_name in transfer.SERVICE_COMPONENT_CHILD_TABLES:
                    source.executemany(
                        f'INSERT INTO "{table_name}" VALUES (?, ?)',
                        (("service-component-target", "target"), ("service-component-other", "other")),
                    )
                source.executemany(
                    "INSERT INTO route VALUES (?, ?, ?, ?)",
                    (("route-target", "project-target", "service-target", "target"), ("route-other", "project-other", "service-other", "other")),
                )
                source.commit()
            finally:
                source.close()

            output_path = Path(directory) / "sub2api-default.sqlite.sql"
            transfer.export_service_command(
                argparse.Namespace(
                    sqlite_path=source_path,
                    output=output_path,
                    service_code="sub2api-default",
                )
            )
            self.assertNotIn("DELETE FROM", output_path.read_text(encoding="utf-8"))
            archive = transfer.load_archive(output_path)
            tables = {table["name"]: table for table in archive["tables"]}

            def values(table_name: str, column_name: str) -> set[str]:
                table = tables[table_name]
                index = table["columns"].index(column_name)
                return {str(row[index]) for row in table["rows"]}

            self.assertEqual(set(tables), set(transfer.SERVICE_TABLES))
            self.assertEqual(values("project", "id"), {"project-target"})
            self.assertEqual(values("application", "id"), {"application-target"})
            self.assertEqual(values("version", "id"), {"version-target-base", "version-target-current"})
            self.assertEqual(values("version_component", "id"), {"component-target-base", "component-target-current"})
            self.assertEqual(values("gateway_config", "application_id"), {"application-target"})
            self.assertEqual(values("service", "code"), {"sub2api-default"})
            self.assertEqual(values("service_env", "service_id"), {"service-target"})
            self.assertEqual(values("service_component", "id"), {"service-component-target"})
            self.assertEqual(values("route", "service_id"), {"service-target"})
            for table_name in transfer.VERSION_COMPONENT_CHILD_TABLES:
                self.assertEqual(values(table_name, "component_id"), {"component-target-base", "component-target-current"})
            for table_name in transfer.SERVICE_COMPONENT_CHILD_TABLES:
                self.assertEqual(values(table_name, "service_component_id"), {"service-component-target"})

            target_path = Path(directory) / "target.db"
            target = connect_service_selection(target_path)
            target.close()
            transfer.import_service_command(argparse.Namespace(sqlite_path=target_path, input=output_path))
            restored = sqlite3.connect(target_path)
            try:
                self.assertEqual(restored.execute("SELECT code FROM service").fetchall(), [("sub2api-default",)])
                self.assertEqual(restored.execute("SELECT COUNT(*) FROM version_component").fetchone()[0], 2)
                self.assertEqual(restored.execute("SELECT COUNT(*) FROM route").fetchone()[0], 1)
                self.assertEqual(list(restored.execute("PRAGMA foreign_key_check")), [])
            finally:
                restored.close()

    def test_file_conversion_preserves_payload_and_changes_target_dialect(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source_path = root / "source.db"
            source = connect(source_path)
            self.populate(source)
            source.close()
            sqlite_file = root / "database.sqlite.sql"
            transfer.write_output(sqlite_file, transfer.render_sql_file(transfer.archive_from_sqlite(source_path), "sqlite"))

            mysql_file = root / "database.mysql.sql"
            output = transfer.convert_command(
                argparse.Namespace(source_format="sqlite", target_format="mysql", input=sqlite_file, output=mysql_file)
            )
            self.assertEqual(output, mysql_file.resolve())
            archive = transfer.load_archive(mysql_file)
            self.assertEqual(archive["driver"], "mysql")
            rendered = mysql_file.read_text(encoding="utf-8")
            self.assertIn("SET FOREIGN_KEY_CHECKS = 0;", rendered)
            self.assertIn("plain-text", rendered)

            restored_sqlite_file = root / "restored.sqlite.sql"
            transfer.convert_command(
                argparse.Namespace(source_format="mysql", target_format="sqlite", input=mysql_file, output=restored_sqlite_file)
            )
            restored_archive = transfer.load_archive(restored_sqlite_file)
            self.assertEqual(restored_archive["driver"], "sqlite")
            self.assertIn("PRAGMA foreign_keys = OFF;", restored_sqlite_file.read_text(encoding="utf-8"))

    def test_mysql_export_and_import_use_all_tables_without_domain_rules(self) -> None:
        timestamp = "2026-08-15 10:40:42.2778562 +0800 CST"
        archive = {
            "format": transfer.ARCHIVE_FORMAT,
            "driver": "mysql",
            "tables": [
                {"name": "child", "columns": ["id", "parent_id", "binary_value", "created_at"], "rows": [["child", "parent", {"type": "blob", "base64": "AGJpbmFyeS12YWx1ZQ=="}, timestamp]]},
                {"name": "parent", "columns": ["id", "name"], "rows": [["parent", "Parent"]]},
            ],
        }
        connection = FakeMySQLConnection(archive)
        original_connection = transfer.mysql_connection
        transfer.mysql_connection = lambda args: connection
        try:
            exported = transfer.archive_from_mysql(argparse.Namespace())
            self.assertEqual(exported, archive)
            transfer.import_into_mysql(archive, argparse.Namespace(), replace=True)
        finally:
            transfer.mysql_connection = original_connection

        self.assertTrue(connection.closed)
        self.assertTrue(any("extra NOT LIKE '%% GENERATED'" in statement for statement in connection.statements))
        self.assertIn("DELETE FROM `parent`", connection.statements)
        self.assertIn("DELETE FROM `child`", connection.statements)
        self.assertEqual(connection.inserted[0][1][0], ("child", "parent", b"\x00binary-value", timestamp))

    def test_mysql_values_preserve_original_timestamp_text(self) -> None:
        self.assertEqual(
            transfer.mysql_sql_value("2026-08-15 10:40:42.2778562 +0800 CST"),
            "'2026-08-15 10:40:42.2778562 +0800 CST'",
        )
        self.assertEqual(transfer.mysql_sql_value("2024-03-16T00:00:00Z"), "'2024-03-16T00:00:00Z'")


class FakeMySQLCursor:
    def __init__(self, archive: dict[str, Any], connection: "FakeMySQLConnection") -> None:
        self.archive = archive
        self.connection = connection
        self.rows: list[tuple[Any, ...]] = []

    def __enter__(self) -> "FakeMySQLCursor":
        return self

    def __exit__(self, exc_type: object, exc_value: object, traceback: object) -> None:
        return None

    def execute(self, statement: str, parameters: tuple[str, ...] | None = None) -> None:
        self.connection.statements.append(statement)
        if "information_schema.tables" in statement:
            self.rows = [(table["name"],) for table in self.archive["tables"]]
        elif "information_schema.columns" in statement:
            assert parameters is not None
            table = next(table for table in self.archive["tables"] if table["name"] == parameters[0])
            self.rows = [(column,) for column in table["columns"]]
        elif statement.startswith("SELECT"):
            table_name = statement.split(" FROM `", 1)[1].split("`", 1)[0]
            table = next(table for table in self.archive["tables"] if table["name"] == table_name)
            self.rows = [tuple(transfer.decode_value(value) for value in row) for row in table["rows"]]

    def executemany(self, statement: str, values: list[tuple[Any, ...]]) -> None:
        self.connection.inserted.append((statement, values))

    def fetchall(self) -> list[tuple[Any, ...]]:
        return self.rows


class FakeMySQLConnection:
    def __init__(self, archive: dict[str, Any]) -> None:
        self.archive = archive
        self.statements: list[str] = []
        self.inserted: list[tuple[str, list[tuple[Any, ...]]]] = []
        self.closed = False

    def cursor(self) -> FakeMySQLCursor:
        return FakeMySQLCursor(self.archive, self)

    def commit(self) -> None:
        return None

    def rollback(self) -> None:
        return None

    def close(self) -> None:
        self.closed = True


if __name__ == "__main__":
    unittest.main()
