"""Focused tests for manage.py commands."""

from __future__ import annotations

import importlib.util
import shlex
import shutil
import subprocess
import sys
import tarfile
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch


SCRIPT = Path(__file__).parent.parent / "manage.py"
SPEC = importlib.util.spec_from_file_location("manage", SCRIPT)
assert SPEC is not None
assert SPEC.loader is not None
manage = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = manage
SPEC.loader.exec_module(manage)


class BackupTests(unittest.TestCase):
    def test_backup_prunes_excluded_directories_without_tar_recursion(self) -> None:
        config = SimpleNamespace(ssh_target="ubuntu@example.test")

        with (
            tempfile.TemporaryDirectory() as directory,
            patch.object(manage, "SCRIPT_DIR", Path(directory)),
            patch.object(manage, "run_ssh_command") as run_ssh,
            patch.object(manage.subprocess, "run"),
        ):
            manage.backup(config, "/opt/pomelo-orbit")

        archive_command = run_ssh.call_args_list[0].args[0]
        self.assertIn("find . -path ./data/pipeline -prune -o", archive_command)
        self.assertIn("--no-recursion", archive_command)

    @unittest.skipUnless(shutil.which("sh"), "POSIX shell is required")
    def test_archive_contains_deployment_configuration_and_data(self) -> None:
        config = SimpleNamespace(ssh_target="ubuntu@example.test")
        retained_files = {
            ".env": "ORBIT_SETTING=value\n",
            "data/db/pomelo-orbit.db": "orbit database",
            "data/deployment/mysql-default/docker-compose.yml": "services: {}\n",
            "data/deployment/mysql-default/.env": "MYSQL_SETTING=value\n",
            "data/deployment/mysql-default/env/mysql.env": "MYSQL_SETTING=value\n",
            "data/deployment/mysql-default/env/mysql/my.cnf": "mysql configuration",
            "data/deployment/postgres-default/docker-compose.yml": "services: {}\n",
            "data/deployment/postgres-default/.env": "POSTGRES_SETTING=value\n",
            "data/deployment/postgres-default/env/postgres/postgresql.conf": "pg configuration",
            "data/deployment/alist-default/data/settings.json": "{}\n",
            "data/deployment/mihomo-default/docker-compose.yml": "services: {}\n",
            "data/deployment/mihomo-default/.env": "MIHOMO_SETTING=value\n",
            "data/deployment/sub2api-default/docker-compose.yml": "services: {}\n",
            "data/deployment/sub2api-default/.env": "SUB2API_SETTING=value\n",
            "data/deployment/sub2api-default/app/model_pricing.json": "{}\n",
            "data/deployment/ragflow-integrated-default/.env": "RAGFLOW_SETTING=value\n",
            "data/deployment/ragflow-integrated-default/ragflow/data/document.txt": "document",
            "data/deployment/traefik-default/traefik/certs/example.pem": "certificate",
            "data/deployment/traefik-default/traefik/routes/example.yml": "http: {}\n",
        }
        excluded_files = {
            "data/pipeline/checkout/source.txt": "pipeline source",
            "data/deployment/mysql-default/mysql/pomelo_orbit/database.ibd": "mysql database",
            "data/deployment/ragflow-integrated-default/mysql/ragflow/database.ibd": "ragflow database",
            "data/deployment/postgres-default/postgres/18/docker/PG_VERSION": "18\n",
            "data/deployment/sub2api-default/postgres/base/database": "postgres database",
            "data/deployment/mihomo-default/config/geoip.dat": "geoip database",
            "data/deployment/mihomo-default/config/ui/_nuxt/app.js": "ui assets",
            "data/deployment/sub2api-default/app/logs/sub2api.log": "application log",
            "data/deployment/sub2api-default/app/logs/sub2api-old.log.gz": "archived log",
        }

        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            remote_dir = root / "remote deployment"
            for name, content in (retained_files | excluded_files).items():
                path = remote_dir / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(content, newline="\n")

            with (
                patch.object(manage, "SCRIPT_DIR", root),
                patch.object(manage, "run_ssh_command") as run_ssh,
                patch.object(manage.subprocess, "run") as run,
            ):
                manage.backup(config, remote_dir.as_posix())

            scp_command = run.call_args.args[0]
            remote_archive = scp_command[1].removeprefix(f"{config.ssh_target}:")
            local_archive = Path(scp_command[2])
            archive_command = (
                run_ssh.call_args_list[0]
                .args[0]
                .replace(
                    shlex.quote(remote_archive),
                    shlex.quote(f"../backup/{local_archive.name}"),
                )
            )
            subprocess.run(["sh", "-c", archive_command], check=True)

            with tarfile.open(local_archive, "r:gz") as archive:
                archived_files = {}
                for member in archive.getmembers():
                    if member.isfile():
                        content_file = archive.extractfile(member)
                        assert content_file is not None
                        archived_files[member.name.removeprefix("./")] = (
                            content_file.read()
                        )

            self.assertEqual(
                archived_files,
                {name: content.encode() for name, content in retained_files.items()},
            )


class TunnelTests(unittest.TestCase):
    def test_start_enables_ssh_diagnostics_when_verbose(self) -> None:
        config = SimpleNamespace(
            ssh_host="example.test",
            ssh_target="ubuntu@example.test",
            ssh_user="ubuntu",
        )
        tunnel = manage.SSHTunnel(config)

        with (
            patch.object(tunnel, "_load_tunnels", return_value=[]),
            patch.object(tunnel, "_port_in_use", return_value=False),
            patch.object(tunnel, "_save_tunnels"),
            patch.object(manage.time, "sleep"),
            patch.object(manage.subprocess, "Popen") as popen,
        ):
            tunnel.start(remote_port=5432, local_port=5433, verbose=True)

        command = popen.call_args.args[0]
        self.assertIn("-v", command)
        self.assertIsNone(popen.call_args.kwargs["stderr"])

    def test_main_passes_verbose_to_tunnel_start(self) -> None:
        config = SimpleNamespace()

        with (
            patch.object(manage, "Config", return_value=config),
            patch.object(manage, "SSHTunnel") as tunnel_class,
            patch.object(
                sys,
                "argv",
                ["manage.py", "tunnel", "start", "-v", "5432:5433"],
            ),
        ):
            manage.main()

        tunnel_class.return_value.start.assert_called_once_with(
            remote_port=5432,
            local_port=5433,
            verbose=True,
        )


if __name__ == "__main__":
    unittest.main()
