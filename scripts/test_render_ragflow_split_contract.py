"""Focused tests for the RAGFlow deployment contract renderer."""

from __future__ import annotations

import copy
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPTS_DIRECTORY = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPTS_DIRECTORY))
import render_ragflow_split_contract as renderer  # noqa: E402


class RAGFlowSplitContractRendererTests(unittest.TestCase):
    def setUp(self) -> None:
        self.contract = renderer.load_contract()
        self.rendered = renderer.render_all(self.contract)

    def test_committed_generated_artifacts_match_contract(self) -> None:
        self.assertEqual(renderer.check_outputs(self.rendered), [])

    def test_integrated_generated_artifacts_match_contract(self) -> None:
        integrated = renderer.load_contract(renderer.INTEGRATED_CONTRACT_PATH)

        self.assertEqual(renderer.check_outputs(renderer.render_all(integrated)), [])

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

        with self.assertRaisesRegex(renderer.ContractError, "must match"):
            renderer.validate_contract(changed)

    def test_command_check_succeeds_for_committed_artifacts(self) -> None:
        result = subprocess.run(
            [sys.executable, str(SCRIPTS_DIRECTORY / "render_ragflow_deployment_contract.py"), "--check"],
            cwd=renderer.repository_root(),
            capture_output=True,
            text=True,
            check=False,
        )

        self.assertEqual(result.returncode, 0, result.stderr)


if __name__ == "__main__":
    unittest.main()
