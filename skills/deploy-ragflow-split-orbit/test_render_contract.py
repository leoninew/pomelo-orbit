"""Focused tests for the split RAGFlow deployment contract renderer."""

from __future__ import annotations

import copy
import importlib.util
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


SKILL_DIRECTORY = Path(__file__).resolve().parent
sys.path.insert(0, str(SKILL_DIRECTORY))
sys.path.insert(0, str(SKILL_DIRECTORY.parent))
from _ragflow import contract_renderer  # noqa: E402

RENDERER_SPEC = importlib.util.spec_from_file_location(
    "ragflow_split_contract_renderer",
    SKILL_DIRECTORY / "render_contract.py",
)
assert RENDERER_SPEC and RENDERER_SPEC.loader
renderer = importlib.util.module_from_spec(RENDERER_SPEC)
sys.modules[RENDERER_SPEC.name] = renderer
RENDERER_SPEC.loader.exec_module(renderer)


class RAGFlowSplitContractRendererTests(unittest.TestCase):
    def setUp(self) -> None:
        self.contract, self.rendered = renderer.render_current_contract()

    def test_committed_generated_artifacts_match_contract(self) -> None:
        self.assertEqual(renderer.check_outputs(self.rendered), [])

    def test_contract_only_owns_split_topology(self) -> None:
        application_codes = {application["code"] for application in self.contract["applications"]}

        self.assertEqual(
            application_codes,
            {
                "ragflow-split",
                "ragflow-mysql",
                "ragflow-redis",
                "ragflow-minio",
                "ragflow-elasticsearch",
            },
        )
        self.assertEqual(
            self.contract["model_cache_paths"],
            [
                "data/deployment/ragflow-integrated/default/tei/cache/bge-m3",
                "data/deployment/ragflow-split/default/tei/cache/bge-m3",
            ],
        )

    def test_check_reports_a_changed_generated_artifact(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            renderer.write_outputs(self.rendered, root)
            changed = next(iter(self.rendered))
            (root / changed).write_text("drift\n", encoding="utf-8")

            self.assertEqual(renderer.check_outputs(self.rendered, root), [changed])

    def test_contract_rejects_runtime_key_assignment_drift(self) -> None:
        changed = copy.deepcopy(self.contract)
        changed["runtime_config"]["assignments"][0]["keys"] = []

        with self.assertRaisesRegex(contract_renderer.ContractError, "must match"):
            contract_renderer.validate_contract(changed)

    def test_command_check_succeeds_for_committed_artifacts(self) -> None:
        result = subprocess.run(
            [sys.executable, str(SKILL_DIRECTORY / "render_contract.py"), "--check"],
            cwd=contract_renderer.REPOSITORY_ROOT,
            capture_output=True,
            text=True,
            check=False,
        )

        self.assertEqual(result.returncode, 0, result.stderr)


if __name__ == "__main__":
    unittest.main()
