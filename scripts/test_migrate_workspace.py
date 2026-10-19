"""Run with python -m unittest discover -s scripts -p test_migrate_workspace.py.

Set ORBIT_MIGRATION_TEST_CONTAINER to an isolated PostgreSQL container for SQL tests.
Set ORBIT_MIGRATION_TEST_CYGWIN to a Cygwin installation root for shell path tests.
"""

from __future__ import annotations

import base64
import copy
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import Mock, patch

import migrate_workspace as migration


def snapshot() -> dict:
    environment = {
        "id": "env-1", "code": "project-1", "project_id": "project-1", "project_name": "Project",
        "workspace_root": "D:\\old", "target_revision": 4, "target_type": "ssh",
        "ssh_credential_id": "key-1", "ssh_credential_revision": 2,
    }
    service = {"id": "svc-1", "project_id": "project-1", "application_id": "app-1",
               "application_kind": "standard", "code": "web", "version_id": "v-1",
               "deployment_directory": "D:/old/deployment/web", "directory_target_revision": 4,
               "runtime_directory": "D:/old/deployment/web", "runtime_target_revision": 4,
               "status": "running"}
    current_run = {"id": "run-1", "environment_id": "env-1", "environment_target_type": "ssh",
                   "environment_target_revision": 4, "ssh_credential_id": "key-1",
                   "ssh_credential_revision": 2, "status": "ran_to_completion"}
    return {"environment": environment, "services": [service], "mounts": [],
            "runs": [current_run], "artifacts": [], "routes": []}



class CommandTests(unittest.TestCase):
    def test_apply_previews_and_no_dry_run_executes_with_an_explicit_target(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "scripts").mkdir()
            runtime = Mock()
            runtime.call.return_value = {"profile": "development", "database_url": "postgres://u:secret@localhost/orbit"}
            plan_path = root / "data" / "workspace-migration" / "run" / "plan.json"
            with patch.object(migration, "__file__", str(root / "scripts" / "migrate_workspace.py")), \
                    patch.object(migration, "Runtime", return_value=runtime), \
                    patch.object(migration, "Database"), \
                    patch.object(migration, "secure_directory", side_effect=lambda path: path.mkdir(parents=True, exist_ok=True)), \
                    patch.object(migration, "plan_migration", return_value=(plan_path, {"blockers": []})) as assess, \
                    patch.object(migration, "apply_migration") as execute:
                self.assertEqual(migration.main(["apply", "--target", "E:/orbit"]), 0)
                execute.assert_not_called()
                self.assertEqual(migration.main(["apply", "--target", "E:/orbit", "--no-dry-run"]), 0)
                execute.assert_called_once()
                self.assertEqual(assess.call_count, 2)
                self.assertIsNone(assess.call_args.args[0].environment)
                self.assertEqual(assess.call_args.args[0].target, "E:/orbit")

    def test_missing_or_empty_target_is_rejected_before_loading_configuration(self):
        for argv in (["apply"], ["apply", "--no-dry-run"], ["apply", "--target", " "]):
            with self.subTest(argv=argv), patch("sys.stderr"), patch.object(migration, "Runtime") as runtime:
                with self.assertRaises(SystemExit) as stopped:
                    migration.main(argv)
                self.assertEqual(stopped.exception.code, 2)
                runtime.assert_not_called()

    def test_environment_is_selected_from_the_current_workspace_and_host(self):
        local = {**snapshot()["environment"], "target_type": "local", "workspace_root": "D:/current"}
        remote = {**local, "id": "remote", "code": "remote", "target_type": "ssh", "host": "remote.invalid"}
        db = Mock()
        db.query.return_value = [remote, local]
        self.assertEqual(migration.select_environment(db, {"workspace_root": "D:/current"}, None)["id"], local["id"])
        self.assertEqual(migration.select_environment(db, {"workspace_root": "D:/current"}, "remote")["id"], "remote")

    def test_pending_execution_is_found_without_a_plan_argument(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            bundle = root / "run"
            bundle.mkdir()
            migration.write_json(root / "active.json", {"plan": str(bundle / "plan.json")})
            target = root / "destination"
            migration.write_json(bundle / "plan.json", {"environment_id": "env-1", "project_code": "project-1", "target": str(target),
                                                       "connection": {"environment": {"target_type": "local"}}, "python": sys.executable})
            migration.write_json(bundle / "state.json", {"steps": {"copy-workspace": "done"}})
            args = SimpleNamespace(environment=None, target=str(target))
            self.assertEqual(migration.pending_plan(root, args, Mock()), bundle / "plan.json")
            args.target = str(root / "another-destination")
            with self.assertRaisesRegex(migration.MigrationError, "different destination"):
                migration.pending_plan(root, args, Mock())
            migration.write_json(bundle / "state.json", {"steps": {"preflight": "started"}})
            self.assertIsNone(migration.pending_plan(root, args, Mock()))

    def test_completed_plan_does_not_block_a_different_environment_or_destination(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            bundle = root / "run"
            bundle.mkdir()
            path = bundle / "plan.json"
            migration.write_json(root / "active.json", {"plan": str(path)})
            migration.write_json(path, {"environment_id": "old-environment", "target": "old-target"})
            migration.write_json(bundle / "state.json", {"steps": {"verify": "done", "release-workspace": "done"}})
            args = SimpleNamespace(environment="another-environment", target="another-destination")
            self.assertIsNone(migration.pending_plan(root, args, Mock()))
            self.assertFalse((root / "active.json").exists())
            self.assertTrue(path.exists())


class RemoteAgentTests(unittest.TestCase):
    def test_the_shipped_agent_reads_files_using_the_target_interpreter(self):
        class RemoteRuntime:
            def call(self, action, **values):
                self.request = values
                result = subprocess.run([sys.executable, *values["argv"][1:]], input=base64.b64decode(values["stdin"]),
                                        capture_output=True, check=False)
                if result.returncode:
                    raise AssertionError(result.stderr.decode(errors="replace"))
                return {"exit_code": result.returncode, "stdout": result.stdout.decode(), "fingerprint": "SHA256:reviewed"}
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / "data"
            source.mkdir()
            (source / "marker").write_text("business-data", encoding="utf-8")
            runtime = RemoteRuntime()
            host = migration.Host(runtime, {"environment": {"target_type": "ssh"}}, python=sys.executable)
            paths = host.call("paths", source=str(source), target=str(root / "destination"))
            self.assertEqual(paths["source"], str(source.resolve()))
            self.assertEqual(host.call("read", path=str(source / "marker")), "business-data")
            self.assertEqual(host.connection["environment"]["host_key_fingerprint"], "SHA256:reviewed")
            alias = "~/.pomelo-orbit/deployment/web"
            with patch.object(migration.os.path, "expanduser", side_effect=AssertionError("Must expand on the data host")):
                expanded = host.call("expand-home-paths", paths=[alias, "./data", "postgres-data", None])
            self.assertEqual(expanded, {alias: str((Path.home() / ".pomelo-orbit/deployment/web").resolve())})

    def test_linux_ssh_paths_are_never_converted_by_the_windows_caller(self):
        environment = {"target_type": "ssh", "platform": "linux"}
        with patch.dict(os.environ, {"ORBIT_MIGRATION_CYGPATH": "should-not-run"}), patch.object(migration, "run") as run:
            self.assertEqual(migration.normalize_target_argument("/srv/orbit", environment), "/srv/orbit")
            run.assert_not_called()


@unittest.skipUnless(os.name == "nt" and os.environ.get("ORBIT_MIGRATION_TEST_CYGWIN"),
                     "Set ORBIT_MIGRATION_TEST_CYGWIN to a Cygwin installation root")
class CygwinEntryTests(unittest.TestCase):
    def test_both_apply_forms_and_remote_paths_reach_native_python(self):
        cygwin = Path(os.environ["ORBIT_MIGRATION_TEST_CYGWIN"]) / "bin"
        def posix(path):
            return subprocess.check_output([str(cygwin / "cygpath.exe"), "-au", "--", str(path)], text=True).strip()
        with tempfile.TemporaryDirectory(prefix="orbit shell ") as directory:
            root = Path(directory)
            scripts = root / "repo space" / "scripts"
            scripts.mkdir(parents=True)
            shutil.copy2(Path(__file__).with_name("migrate_workspace.sh"), scripts / "migrate_workspace.sh")
            (scripts / "migrate_workspace.py").write_text(
                "import json, os, sys\nfrom pathlib import Path\n"
                "Path(os.environ['MIGRATION_RESULT']).write_text(json.dumps({'argv':sys.argv[1:],"
                "'profile':os.environ['POMELO_ORBIT_APP__ENV'],'caller':os.environ['ORBIT_MIGRATION_CALLER_CWD'],"
                "'cygpath':os.environ['ORBIT_MIGRATION_CYGPATH']}))\n", encoding="utf-8")
            for args in (["apply", "--target", posix(root / "destination")],
                         ["apply", "--no-dry-run", "--target", posix(root / "destination")],
                         ["apply", "--environment", "remote", "--target", "/srv/orbit"]):
                with self.subTest(args=args):
                    result = root / "result.json"
                    env = {**os.environ, "PYTHON": posix(Path(sys.executable)), "MIGRATION_RESULT": str(result)}
                    env.pop("POMELO_ORBIT_APP__ENV", None)
                    completed = subprocess.run([str(cygwin / "env.exe"), "PATH=/usr/bin:/bin", "/bin/sh",
                                                posix(scripts / "migrate_workspace.sh"), *args],
                                               cwd=root, env=env, capture_output=True, text=True, check=False)
                    self.assertEqual(completed.returncode, 0, completed.stderr)
                    saved = json.loads(result.read_text())
                    self.assertEqual(saved["argv"], args)
                    self.assertEqual(saved["profile"], "development")
                    self.assertEqual(migration.canonical(saved["caller"]), migration.canonical(root))
                    with patch.dict(os.environ, {"ORBIT_MIGRATION_CYGPATH": saved["cygpath"]}):
                        self.assertEqual(migration.canonical(migration.normalize_target_argument(posix(root), {"target_type": "ssh", "platform": "windows"})),
                                         migration.canonical(root))


class PlanningTests(unittest.TestCase):
    def test_home_workspace_overlap_is_blocked_without_running_containers(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as directory:
            root = Path(directory)
            source, target, cwd = root / "source", root / "target", root / "repo"
            source.mkdir()
            cwd.mkdir()
            alias = "~/" + source.relative_to(Path.home()).as_posix()
            parent_alias = "~/" + root.relative_to(Path.home()).as_posix()
            data = snapshot()
            data["environment"].update(target_type="local", workspace_root=alias)
            for field in ("services", "mounts", "runs", "artifacts", "routes"):
                data[field] = []
            config = {"workspace_root": str(source), "deployment_root": str(source / "deployment-logs"), "profile": "development"}
            for other_root, overlaps in ((alias, True), (alias + "/deployment", True), (parent_alias, True), (alias + "-other", False)):
                with self.subTest(other_root=other_root):
                    other = {"id": "env-2", "target_type": "local", "workspace_root": other_root}
                    db = Mock()
                    db.query.side_effect = lambda sql: [other] if "FROM environment WHERE id<>" in sql else [
                        {"name": "orbit", "address": "127.0.0.1", "port": 5432}]
                    with patch.object(migration, "select_environment", return_value=data["environment"]), \
                            patch.object(migration, "inventory", return_value=data), \
                            patch.object(migration, "active_jobs", return_value=False), \
                            patch.object(migration, "docker_inventory", return_value=("daemon", [])), \
                            patch.object(migration, "secure_directory", side_effect=lambda path: path.mkdir(parents=True, exist_ok=True)), \
                            self.assertLogs(level="INFO"):
                        _, plan = migration.plan_migration(SimpleNamespace(environment=None, target=str(target)), Mock(), config, db, cwd)
                    self.assertEqual(plan["blockers"], ["Another Environment on this data host overlaps the source"] if overlaps else [])
                    self.assertEqual(other["workspace_root"], other_root)

    def test_home_service_paths_are_resolved_for_planning_without_changing_database_guards(self):
        with tempfile.TemporaryDirectory(dir=Path.home()) as directory:
            root = Path(directory)
            source, target, cwd = root / "source", root / "target", root / "repo"
            service_directory = source / "deployment" / "web"
            service_directory.mkdir(parents=True)
            cwd.mkdir()
            (service_directory / "docker-compose.yml").write_text("original", encoding="utf-8")
            alias = "~/" + source.relative_to(Path.home()).as_posix()
            service_alias = alias + "/deployment/web"
            data = snapshot()
            data["environment"].update(target_type="local", workspace_root=alias)
            data["services"][0].update(deployment_directory=service_alias, runtime_directory=service_alias)
            data["artifacts"] = [{"id": "artifact-1", "location": alias + "/pipeline/artifact.tar"}]
            data["mounts"] = [{"service_id": "svc-1", "component_id": "component-1", "component_name": "web",
                               "target": "/data", "source": service_alias + "/data", "source_is_host_path": 0,
                               "overlay_id": None, "mount_id": None, "overlay_source": None,
                               "overlay_state": None, "overlay_host_path": None}]
            containers = [{"id": "container-1", "running": True, "project": "web", "component": "web",
                           "oneoff": False, "image": "web", "working_dir": str(service_directory),
                           "mounts": [{"Type": "bind", "Source": str(service_directory / "data"), "Destination": "/data"}]}]
            rendered = {"services": {"web": {"volumes": [{"type": "bind", "source": service_alias + "/data", "target": "/data"}]}}}
            db = Mock()
            db.query.side_effect = lambda sql: [{"name": "orbit", "address": "127.0.0.1", "port": 5432}] if "current_database" in sql else []
            config = {"workspace_root": str(source), "deployment_root": str(source / "deployment-logs"), "profile": "development"}
            with patch.object(migration, "select_environment", return_value=data["environment"]), \
                    patch.object(migration, "inventory", return_value=data), \
                    patch.object(migration, "active_jobs", return_value=False), \
                    patch.object(migration, "docker_inventory", return_value=("daemon", containers)), \
                    patch.object(migration, "run", return_value=json.dumps(rendered)), \
                    patch.object(migration, "secure_directory", side_effect=lambda path: path.mkdir(parents=True, exist_ok=True)):
                _, plan = migration.plan_migration(SimpleNamespace(environment=None, target=str(target)), Mock(), config, db, cwd)
            self.assertEqual(plan["blockers"], [])
            self.assertEqual(plan["services"][0]["source_directory"], str(service_directory))
            service = next(op for op in plan["mutations"] if op["table"] == "service")
            self.assertEqual(service["before"]["deployment_directory"], service_alias)
            self.assertEqual(service["after"]["deployment_directory"], migration.native_slashes(target / "deployment/web"))
            mount = next(op for op in plan["mutations"] if op["table"] == "service_component_mount")
            self.assertEqual(mount["after"]["source"], migration.native_slashes(target / "deployment/web/data"))
            artifact = next(op for op in plan["mutations"] if op["table"] == "artifact")
            self.assertEqual(artifact["before"]["location"], alias + "/pipeline/artifact.tar")
            self.assertEqual(artifact["after"]["location"], migration.native_slashes(target / "pipeline/artifact.tar"))
            self.assertEqual(data["services"][0]["runtime_directory"], service_alias)


class ExecutionTests(unittest.TestCase):
    def test_cold_copy_bootstrap_and_resume_preserve_new_database_writes(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source, target, bundle = root / "data", root / "target", root / "plan"
            source.mkdir()
            bundle.mkdir()
            services, containers = [], []
            for code, running, bootstrap in (("pg", True, True), ("web", True, False), ("cache", False, False)):
                path = source / "deployment" / code
                path.mkdir(parents=True)
                (path / "docker-compose.yml").write_text("original", encoding="utf-8")
                (path / "data").write_bytes(code.encode())
                services.append({"id": code, "code": code, "kind": "standard", "running": running,
                                 "components": {code: 1} if running else {}, "bootstrap": bootstrap,
                                 "source_directory": str(path), "target_directory": str(target / "deployment" / code),
                                 "compose_hash": migration.file_hash(path / "docker-compose.yml")})
                if running:
                    containers.append({"id": code + "-container", "project": code, "component": code, "running": True,
                                       "oneoff": False, "image": code, "addresses": [], "working_dir": str(path),
                                       "mounts": [{"Type": "bind", "Source": str(path / "data"), "Destination": "/data"}]})
            data = snapshot()
            data["environment"].update(workspace_root=str(source), last_probe_status="failed", last_probe_revision=4,
                                       last_probe_diagnostic="Environment was already unreachable")
            plan = {"format": 2, "blockers": [], "host": migration.socket.gethostname(), "profile": "development",
                    "docker_daemon": "daemon", "source": str(source), "target": str(target), "python": None,
                    "connection": {"environment": {"target_type": "local"}}, "environment_id": "env-1",
                    "project_code": "project-1", "new_revision": 5, "snapshot": data, "services": services,
                    "containers": copy.deepcopy(containers), "overrides": [],
                    "mutations": [migration.mutation("environment", {"id": "env-1"}, {"target_revision": 4}, {"target_revision": 5})]}
            path = bundle / "plan.json"
            migration.write_json(path, plan)
            migration.write_private(bundle / "plan.sha256", migration.digest(plan))
            migration.write_json(root / "active.json", {"plan": str(path)})
            events, live = [], {"updated": False, "fail": True}

            def fake_run(argv, cwd, timeout=120):
                if argv[:2] == ["docker", "stop"]:
                    item = next(item for item in containers if item["id"] == argv[-1])
                    item["running"] = False
                    events.append("stop-" + item["project"])
                elif "config" in argv:
                    code = argv[argv.index("-p") + 1]
                    return json.dumps({"services": {code: {"environment": {"KEY": "literal$VALUE"},
                                "volumes": [{"type": "bind", "source": str(source / "deployment" / code / "data"), "target": "/data"}]}}})
                elif "up" in argv:
                    self.assertNotIn("--wait", argv)
                    code = argv[argv.index("-p") + 1]
                    events.append("up-" + code)
                    item = next(item for item in containers if item["project"] == code)
                    item["running"] = True
                    item["working_dir"] = str(target / "deployment" / code)
                    item["mounts"][0]["Source"] = str(target / "deployment" / code / "data")
                return ""

            def read_inventory(db, environment):
                value = copy.deepcopy(data)
                if live["updated"]:
                    value["environment"]["workspace_root"] = str(target)
                return value

            def update(file):
                if file.name == "update.sql":
                    if live["fail"]:
                        live["fail"] = False
                        raise migration.MigrationError("interrupted after database bootstrap")
                    events.append("sql")
                    live["updated"] = True

            db, runtime = Mock(), Mock()
            db.query.side_effect = lambda sql: [{"workspace_root": str(target), "last_probe_status": "failed",
                                                "last_probe_revision": 4, "target_revision": 5}] if "SELECT workspace_root" in sql else [{"value": 1}]
            db.execute.side_effect = update
            db.backup.side_effect = lambda file: events.append("backup") or file.write_bytes(b"dump")
            runtime.call.side_effect = AssertionError("A local migration must not call Environment Probe")
            config = {"profile": "development"}
            with patch.object(migration, "run", side_effect=fake_run), \
                    patch.object(migration, "docker_inventory", side_effect=lambda cwd: ("daemon", copy.deepcopy(containers))), \
                    patch.object(migration, "inventory", side_effect=read_inventory), \
                    patch.object(migration, "active_jobs", return_value=False), \
                    patch.object(migration, "secure_directory", side_effect=lambda path: path.mkdir(parents=True, exist_ok=True)), \
                    self.assertLogs(level="INFO") as logs:
                with self.assertRaisesRegex(migration.MigrationError, "interrupted"):
                    migration.apply_migration(path, runtime, config, db, root)
                database_file = target / "deployment" / "pg" / "data"
                database_file.write_bytes(b"new-database-writes")
                migration.apply_migration(path, runtime, config, db, root)
                self.assertEqual(database_file.read_bytes(), b"new-database-writes")
                self.assertFalse((root / "active.json").exists())
                completed = events.copy()
                migration.apply_migration(path, runtime, config, db, root)
                self.assertEqual(events, completed)
            self.assertEqual(events, ["backup", "stop-web", "stop-pg", "up-pg", "sql", "up-web"])
            self.assertTrue(any("Start Orbit manually" in message for message in logs.output))
            runtime.call.assert_not_called()
            self.assertFalse((source / ".orbit-workspace-migration").exists())
            self.assertTrue((source / "deployment" / "pg" / "data").exists())
            stopped = json.loads((target / "deployment" / "cache" / "docker-compose.yml").read_text())
            self.assertEqual(stopped["services"]["cache"]["environment"]["KEY"], "literal$$VALUE")
            self.assertTrue((bundle / "verification.json").is_file())

class FilesystemTests(unittest.TestCase):
    def test_relocation_is_case_aware_and_respects_directory_boundaries(self):
        self.assertEqual(migration.relocate("d:\\OLD\\pipeline\\a", "D:/old", "E:/new"), "E:/new/pipeline/a")
        self.assertEqual(migration.relocate("D:/older/db", "D:/old", "E:/new"), "D:/older/db")
        self.assertEqual(migration.relocate("/srv/OLD/db", "/srv/old", "/srv/new"), "/srv/OLD/db")
        self.assertTrue(migration.within("/run/desktop/mnt/host/d/old/db", "D:/old"))
        self.assertEqual(migration.relocate("/run/desktop/mnt/host/d/old/db", "D:/old", "E:/new"), "E:/new/db")

    def test_copy_preserves_source_and_does_not_move_other_state(self):
        with tempfile.TemporaryDirectory() as directory:
            source, target = Path(directory) / "source", Path(directory) / "target"
            (source / "deployment" / "pg").mkdir(parents=True)
            (source / "config").mkdir()
            (source / "deployment" / "pg" / "data").write_bytes(b"persistent-data")
            (source / "config" / "lock").touch()
            result = migration.copy_workspace(source, target)
            self.assertEqual(result["files"], 1)
            self.assertEqual((target / "deployment" / "pg" / "data").read_bytes(), b"persistent-data")
            self.assertTrue((source / "deployment" / "pg" / "data").exists())
            self.assertFalse((target / "config").exists())
            (target / "deployment" / "pg" / "data").write_bytes(b"interrupted-copy")
            migration.copy_workspace(source, target)
            self.assertEqual((target / "deployment" / "pg" / "data").read_bytes(), b"persistent-data")

    def test_nested_target_is_rejected_before_copy(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / "source"
            source.mkdir()
            with self.assertRaises(migration.MigrationError):
                migration.validate_paths(source, source / "nested", root / "plan")

    def test_migration_bundle_in_data_stays_out_of_the_copy(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source, target = root / "data", root / "new-workspace"
            bundle = source / "workspace-migration" / "run"
            bundle.mkdir(parents=True)
            (bundle / "plan.json").write_text("private migration state", encoding="utf-8")
            (source / "deployment").mkdir()
            (source / "deployment" / "compose.yml").write_text("services: {}", encoding="utf-8")
            migration.validate_paths(source, target, bundle)
            self.assertEqual(migration.copy_workspace(source, target)["files"], 1)
            self.assertFalse((target / "workspace-migration").exists())
            for child in migration.WORKSPACE_CHILDREN:
                with self.subTest(child=child), self.assertRaises(migration.MigrationError):
                    migration.validate_paths(source, target, source / child / "migration")

    def test_linked_source_file_is_not_followed(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "source" / "pipeline").mkdir(parents=True)
            secret = root / "outside"
            secret.write_bytes(b"outside-data")
            link = root / "source" / "pipeline" / "link"
            try:
                link.symlink_to(secret)
            except OSError:
                self.skipTest("Creating symlinks is unavailable for this Windows account")
            with self.assertRaises(migration.MigrationError):
                migration.copy_workspace(root / "source", root / "target")
            self.assertEqual(secret.read_bytes(), b"outside-data")

    def test_mysql_runtime_socket_link_is_excluded_from_assessment_and_copy(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source, target = root / "source", root / "target"
            data = source / "deployment" / "mysql" / "data"
            data.mkdir(parents=True)
            (data / "database").write_bytes(b"business-data")
            link = data / "mysql.sock"
            try:
                link.symlink_to("/run/mysqld/mysqld.sock")
            except OSError:
                self.skipTest("Creating symlinks is unavailable for this Windows account")
            assessment = migration.assess_files(source, target)
            self.assertEqual(assessment["unreadable_or_linked"], [])
            self.assertEqual(assessment["runtime_sockets_skipped"], [str(link)])
            self.assertEqual(migration.copy_workspace(source, target)["files"], 1)
            self.assertFalse((target / "deployment" / "mysql" / "data" / "mysql.sock").exists())
            self.assertEqual((target / "deployment" / "mysql" / "data" / "database").read_bytes(), b"business-data")


class BusinessRulesTests(unittest.TestCase):
    def test_only_current_proven_run_binding_is_carried_forward(self):
        data = snapshot()
        for key, value in (("environment_target_revision", 1), ("ssh_credential_revision", 1),
                           ("environment_id", "other-env")):
            old = {**data["runs"][0], "id": "old-" + key, key: value}
            data["runs"].append(old)
        ops = migration.build_mutations(data, "D:/old", "E:/new", "2026-10-19T00:00:00Z", {})
        runs = [op for op in ops if op["table"] == "pipeline_run"]
        self.assertEqual([op["key"]["id"] for op in runs], ["run-1"])
        self.assertEqual(runs[0]["after"]["environment_target_revision"], 5)
        self.assertEqual(data["environment"]["target_revision"], 4)

    def test_mount_relocation_creates_service_overrides_without_editing_published_versions(self):
        data = snapshot()
        mount = {"service_id": "svc-1", "component_id": "component-1", "component_name": "web",
                 "target": "/data", "source": "D:/old/deployment/web/data", "source_is_host_path": 0,
                 "overlay_id": None, "mount_id": None, "overlay_source": None,
                 "overlay_state": None, "overlay_host_path": None}
        data["mounts"] = [mount, {**mount, "target": "/config", "source": "D:/old/deployment/web/config"}]
        ops = migration.build_mutations(data, "D:/old", "E:/new", "2026-10-19T00:00:00Z", {})
        self.assertEqual(sum(op["table"] == "service_component" for op in ops), 1)
        mounts = [op for op in ops if op["table"] == "service_component_mount"]
        self.assertEqual(len(mounts), 2)
        self.assertEqual({op["after"]["service_component_id"] for op in mounts},
                         {next(op["key"]["id"] for op in ops if op["table"] == "service_component")})
        self.assertTrue(all(not op["table"].startswith("version") for op in ops))
        self.assertEqual(mounts[0]["after"]["source"], "E:/new/deployment/web/data")

    def test_stopped_service_and_failed_probe_history_are_preserved(self):
        data = snapshot()
        data["services"][0]["status"] = "stopped"
        history = {"last_probe_revision": 4, "last_probe_status": "failed",
                   "last_probe_at": "2026-10-18T00:00:00Z", "last_probe_diagnostic": "SSH unavailable"}
        data["environment"].update(history)
        ops = migration.build_mutations(data, "D:/old", "E:/new", "2026-10-19T00:00:00Z", {})
        service = next(op for op in ops if op["table"] == "service")
        self.assertNotIn("status", service["after"])
        self.assertTrue(history.keys().isdisjoint(ops[0]["after"]))
        self.assertEqual({key: data["environment"][key] for key in history}, history)
        self.assertEqual(service["after"]["runtime_directory"], "E:/new/deployment/web")

    def test_uncopied_workspace_paths_cannot_be_rewritten(self):
        for path in ("D:/old/database", "D:/old", "D:/old/deployment-other/database"):
            with self.subTest(path=path), self.assertRaisesRegex(migration.MigrationError, "outside the copied"):
                data = snapshot()
                data["artifacts"] = [{"id": "artifact-1", "location": path}]
                migration.build_mutations(data, "D:/old", "E:/new", "2026-10-19T00:00:00Z", {})

    def test_compose_freeze_rejects_uncopied_data_and_file_references(self):
        for config in (
            {"services": {"db": {"volumes": [{"type": "bind", "source": "D:/old/database", "target": "/data"}]}}},
            {"services": {"web": {"build": {"context": "D:/old/build"}}}},
            {"secrets": {"password": {"file": "D:/old/config/password"}}},
        ):
            with self.subTest(config=config), tempfile.TemporaryDirectory() as directory:
                bundle = Path(directory)
                host = migration.Host(Mock(), {"environment": {"target_type": "local"}})
                plan = {"source": "D:/old", "target": "E:/new", "services": [
                    {"code": "db", "compose_hash": "present", "source_directory": "D:/old/deployment/db"}]}
                with patch.object(migration, "run", return_value=json.dumps(config)):
                    with self.assertRaisesRegex(migration.MigrationError, "outside the copied"):
                        migration.freeze_compose(plan, bundle, host)
                self.assertEqual(list(bundle.iterdir()), [])

    def test_compose_relocates_only_copied_host_paths_and_preserves_named_and_external_volumes(self):
        config = {"services": {"db": {"volumes": [
            {"type": "bind", "source": "/run/desktop/mnt/host/d/old/deployment/db/data", "target": "/data"},
            {"type": "bind", "source": "D:/shared", "target": "/shared"},
            {"type": "volume", "source": "database", "target": "/volume"},
        ]}}}
        host = migration.Host(Mock(), {"environment": {"target_type": "local"}})
        migration.relocate_compose(config, "D:/old", "E:/new", host)
        self.assertEqual([volume["source"] for volume in config["services"]["db"]["volumes"]],
                         ["E:/new/deployment/db/data", "D:/shared", "database"])


@unittest.skipUnless(os.environ.get("ORBIT_MIGRATION_TEST_CONTAINER"), "Set an isolated PostgreSQL test container")
class PostgresTransactionTests(unittest.TestCase):
    def sql(self, statement: str, check: bool = True) -> str:
        result = subprocess.run(
            ["docker", "exec", "-i", os.environ["ORBIT_MIGRATION_TEST_CONTAINER"], "psql", "-X", "-qAt",
             "-v", "ON_ERROR_STOP=1", "-U", "postgres"], input=statement,
            capture_output=True, text=True, encoding="utf-8", check=False,
        )
        if check and result.returncode:
            self.fail(result.stderr)
        if not check:
            self.assertNotEqual(result.returncode, 0, result.stdout)
        return result.stdout.strip()

    def setUp(self):
        self.sql("""
        DROP SCHEMA IF EXISTS migration_test CASCADE;
        CREATE SCHEMA migration_test;
        SET search_path TO migration_test;
        CREATE TABLE environment (id text primary key, workspace_root text, target_revision int,
            last_probe_revision int, last_probe_status text, last_probe_diagnostic text, last_probe_at timestamptz);
        CREATE TABLE service (id text primary key, deployment_directory text);
        CREATE TABLE pipeline_run (id text primary key);
        CREATE TABLE artifact (id text primary key);
        CREATE TABLE service_component (id text primary key);
        CREATE TABLE service_component_mount (id text primary key);
        INSERT INTO environment (id,workspace_root,target_revision,last_probe_revision,last_probe_status,last_probe_diagnostic,last_probe_at)
            VALUES ('env-1','D:/old',4,4,'failed','SSH unavailable','2026-10-18T00:00:00Z'),
                   ('other-env','D:/old',4,NULL,NULL,NULL,NULL);
        INSERT INTO service VALUES ('svc-1','D:/old/deployment/web');
        """)
        self.ops = [migration.mutation("environment", {"id": "env-1"},
                                     {"workspace_root": "D:/old", "target_revision": 4},
                                     {"workspace_root": "E:/new", "target_revision": 5}),
                    migration.mutation("service", {"id": "svc-1"},
                                       {"deployment_directory": "D:/old/deployment/web"},
                                       {"deployment_directory": "E:/new/deployment/web"})]

    def execute(self, operations: list, check: bool = True):
        self.sql("SET search_path TO migration_test;\n" + migration.migration_sql(operations), check=check)

    def test_transaction_relocates_only_selected_environment_and_is_replayable(self):
        history_sql = "SET search_path TO migration_test; SELECT last_probe_status||':'||last_probe_revision||':'||last_probe_diagnostic||':'||last_probe_at FROM environment WHERE id='env-1';"
        history = self.sql(history_sql)
        self.execute(self.ops)
        self.execute(self.ops)
        self.assertEqual(self.sql("SET search_path TO migration_test; SELECT workspace_root||':'||target_revision FROM environment WHERE id='env-1';"), "E:/new:5")
        self.assertEqual(self.sql("SET search_path TO migration_test; SELECT workspace_root FROM environment WHERE id='other-env';"), "D:/old")
        self.assertEqual(self.sql(history_sql), history)

    def test_stale_service_aborts_every_update(self):
        self.sql("SET search_path TO migration_test; UPDATE service SET deployment_directory='D:/changed';")
        self.execute(self.ops, check=False)
        self.assertEqual(self.sql("SET search_path TO migration_test; SELECT target_revision FROM environment WHERE id='env-1';"), "4")

    def test_constraint_failure_rolls_back_prior_updates(self):
        ops = [*self.ops, migration.mutation("service_component", {"id": None}, None, {})]
        self.execute(ops, check=False)
        self.assertEqual(self.sql("SET search_path TO migration_test; SELECT target_revision FROM environment WHERE id='env-1';"), "4")

    def test_quoted_path_is_treated_as_data(self):
        ops = copy.deepcopy(self.ops)
        ops[0]["after"]["workspace_root"] = "E:/O'Brien/$orbit_/new"
        self.execute(ops)
        self.assertEqual(self.sql("SET search_path TO migration_test; SELECT workspace_root FROM environment WHERE id='env-1';"), "E:/O'Brien/$orbit_/new")


if __name__ == "__main__":
    unittest.main()
