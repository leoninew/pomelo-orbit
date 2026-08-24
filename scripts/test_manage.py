"""Focused tests for the remote backup command."""

from __future__ import annotations

import importlib.util
import sys
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch


SCRIPT = Path(__file__).with_name("manage.py")
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


if __name__ == "__main__":
    unittest.main()
