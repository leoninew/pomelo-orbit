"""Integration tests for the release version calculator."""

from __future__ import annotations

import importlib.util
import subprocess
import tempfile
import unittest
from pathlib import Path


SCRIPT_PATH = Path(__file__).with_name("version-calc.py")
SPEC = importlib.util.spec_from_file_location("version_calc", SCRIPT_PATH)
assert SPEC is not None and SPEC.loader is not None
version_calc = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(version_calc)


def git(directory: Path, *args: str) -> str:
    return subprocess.run(
        ["git", *args],
        cwd=directory,
        check=True,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    ).stdout.strip()


class VersionCalcTests(unittest.TestCase):
    def setUp(self) -> None:
        self.directory = tempfile.TemporaryDirectory()
        self.root = Path(self.directory.name)
        self.remote = self.root / "remote.git"
        self.repository = self.root / "repository"
        git(self.root, "init", "--bare", str(self.remote))
        git(self.root, "init", str(self.repository))
        git(self.repository, "config", "user.name", "Version Test")
        git(self.repository, "config", "user.email", "version-test@example.test")
        (self.repository / "configs").mkdir()
        (self.repository / "web").mkdir()
        (self.repository / "VERSION").write_text("0.0.0\n", encoding="utf-8")
        (self.repository / "configs" / "config.yaml").write_text(
            "app:\n  version: 0.0.0\n", encoding="utf-8"
        )
        (self.repository / ".env.example").write_text(
            "# POMELO_ORBIT_APP__VERSION=0.0.0\n", encoding="utf-8"
        )
        (self.repository / "web" / "package.json").write_text(
            '{\n  "version": "0.0.0"\n}\n', encoding="utf-8"
        )
        git(self.repository, "add", ".")
        git(self.repository, "commit", "-m", "chore: bootstrap")
        git(self.repository, "branch", "-M", "develop")
        git(self.repository, "remote", "add", "origin", str(self.remote))
        git(self.repository, "push", "-u", "origin", "develop")

        self.original_paths = (
            version_calc.REPO_ROOT,
            version_calc.VERSION_FILE,
            version_calc.CONFIG_FILE,
            version_calc.ENV_EXAMPLE_FILE,
            version_calc.PACKAGE_JSON,
        )
        version_calc.REPO_ROOT = self.repository
        version_calc.VERSION_FILE = self.repository / "VERSION"
        version_calc.CONFIG_FILE = self.repository / "configs" / "config.yaml"
        version_calc.ENV_EXAMPLE_FILE = self.repository / ".env.example"
        version_calc.PACKAGE_JSON = self.repository / "web" / "package.json"

    def tearDown(self) -> None:
        (
            version_calc.REPO_ROOT,
            version_calc.VERSION_FILE,
            version_calc.CONFIG_FILE,
            version_calc.ENV_EXAMPLE_FILE,
            version_calc.PACKAGE_JSON,
        ) = self.original_paths
        self.directory.cleanup()

    def test_apply_amend_updates_version_files_amends_unpushed_head_and_tags(
        self,
    ) -> None:
        (self.repository / "change.txt").write_text("change\n", encoding="utf-8")
        git(self.repository, "add", "change.txt")
        git(self.repository, "commit", "-m", "fix: change source")
        before = git(self.repository, "rev-parse", "HEAD")
        parent = git(self.repository, "rev-parse", "HEAD^")
        expected = version_calc.calculate_version(print_history=False)
        git(self.repository, "tag", f"v{expected}")

        self.assertEqual(0, version_calc.main(["--quiet", "--apply-amend"]))

        self.assertNotEqual(before, git(self.repository, "rev-parse", "HEAD"))
        self.assertEqual(parent, git(self.repository, "rev-parse", "HEAD^"))
        self.assertEqual(
            "fix: change source", git(self.repository, "log", "-1", "--format=%s")
        )
        self.assertEqual("", git(self.repository, "status", "--porcelain=v1"))
        self.assertEqual(
            expected, (self.repository / "VERSION").read_text(encoding="utf-8").strip()
        )
        self.assertIn(
            expected,
            (self.repository / "configs" / "config.yaml").read_text(encoding="utf-8"),
        )
        self.assertIn(
            expected, (self.repository / ".env.example").read_text(encoding="utf-8")
        )
        self.assertIn(
            expected,
            (self.repository / "web" / "package.json").read_text(encoding="utf-8"),
        )
        self.assertEqual(
            git(self.repository, "rev-parse", "HEAD"),
            git(self.repository, "rev-list", "-n", "1", f"v{expected}"),
        )

    def test_apply_amend_includes_pre_staged_version_metadata(self) -> None:
        (self.repository / "change.txt").write_text("change\n", encoding="utf-8")
        git(self.repository, "add", "change.txt")
        git(self.repository, "commit", "-m", "fix: change source")
        expected = version_calc.calculate_version(print_history=False)
        version_calc.apply_version(expected)
        version_calc.stage_version_files()
        before = git(self.repository, "rev-parse", "HEAD")
        git(self.repository, "tag", f"v{expected}")

        self.assertEqual(0, version_calc.main(["--quiet", "--apply-amend"]))

        self.assertNotEqual(before, git(self.repository, "rev-parse", "HEAD"))
        self.assertEqual("", git(self.repository, "status", "--porcelain=v1"))
        self.assertEqual(
            git(self.repository, "rev-parse", "HEAD"),
            git(self.repository, "rev-list", "-n", "1", f"v{expected}"),
        )

    def test_apply_amend_preserves_unrelated_index_and_worktree_changes(self) -> None:
        (self.repository / "tracked.txt").write_text("before\n", encoding="utf-8")
        git(self.repository, "add", "tracked.txt")
        git(self.repository, "commit", "-m", "chore: add tracked file")
        (self.repository / "change.txt").write_text("change\n", encoding="utf-8")
        git(self.repository, "add", "change.txt")
        git(self.repository, "commit", "-m", "fix: change source")
        (self.repository / "staged.txt").write_text("staged\n", encoding="utf-8")
        (self.repository / "tracked.txt").write_text("after\n", encoding="utf-8")
        git(self.repository, "add", "staged.txt")

        self.assertEqual(0, version_calc.main(["--quiet", "--apply-amend"]))

        self.assertEqual(
            "staged.txt", git(self.repository, "diff", "--cached", "--name-only")
        )
        self.assertEqual("tracked.txt", git(self.repository, "diff", "--name-only"))
        self.assertEqual(
            "",
            git(
                self.repository,
                "show",
                "--format=",
                "--name-only",
                "HEAD",
                "--",
                "staged.txt",
            ),
        )

    def test_apply_amend_rejects_head_already_on_upstream(self) -> None:
        before = (self.repository / "VERSION").read_text(encoding="utf-8")

        with self.assertRaisesRegex(RuntimeError, "requires an unpushed HEAD"):
            version_calc.main(["--quiet", "--apply-amend"])

        self.assertEqual(
            before, (self.repository / "VERSION").read_text(encoding="utf-8")
        )
        self.assertEqual("", git(self.repository, "status", "--porcelain=v1"))

    def test_apply_and_apply_amend_are_mutually_exclusive(self) -> None:
        with self.assertRaises(SystemExit):
            version_calc.parse_args(["--apply", "--apply-amend"])


if __name__ == "__main__":
    unittest.main()
