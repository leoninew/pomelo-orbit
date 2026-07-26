#!/usr/bin/env python3
"""Initialize the split RAGFlow deployment through the local stdio MCP server.

The fixture is intentionally non-secret. `plan` is offline, while `check`,
`apply`, and `verify` use only MCP tools. Docker lifecycle commands and direct
Orbit HTTP calls are deliberately absent from this orchestrator.
"""

from __future__ import annotations

import argparse
import asyncio
import hashlib
import json
import os
import sys
import tempfile
from collections.abc import AsyncIterator, Mapping, Sequence
from contextlib import asynccontextmanager
from datetime import UTC, datetime
from pathlib import Path
from typing import Any, Protocol

import yaml
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client

REPO_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_FIXTURE = REPO_ROOT / "docs" / "guides" / "ragflow-split-deployment.fixture.yaml"
DEFAULT_RUN_ROOT = REPO_ROOT / "data" / "ragflow" / "runs"
REQUIRED_COMPONENT_FIELDS = frozenset({"restart_policy", "tmpfs_json", "ulimits_json", "secret_env_refs"})
REQUIRED_EXTERNAL_NETWORK = "traefik"
REQUIRED_MCP_TOOLS = frozenset(
    {
        "orbit_list_applications",
        "orbit_bootstrap_application",
        "orbit_update_version",
        "orbit_create_runtime_env_credential",
        "orbit_list_runtime_env_credentials",
        "orbit_preview_version",
        "orbit_publish_version",
        "orbit_deploy",
        "orbit_wait_deployment",
        "runtime_doctor",
        "runtime_http_probe",
        "verify_deployment",
    }
)


class InitializerError(Exception):
    """Base error that is safe to render to the operator."""

    code = "initializer_error"


class FixtureError(InitializerError):
    code = "invalid_fixture"


class GateBlocked(InitializerError):
    code = "blocked"

    def __init__(self, blockers: Sequence[str]):
        super().__init__("required capabilities or preconditions are unavailable")
        self.blockers = sorted(set(blockers))


class ToolCallError(InitializerError):
    code = "mcp_tool_failed"

    def __init__(self, tool_name: str):
        super().__init__(f"MCP tool failed: {tool_name}")
        self.tool_name = tool_name


class ToolGateway(Protocol):
    async def list_tool_names(self) -> set[str]: ...

    async def call(self, name: str, arguments: Mapping[str, Any]) -> dict[str, Any]: ...


def utc_now() -> str:
    return datetime.now(UTC).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def canonical_json(value: Any) -> str:
    return json.dumps(value, ensure_ascii=True, sort_keys=True, separators=(",", ":"))


def fixture_digest(fixture: Mapping[str, Any]) -> str:
    return hashlib.sha256(canonical_json(fixture).encode("utf-8")).hexdigest()


def load_yaml_mapping(path: Path) -> dict[str, Any]:
    try:
        raw = yaml.safe_load(path.read_text(encoding="utf-8"))
    except (OSError, yaml.YAMLError) as error:
        raise FixtureError("fixture could not be read") from error
    if not isinstance(raw, dict):
        raise FixtureError("fixture root must be an object")
    return raw


def as_mapping(value: Any, label: str) -> dict[str, Any]:
    if not isinstance(value, dict):
        raise FixtureError(f"{label} must be an object")
    return value


def as_list(value: Any, label: str) -> list[Any]:
    if not isinstance(value, list):
        raise FixtureError(f"{label} must be an array")
    return value


def string_value(value: Any, label: str) -> str:
    if not isinstance(value, str) or not value:
        raise FixtureError(f"{label} must be a non-empty string")
    return value


def contains_key(value: Any, prohibited: str) -> bool:
    if isinstance(value, dict):
        return prohibited in value or any(contains_key(item, prohibited) for item in value.values())
    if isinstance(value, list):
        return any(contains_key(item, prohibited) for item in value)
    return False


def validate_fixture(fixture: Mapping[str, Any]) -> None:
    if fixture.get("schema_version") != 1:
        raise FixtureError("schema_version must be 1")
    if fixture.get("kind") != "pomelo-orbit-ragflow-desired-state":
        raise FixtureError("unexpected fixture kind")
    if contains_key(fixture, "environment_id"):
        raise FixtureError("fixture must not contain environment_id")

    templates = as_list(fixture.get("credential_templates"), "credential_templates")
    credential_keys: dict[str, set[str]] = {}
    for index, raw_template in enumerate(templates):
        template = as_mapping(raw_template, f"credential_templates[{index}]")
        name = string_value(template.get("name"), f"credential_templates[{index}].name")
        if template.get("type") != "runtime_env":
            raise FixtureError(f"credential {name} must have type runtime_env")
        if "values" in template:
            raise FixtureError(f"credential {name} must not contain values")
        keys = {
            string_value(item, f"credential {name}.keys")
            for item in as_list(template.get("keys"), f"credential {name}.keys")
        }
        if not keys:
            raise FixtureError(f"credential {name} must declare at least one key")
        credential_keys[name] = keys

    applications = as_list(fixture.get("applications"), "applications")
    expected_codes = {
        "ragflow-mysql",
        "ragflow-redis",
        "ragflow-minio",
        "ragflow-elasticsearch",
        "ragflow",
    }
    codes: set[str] = set()
    for index, raw_application in enumerate(applications):
        application = as_mapping(raw_application, f"applications[{index}]")
        code = string_value(application.get("code"), f"applications[{index}].code")
        if code in codes:
            raise FixtureError(f"duplicate application code {code}")
        codes.add(code)
        components = as_list(application.get("components"), f"application {code}.components")
        if len(components) != 1:
            raise FixtureError(f"application {code} must contain exactly one component")
        component = as_mapping(components[0], f"application {code}.components[0]")
        string_value(component.get("name"), f"application {code}.component.name")
        string_value(component.get("image"), f"application {code}.component.image")
        for raw_ref in as_list(component.get("secret_env_refs", []), f"application {code}.secret_env_refs"):
            ref = as_mapping(raw_ref, f"application {code}.secret_env_ref")
            credential_name = string_value(ref.get("credential_name"), "secret_env_ref.credential_name")
            data_key = string_value(ref.get("data_key"), "secret_env_ref.data_key")
            string_value(ref.get("env_key"), "secret_env_ref.env_key")
            if data_key not in credential_keys.get(credential_name, set()):
                raise FixtureError(f"application {code} references unknown credential key {credential_name}.{data_key}")
        if code != "ragflow" and application.get("exposes") != []:
            raise FixtureError(f"dependency application {code} must not define exposes")

    if codes != expected_codes:
        raise FixtureError("fixture must define exactly the five selected RAGFlow applications")
    ragflow = next(
        as_mapping(item, "ragflow application")
        for item in applications
        if as_mapping(item, "application").get("code") == "ragflow"
    )
    variants = as_mapping(ragflow.get("expose_variants"), "ragflow.expose_variants")
    if set(variants) != {"local", "public"}:
        raise FixtureError("ragflow must define local and public expose variants")
    gate = as_mapping(ragflow.get("readiness_gate"), "ragflow.readiness_gate")
    if gate.get("status") != "required_after_deploy":
        raise FixtureError("ragflow readiness gate must remain required_after_deploy")
    probe = as_mapping(gate.get("http_probe"), "ragflow.readiness_gate.http_probe")
    component = as_mapping(as_list(ragflow["components"], "ragflow.components")[0], "ragflow.component")
    if probe.get("component_name") != component.get("name"):
        raise FixtureError("ragflow readiness probe must target the ragflow component")
    port = probe.get("port")
    if not isinstance(port, int) or not 1 <= port <= 65_535:
        raise FixtureError("ragflow readiness probe port must be between 1 and 65535")
    path = probe.get("path")
    if not isinstance(path, str) or not path.startswith("/") or len(path) > 512 or any(item.isspace() for item in path):
        raise FixtureError("ragflow readiness probe path is invalid")


def planned_operations(fixture: Mapping[str, Any]) -> list[dict[str, str]]:
    applications = as_list(fixture["applications"], "applications")
    operations = [{"id": "check", "operation": "check MCP capabilities and target conflicts"}]
    for credential in as_list(fixture["credential_templates"], "credential_templates"):
        name = string_value(as_mapping(credential, "credential").get("name"), "credential.name")
        operations.append({"id": f"credential:{name}", "operation": "create runtime_env credential"})
    for application in applications:
        app = as_mapping(application, "application")
        code = string_value(app.get("code"), "application.code")
        operations.extend(
            [
                {"id": f"bootstrap:{code}", "operation": "create application and initial version"},
                {"id": f"preview:{code}", "operation": "preview and inspect redacted compose"},
                {"id": f"publish:{code}", "operation": "publish version"},
                {"id": f"deploy:{code}", "operation": "deploy application"},
                {"id": f"verify:{code}", "operation": "wait and verify deployment"},
            ]
        )
    return operations


def make_run_directory(root: Path) -> Path:
    name = datetime.now(UTC).strftime("%Y%m%dT%H%M%SZ")
    candidate = root / name
    suffix = 1
    while candidate.exists():
        suffix += 1
        candidate = root / f"{name}-{suffix}"
    candidate.mkdir(parents=True, exist_ok=False)
    return candidate


def explicit_run_directory(path: Path) -> Path:
    if path.exists():
        raise InitializerError("run directory already exists; use --resume to continue it")
    path.mkdir(parents=True, exist_ok=False)
    return path


class RunJournal:
    def __init__(self, directory: Path, state: dict[str, Any]):
        self.directory = directory
        self.state_path = directory / "state.json"
        self.events_path = directory / "events.jsonl"
        self.state = state

    @classmethod
    def create(cls, directory: Path, fixture: Mapping[str, Any], parameters: Mapping[str, str]) -> RunJournal:
        operations = planned_operations(fixture)
        state = {
            "schema_version": 1,
            "fixture_name": DEFAULT_FIXTURE.name,
            "fixture_digest": fixture_digest(fixture),
            "plan_digest": hashlib.sha256(canonical_json(operations).encode("utf-8")).hexdigest(),
            "parameters": dict(parameters),
            "created_at": utc_now(),
            "updated_at": utc_now(),
            "steps": {item["id"]: {"status": "pending"} for item in operations},
            "resources": {"credentials": {}, "applications": {}, "versions": {}, "deployments": {}},
        }
        journal = cls(directory, state)
        journal._write_state()
        journal.event("run_created", {"operation_count": len(operations)})
        return journal

    @classmethod
    def resume(cls, directory: Path, fixture: Mapping[str, Any]) -> RunJournal:
        try:
            state = json.loads((directory / "state.json").read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError) as error:
            raise InitializerError("resume state could not be read") from error
        if "values" in canonical_json(state).lower():
            raise InitializerError("resume state contains a prohibited secret value field")
        resumed_state = as_mapping(state, "resume state")
        current_digest = fixture_digest(fixture)
        if resumed_state.get("fixture_digest") == current_digest:
            return cls(directory, resumed_state)

        steps = as_mapping(resumed_state.get("steps"), "resume steps")
        deployments = as_mapping(as_mapping(resumed_state.get("resources"), "resume resources").get("deployments"), "deployments")
        has_published_version = any(
            as_mapping(step, "resume step").get("status") == "complete"
            for step_id, step in steps.items()
            if str(step_id).startswith("publish:")
        )
        if deployments or has_published_version:
            raise FixtureError("cannot rebase a run after a Version is published or deployed")

        previous_digest = str(resumed_state.get("fixture_digest") or "")
        resumed_state["fixture_digest"] = current_digest
        resumed_state["plan_digest"] = hashlib.sha256(
            canonical_json(planned_operations(fixture)).encode("utf-8")
        ).hexdigest()
        resumed_state["fixture_reconciliation"] = {
            "status": "pending",
            "from_fixture_digest": previous_digest,
        }
        journal = cls(directory, resumed_state)
        journal._write_state()
        journal.event("fixture_rebased", {"from_fixture_digest": previous_digest, "to_fixture_digest": current_digest})
        return journal

    @property
    def needs_fixture_reconciliation(self) -> bool:
        reconciliation = as_mapping(self.state.get("fixture_reconciliation"), "fixture reconciliation")
        return reconciliation.get("status") == "pending"

    def complete_fixture_reconciliation(self) -> None:
        reconciliation = as_mapping(self.state.get("fixture_reconciliation"), "fixture reconciliation")
        reconciliation["status"] = "complete"
        reconciliation["updated_at"] = utc_now()
        self.state["updated_at"] = utc_now()
        self._write_state()
        self.event("fixture_reconciliation", {"status": "complete"})

    def event(self, event: str, details: Mapping[str, Any]) -> None:
        record = {"time": utc_now(), "event": event, "details": dict(details)}
        with self.events_path.open("a", encoding="utf-8", newline="\n") as handle:
            handle.write(canonical_json(record) + "\n")

    def step(self, step_id: str, status: str, details: Mapping[str, Any] | None = None) -> None:
        step = as_mapping(self.state["steps"].get(step_id), f"step {step_id}")
        step["status"] = status
        step["updated_at"] = utc_now()
        if details:
            step["details"] = dict(details)
        self.state["updated_at"] = utc_now()
        self._write_state()
        self.event("step", {"id": step_id, "status": status, **(dict(details) if details else {})})

    def resource(self, category: str, name: str, resource_id: str) -> None:
        resources = as_mapping(self.state["resources"], "resources")
        as_mapping(resources.get(category), f"resources.{category}")[name] = resource_id
        self.state["updated_at"] = utc_now()
        self._write_state()

    def _write_state(self) -> None:
        descriptor, temp_name = tempfile.mkstemp(prefix=".ragflow-run-", suffix=".tmp", dir=self.directory)
        try:
            with os.fdopen(descriptor, "w", encoding="utf-8", newline="\n") as handle:
                handle.write(json.dumps(self.state, ensure_ascii=True, indent=2, sort_keys=True) + "\n")
            Path(temp_name).replace(self.state_path)
        finally:
            temporary = Path(temp_name)
            if temporary.exists():
                temporary.unlink()


class StdioGateway:
    def __init__(self, session: ClientSession):
        self._session = session

    async def list_tool_names(self) -> set[str]:
        response = await self._session.list_tools()
        return {tool.name for tool in response.tools}

    async def call(self, name: str, arguments: Mapping[str, Any]) -> dict[str, Any]:
        response = await self._session.call_tool(name, arguments=dict(arguments))
        if response.isError:
            raise ToolCallError(name)
        if response.structuredContent is not None:
            return dict(response.structuredContent)
        texts = [getattr(block, "text", None) for block in response.content]
        for text in texts:
            if isinstance(text, str):
                try:
                    decoded = json.loads(text)
                except json.JSONDecodeError:
                    continue
                if isinstance(decoded, dict):
                    return decoded
        return {}


def mcp_child_environment() -> dict[str, str]:
    """Preserve only the Windows path needed for Docker Compose plugin discovery."""
    program_files = os.environ.get("ProgramFiles")
    return {"ProgramFiles": program_files} if program_files else {}


@asynccontextmanager
async def open_mcp() -> AsyncIterator[StdioGateway]:
    parameters = StdioServerParameters(
        command=sys.executable,
        args=["-m", "pomelo_orbit_mcp.server"],
        cwd=REPO_ROOT / "mcp",
        env=mcp_child_environment(),
    )
    async with stdio_client(parameters) as (read_stream, write_stream):
        async with ClientSession(read_stream, write_stream) as session:
            await session.initialize()
            yield StdioGateway(session)


def load_capability_contract(path: Path | None) -> set[str]:
    if path is None:
        raise GateBlocked(["task1_component_contract_file"])
    try:
        contract = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        raise GateBlocked(["task1_component_contract_file"]) from error
    if not isinstance(contract, dict):
        raise GateBlocked(["task1_component_contract_file"])
    fields = contract.get("version_component_fields")
    if not isinstance(fields, list) or not all(isinstance(item, str) for item in fields):
        raise GateBlocked(["task1_component_contract_file"])
    return set(fields)


def ragflow_http_probe(fixture: Mapping[str, Any]) -> dict[str, Any]:
    ragflow = next(
        as_mapping(item, "ragflow application")
        for item in as_list(fixture["applications"], "applications")
        if as_mapping(item, "application").get("code") == "ragflow"
    )
    gate = as_mapping(ragflow["readiness_gate"], "ragflow.readiness_gate")
    return dict(as_mapping(gate["http_probe"], "ragflow.readiness_gate.http_probe"))


def recorded_resource_id(
    resources: Mapping[str, Any] | None,
    category: str,
    name: str,
) -> str | None:
    if resources is None:
        return None
    category_resources = resources.get(category)
    if not isinstance(category_resources, Mapping):
        return None
    resource_id = category_resources.get(name)
    return resource_id if isinstance(resource_id, str) and resource_id else None


async def check_live_gate(
    gateway: ToolGateway,
    fixture: Mapping[str, Any],
    project_id: str,
    capability_contract: Path | None,
    gateway_application_id: str,
    gateway_instance_key: str,
    known_resources: Mapping[str, Any] | None = None,
) -> dict[str, Any]:
    available = await gateway.list_tool_names()
    blockers = [f"mcp_tool:{name}" for name in sorted(REQUIRED_MCP_TOOLS - available)]
    try:
        fields = load_capability_contract(capability_contract)
    except GateBlocked as error:
        blockers.extend(error.blockers)
        fields = set()
    blockers.extend(f"component_field:{name}" for name in sorted(REQUIRED_COMPONENT_FIELDS - fields))
    if blockers:
        raise GateBlocked(blockers)

    listed = await gateway.call("orbit_list_applications", {"project_id": project_id, "kind": "standard"})
    applications = listed.get("applications")
    if not isinstance(applications, list):
        raise ToolCallError("orbit_list_applications")
    desired_codes = {
        str(as_mapping(item, "application").get("code")) for item in as_list(fixture["applications"], "applications")
    }
    existing_applications = {
        str(item.get("code")): item
        for item in applications
        if isinstance(item, dict) and isinstance(item.get("code"), str)
    }
    for code in sorted(desired_codes):
        existing = existing_applications.get(code)
        if existing is None:
            continue
        known_id = recorded_resource_id(known_resources, "applications", code)
        if known_id is not None and existing.get("id") == known_id:
            continue
        blockers.append(f"application_conflict:{code}")

    credential_result = await gateway.call("orbit_list_runtime_env_credentials", {"project_id": project_id})
    credentials = credential_result.get("credentials")
    if not isinstance(credentials, list):
        raise ToolCallError("orbit_list_runtime_env_credentials")
    existing_credentials = {
        str(item.get("name")): item
        for item in credentials
        if isinstance(item, dict) and isinstance(item.get("name"), str)
    }
    for raw_credential in as_list(fixture["credential_templates"], "credential_templates"):
        credential = as_mapping(raw_credential, "credential")
        name = string_value(credential.get("name"), "credential.name")
        existing = existing_credentials.get(name)
        if existing is None:
            continue
        known_id = recorded_resource_id(known_resources, "credentials", name)
        if known_id is not None and existing.get("id") == known_id:
            continue
        blockers.append(f"credential_conflict:{name}")

    doctor = await gateway.call(
        "runtime_doctor",
        {
            "gateway_application_id": gateway_application_id,
            "gateway_instance_key": gateway_instance_key,
        },
    )
    if doctor.get("healthy") is not True:
        blockers.append("runtime_doctor:unhealthy")
    networks = doctor.get("external_networks")
    if not isinstance(networks, list) or REQUIRED_EXTERNAL_NETWORK not in networks:
        blockers.append(f"runtime_doctor:external_network:{REQUIRED_EXTERNAL_NETWORK}")
    if blockers:
        raise GateBlocked(blockers)
    return {
        "status": "ready",
        "available_tools": len(available),
        "conflicts": [],
        "runtime_doctor": {"healthy": True, "external_network": REQUIRED_EXTERNAL_NETWORK},
    }


def load_secrets(path: Path, fixture: Mapping[str, Any]) -> dict[str, dict[str, str]]:
    try:
        resolved = path.resolve(strict=True)
    except OSError as error:
        raise InitializerError("secrets file could not be read") from error
    if resolved.is_relative_to(REPO_ROOT):
        raise InitializerError("secrets file must be outside the repository")
    try:
        raw = json.loads(resolved.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        raise InitializerError("secrets file must be valid JSON") from error
    if not isinstance(raw, dict):
        raise InitializerError("secrets file root must be an object")
    expected: dict[str, set[str]] = {}
    for item in as_list(fixture["credential_templates"], "credential_templates"):
        template = as_mapping(item, "credential template")
        expected[string_value(template["name"], "credential name")] = set(as_list(template["keys"], "credential keys"))
    if set(raw) != set(expected):
        raise InitializerError("secrets file credential names do not match fixture")
    values: dict[str, dict[str, str]] = {}
    for credential_name, keys in expected.items():
        credential_values = raw.get(credential_name)
        if not isinstance(credential_values, dict) or set(credential_values) != keys:
            raise InitializerError("secrets file keys do not match fixture")
        if not all(isinstance(value, str) and value for value in credential_values.values()):
            raise InitializerError("secrets file values must be non-empty strings")
        values[credential_name] = {key: str(value) for key, value in credential_values.items()}
    return values


def encode_component(component: Mapping[str, Any], credential_ids: Mapping[str, str]) -> dict[str, Any]:
    encoded: dict[str, Any] = {"name": component["name"], "image": component["image"]}
    mapping = {
        "command": "command_json",
        "environment": "env_json",
        "mounts": "mounts_json",
        "healthcheck": "healthcheck_json",
        "resources": "resources_json",
        "tmpfs": "tmpfs_json",
        "ulimits": "ulimits_json",
    }
    for fixture_key, payload_key in mapping.items():
        if fixture_key in component:
            encoded[payload_key] = json.dumps(component[fixture_key], ensure_ascii=True, separators=(",", ":"))
    if "restart_policy" in component:
        encoded["restart_policy"] = component["restart_policy"]
    refs: list[dict[str, str]] = []
    for item in as_list(component.get("secret_env_refs", []), "secret_env_refs"):
        ref = as_mapping(item, "secret_env_ref")
        credential_name = string_value(ref.get("credential_name"), "secret_env_ref.credential_name")
        credential_id = credential_ids.get(credential_name)
        if not credential_id:
            raise InitializerError("credential reference was not created")
        refs.append(
            {
                "env_key": string_value(ref.get("env_key"), "secret_env_ref.env_key"),
                "credential_id": credential_id,
                "data_key": string_value(ref.get("data_key"), "secret_env_ref.data_key"),
            }
        )
    if refs:
        encoded["secret_env_refs"] = refs
    return encoded


def component_for_apply(application: Mapping[str, Any]) -> dict[str, Any]:
    component = dict(as_mapping(as_list(application["components"], "components")[0], "component"))
    if application.get("code") == "ragflow":
        gate = as_mapping(application.get("readiness_gate"), "ragflow.readiness_gate")
        component["healthcheck"] = as_mapping(gate.get("candidate_healthcheck"), "ragflow candidate healthcheck")
    return component


def selected_exposes(application: Mapping[str, Any], access: str, local_http_port: int | None) -> list[dict[str, Any]]:
    if application.get("code") != "ragflow":
        return as_list(application.get("exposes", []), "exposes")
    variants = as_mapping(application.get("expose_variants"), "ragflow.expose_variants")
    selected = [dict(as_mapping(item, "expose")) for item in as_list(variants.get(access), f"expose variant {access}")]
    if access == "local":
        if local_http_port is None:
            raise InitializerError("local_http_port is required for local exposure")
        for expose in selected:
            expose["listen_port"] = local_http_port
    return selected


def identifier(result: Mapping[str, Any], name: str) -> str:
    resource_ids = result.get("resource_ids")
    if isinstance(resource_ids, dict):
        candidate = resource_ids.get(name)
        if isinstance(candidate, str) and candidate:
            return candidate
    value = result.get(name)
    if isinstance(value, str) and value:
        return value
    raise ToolCallError(f"response:{name}")


async def apply_fixture(
    gateway: ToolGateway,
    journal: RunJournal,
    fixture: Mapping[str, Any],
    project_id: str,
    instance_key: str,
    access: str,
    local_http_port: int | None,
    secrets: Mapping[str, Mapping[str, str]],
) -> None:
    resources = as_mapping(journal.state["resources"], "resources")
    credential_ids = as_mapping(resources["credentials"], "resources.credentials")
    for raw_credential in as_list(fixture["credential_templates"], "credential_templates"):
        credential = as_mapping(raw_credential, "credential")
        name = string_value(credential["name"], "credential.name")
        step_id = f"credential:{name}"
        if name in credential_ids:
            continue
        result = await gateway.call(
            "orbit_create_runtime_env_credential",
            {"project_id": project_id, "name": name, "values": dict(secrets[name])},
        )
        credential_id = identifier(result, "credential_id")
        journal.resource("credentials", name, credential_id)
        journal.step(step_id, "complete", {"credential_id": credential_id})

    application_ids = as_mapping(resources["applications"], "resources.applications")
    version_ids = as_mapping(resources["versions"], "resources.versions")
    for raw_application in as_list(fixture["applications"], "applications"):
        application = as_mapping(raw_application, "application")
        code = string_value(application["code"], "application.code")
        if code in application_ids and code in version_ids:
            continue
        if code in application_ids or code in version_ids:
            raise GateBlocked([f"incomplete_bootstrap_state:{code}"])
        component = component_for_apply(application)
        result = await gateway.call(
            "orbit_bootstrap_application",
            {
                "project_id": project_id,
                "name": string_value(application["name"], f"{code}.name"),
                "code": code,
                "version_label": string_value(application["version_label"], f"{code}.version_label"),
                "components": [encode_component(component, credential_ids)],
                "exposes": selected_exposes(application, access, local_http_port),
                "image_pull_policy": "missing",
                "kind": "standard",
            },
        )
        application_id = identifier(result, "application_id")
        version_id = identifier(result, "version_id")
        journal.resource("applications", code, application_id)
        journal.resource("versions", code, version_id)
        journal.step(f"bootstrap:{code}", "complete", {"application_id": application_id, "version_id": version_id})

    if journal.needs_fixture_reconciliation:
        await reconcile_draft_versions(gateway, journal, fixture, access, local_http_port)

    for raw_application in as_list(fixture["applications"], "applications"):
        application = as_mapping(raw_application, "application")
        code = string_value(application["code"], "application.code")
        application_id = string_value(application_ids.get(code), f"application id {code}")
        version_id = string_value(version_ids.get(code), f"version id {code}")
        if as_mapping(journal.state["steps"], "steps")[f"preview:{code}"].get("status") != "complete":
            await gateway.call("orbit_preview_version", {"version_id": version_id, "instance_key": instance_key})
            journal.step(f"preview:{code}", "complete", {"version_id": version_id})
        if as_mapping(journal.state["steps"], "steps")[f"publish:{code}"].get("status") != "complete":
            await gateway.call("orbit_publish_version", {"version_id": version_id})
            journal.step(f"publish:{code}", "complete", {"version_id": version_id})
        deployments = as_mapping(resources["deployments"], "resources.deployments")
        verification_step = as_mapping(journal.state["steps"], "steps")[f"verify:{code}"]
        if code in deployments and verification_step.get("status") != "complete":
            previous = await gateway.call("orbit_wait_deployment", {"deployment_id": deployments[code]})
            previous_deployment = as_mapping(previous.get("deployment"), "deployment")
            if previous_deployment.get("status") in {"faulted", "canceled"}:
                result = await gateway.call(
                    "orbit_deploy",
                    {"application_id": application_id, "version_id": version_id, "instance_key": instance_key},
                )
                deployment_id = identifier(result, "deployment_id")
                journal.resource("deployments", code, deployment_id)
                journal.step(f"deploy:{code}", "complete", {"deployment_id": deployment_id, "retry": True})
        if code not in deployments:
            result = await gateway.call(
                "orbit_deploy",
                {"application_id": application_id, "version_id": version_id, "instance_key": instance_key},
            )
            deployment_id = identifier(result, "deployment_id")
            journal.resource("deployments", code, deployment_id)
            journal.step(f"deploy:{code}", "complete", {"deployment_id": deployment_id})
        if code != "ragflow":
            await wait_and_verify_one(gateway, journal, fixture, code)


async def reconcile_draft_versions(
    gateway: ToolGateway,
    journal: RunJournal,
    fixture: Mapping[str, Any],
    access: str,
    local_http_port: int | None,
) -> None:
    resources = as_mapping(journal.state["resources"], "resources")
    credential_ids = as_mapping(resources["credentials"], "resources.credentials")
    version_ids = as_mapping(resources["versions"], "resources.versions")
    for raw_application in as_list(fixture["applications"], "applications"):
        application = as_mapping(raw_application, "application")
        code = string_value(application["code"], "application.code")
        version_id = string_value(version_ids.get(code), f"version id {code}")
        await gateway.call(
            "orbit_update_version",
            {
                "version_id": version_id,
                "label": string_value(application["version_label"], f"{code}.version_label"),
                "components": [encode_component(component_for_apply(application), credential_ids)],
                "exposes": selected_exposes(application, access, local_http_port),
            },
        )
    journal.complete_fixture_reconciliation()


async def wait_and_verify_one(
    gateway: ToolGateway,
    journal: RunJournal,
    fixture: Mapping[str, Any],
    code: str,
) -> None:
    resources = as_mapping(journal.state["resources"], "resources")
    applications = as_mapping(resources["applications"], "resources.applications")
    deployments = as_mapping(resources["deployments"], "resources.deployments")
    application_id = string_value(applications.get(code), f"application id {code}")
    deployment_id = string_value(deployments.get(code), f"deployment id {code}")
    await gateway.call("orbit_wait_deployment", {"deployment_id": deployment_id})
    verification = await gateway.call(
        "verify_deployment", {"application_id": application_id, "deployment_id": deployment_id}
    )
    conclusion = verification.get("conclusion")
    if conclusion != "consistent":
        raise GateBlocked([f"verification:{code}:{conclusion or 'unknown'}"])
    details: dict[str, Any] = {"deployment_id": deployment_id, "conclusion": conclusion}
    if code == "ragflow":
        probe = ragflow_http_probe(fixture)
        probe_result = await gateway.call(
            "runtime_http_probe",
            {
                "application_id": application_id,
                "component_name": probe["component_name"],
                "port": probe["port"],
                "path": probe["path"],
                "instance_key": str(journal.state["parameters"].get("instance_key", "")),
            },
        )
        if probe_result.get("status") != "reachable":
            raise GateBlocked(["ragflow_http_probe"])
        details["http_probe"] = {
            "status": "reachable",
            "component_name": probe["component_name"],
            "port": probe["port"],
            "path": probe["path"],
        }
    journal.step(f"verify:{code}", "complete", details)


async def verify_deployments(gateway: ToolGateway, journal: RunJournal, fixture: Mapping[str, Any]) -> None:
    steps = as_mapping(journal.state["steps"], "steps")
    for code in ("ragflow-mysql", "ragflow-redis", "ragflow-minio", "ragflow-elasticsearch", "ragflow"):
        if as_mapping(steps.get(f"verify:{code}"), f"verify step {code}").get("status") == "complete":
            continue
        await wait_and_verify_one(gateway, journal, fixture, code)


def output(value: Mapping[str, Any]) -> None:
    print(json.dumps(value, ensure_ascii=True, indent=2, sort_keys=True))


def parser() -> argparse.ArgumentParser:
    result = argparse.ArgumentParser(description=__doc__)
    result.add_argument("--fixture", type=Path, default=DEFAULT_FIXTURE)
    result.add_argument("--run-root", type=Path, default=DEFAULT_RUN_ROOT)
    subparsers = result.add_subparsers(dest="mode", required=True)
    subparsers.add_parser("plan", help="validate the fixture and create an offline run plan")
    for mode in ("check", "apply"):
        command = subparsers.add_parser(mode)
        command.add_argument("--project-id", required=True)
        command.add_argument("--instance-key", required=True)
        command.add_argument("--access", choices=("local", "public"), required=True)
        command.add_argument("--local-http-port", type=int)
        command.add_argument("--capability-contract", type=Path)
        command.add_argument("--gateway-application-id", required=True)
        command.add_argument("--gateway-instance-key", default="default")
        command.add_argument("--run-dir", type=Path)
    apply = subparsers.choices["apply"]
    apply.add_argument("--secrets-file", type=Path, required=True)
    apply.add_argument("--confirm", action="store_true")
    apply.add_argument("--resume", type=Path)
    verify = subparsers.add_parser("verify", help="resume a prior apply run and verify deployments")
    verify.add_argument("--resume", type=Path, required=True)
    verify.add_argument("--capability-contract", type=Path, required=True)
    return result


async def run(arguments: argparse.Namespace) -> dict[str, Any]:
    fixture = load_yaml_mapping(arguments.fixture)
    validate_fixture(fixture)
    digest = fixture_digest(fixture)
    if arguments.mode == "plan":
        journal = RunJournal.create(make_run_directory(arguments.run_root), fixture, {})
        return {
            "status": "planned",
            "fixture_digest": digest,
            "plan_digest": journal.state["plan_digest"],
            "run_directory": str(journal.directory),
            "operations": planned_operations(fixture),
        }

    if arguments.mode == "verify":
        journal = RunJournal.resume(arguments.resume, fixture)
        async with open_mcp() as gateway:
            await check_live_gate(
                gateway,
                fixture,
                str(journal.state["parameters"].get("project_id", "")),
                arguments.capability_contract,
                str(journal.state["parameters"].get("gateway_application_id", "")),
                str(journal.state["parameters"].get("gateway_instance_key", "")),
                as_mapping(journal.state["resources"], "resources"),
            )
            await verify_deployments(gateway, journal, fixture)
        return {"status": "verified", "run_directory": str(journal.directory), "fixture_digest": digest}

    if arguments.access == "local" and arguments.local_http_port is None:
        raise InitializerError("--local-http-port is required when --access=local")
    if arguments.mode == "apply" and not arguments.confirm:
        raise InitializerError("apply requires --confirm")
    parameters = {
        "project_id": arguments.project_id,
        "instance_key": arguments.instance_key,
        "access": arguments.access,
        "gateway_application_id": arguments.gateway_application_id,
        "gateway_instance_key": arguments.gateway_instance_key,
    }
    if arguments.local_http_port is not None:
        parameters["local_http_port"] = str(arguments.local_http_port)
    if arguments.mode == "apply" and arguments.resume is not None:
        if arguments.run_dir is not None:
            raise InitializerError("--run-dir and --resume cannot be used together")
        journal = RunJournal.resume(arguments.resume, fixture)
        if journal.state.get("parameters") != parameters:
            raise InitializerError("resume parameters do not match the existing run")
    else:
        directory = (
            explicit_run_directory(arguments.run_dir) if arguments.run_dir else make_run_directory(arguments.run_root)
        )
        journal = RunJournal.create(directory, fixture, parameters)
    async with open_mcp() as gateway:
        try:
            result = await check_live_gate(
                gateway,
                fixture,
                arguments.project_id,
                arguments.capability_contract,
                arguments.gateway_application_id,
                arguments.gateway_instance_key,
                as_mapping(journal.state["resources"], "resources"),
            )
        except GateBlocked as error:
            journal.step("check", "blocked", {"blockers": error.blockers})
            return {"status": "blocked", "blockers": error.blockers, "run_directory": str(journal.directory)}
        journal.step("check", "complete", result)
        if arguments.mode == "check":
            return {"status": "ready", "run_directory": str(journal.directory), "fixture_digest": digest}
        secrets = load_secrets(arguments.secrets_file, fixture)
        await apply_fixture(
            gateway,
            journal,
            fixture,
            arguments.project_id,
            arguments.instance_key,
            arguments.access,
            arguments.local_http_port,
            secrets,
        )
    return {"status": "applied", "run_directory": str(journal.directory), "fixture_digest": digest}


def main() -> int:
    arguments = parser().parse_args()
    try:
        output(asyncio.run(run(arguments)))
    except GateBlocked as error:
        output({"status": "blocked", "blockers": error.blockers})
        return 2
    except InitializerError as error:
        output({"status": "failed", "code": error.code})
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
