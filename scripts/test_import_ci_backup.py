"""End-to-end tests for the legacy CI backup importer."""

from __future__ import annotations

import argparse
import json
import sqlite3
import sys
import tempfile
import unittest
from pathlib import Path

from cryptography.fernet import Fernet, InvalidToken

sys.path.insert(0, str(Path(__file__).resolve().parent))
import import_ci_backup as importer


SOURCE_SCHEMA = """
CREATE TABLE project (id TEXT PRIMARY KEY);
CREATE TABLE credential (id TEXT PRIMARY KEY, name TEXT NOT NULL, type TEXT NOT NULL, encrypted_data TEXT NOT NULL, created_at TEXT NOT NULL, project_id TEXT);
CREATE TABLE repository (id TEXT PRIMARY KEY, name TEXT NOT NULL, code TEXT NOT NULL, repository_type TEXT NOT NULL, repository_url TEXT NOT NULL, git_credential_id TEXT, variable_overrides TEXT NOT NULL, default_branch TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL, project_id TEXT);
CREATE TABLE pipeline_template (id TEXT PRIMARY KEY, name TEXT NOT NULL, description TEXT NOT NULL, variable_declarations TEXT NOT NULL, version INTEGER NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL, project_id TEXT);
CREATE TABLE pipeline_template_stage (id TEXT PRIMARY KEY, template_id TEXT NOT NULL, stage_id TEXT NOT NULL, stage_name TEXT NOT NULL, stage_version INTEGER NOT NULL, depends_on TEXT NOT NULL, sort_order INTEGER NOT NULL);
CREATE TABLE pipeline_stage (id TEXT PRIMARY KEY, name TEXT NOT NULL, image TEXT NOT NULL, script TEXT NOT NULL, artifacts TEXT, description TEXT NOT NULL, version INTEGER NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL, project_id TEXT);
CREATE TABLE pipeline_snapshot (id TEXT PRIMARY KEY, template_id TEXT NOT NULL, version INTEGER NOT NULL, stages_snapshot TEXT NOT NULL, variables_snapshot TEXT NOT NULL, created_at TEXT NOT NULL, project_id TEXT);
CREATE TABLE pipeline_run (id TEXT PRIMARY KEY, repository_id TEXT NOT NULL, repository_name TEXT NOT NULL, snapshot_id TEXT NOT NULL, template_id TEXT NOT NULL, template_name TEXT NOT NULL, template_version INTEGER NOT NULL, trigger TEXT NOT NULL, trigger_ref TEXT NOT NULL, variables_snapshot TEXT NOT NULL, status TEXT NOT NULL, retry_of TEXT, started_at TEXT, finished_at TEXT, error_message TEXT, created_at TEXT NOT NULL, project_id TEXT);
CREATE TABLE pipeline_stage_run (id TEXT PRIMARY KEY, pipeline_run_id TEXT NOT NULL, stage_id TEXT NOT NULL, stage_name TEXT NOT NULL, status TEXT NOT NULL, started_at TEXT, finished_at TEXT, exit_code INTEGER, error_message TEXT);
CREATE TABLE pipeline_stage_build_version_binding (pipeline_stage_id TEXT PRIMARY KEY);
CREATE TABLE pipeline_run_build_version_binding (pipeline_run_id TEXT PRIMARY KEY);
CREATE TABLE artifact (id TEXT PRIMARY KEY, pipeline_run_id TEXT NOT NULL, pipeline_stage_id TEXT NOT NULL, stage_name TEXT NOT NULL, collector TEXT NOT NULL, name TEXT NOT NULL, location TEXT, value TEXT, value_format TEXT, image_ref TEXT, local_image_sha256 TEXT, source_artifact_id TEXT, created_at TEXT NOT NULL, repository_id TEXT NOT NULL, repository_name TEXT NOT NULL, template_id TEXT NOT NULL, template_name TEXT NOT NULL, project_id TEXT);
CREATE TABLE repository_webhook (id TEXT PRIMARY KEY);
"""

TARGET_SCHEMA = """
CREATE TABLE project (id TEXT PRIMARY KEY);
CREATE TABLE credential (id TEXT PRIMARY KEY, name TEXT NOT NULL, type TEXT NOT NULL, encrypted_data TEXT NOT NULL, created_at TEXT NOT NULL, project_id TEXT);
CREATE TABLE repository (id TEXT PRIMARY KEY, name TEXT NOT NULL, code TEXT NOT NULL UNIQUE, repository_type TEXT NOT NULL, repository_url TEXT NOT NULL, git_credential_id TEXT, variable_overrides TEXT NOT NULL, default_branch TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL, project_id TEXT);
CREATE TABLE pipeline (id TEXT PRIMARY KEY, project_id TEXT, kind TEXT NOT NULL, source_pipeline_id TEXT, source_template_name TEXT, source_template_version INTEGER, application_id TEXT, application_name TEXT, repository_id TEXT, repository_name TEXT, version_fork_strategy TEXT, fixed_version_id TEXT, fixed_version_label TEXT, name TEXT NOT NULL, description TEXT NOT NULL, variable_declarations TEXT NOT NULL, version INTEGER NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL, UNIQUE(project_id, name));
CREATE TABLE pipeline_stage (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, kind TEXT NOT NULL, pipeline_id TEXT, name TEXT NOT NULL, image TEXT NOT NULL, script TEXT NOT NULL, description TEXT NOT NULL, version INTEGER, source_template_stage_id TEXT, source_template_stage_name TEXT, source_template_stage_version INTEGER, source_template_stage_description TEXT, artifacts TEXT, depends_on TEXT, sort_order INTEGER, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE pipeline_stage_reference (id TEXT PRIMARY KEY, pipeline_id TEXT NOT NULL, source_template_stage_id TEXT NOT NULL, source_template_stage_name TEXT NOT NULL, source_template_stage_version INTEGER NOT NULL, source_template_stage_description TEXT NOT NULL, name TEXT NOT NULL, image TEXT NOT NULL, script TEXT NOT NULL, description TEXT NOT NULL, artifacts TEXT NOT NULL, depends_on TEXT NOT NULL, sort_order INTEGER NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE pipeline_snapshot (id TEXT PRIMARY KEY, project_id TEXT, pipeline_id TEXT NOT NULL, pipeline_name TEXT NOT NULL, pipeline_version INTEGER NOT NULL, source_pipeline_id TEXT NOT NULL, source_template_name TEXT NOT NULL, source_template_version INTEGER NOT NULL, application_id TEXT, application_name TEXT, repository_id TEXT NOT NULL, repository_name TEXT NOT NULL, version_fork_strategy TEXT, fixed_version_id TEXT, fixed_version_label TEXT, stages_snapshot TEXT NOT NULL, variables_snapshot TEXT NOT NULL, created_at TEXT NOT NULL, UNIQUE(pipeline_id, pipeline_version));
CREATE TABLE pipeline_run (id TEXT PRIMARY KEY, project_id TEXT, repository_id TEXT NOT NULL, repository_name TEXT NOT NULL, snapshot_id TEXT NOT NULL, pipeline_id TEXT NOT NULL, pipeline_name TEXT NOT NULL, pipeline_version INTEGER NOT NULL, trigger TEXT NOT NULL, repository_ref TEXT NOT NULL, variables_snapshot TEXT NOT NULL, status TEXT NOT NULL, retry_of TEXT, started_at TEXT, finished_at TEXT, error_message TEXT, created_at TEXT NOT NULL, FOREIGN KEY(repository_id) REFERENCES repository(id), FOREIGN KEY(snapshot_id) REFERENCES pipeline_snapshot(id), FOREIGN KEY(retry_of) REFERENCES pipeline_run(id));
CREATE TABLE pipeline_stage_run (id TEXT PRIMARY KEY, pipeline_run_id TEXT NOT NULL, stage_id TEXT NOT NULL, stage_name TEXT NOT NULL, status TEXT NOT NULL, started_at TEXT, finished_at TEXT, exit_code INTEGER, error_message TEXT, FOREIGN KEY(pipeline_run_id) REFERENCES pipeline_run(id));
CREATE TABLE pipeline_run_version_binding (pipeline_run_id TEXT PRIMARY KEY);
CREATE TABLE artifact (id TEXT PRIMARY KEY, project_id TEXT, pipeline_run_id TEXT NOT NULL, repository_id TEXT NOT NULL, repository_name TEXT NOT NULL, pipeline_id TEXT NOT NULL, pipeline_name TEXT NOT NULL, pipeline_stage_id TEXT NOT NULL, stage_name TEXT NOT NULL, collector TEXT NOT NULL, name TEXT NOT NULL, location TEXT, value TEXT, value_format TEXT, image_ref TEXT, local_image_sha256 TEXT, source_artifact_id TEXT, created_at TEXT NOT NULL, FOREIGN KEY(pipeline_run_id) REFERENCES pipeline_run(id), FOREIGN KEY(source_artifact_id) REFERENCES artifact(id));
"""


class LegacyCIImporterTests(unittest.TestCase):
    def write_env(self, path: Path, key: bytes) -> None:
        path.write_text(f"POMELO_ORBIT_JWT__SECRET_KEY={key.decode('ascii')}\n", encoding="utf-8")

    def populate_source(self, path: Path, source_key: bytes) -> None:
        connection = sqlite3.connect(path)
        try:
            connection.executescript(SOURCE_SCHEMA)
            connection.execute("INSERT INTO project VALUES ('project')")
            connection.execute(
                "INSERT INTO credential VALUES ('credential', 'git', 'github_token', ?, '2026-08-01', 'project')",
                (Fernet(source_key).encrypt(b'test-token').decode("ascii"),),
            )
            connection.execute(
                "INSERT INTO repository VALUES ('repository', 'Repository', 'repository', 'remote_git', 'https://example.test/repository.git', 'credential', '[]', 'main', '2026-08-01', '2026-08-01', 'project')"
            )
            connection.execute(
                "INSERT INTO pipeline_template VALUES ('template', 'Build', 'Build a container', ?, 2, '2026-08-01', '2026-08-01', 'project')",
                ('[{"name":"working_dir","default":".","value":null,"secret":false,"source":"template_custom","editable":true}]',),
            )
            stages = (
                ('clone', 'git clone', 'alpine/git', 'git clone {{ repository_url }}', None, 'Clone', 2),
                ('build', 'docker build', 'docker:29', 'docker build {{ repository_code }}', '[{"type":"docker_image","name":"image","path":"example/image:latest"}]', 'Build', 3),
            )
            for stage in stages:
                connection.execute(
                    "INSERT INTO pipeline_stage VALUES (?, ?, ?, ?, ?, ?, ?, '2026-08-01', '2026-08-01', 'project')", stage
                )
            connection.executemany(
                "INSERT INTO pipeline_template_stage VALUES (?, 'template', ?, ?, ?, ?, ?)",
                (
                    ('reference-clone', 'clone', 'git clone', 2, '[]', 0),
                    ('reference-build', 'build', 'docker build', 3, '["clone"]', 1),
                ),
            )
            snapshot_stages = [
                {'id': 'clone', 'name': 'git clone', 'image': 'alpine/git', 'version': 2, 'depends_on': [], 'script': 'git clone {{ repository_url }}', 'artifacts': None},
                {'id': 'build', 'name': 'docker build', 'image': 'docker:29', 'version': 3, 'depends_on': ['clone'], 'script': 'docker build {{ repository_code }}', 'artifacts': [{'type': 'docker_image', 'name': 'image', 'path': 'example/image:latest'}]},
            ]
            variables = [{'name': 'working_dir', 'description': '', 'default': '.', 'value': None, 'secret': False, 'source': 'template_custom', 'editable': True}]
            connection.execute(
                "INSERT INTO pipeline_snapshot VALUES ('snapshot', 'template', 2, ?, ?, '2026-08-01', 'project')",
                (json.dumps(snapshot_stages), json.dumps(variables)),
            )
            connection.execute(
                "INSERT INTO pipeline_run VALUES ('run', 'repository', 'Repository', 'snapshot', 'template', 'Build', 2, 'manual', 'main', ?, 'ran_to_completion', NULL, '2026-08-01', '2026-08-01', NULL, '2026-08-01', 'project')",
                (json.dumps(variables),),
            )
            connection.executemany(
                "INSERT INTO pipeline_stage_run VALUES (?, 'run', ?, ?, 'ran_to_completion', '2026-08-01', '2026-08-01', 0, NULL)",
                (('stage-run-clone', 'clone', 'git clone'), ('stage-run-build', 'build', 'docker build')),
            )
            connection.execute(
                "INSERT INTO artifact VALUES ('artifact', 'run', '', 'docker build', 'docker_image', 'image', NULL, NULL, NULL, 'example/image:latest', 'sha256:test', NULL, '2026-08-01', 'repository', 'Repository', 'template', 'Build', 'project')"
            )
            connection.commit()
        finally:
            connection.close()

    def populate_target(self, path: Path) -> None:
        connection = sqlite3.connect(path)
        try:
            connection.executescript(TARGET_SCHEMA)
            connection.execute("INSERT INTO project VALUES ('project')")
            connection.execute(
                "INSERT INTO pipeline_stage VALUES ('existing-stage', 'project', 'template', NULL, 'git clone', 'alpine/git', 'true', 'Existing', 1, NULL, NULL, NULL, NULL, '[]', NULL, NULL, '2026-08-01', '2026-08-01')"
            )
            connection.commit()
        finally:
            connection.close()

    def arguments(self, source: Path, source_env: Path, target: Path, target_env: Path) -> argparse.Namespace:
        return argparse.Namespace(source=source, source_env=source_env, database=target, target_env=target_env)

    def test_import_converts_legacy_data_and_is_idempotent(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source, target = root / 'source.db', root / 'target.db'
            source_env, target_env = root / 'source.env', root / 'target.env'
            source_key, target_key = Fernet.generate_key(), Fernet.generate_key()
            self.write_env(source_env, source_key)
            self.write_env(target_env, target_key)
            self.populate_source(source, source_key)
            self.populate_target(target)

            status, counts = importer.import_command(self.arguments(source, source_env, target, target_env))
            self.assertEqual(status, 'imported')
            self.assertEqual(counts['pipeline_runs'], 1)
            self.assertEqual(counts['application_pipelines'], 1)

            connection = sqlite3.connect(target)
            connection.row_factory = sqlite3.Row
            try:
                encrypted = connection.execute("SELECT encrypted_data FROM credential WHERE id = 'credential'").fetchone()[0]
                self.assertEqual(Fernet(target_key).decrypt(encrypted.encode('ascii')), b'test-token')
                with self.assertRaises(InvalidToken):
                    Fernet(source_key).decrypt(encrypted.encode('ascii'))
                self.assertEqual(connection.execute("SELECT COUNT(*) FROM pipeline WHERE kind = 'application'").fetchone()[0], 1)
                self.assertEqual(connection.execute("SELECT COUNT(*) FROM pipeline_run WHERE pipeline_id IN (SELECT id FROM pipeline WHERE kind = 'application')").fetchone()[0], 1)
                self.assertEqual(connection.execute("SELECT COUNT(*) FROM pipeline_stage WHERE kind = 'template' AND name LIKE 'git clone (backup %)' ").fetchone()[0], 1)
                self.assertEqual(connection.execute("SELECT COUNT(*) FROM pipeline_stage WHERE artifacts LIKE '%\"type\"%' ").fetchone()[0], 0)
                self.assertEqual(connection.execute("PRAGMA integrity_check").fetchone()[0], 'ok')
                self.assertEqual(list(connection.execute("PRAGMA foreign_key_check")), [])
            finally:
                connection.close()

            status, repeated = importer.import_command(self.arguments(source, source_env, target, target_env))
            self.assertEqual(status, 'already imported')
            self.assertEqual(repeated, counts)
            self.assertEqual(importer.verify_command(self.arguments(source, source_env, target, target_env)), counts)


if __name__ == '__main__':
    unittest.main()
