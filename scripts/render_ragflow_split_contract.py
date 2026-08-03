"""Render and check the generated artifacts of the split RAGFlow contract."""

from __future__ import annotations

import argparse
import json
import re
import sys
from collections.abc import Mapping, Sequence
from pathlib import Path
from typing import Any


REPOSITORY_ROOT = Path(__file__).resolve().parent.parent
DEFAULT_CONTRACT_PATH = REPOSITORY_ROOT / "scripts" / "ragflow-split" / "deployment-contract.json"
INTEGRATED_CONTRACT_PATH = REPOSITORY_ROOT / "scripts" / "ragflow-integrated" / "deployment-contract.json"
ALL_CONTRACT_PATHS = (DEFAULT_CONTRACT_PATH, INTEGRATED_CONTRACT_PATH)
SAFE_YAML_STRING = re.compile(r"^[A-Za-z0-9_./@+%=-]+$")
YAML_KEYWORDS = {"null", "true", "false", "yes", "no", "on", "off", "~"}


class ContractError(ValueError):
    """The deployment contract is incomplete or internally inconsistent."""


def repository_root() -> Path:
    return REPOSITORY_ROOT


def load_contract(path: Path = DEFAULT_CONTRACT_PATH) -> dict[str, Any]:
    try:
        document = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError as error:
        raise ContractError(f"contract does not exist: {path}") from error
    except json.JSONDecodeError as error:
        raise ContractError(f"contract is not valid JSON: {error}") from error

    if not isinstance(document, dict):
        raise ContractError("contract root must be an object")
    validate_contract(document)
    return document


def validate_contract(contract: Mapping[str, Any]) -> None:
    required = {
        "schema_version",
        "contract_path",
        "primitive_path",
        "primitive_title",
        "primitive_description",
        "network",
        "compose_header",
        "model_cache_paths",
        "runtime_config",
        "applications",
        "components",
        "compose_targets",
        "manual_provider",
    }
    missing = sorted(required.difference(contract))
    if missing:
        raise ContractError(f"contract is missing required keys: {', '.join(missing)}")
    if contract["schema_version"] != 1:
        raise ContractError("unsupported contract schema_version")
    for key in ("contract_path", "primitive_path", "primitive_title", "primitive_description"):
        require_string(contract[key], key)

    network = require_mapping(contract["network"], "network")
    if not isinstance(network.get("name"), str) or not network["name"]:
        raise ContractError("network.name must be a non-empty string")

    components = require_list(contract["components"], "components")
    applications = require_list(contract["applications"], "applications")
    component_names: set[str] = set()
    application_codes: set[str] = set()
    compose_paths: set[str] = set()

    for application in applications:
        application_mapping = require_mapping(application, "application")
        code = require_string(application_mapping.get("code"), "application.code")
        if code in application_codes:
            raise ContractError(f"duplicate application code: {code}")
        application_codes.add(code)
        require_string(application_mapping.get("service"), f"application {code}.service")
        versions = require_list(application_mapping.get("versions"), f"application {code}.versions")
        if not versions:
            raise ContractError(f"application {code} must declare a Version")
        for version in versions:
            version_mapping = require_mapping(version, f"application {code}.version")
            require_string(version_mapping.get("role"), f"application {code}.version.role")
            require_string(version_mapping.get("label"), f"application {code}.version.label")
            require_list(version_mapping.get("components"), f"application {code}.version.components")
            require_string(
                version_mapping.get("hostname_component"),
                f"application {code}.version.hostname_component",
            )

    for component in components:
        component_mapping = require_mapping(component, "component")
        name = require_string(component_mapping.get("name"), "component.name")
        if name in component_names:
            raise ContractError(f"duplicate component name: {name}")
        component_names.add(name)
        application = require_string(component_mapping.get("application"), f"component {name}.application")
        if application not in application_codes:
            raise ContractError(f"component {name} references unknown Application {application}")
        compose_file = require_string(component_mapping.get("compose_file"), f"component {name}.compose_file")
        compose_paths.add(compose_file)
        require_string(component_mapping.get("service_name"), f"component {name}.service_name")
        if "image" not in component_mapping and "images" not in component_mapping:
            raise ContractError(f"component {name} must define image or images")
        if "images" in component_mapping:
            images = require_mapping(component_mapping["images"], f"component {name}.images")
            if not images:
                raise ContractError(f"component {name}.images must not be empty")
            for profile, image in images.items():
                require_string(profile, f"component {name}.images profile")
                require_string(image, f"component {name}.images.{profile}")
        if "environment" in component_mapping:
            for item in require_list(component_mapping["environment"], f"component {name}.environment"):
                environment = require_mapping(item, f"component {name}.environment item")
                require_string(environment.get("key"), f"component {name}.environment.key")
                if ("value" in environment) == ("runtime_key" in environment):
                    raise ContractError(
                        f"component {name}.environment.{environment['key']} must have exactly one value source"
                    )
        for endpoint in require_list(component_mapping.get("endpoints", []), f"component {name}.endpoints"):
            endpoint_mapping = require_mapping(endpoint, f"component {name}.endpoint")
            for endpoint_key in ("name", "protocol", "container_port", "mode"):
                if endpoint_key not in endpoint_mapping:
                    raise ContractError(f"component {name}.endpoint is missing {endpoint_key}")

    for application in applications:
        application_mapping = require_mapping(application, "application")
        code = application_mapping["code"]
        for version in application_mapping["versions"]:
            version_mapping = require_mapping(version, f"application {code}.version")
            for component_name in version_mapping["components"]:
                if component_name not in component_names:
                    raise ContractError(f"Application {code} references unknown Component {component_name}")
            if version_mapping["hostname_component"] not in version_mapping["components"]:
                raise ContractError(f"Application {code} hostname_component must belong to its Version")

    targets = require_list(contract["compose_targets"], "compose_targets")
    target_paths: set[str] = set()
    base_target_paths: set[str] = set()
    for target in targets:
        target_mapping = require_mapping(target, "compose target")
        path = require_string(target_mapping.get("path"), "compose target.path")
        if path in target_paths:
            raise ContractError(f"duplicate Compose target: {path}")
        target_paths.add(path)
        for component_name in require_list(target_mapping.get("components"), f"Compose target {path}.components"):
            if component_name not in component_names:
                raise ContractError(f"Compose target {path} references unknown Component {component_name}")
            component = component_by_name(contract, component_name)
            if not target_mapping.get("overlay") and component["compose_file"] != path:
                raise ContractError(f"Component {component_name} is assigned to the wrong Compose target")
        if target_mapping.get("overlay") and not target_mapping.get("profile"):
            raise ContractError(f"Compose overlay {path} must declare a profile")
        if not target_mapping.get("overlay"):
            base_target_paths.add(path)
    if base_target_paths != compose_paths:
        raise ContractError("compose_targets must cover every component Compose file exactly once")

    runtime_config = require_mapping(contract["runtime_config"], "runtime_config")
    runtime_keys = set(require_list(runtime_config.get("keys"), "runtime_config.keys"))
    if not all(isinstance(key, str) and key for key in runtime_keys):
        raise ContractError("runtime_config.keys must contain non-empty strings")
    assignments_by_component: dict[str, set[str]] = {}
    for assignment in require_list(runtime_config.get("assignments"), "runtime_config.assignments"):
        assignment_mapping = require_mapping(assignment, "runtime_config.assignment")
        component_name = require_string(assignment_mapping.get("component"), "runtime_config.assignment.component")
        if component_name not in component_names:
            raise ContractError(f"runtime assignment references unknown Component {component_name}")
        if component_name in assignments_by_component:
            raise ContractError(f"duplicate runtime assignment for Component {component_name}")
        assigned_keys = set(require_list(assignment_mapping.get("keys"), "runtime_config.assignment.keys"))
        if not assigned_keys.issubset(runtime_keys):
            raise ContractError(f"runtime assignment for {component_name} contains an unknown key")
        assignments_by_component[component_name] = assigned_keys
    for component in components:
        component_mapping = require_mapping(component, "component")
        component_name = component_mapping["name"]
        configured_keys = {
            item["runtime_key"]
            for item in component_mapping.get("environment", [])
            if "runtime_key" in item
        }
        if not configured_keys.issubset(runtime_keys):
            raise ContractError(f"component {component_name} uses an unknown runtime key")
        if configured_keys != assignments_by_component.get(component_name, set()):
            raise ContractError(
                f"runtime assignment for {component_name} must match its environment runtime keys"
            )


def require_mapping(value: Any, description: str) -> Mapping[str, Any]:
    if not isinstance(value, Mapping):
        raise ContractError(f"{description} must be an object")
    return value


def require_list(value: Any, description: str) -> list[Any]:
    if not isinstance(value, list):
        raise ContractError(f"{description} must be an array")
    return value


def require_string(value: Any, description: str) -> str:
    if not isinstance(value, str) or not value:
        raise ContractError(f"{description} must be a non-empty string")
    return value


def component_by_name(contract: Mapping[str, Any], name: str) -> Mapping[str, Any]:
    for component in contract["components"]:
        if component["name"] == name:
            return require_mapping(component, f"component {name}")
    raise ContractError(f"unknown Component {name}")


def generated_header(contract: Mapping[str, Any]) -> str:
    return (
        "Generated by scripts/render_ragflow_deployment_contract.py "
        f"from {contract['contract_path']}."
    )


def image_for(component: Mapping[str, Any], profile: str | None = None) -> str:
    if "image" in component:
        return require_string(component["image"], f"component {component['name']}.image")
    images = require_mapping(component["images"], f"component {component['name']}.images")
    selected_profile = profile or "cpu"
    image = images.get(selected_profile)
    if image is None:
        raise ContractError(f"component {component['name']} has no image for profile {selected_profile}")
    return require_string(image, f"component {component['name']}.images.{selected_profile}")


def runtime_value(key: str) -> str:
    return f"${{{key}:?{key} must be set}}"


def compose_service(contract: Mapping[str, Any], component: Mapping[str, Any], profile: str | None = None) -> dict[str, Any]:
    service: dict[str, Any] = {"image": image_for(component, profile)}
    if "command" in component:
        service["command"] = component["command"]
    environment: dict[str, str] = {}
    for item in component.get("environment", []):
        if "value" in item:
            environment[item["key"]] = item["value"]
        else:
            environment[item["key"]] = runtime_value(item["runtime_key"])
    if environment:
        service["environment"] = environment
    if "depends_on" in component:
        service["depends_on"] = component["depends_on"]
    mounts = component.get("mounts", [])
    if mounts:
        service["volumes"] = [f"{mount['source']}:{mount['target']}" for mount in mounts]
    for key in ("tmpfs", "ulimits", "deploy"):
        if key in component:
            service[key] = component[key]
    service["networks"] = {
        contract["network"]["name"]: {"aliases": [component["network_alias"]]}
    }
    if "healthcheck" in component:
        service["healthcheck"] = component["healthcheck"]
    service["restart"] = component["restart"]
    return service


def compose_overlay_service(component: Mapping[str, Any], profile: str) -> dict[str, Any]:
    profile_overrides = require_mapping(component.get("profile_overrides", {}), f"component {component['name']}.profile_overrides")
    override = require_mapping(profile_overrides.get(profile), f"component {component['name']}.{profile} override")
    service: dict[str, Any] = {"image": image_for(component, profile)}
    for key, value in override.items():
        service[key] = value
    return service


def render_compose(contract: Mapping[str, Any], target: Mapping[str, Any]) -> str:
    services: dict[str, Any] = {}
    volumes: dict[str, Any] = {}
    profile = target.get("profile")
    for component_name in target["components"]:
        component = component_by_name(contract, component_name)
        if target.get("overlay"):
            services[component["service_name"]] = compose_overlay_service(component, profile)
        else:
            services[component["service_name"]] = compose_service(contract, component, profile)
            for mount in component.get("mounts", []):
                if mount["type"] == "volume":
                    volumes[mount["source"]] = {}

    document: dict[str, Any] = {"services": services}
    if volumes:
        document["volumes"] = volumes
    if not target.get("overlay"):
        document["networks"] = {
            contract["network"]["name"]: {"external": contract["network"]["external"]}
        }

    header = [generated_header(contract), "Do not edit this file directly.", *contract["compose_header"], *target.get("header", [])]
    return "".join(f"# {line}\n" for line in header) + dump_yaml(document)


def yaml_scalar(value: Any) -> str:
    if value is None:
        return "null"
    if value is True:
        return "true"
    if value is False:
        return "false"
    if isinstance(value, (int, float)):
        return str(value)
    if not isinstance(value, str):
        raise ContractError(f"cannot encode YAML scalar of type {type(value).__name__}")
    if (
        SAFE_YAML_STRING.fullmatch(value)
        and value.lower() not in YAML_KEYWORDS
        and not value.isdigit()
        and value != "%"
    ):
        return value
    return json.dumps(value, ensure_ascii=False)


def dump_yaml(value: Any, indent: int = 0) -> str:
    prefix = " " * indent
    if isinstance(value, Mapping):
        if not value:
            return f"{prefix}{{}}\n"
        lines: list[str] = []
        for key, item in value.items():
            if isinstance(item, Mapping):
                if item:
                    lines.append(f"{prefix}{key}:\n")
                    lines.append(dump_yaml(item, indent + 2))
                else:
                    lines.append(f"{prefix}{key}: {{}}\n")
            elif is_sequence(item):
                if item:
                    lines.append(f"{prefix}{key}:\n")
                    lines.append(dump_yaml(item, indent + 2))
                else:
                    lines.append(f"{prefix}{key}: []\n")
            else:
                lines.append(f"{prefix}{key}: {yaml_scalar(item)}\n")
        return "".join(lines)
    if is_sequence(value):
        lines = []
        for item in value:
            if isinstance(item, Mapping) or is_sequence(item):
                lines.append(f"{prefix}-\n")
                lines.append(dump_yaml(item, indent + 2))
            else:
                lines.append(f"{prefix}- {yaml_scalar(item)}\n")
        return "".join(lines)
    return f"{prefix}{yaml_scalar(value)}\n"


def is_sequence(value: Any) -> bool:
    return isinstance(value, Sequence) and not isinstance(value, (str, bytes, bytearray))


def markdown_code(value: object) -> str:
    return f"`{value}`"


def markdown_cell(value: object) -> str:
    return str(value).replace("|", "\\|").replace("\n", "<br>")


def markdown_table(headers: Sequence[str], rows: Sequence[Sequence[object]]) -> str:
    output = ["| " + " | ".join(headers) + " |", "| " + " | ".join("---" for _ in headers) + " |"]
    output.extend("| " + " | ".join(markdown_cell(cell) for cell in row) + " |" for row in rows)
    return "\n".join(output) + "\n"


def component_mounts(component: Mapping[str, Any]) -> str:
    mounts = component.get("mounts", [])
    if not mounts:
        return "none"
    return "; ".join(
        f"{mount['type']} {markdown_code(mount['logical_source'])} -> {markdown_code(mount['target'])}"
        for mount in mounts
    )


def component_image(component: Mapping[str, Any]) -> str:
    if "image" in component:
        return markdown_code(component["image"])
    return "; ".join(f"{profile}: {markdown_code(image)}" for profile, image in component["images"].items())


def render_primitive(contract: Mapping[str, Any]) -> str:
    lines = [
        f"# {contract['primitive_title']}\n",
        f"> {generated_header(contract)} Do not edit this file directly.\n",
        "\n",
        f"{contract['primitive_description']}\n",
        "\n",
        "## Topology\n",
        "\n",
    ]
    topology_rows: list[list[str]] = []
    for application in contract["applications"]:
        for version in application["versions"]:
            hostname_component = component_by_name(contract, version["hostname_component"])
            topology_rows.append(
                [
                    markdown_code(version["role"]),
                    markdown_code(application["code"]),
                    markdown_code(version["label"]),
                    ", ".join(markdown_code(component) for component in version["components"]),
                    markdown_code(application["service"]),
                    markdown_code(hostname_component["network_alias"]),
                ]
            )
    lines.append(
        markdown_table(
            ["Role", "Application code", "Version label", "Components", "Service", "External network hostname"],
            topology_rows,
        )
    )
    lines.extend(
        [
            "\n",
            "Every Component joins the external `traefik` network. Hostnames and Version labels above are deployment contracts, not compatibility suggestions.\n",
            "\n",
            "## Model Caches\n",
            "\n",
            "Run the selected CPU or GPU TEI preflight against every path below before deployment:\n",
            "\n",
            *[f"- {markdown_code(path)}\n" for path in contract["model_cache_paths"]],
            "\n",
            "## Runtime Config Primitive\n",
            "\n",
            f"{contract['runtime_config']['name']} = {{ " + ", ".join(contract["runtime_config"]["keys"]) + " }\n",
            "\n",
            "Runtime values are never stored in this contract. Create a single in-memory value object for a new complete resource set; retain matching authorized values when reusing a resource.\n",
            "\n",
        ]
    )
    assignments_by_service: dict[str, set[str]] = {}
    for assignment in contract["runtime_config"]["assignments"]:
        component = component_by_name(contract, assignment["component"])
        service = f"{component['application']}/default"
        assignments_by_service.setdefault(service, set()).update(assignment["keys"])
    assignment_rows = [
        [
            "/".join(markdown_code(part) for part in service.split("/")),
            ", ".join(markdown_code(key) for key in contract["runtime_config"]["keys"] if key in keys),
        ]
        for service, keys in assignments_by_service.items()
    ]
    lines.append(markdown_table(["Service", "Required runtime keys"], assignment_rows))

    lines.extend(["\n", "## Component Primitive\n", "\n"])
    component_rows: list[list[str]] = []
    for component in contract["components"]:
        command = " ".join(component.get("command", [])) or "default"
        component_rows.append(
            [
                markdown_code(component["name"]),
                component_image(component),
                markdown_code(command),
                component_mounts(component),
                markdown_code(component["network_alias"]),
            ]
        )
    lines.append(markdown_table(["Component", "Image", "Command", "Logical mount", "Network alias"], component_rows))

    lines.extend(["\n", "### Component environment\n", "\n"])
    environment_rows: list[list[str]] = []
    for component in contract["components"]:
        environment = component.get("environment", [])
        if not environment:
            continue
        fixed = ", ".join(
            f"{item['key']}={item['value']}" for item in environment if "value" in item
        ) or "none"
        runtime = ", ".join(markdown_code(item["runtime_key"]) for item in environment if "runtime_key" in item) or "none"
        environment_rows.append([markdown_code(component["name"]), fixed, runtime])
    lines.append(markdown_table(["Component", "Fixed environment", "Runtime keys"], environment_rows))

    lines.extend(["\n", "### Health and special settings\n", "\n"])
    health_rows: list[list[str]] = []
    for component in contract["components"]:
        healthcheck = component.get("healthcheck", {})
        test = " ".join(healthcheck.get("test", [])) or "none"
        settings: list[str] = []
        if "tmpfs" in component:
            settings.append("tmpfs " + ", ".join(component["tmpfs"]))
        if "ulimits" in component:
            settings.append("ulimits " + json.dumps(component["ulimits"], separators=(",", ":")))
        if "deploy" in component:
            settings.append("resources " + json.dumps(component["deploy"], separators=(",", ":")))
        if "profile_overrides" in component:
            settings.append("profile overrides " + json.dumps(component["profile_overrides"], separators=(",", ":")))
        health_rows.append([markdown_code(component["name"]), markdown_code(test), "; ".join(settings) or "none"])
    lines.append(markdown_table(["Component", "Health check", "Special settings"], health_rows))

    lines.extend(["\n", "## Endpoint Primitive\n", "\n", "```json\n"])
    endpoint_data = [
        {"component_name": component["name"], **endpoint}
        for component in contract["components"]
        for endpoint in component.get("endpoints", [])
    ]
    lines.extend([json.dumps(endpoint_data, indent=2), "\n```\n", "\n"])

    provider = contract["manual_provider"]
    lines.extend(
        [
            "## Manual Provider Initialization\n",
            "\n",
            "Configure this HuggingFace embedding provider manually in the RAGFlow UI after the selected Version is healthy:\n",
            "\n",
            markdown_table(
                ["Name", "Model", "Base URL", "Max tokens"],
                [[markdown_code(provider["name"]), markdown_code(provider["model"]), markdown_code(provider["tei_base_url"]), provider["max_tokens"]]],
            ),
            "\n",
            "`TEI_BASE_URL` is a manual Provider setting, not a `ragflow-cpu` container environment variable.\n",
            "\n",
            "## MCP Workflow Primitive\n",
            "\n",
            "```text\n",
            "discover:  orbit_list_projects -> orbit_list_applications -> orbit_get_application\n",
            "           -> orbit_list_versions / orbit_get_version -> orbit_list_application_services\n",
            "preflight: runtime_doctor(network_name=\"traefik\")\n",
            "write:     orbit_create_application -> orbit_create_version\n",
            "           -> orbit_create_service -> orbit_preview_service -> orbit_deploy\n",
            "verify:    orbit_wait_deployment -> runtime_compose_ps -> runtime_http_probe\n",
            "           -> verify_deployment\n",
            "```\n",
            "\n",
            "Inspect each write tool schema immediately before use. Use `orbit_update_*` only to repair the matching target resource, and sanitize tool results before reporting them.\n",
        ]
    )
    return "".join(lines)


def render_all(contract: Mapping[str, Any]) -> dict[Path, str]:
    rendered = {
        Path(target["path"]): render_compose(contract, target)
        for target in contract["compose_targets"]
    }
    rendered[Path(contract["primitive_path"])] = render_primitive(contract)
    return rendered


def render_contracts(contracts: Sequence[Mapping[str, Any]]) -> dict[Path, str]:
    rendered: dict[Path, str] = {}
    for contract in contracts:
        for path, content in render_all(contract).items():
            if path in rendered:
                raise ContractError(f"multiple contracts generate {path.as_posix()}")
            rendered[path] = content
    return rendered


def write_outputs(rendered: Mapping[Path, str], root: Path = REPOSITORY_ROOT) -> None:
    for relative_path, content in rendered.items():
        target = root / relative_path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content, encoding="utf-8", newline="\n")


def check_outputs(rendered: Mapping[Path, str], root: Path = REPOSITORY_ROOT) -> list[Path]:
    drifted: list[Path] = []
    for relative_path, expected in rendered.items():
        target = root / relative_path
        if not target.is_file() or target.read_text(encoding="utf-8") != expected:
            drifted.append(relative_path)
    return drifted


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    action = parser.add_mutually_exclusive_group(required=True)
    action.add_argument("--write", action="store_true", help="write all generated artifacts")
    action.add_argument("--check", action="store_true", help="fail when a generated artifact has drifted")
    parser.add_argument("--all", action="store_true", help="render or check both deployment contracts")
    parser.add_argument("--contract", type=Path, default=DEFAULT_CONTRACT_PATH, help="path to deployment contract")
    return parser.parse_args()


def main(default_all: bool = False) -> int:
    args = parse_args()
    try:
        contracts = [load_contract(path) for path in ALL_CONTRACT_PATHS] if args.all or default_all else [load_contract(args.contract)]
        rendered = render_contracts(contracts)
    except ContractError as error:
        print(f"deployment contract error: {error}", file=sys.stderr)
        return 2

    if args.write:
        write_outputs(rendered)
        return 0

    drifted = check_outputs(rendered)
    if drifted:
        print("generated RAGFlow deployment artifacts have drifted:", file=sys.stderr)
        for path in drifted:
            print(f"  {path.as_posix()}", file=sys.stderr)
        print("run: python scripts/render_ragflow_deployment_contract.py --write", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
