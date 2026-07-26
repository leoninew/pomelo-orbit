from __future__ import annotations

import asyncio
import copy
import importlib.util
import tempfile
import unittest
from collections.abc import Mapping
from pathlib import Path
from typing import Any
from unittest.mock import patch

SCRIPT_PATH = Path(__file__).with_name("ragflow_initialize.py")
SPEC = importlib.util.spec_from_file_location("ragflow_initialize", SCRIPT_PATH)
assert SPEC is not None and SPEC.loader is not None
initializer = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(initializer)


class FakeGateway:
    def __init__(
        self,
        tools: set[str],
        *,
        applications: list[dict[str, str]] | None = None,
        credentials: list[dict[str, str]] | None = None,
        doctor: Mapping[str, Any] | None = None,
    ) -> None:
        self.tools = tools
        self.applications = applications or []
        self.credentials = credentials or []
        self.doctor = (
            dict(doctor)
            if doctor is not None
            else {
                "healthy": True,
                "external_networks": [initializer.REQUIRED_EXTERNAL_NETWORK],
            }
        )
        self.calls: list[tuple[str, Mapping[str, Any]]] = []

    async def list_tool_names(self) -> set[str]:
        return self.tools

    async def call(self, name: str, arguments: Mapping[str, Any]) -> dict[str, Any]:
        self.calls.append((name, arguments))
        if name == "orbit_list_applications":
            return {"applications": self.applications}
        if name == "orbit_list_runtime_env_credentials":
            return {"credentials": self.credentials}
        if name == "runtime_doctor":
            return self.doctor
        if name == "orbit_wait_deployment":
            return {"deployment": {"status": "ran_to_completion"}}
        if name == "verify_deployment":
            return {"conclusion": "consistent"}
        if name == "runtime_http_probe":
            return {"status": "reachable"}
        if name == "orbit_update_version":
            return {}
        raise AssertionError(name)


class RAGFlowInitializerTests(unittest.TestCase):
    def setUp(self) -> None:
        self.fixture = initializer.load_yaml_mapping(initializer.DEFAULT_FIXTURE)

    def test_fixture_is_valid_and_plan_is_deterministic(self) -> None:
        initializer.validate_fixture(self.fixture)
        operations = initializer.planned_operations(self.fixture)
        self.assertEqual(len(operations), 30)
        self.assertEqual(initializer.fixture_digest(self.fixture), initializer.fixture_digest(self.fixture))
        self.assertIn("bootstrap:ragflow", {item["id"] for item in operations})

    def test_plan_writes_a_secret_free_run_record(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            result = asyncio.run(
                initializer.run(
                    initializer.argparse.Namespace(
                        fixture=initializer.DEFAULT_FIXTURE,
                        mode="plan",
                        run_root=Path(temporary),
                    )
                )
            )
            state = (Path(result["run_directory"]) / "state.json").read_text(encoding="utf-8")
        self.assertEqual(result["status"], "planned")
        self.assertNotIn("credential_values", state)
        self.assertNotIn("environment_id", state)

    def test_encode_component_resolves_only_credential_ids(self) -> None:
        application = next(item for item in self.fixture["applications"] if item["code"] == "ragflow")
        component = application["components"][0]
        encoded = initializer.encode_component(
            component,
            {
                "ragflow-mysql": "credential-mysql",
                "ragflow-redis": "credential-redis",
                "ragflow-minio": "credential-minio",
                "ragflow-elasticsearch": "credential-elasticsearch",
            },
        )
        refs = encoded["secret_env_refs"]
        self.assertEqual(refs[0]["credential_id"], "credential-mysql")
        self.assertNotIn("environment_id", encoded)
        self.assertNotIn("values", encoded)

    def test_missing_capabilities_block_without_read_calls(self) -> None:
        gateway = FakeGateway({"orbit_list_applications"})
        with self.assertRaises(initializer.GateBlocked) as raised:
            asyncio.run(initializer.check_live_gate(gateway, self.fixture, "project-1", None, "gateway-1", "default"))
        self.assertIn("task1_component_contract_file", raised.exception.blockers)
        self.assertEqual(gateway.calls, [])

    def test_ready_capabilities_only_use_read_tools(self) -> None:
        gateway = FakeGateway(set(initializer.REQUIRED_MCP_TOOLS))
        with tempfile.TemporaryDirectory() as temporary:
            contract_path = Path(temporary) / "contract.json"
            contract_path.write_text(
                initializer.json.dumps({"version_component_fields": sorted(initializer.REQUIRED_COMPONENT_FIELDS)}),
                encoding="utf-8",
            )
            result = asyncio.run(
                initializer.check_live_gate(gateway, self.fixture, "project-1", contract_path, "gateway-1", "default")
            )
        self.assertEqual(result["status"], "ready")
        self.assertEqual(
            [call[0] for call in gateway.calls],
            ["orbit_list_applications", "orbit_list_runtime_env_credentials", "runtime_doctor"],
        )
        self.assertEqual(
            gateway.calls[-1][1],
            {"gateway_application_id": "gateway-1", "gateway_instance_key": "default"},
        )

    def test_recorded_resources_are_allowed_only_when_ids_match(self) -> None:
        gateway = FakeGateway(
            set(initializer.REQUIRED_MCP_TOOLS),
            applications=[{"id": "application-mysql", "code": "ragflow-mysql"}],
            credentials=[{"id": "credential-mysql", "name": "ragflow-mysql"}],
        )
        with tempfile.TemporaryDirectory() as temporary:
            contract_path = Path(temporary) / "contract.json"
            contract_path.write_text(
                initializer.json.dumps({"version_component_fields": sorted(initializer.REQUIRED_COMPONENT_FIELDS)}),
                encoding="utf-8",
            )
            result = asyncio.run(
                initializer.check_live_gate(
                    gateway,
                    self.fixture,
                    "project-1",
                    contract_path,
                    "gateway-1",
                    "default",
                    {
                        "applications": {"ragflow-mysql": "application-mysql"},
                        "credentials": {"ragflow-mysql": "credential-mysql"},
                    },
                )
            )
        self.assertEqual(result["status"], "ready")

    def test_missing_external_network_proof_blocks_live_gate(self) -> None:
        gateway = FakeGateway(set(initializer.REQUIRED_MCP_TOOLS), doctor={"healthy": True})
        with tempfile.TemporaryDirectory() as temporary:
            contract_path = Path(temporary) / "contract.json"
            contract_path.write_text(
                initializer.json.dumps({"version_component_fields": sorted(initializer.REQUIRED_COMPONENT_FIELDS)}),
                encoding="utf-8",
            )
            with self.assertRaises(initializer.GateBlocked) as raised:
                asyncio.run(
                    initializer.check_live_gate(
                        gateway,
                        self.fixture,
                        "project-1",
                        contract_path,
                        "gateway-1",
                        "default",
                    )
                )
        self.assertIn("runtime_doctor:external_network:traefik", raised.exception.blockers)

    def test_journal_rebases_unpublished_draft_fixture(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary) / "run"
            directory.mkdir()
            journal = initializer.RunJournal.create(directory, self.fixture, {"project_id": "project-1"})
            changed = copy.deepcopy(self.fixture)
            changed["payload_encoding"]["secret_env_refs"] = "structured secret reference array"
            resumed = initializer.RunJournal.resume(journal.directory, changed)
        self.assertTrue(resumed.needs_fixture_reconciliation)
        self.assertEqual(resumed.state["fixture_digest"], initializer.fixture_digest(changed))

    def test_journal_refuses_rebase_after_publish(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary) / "run"
            directory.mkdir()
            journal = initializer.RunJournal.create(directory, self.fixture, {"project_id": "project-1"})
            journal.step("publish:ragflow-mysql", "complete")
            changed = copy.deepcopy(self.fixture)
            changed["payload_encoding"]["secret_env_refs"] = "structured secret reference array"
            with self.assertRaises(initializer.FixtureError):
                initializer.RunJournal.resume(journal.directory, changed)

    def test_reconcile_rebased_versions_updates_each_draft(self) -> None:
        gateway = FakeGateway(set(initializer.REQUIRED_MCP_TOOLS))
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary) / "run"
            directory.mkdir()
            journal = initializer.RunJournal.create(directory, self.fixture, {"instance_key": "default"})
            for application in self.fixture["applications"]:
                code = application["code"]
                journal.resource("versions", code, f"version-{code}")
            for credential in self.fixture["credential_templates"]:
                journal.resource("credentials", credential["name"], f"credential-{credential['name']}")
            journal.state["fixture_reconciliation"] = {"status": "pending"}
            journal._write_state()
            asyncio.run(initializer.reconcile_draft_versions(gateway, journal, self.fixture, "local", 9380))
        updates = [arguments for name, arguments in gateway.calls if name == "orbit_update_version"]
        self.assertEqual(len(updates), 5)
        self.assertFalse(journal.needs_fixture_reconciliation)

    def test_explicit_run_directory_requires_new_path(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary) / "new-run"
            self.assertEqual(initializer.explicit_run_directory(directory), directory)
            with self.assertRaises(initializer.InitializerError):
                initializer.explicit_run_directory(directory)

    def test_ragflow_probe_is_defined_for_the_ragflow_component(self) -> None:
        probe = initializer.ragflow_http_probe(self.fixture)
        self.assertEqual(probe, {"component_name": "ragflow-cpu", "port": 80, "path": "/"})

    def test_mcp_child_environment_exposes_only_docker_plugin_discovery_path(self) -> None:
        with patch.dict(
            initializer.os.environ,
            {"ProgramFiles": r"C:\Program Files", "POMELO_ORBIT_PASSWORD": "must-not-be-inherited"},
            clear=True,
        ):
            self.assertEqual(initializer.mcp_child_environment(), {"ProgramFiles": r"C:\Program Files"})

    def test_ragflow_verification_records_managed_http_probe(self) -> None:
        gateway = FakeGateway(set(initializer.REQUIRED_MCP_TOOLS))
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary) / "run"
            directory.mkdir()
            journal = initializer.RunJournal.create(directory, self.fixture, {"instance_key": "default"})
            journal.resource("applications", "ragflow", "application-ragflow")
            journal.resource("deployments", "ragflow", "deployment-ragflow")
            asyncio.run(initializer.wait_and_verify_one(gateway, journal, self.fixture, "ragflow"))
        self.assertEqual(
            [call[0] for call in gateway.calls],
            ["orbit_wait_deployment", "verify_deployment", "runtime_http_probe"],
        )
        self.assertEqual(
            gateway.calls[-1][1],
            {
                "application_id": "application-ragflow",
                "component_name": "ragflow-cpu",
                "port": 80,
                "path": "/",
                "instance_key": "default",
            },
        )


if __name__ == "__main__":
    unittest.main()
