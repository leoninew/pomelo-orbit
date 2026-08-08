from __future__ import annotations

import importlib.util
import json
import sqlite3
import tempfile
import unittest
from pathlib import Path


SCRIPT_PATH = Path(__file__).with_name("reconcile_legacy_service_deploying.py")
SPEC = importlib.util.spec_from_file_location("reconcile_legacy_service_deploying", SCRIPT_PATH)
assert SPEC is not None and SPEC.loader is not None
reconciler = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(reconciler)


class ReconcileLegacyServiceDeployingTest(unittest.TestCase):
    def create_database(self, root: Path) -> Path:
        path = root / "orbit.db"
        connection = sqlite3.connect(path)
        connection.executescript(
            """
            CREATE TABLE service (id TEXT PRIMARY KEY, status TEXT NOT NULL, updated_at TEXT);
            INSERT INTO service (id, status) VALUES ('service-deploying', 'deploying');
            INSERT INTO service (id, status) VALUES ('service-running', 'running');
            """
        )
        connection.close()
        return path

    def write_mapping(self, root: Path, mapping: dict[str, str]) -> Path:
        path = root / "mapping.json"
        path.write_text(json.dumps(mapping), encoding="utf-8")
        return path

    def test_dry_run_preserves_database(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            database = self.create_database(root)
            mapping = self.write_mapping(root, {"service-deploying": "stopped"})

            self.assertEqual(0, reconciler.main(["--database", str(database), "--mapping", str(mapping)]))

            connection = sqlite3.connect(database)
            try:
                status = connection.execute("SELECT status FROM service WHERE id = 'service-deploying'").fetchone()[0]
            finally:
                connection.close()
            self.assertEqual("deploying", status)

    def test_apply_creates_backup_and_updates_only_legacy_status(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            database = self.create_database(root)
            mapping = self.write_mapping(root, {"service-deploying": "faulted"})
            backup = root / "backup.db"

            self.assertEqual(0, reconciler.main(["--database", str(database), "--mapping", str(mapping), "--apply", "--backup", str(backup)]))

            restored = sqlite3.connect(database)
            self.assertEqual("faulted", restored.execute("SELECT status FROM service WHERE id = 'service-deploying'").fetchone()[0])
            self.assertEqual("running", restored.execute("SELECT status FROM service WHERE id = 'service-running'").fetchone()[0])
            restored.close()
            self.assertTrue(backup.is_file())
            backup_connection = sqlite3.connect(backup)
            try:
                backup_status = backup_connection.execute("SELECT status FROM service WHERE id = 'service-deploying'").fetchone()[0]
            finally:
                backup_connection.close()
            self.assertEqual("deploying", backup_status)

    def test_apply_rejects_non_legacy_source_status(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            database = self.create_database(root)
            mapping = self.write_mapping(root, {"service-deploying": "stopped", "service-running": "stopped"})
            backup = root / "backup.db"

            self.assertEqual(1, reconciler.main(["--database", str(database), "--mapping", str(mapping), "--apply", "--backup", str(backup)]))
            self.assertFalse(backup.exists())


if __name__ == "__main__":
    unittest.main()
