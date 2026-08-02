"""Focused round-trip test for the deterministic RAGFlow TEI SQL exporter."""

from __future__ import annotations

import argparse
import sqlite3
import tempfile
import unittest
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
import export_ragflow_tei_baseline as exporter


SCHEMA = """
CREATE TABLE project (id TEXT PRIMARY KEY, name TEXT NOT NULL);
CREATE TABLE application (id TEXT PRIMARY KEY, project_id TEXT, code TEXT NOT NULL, name TEXT NOT NULL, kind TEXT NOT NULL);
CREATE TABLE gateway_config (application_id TEXT PRIMARY KEY, rest_api_url TEXT NOT NULL, base_domain TEXT NOT NULL, default_entrypoint TEXT NOT NULL, tls_mode TEXT NOT NULL);
CREATE TABLE version (id TEXT PRIMARY KEY, application_id TEXT NOT NULL, label TEXT NOT NULL, note TEXT);
CREATE TABLE version_component (id TEXT PRIMARY KEY, version_id TEXT NOT NULL, name TEXT NOT NULL, image TEXT NOT NULL, pull_policy TEXT NOT NULL CHECK (pull_policy IN ('always', 'missing', 'never')), restart_policy TEXT, command_json TEXT);
CREATE TABLE version_component_dependency (component_id TEXT NOT NULL, position INTEGER NOT NULL, name TEXT NOT NULL);
CREATE TABLE version_component_env (component_id TEXT NOT NULL, env_key TEXT NOT NULL, value TEXT NOT NULL, position INTEGER NOT NULL);
CREATE TABLE version_component_healthcheck (component_id TEXT PRIMARY KEY, test TEXT NOT NULL);
CREATE TABLE version_component_mount (component_id TEXT NOT NULL, position INTEGER NOT NULL, source TEXT NOT NULL);
CREATE TABLE version_component_endpoint (component_id TEXT NOT NULL, name TEXT NOT NULL, protocol TEXT NOT NULL, container_port INTEGER NOT NULL, mode TEXT NOT NULL, bind_address TEXT, listen_port INTEGER, entrypoint TEXT, path_prefix TEXT, position INTEGER NOT NULL);
CREATE TABLE version_component_resource (component_id TEXT PRIMARY KEY, reservation_memory TEXT);
CREATE TABLE version_component_tmpfs (component_id TEXT NOT NULL, position INTEGER NOT NULL, target TEXT NOT NULL);
CREATE TABLE version_component_ulimit (component_id TEXT NOT NULL, name TEXT NOT NULL, soft INTEGER NOT NULL);
CREATE TABLE version_component_device (component_id TEXT NOT NULL, position INTEGER NOT NULL, driver TEXT NOT NULL, device_count TEXT NOT NULL, capabilities_json TEXT NOT NULL);
CREATE TABLE service (id TEXT PRIMARY KEY, application_id TEXT NOT NULL, instance_key TEXT NOT NULL, version_id TEXT NOT NULL, status TEXT NOT NULL);
CREATE TABLE service_component (id TEXT PRIMARY KEY, service_id TEXT NOT NULL, source_version_component_id TEXT NOT NULL, component_name TEXT NOT NULL, status TEXT NOT NULL);
CREATE TABLE service_component_env (service_component_id TEXT NOT NULL, env_key TEXT NOT NULL, value TEXT, state TEXT NOT NULL);
CREATE TABLE service_component_mount (id TEXT PRIMARY KEY, service_component_id TEXT NOT NULL, target TEXT NOT NULL, source TEXT, state TEXT NOT NULL);
CREATE TABLE service_component_resource (service_component_id TEXT PRIMARY KEY, limit_cpus TEXT, limit_memory TEXT, reservation_cpus TEXT, reservation_memory TEXT, state TEXT NOT NULL);
CREATE TABLE service_component_endpoint (service_component_id TEXT NOT NULL, name TEXT NOT NULL, mode TEXT, bind_address TEXT, listen_port INTEGER, entrypoint TEXT, path_prefix TEXT, state TEXT NOT NULL);
"""

VERSION_A_IMAGE = "example/tei:a"
VERSION_B_IMAGE = "example/tei:b"
COMPONENT_NAMES = ("es01", "minio", "mysql", "ragflow", "redis", "tei")


def connect(path: Path) -> sqlite3.Connection:
    connection = sqlite3.connect(path)
    connection.executescript(SCHEMA)
    return connection


class BaselineExportTests(unittest.TestCase):
    def populate(self, connection: sqlite3.Connection) -> None:
        connection.execute("INSERT INTO project VALUES ('project', 'Project')")
        connection.executemany(
            "INSERT INTO application VALUES (?, 'project', ?, ?, ?)",
            (("gateway", "traefik", "Gateway", "gateway"), ("ragflow", "ragflow", "RAGFlow", "standard")),
        )
        connection.execute("INSERT INTO gateway_config VALUES ('gateway', 'http://127.0.0.1:8080', 'example.test', 'web', 'none')")
        connection.executemany(
            "INSERT INTO version VALUES (?, ?, ?, ?)",
            (
                ("gateway-v1", "gateway", "default", None),
                ("version-a", "ragflow", "version-a", "baseline A"),
                ("version-b", "ragflow", "version-b", "baseline B"),
            ),
        )
        components = [("gateway-traefik", "gateway-v1", "traefik", "traefik:3.6", "missing", None, "[]")]
        for version_id in ("version-a", "version-b"):
            tei_image = VERSION_A_IMAGE if version_id == "version-a" else VERSION_B_IMAGE
            components.extend(
                (
                    f"{version_id}-{name}",
                    version_id,
                    name,
                    tei_image if name == "tei" else f"example/{name}:1",
                    "missing",
                    "unless-stopped",
                    "[]",
                )
                for name in COMPONENT_NAMES
            )
        connection.executemany("INSERT INTO version_component VALUES (?, ?, ?, ?, ?, ?, ?)", components)
        connection.executemany(
            "INSERT INTO version_component_healthcheck VALUES (?, 'curl -fsS /health')",
            (("version-a-tei",), ("version-b-tei",)),
        )
        connection.executemany(
            "INSERT INTO version_component_resource VALUES (?, '4g')",
            (("version-a-tei",), ("version-b-tei",)),
        )
        connection.execute("INSERT INTO version_component_device VALUES ('version-b-tei', 0, 'accelerator', 'all', '[\"compute\"]')")
        connection.executemany(
            "INSERT INTO version_component_endpoint VALUES (?, 'http', 'http', 80, 'internal', NULL, NULL, NULL, NULL, 0)",
            (("version-a-ragflow",), ("version-b-ragflow",)),
        )
        connection.executemany(
            "INSERT INTO service VALUES (?, ?, 'default', ?, ?)",
            (
                ("gateway-service", "gateway", "gateway-v1", "running"),
                ("ragflow-service", "ragflow", "version-a", "running"),
            ),
        )
        service_components = [("gateway-component", "gateway-service", "gateway-traefik", "traefik", "active")]
        service_components.extend(
            (
                f"service-{name}",
                "ragflow-service",
                f"version-a-{name}",
                name,
                "active",
            )
            for name in COMPONENT_NAMES
        )
        connection.executemany("INSERT INTO service_component VALUES (?, ?, ?, ?, ?)", service_components)
        connection.execute(
            "INSERT INTO service_component_endpoint VALUES ('service-ragflow', 'http', 'local', '127.0.0.1', 9380, NULL, NULL, 'override')"
        )
        connection.execute(
            "INSERT INTO service_component_env VALUES ('service-ragflow', 'MYSQL_PASSWORD', 'exported-value', 'override')"
        )
        connection.execute(
            "INSERT INTO service_component_mount VALUES ('mount-overlay', 'service-ragflow', '/data', 'runtime-data', 'override')"
        )
        connection.commit()

    def test_export_round_trips_with_new_ragflow_runtime_config(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source_path = root / "source.db"
            source = connect(source_path)
            self.populate(source)
            source.close()

            output = root / "baseline.sql"
            args = argparse.Namespace(database=str(source_path), output=str(output), project_id=None)
            connection = exporter.connect_read_only(source_path)
            try:
                exporter.ensure_database_integrity(connection)
                rendered = exporter.render_sql(connection, exporter.selected_baseline(connection, args))
            finally:
                connection.close()
            exporter.write_output(output, rendered, replace=False)

            restored_path = root / "restored.db"
            restored = connect(restored_path)
            try:
                restored.execute("INSERT INTO project VALUES ('project', 'Project')")
                restored.executescript(rendered)
                self.assertEqual(restored.execute("SELECT COUNT(*) FROM project").fetchone()[0], 1)
                self.assertIn("exported-value", rendered)
                self.assertEqual(restored.execute("SELECT COUNT(*) FROM service_component_env").fetchone()[0], 1)
                self.assertEqual(
                    restored.execute(
                        "SELECT target, source, state FROM service_component_mount"
                    ).fetchall(),
                    [("/data", "runtime-data", "override")],
                )
                self.assertEqual(restored.execute("SELECT DISTINCT status FROM service").fetchall(), [("running",)])
                self.assertEqual(restored.execute("SELECT COUNT(*) FROM version_component_device").fetchone()[0], 1)
                self.assertEqual(restored.execute("SELECT COUNT(*) FROM version_component_healthcheck").fetchone()[0], 2)
                self.assertEqual(restored.execute("SELECT COUNT(*) FROM version_component_resource").fetchone()[0], 2)
                self.assertEqual(
                    restored.execute(
                        "SELECT mode, bind_address, listen_port FROM service_component_endpoint"
                    ).fetchall(),
                    [("local", "127.0.0.1", 9380)],
                )
                self.assertEqual(
                    restored.execute("SELECT label FROM version WHERE application_id = 'ragflow' ORDER BY label").fetchall(),
                    [("version-a",), ("version-b",)],
                )
                self.assertEqual(
                    restored.execute("SELECT image FROM version_component WHERE id = 'version-b-tei'").fetchone()[0],
                    VERSION_B_IMAGE,
                )
            finally:
                restored.close()

    def test_export_reads_tables_without_domain_specific_validation(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source_path = Path(directory) / "source.db"
            source = connect(source_path)
            self.populate(source)
            source.execute("DELETE FROM version_component_device WHERE component_id = 'version-b-tei'")
            source.commit()
            source.row_factory = sqlite3.Row
            try:
                args = argparse.Namespace(database=str(source_path), output=str(Path(directory) / "baseline.sql"), project_id=None)
                data = exporter.selected_baseline(source, args)
                self.assertEqual(data["version_component_device"], [])
            finally:
                source.close()

    def test_existing_output_requires_explicit_replace(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "baseline.sql"
            output.write_text("previous", encoding="utf-8")
            with self.assertRaisesRegex(exporter.ExportError, "refusing to overwrite"):
                exporter.write_output(output, "next", replace=False)
            exporter.write_output(output, "next", replace=True)
            self.assertEqual(output.read_text(encoding="utf-8"), "next")


if __name__ == "__main__":
    unittest.main()
