"""Focused round-trip test for the continuous-delivery SQL exporter."""

from __future__ import annotations

import sqlite3
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import export_cd_baseline as exporter


SCHEMA = """
CREATE TABLE project (id TEXT PRIMARY KEY, name TEXT NOT NULL);
CREATE TABLE application (id TEXT PRIMARY KEY, project_id TEXT, code TEXT NOT NULL, name TEXT NOT NULL, kind TEXT NOT NULL);
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
CREATE TABLE gateway_config (application_id TEXT PRIMARY KEY, rest_api_url TEXT NOT NULL, base_domain TEXT NOT NULL, default_entrypoint TEXT NOT NULL, tls_mode TEXT NOT NULL);
CREATE TABLE service (id TEXT PRIMARY KEY, application_id TEXT NOT NULL, instance_key TEXT NOT NULL, version_id TEXT NOT NULL, status TEXT NOT NULL);
CREATE TABLE service_env (service_id TEXT NOT NULL, env_key TEXT NOT NULL, value TEXT NOT NULL);
CREATE TABLE service_component (id TEXT PRIMARY KEY, service_id TEXT NOT NULL, source_version_component_id TEXT NOT NULL, component_name TEXT NOT NULL, status TEXT NOT NULL);
CREATE TABLE service_component_env (service_component_id TEXT NOT NULL, env_key TEXT NOT NULL, value TEXT, state TEXT NOT NULL);
CREATE TABLE service_component_mount (id TEXT PRIMARY KEY, service_component_id TEXT NOT NULL, target TEXT NOT NULL, source TEXT, state TEXT NOT NULL);
CREATE TABLE service_component_resource (service_component_id TEXT PRIMARY KEY, limit_cpus TEXT, limit_memory TEXT, reservation_cpus TEXT, reservation_memory TEXT, state TEXT NOT NULL);
CREATE TABLE service_component_endpoint (service_component_id TEXT NOT NULL, name TEXT NOT NULL, mode TEXT, bind_address TEXT, listen_port INTEGER, entrypoint TEXT, path_prefix TEXT, state TEXT NOT NULL);
CREATE TABLE route (id TEXT PRIMARY KEY, project_id TEXT, name TEXT NOT NULL, domain TEXT NOT NULL);
CREATE TABLE deployment (id TEXT PRIMARY KEY, application_id TEXT, status TEXT NOT NULL);
"""


def connect(path: Path) -> sqlite3.Connection:
    connection = sqlite3.connect(path)
    connection.executescript(SCHEMA)
    return connection


class ContinuousDeliveryExportTests(unittest.TestCase):
    def populate(self, connection: sqlite3.Connection) -> None:
        connection.executemany(
            "INSERT INTO project VALUES (?, ?)",
            (("project", "Project"), ("gateway-project", "Gateway Project")),
        )
        connection.executemany(
            "INSERT INTO application VALUES (?, ?, ?, ?, ?)",
            (
                ("gateway", "gateway-project", "traefik", "Gateway", "gateway"),
                ("ragflow", "project", "ragflow", "RAGFlow", "standard"),
            ),
        )
        connection.execute(
            "INSERT INTO gateway_config VALUES ('gateway', 'http://127.0.0.1:8080', 'example.test', 'web', 'none')"
        )
        connection.executemany(
            "INSERT INTO version VALUES (?, ?, ?, ?)",
            (
                ("gateway-v1", "gateway", "default", None),
                ("version-a", "ragflow", "version-a", "baseline A"),
            ),
        )
        connection.executemany(
            "INSERT INTO version_component VALUES (?, ?, ?, ?, ?, ?, ?)",
            (
                ("gateway-traefik", "gateway-v1", "traefik", "traefik:3.6", "missing", None, "[]"),
                ("version-a-ragflow", "version-a", "ragflow", "example/ragflow:1", "missing", "unless-stopped", "[]"),
            ),
        )
        connection.execute("INSERT INTO version_component_env VALUES ('version-a-ragflow', 'IMAGE_ENV', 'image-value', 0)")
        connection.execute("INSERT INTO version_component_healthcheck VALUES ('version-a-ragflow', 'curl -fsS /health')")
        connection.execute("INSERT INTO version_component_resource VALUES ('version-a-ragflow', '4g')")
        connection.execute("INSERT INTO version_component_device VALUES ('version-a-ragflow', 0, 'accelerator', 'all', '[\"compute\"]')")
        connection.execute(
            "INSERT INTO version_component_endpoint VALUES ('version-a-ragflow', 'http', 'http', 80, 'internal', NULL, NULL, NULL, NULL, 0)"
        )
        connection.executemany(
            "INSERT INTO service VALUES (?, ?, 'default', ?, 'running')",
            (("gateway-service", "gateway", "gateway-v1"), ("ragflow-service", "ragflow", "version-a")),
        )
        connection.executemany(
            "INSERT INTO service_env VALUES (?, ?, ?)",
            (("gateway-service", "GATEWAY_VALUE", "gateway-value"), ("ragflow-service", "RUNTIME_VALUE", "runtime-value")),
        )
        connection.executemany(
            "INSERT INTO service_component VALUES (?, ?, ?, ?, 'active')",
            (
                ("gateway-component", "gateway-service", "gateway-traefik", "traefik"),
                ("service-ragflow", "ragflow-service", "version-a-ragflow", "ragflow"),
            ),
        )
        connection.execute(
            "INSERT INTO service_component_endpoint VALUES ('service-ragflow', 'http', 'local', '127.0.0.1', 9380, NULL, NULL, 'override')"
        )
        connection.execute(
            "INSERT INTO service_component_env VALUES ('service-ragflow', 'MYSQL_PASSWORD', 'exported-value', 'override')"
        )
        connection.execute(
            "INSERT INTO service_component_mount VALUES ('mount-overlay', 'service-ragflow', '/data', 'runtime-data', 'override')"
        )
        connection.execute("INSERT INTO route VALUES ('route', 'project', 'RAGFlow', 'ragflow.example.test')")
        connection.execute("INSERT INTO deployment VALUES ('deployment', 'ragflow', 'succeeded')")
        connection.commit()

    def test_export_round_trips_all_selected_tables_without_deployments(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source_path = root / "source.db"
            source = connect(source_path)
            self.populate(source)
            source.close()

            connection = exporter.connect_read_only(source_path)
            try:
                exporter.ensure_database_integrity(connection)
                rendered = exporter.render_sql(connection, exporter.exported_table_rows(connection))
            finally:
                connection.close()

            restored_path = root / "restored.db"
            restored = connect(restored_path)
            try:
                restored.executescript(rendered)
                self.assertNotIn('INSERT INTO "deployment"', rendered)
                self.assertNotIn("INSERT OR IGNORE", rendered)
                self.assertEqual(restored.execute("SELECT COUNT(*) FROM gateway_config").fetchone()[0], 1)
                self.assertEqual(restored.execute("SELECT COUNT(*) FROM application WHERE kind = 'gateway'").fetchone()[0], 1)
                self.assertEqual(restored.execute("SELECT COUNT(*) FROM route").fetchone()[0], 1)
                self.assertEqual(restored.execute("SELECT COUNT(*) FROM deployment").fetchone()[0], 0)
                self.assertEqual(
                    restored.execute("SELECT env_key, value FROM service_env ORDER BY service_id").fetchall(),
                    [("GATEWAY_VALUE", "gateway-value"), ("RUNTIME_VALUE", "runtime-value")],
                )
                self.assertEqual(restored.execute("SELECT COUNT(*) FROM service_component_env").fetchone()[0], 1)
                self.assertEqual(
                    restored.execute("SELECT value FROM service_component_env").fetchone()[0],
                    "exported-value",
                )
                self.assertEqual(
                    restored.execute("SELECT value FROM version_component_env").fetchone()[0],
                    "image-value",
                )
                self.assertEqual(restored.execute("SELECT COUNT(*) FROM version_component_device").fetchone()[0], 1)
            finally:
                restored.close()

    def test_export_reads_every_configured_table_without_business_filters(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source_path = Path(directory) / "source.db"
            source = connect(source_path)
            self.populate(source)
            source.row_factory = sqlite3.Row
            try:
                data = exporter.exported_table_rows(source)
                self.assertEqual(tuple(data), exporter.TABLES)
                self.assertEqual(data["gateway_config"][0]["application_id"], "gateway")
                self.assertEqual(data["route"][0]["id"], "route")
                self.assertEqual(len(data["service_env"]), 2)
                self.assertNotIn("deployment", data)
            finally:
                source.close()

    def test_existing_output_requires_explicit_replacement(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "baseline.sql"
            output.write_text("previous", encoding="utf-8")
            with self.assertRaises(exporter.ExportError):
                exporter.write_output(output, "next", replace=False)
            exporter.write_output(output, "next", replace=True)
            self.assertEqual(output.read_text(encoding="utf-8"), "next")


if __name__ == "__main__":
    unittest.main()
