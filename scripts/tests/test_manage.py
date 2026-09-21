"""Focused tests for manage.py commands."""

from __future__ import annotations

import importlib.util
import sys
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
    def test_backup_prunes_pipeline_and_keeps_only_deployment_compose_files(
        self,
    ) -> None:
        config = SimpleNamespace(ssh_target="ubuntu@example.test")

        with (
            patch.object(manage, "run_ssh_command") as run_ssh,
            patch.object(manage.subprocess, "run"),
        ):
            manage.backup(config, "/opt/pomelo-orbit")

        archive_command = run_ssh.call_args_list[0].args[0]
        self.assertIn("find . -path ./data/pipeline -prune", archive_command)
        self.assertIn("-path ./data/deployment -prune -o -print0", archive_command)
        self.assertIn(
            "find ./data/deployment -mindepth 2 -maxdepth 2 "
            "-type f -name docker-compose.yml -print0",
            archive_command,
        )
        self.assertIn("--no-recursion", archive_command)
        self.assertNotIn("--exclude-workspace", archive_command)


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
