"""Focused round-trip test for the deterministic RAGFlow TEI SQL exporter."""

from __future__ import annotations

import argparse
import json
import sqlite3
import tempfile
import unittest
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
import export_ragflow_tei_baseline as exporter


SCHEMA = """
CREATE TABLE project (id TEXT PRIMARY KEY, name TEXT NOT NULL);
CREATE TABLE application (id TEXT PRIMARY KEY, project_id TEXT, code TEXT NOT NULL, name TEXT NOT NULL);
CREATE TABLE gateway_config (application_id TEXT PRIMARY KEY, base_domain TEXT NOT NULL);
CREATE TABLE version (id TEXT PRIMARY KEY, application_id TEXT NOT NULL, label TEXT NOT NULL, note TEXT);
CREATE TABLE version_component (id TEXT PRIMARY KEY, version_id TEXT NOT NULL, name TEXT NOT NULL, image TEXT NOT NULL, pull_policy TEXT NOT NULL CHECK (pull_policy IN ('always', 'missing', 'never')), restart_policy TEXT, command_json TEXT);
CREATE TABLE version_component_dependency (component_id TEXT NOT NULL, position INTEGER NOT NULL, name TEXT NOT NULL);
CREATE TABLE version_component_env (component_id TEXT NOT NULL, position INTEGER NOT NULL, key TEXT NOT NULL, value TEXT NOT NULL);
CREATE TABLE version_component_healthcheck (component_id TEXT PRIMARY KEY, test TEXT NOT NULL);
CREATE TABLE version_component_mount (component_id TEXT NOT NULL, position INTEGER NOT NULL, source TEXT NOT NULL);
CREATE TABLE version_component_port (component_id TEXT NOT NULL, position INTEGER NOT NULL, container_port INTEGER NOT NULL);
CREATE TABLE version_component_resource (component_id TEXT PRIMARY KEY, reservation_memory TEXT);
CREATE TABLE version_component_tmpfs (component_id TEXT NOT NULL, position INTEGER NOT NULL, target TEXT NOT NULL);
CREATE TABLE version_component_ulimit (component_id TEXT NOT NULL, name TEXT NOT NULL, soft INTEGER NOT NULL);
CREATE TABLE version_component_device (component_id TEXT NOT NULL, position INTEGER NOT NULL, driver TEXT NOT NULL, device_count TEXT NOT NULL, capabilities_json TEXT NOT NULL);
CREATE TABLE service (id TEXT PRIMARY KEY, application_id TEXT NOT NULL, instance_key TEXT NOT NULL, version_id TEXT NOT NULL, runtime_config_json TEXT NOT NULL, status TEXT NOT NULL);
CREATE TABLE service_expose (id TEXT PRIMARY KEY, service_id TEXT NOT NULL, component_name TEXT NOT NULL, protocol TEXT NOT NULL, container_port INTEGER NOT NULL, access TEXT NOT NULL, listen_port INTEGER NOT NULL);
"""


def connect(path: Path) -> sqlite3.Connection:
    connection = sqlite3.connect(path)
    connection.executescript(SCHEMA)
    return connection


class BaselineExportTests(unittest.TestCase):
    def populate(self, connection: sqlite3.Connection) -> None:
        connection.execute("INSERT INTO project VALUES ('project', 'Project')")
        connection.executemany(
            "INSERT INTO application VALUES (?, 'project', ?, ?)",
            (("gateway", "traefik", "Gateway"), ("ragflow", "ragflow", "RAGFlow")),
        )
        connection.execute("INSERT INTO gateway_config VALUES ('gateway', 'example.test')")
        connection.executemany(
            "INSERT INTO version VALUES (?, ?, ?, ?)",
            (
                ("gateway-v1", "gateway", "default", None),
                ("cpu-v1", "ragflow", "ragflow-tei-cpu", "CPU baseline"),
                ("gpu-v1", "ragflow", "ragflow-tei-gpu", "GPU runtime unverified; static specification only"),
            ),
        )
        components = [("gateway-traefik", "gateway-v1", "traefik", "traefik:3.6", "missing", None, "[]")]
        for version_id in ("cpu-v1", "gpu-v1"):
            tei_image = exporter.CPU_TEI_IMAGE if version_id == "cpu-v1" else exporter.GPU_TEI_IMAGE
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
                for name in exporter.COMPONENT_NAMES
            )
        connection.executemany("INSERT INTO version_component VALUES (?, ?, ?, ?, ?, ?, ?)", components)
        connection.executemany(
            "INSERT INTO version_component_healthcheck VALUES (?, 'curl -fsS /health')",
            (("cpu-v1-tei",), ("gpu-v1-tei",)),
        )
        connection.executemany(
            "INSERT INTO version_component_resource VALUES (?, '4g')",
            (("cpu-v1-tei",), ("gpu-v1-tei",)),
        )
        connection.execute("INSERT INTO version_component_device VALUES ('gpu-v1-tei', 0, 'nvidia', 'all', '[\"gpu\"]')")
        connection.executemany(
            "INSERT INTO service VALUES (?, ?, 'default', ?, ?, ?)",
            (
                ("gateway-service", "gateway", "gateway-v1", '{\"gateway_secret\":\"do-not-export\"}', "running"),
                ("ragflow-service", "ragflow", "cpu-v1", '{\"mysql_password\":\"do-not-export\"}', "running"),
            ),
        )
        connection.executemany(
            "INSERT INTO service_expose VALUES (?, ?, ?, ?, ?, ?, ?)",
            (
                ("gateway-expose", "gateway-service", "traefik", "http", 80, "local", 8080),
                ("ragflow-expose", "ragflow-service", "ragflow-cpu", "http", 80, "local", 9380),
            ),
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
                services = dict(restored.execute("SELECT application_id, runtime_config_json FROM service"))
                self.assertEqual(services["gateway"], "{}")
                ragflow_runtime_config = json.loads(services["ragflow"])
                self.assertEqual(set(ragflow_runtime_config), set(exporter.RAGFLOW_RUNTIME_CONFIG_KEYS))
                self.assertTrue(all(value for value in ragflow_runtime_config.values()))
                self.assertEqual(
                    restored.execute("SELECT DISTINCT status FROM service").fetchall(), [("stopped",)]
                )
                self.assertEqual(restored.execute("SELECT COUNT(*) FROM version_component_device").fetchone()[0], 1)
                self.assertEqual(restored.execute("SELECT COUNT(*) FROM version_component_healthcheck").fetchone()[0], 2)
                self.assertEqual(restored.execute("SELECT COUNT(*) FROM version_component_resource").fetchone()[0], 2)
                self.assertEqual(
                    restored.execute("SELECT label FROM version WHERE application_id = 'ragflow' ORDER BY label").fetchall(),
                    [("ragflow-tei-cpu",), ("ragflow-tei-gpu",)],
                )
                self.assertEqual(
                    restored.execute("SELECT image FROM version_component WHERE id = 'gpu-v1-tei'").fetchone()[0],
                    exporter.GPU_TEI_IMAGE,
                )
            finally:
                restored.close()

    def test_export_rejects_gpu_without_the_required_device_request(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source_path = Path(directory) / "source.db"
            source = connect(source_path)
            self.populate(source)
            source.execute("DELETE FROM version_component_device WHERE component_id = 'gpu-v1-tei'")
            source.commit()
            source.row_factory = sqlite3.Row
            try:
                args = argparse.Namespace(database=str(source_path), output=str(Path(directory) / "baseline.sql"), project_id=None)
                with self.assertRaisesRegex(exporter.ExportError, "GPU Version must contain exactly one device request"):
                    exporter.selected_baseline(source, args)
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
