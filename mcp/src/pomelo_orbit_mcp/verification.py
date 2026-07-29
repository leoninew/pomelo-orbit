"""Deployment state, Compose, and Docker runtime consistency checks."""

from __future__ import annotations

import asyncio
from collections.abc import Awaitable, Callable, Collection, Mapping
from dataclasses import dataclass, field
from typing import Any, cast

import yaml

from .docker_runtime import DockerRuntime, DockerRuntimeError, compose_container_summaries
from .orbit_client import OrbitClient
from .settings import Settings
from .workspace import RuntimeTarget, RuntimeTargetError, build_runtime_target


@dataclass
class VerificationResult:
    conclusion: str
    evidence: dict[str, Any] = field(default_factory=dict)
    differences: list[str] = field(default_factory=list)

    def as_dict(self, *, detail: bool = False) -> dict[str, Any]:
        result: dict[str, Any] = {
            "conclusion": self.conclusion,
            "differences": self.differences,
        }
        if detail:
            result["evidence"] = self.evidence
        else:
            result.update(_verification_summary(self.evidence, self.differences))
        return result


async def resolve_runtime_target(
    client: OrbitClient,
    settings: Settings,
    application_id: str,
    instance_key: str,
    *,
    allowed_application_kinds: Collection[str] = ("standard",),
) -> RuntimeTarget:
    application = await client.get_application(application_id)
    services = await client.list_application_services(application_id)
    matches = [service for service in services if service.get("instance_key") == instance_key]
    if len(matches) != 1:
        raise RuntimeTargetError("expected exactly one managed service for application and instance key")
    return build_runtime_target(
        settings,
        application,
        matches[0],
        instance_key,
        allowed_application_kinds=allowed_application_kinds,
    )


async def verify_deployment(
    client: OrbitClient,
    runtime: DockerRuntime,
    settings: Settings,
    application_id: str,
    deployment_id: str,
    *,
    sleep: Callable[[float], Awaitable[None]] = asyncio.sleep,
) -> VerificationResult:
    deployment = await client.get_deployment(deployment_id)
    if deployment.get("application_id") != application_id:
        return VerificationResult(
            "inconclusive", {"deployment": deployment}, ["deployment does not belong to application"]
        )
    if deployment.get("status") in {"faulted", "canceled"}:
        return VerificationResult("failed", {"deployment": deployment}, ["deployment reached a failed terminal state"])
    if deployment.get("status") != "ran_to_completion":
        return VerificationResult("inconclusive", {"deployment": deployment}, ["deployment is not complete"])
    if deployment.get("operation_type") not in {"deploy", "restart"}:
        return VerificationResult(
            "inconclusive", {"deployment": deployment}, ["stability verification only applies to deploy or restart"]
        )

    service_id = _nonempty_string(deployment.get("service_id"))
    version_id = _nonempty_string(deployment.get("version_id"))
    if service_id is None or version_id is None:
        return VerificationResult(
            "inconclusive", {"deployment": deployment}, ["deployment lacks service or version linkage"]
        )
    application = await client.get_application(application_id)
    version = await client.get_version(version_id)
    service = await client.get_service(service_id)
    target = build_runtime_target(settings, application, service, _nonempty_string(service.get("instance_key")) or "")
    preview = await client.preview_version(version_id, target.instance_key)
    compose_config = await runtime.compose_config(target)
    ps = await runtime.compose_ps(target)
    inspections = await _inspect_containers(runtime, target, ps["containers"])

    stability = await observe_stability(runtime, target, settings, sleep=sleep)
    evidence: dict[str, Any] = {
        "deployment": deployment,
        "application": application,
        "service": service,
        "version": version,
        "target": target.as_dict(),
        "preview_compose": preview.get("compose_yaml"),
        "compose_config": compose_config.get("compose_yaml"),
        "containers": ps["containers"],
        "inspections": inspections,
        "stability": stability,
    }
    if stability["state"] == "inconclusive":
        return VerificationResult("inconclusive", evidence, stability["issues"])
    if stability["state"] == "failed":
        return VerificationResult("failed", evidence, stability["issues"])

    differences = _compare_runtime(
        str(preview.get("compose_yaml") or ""),
        str(compose_config.get("compose_yaml") or ""),
        ps["containers"],
        inspections,
        target,
    )
    return VerificationResult("drift" if differences else "consistent", evidence, differences)


async def observe_stability(
    runtime: DockerRuntime,
    target: RuntimeTarget,
    settings: Settings,
    *,
    sleep: Callable[[float], Awaitable[None]] = asyncio.sleep,
    monotonic: Callable[[], float] | None = None,
) -> dict[str, Any]:
    loop = asyncio.get_running_loop()
    clock = monotonic or loop.time
    deadline = clock() + settings.stability_window_seconds
    baseline_restarts: dict[str, int] = {}
    samples: list[dict[str, Any]] = []
    while True:
        ps = await runtime.compose_ps(target)
        containers = ps["containers"]
        if not containers:
            return {
                "state": "failed",
                "issues": ["no containers are running for the managed Compose project"],
                "samples": samples,
            }
        inspections = await _inspect_containers(runtime, target, containers)

        current: list[dict[str, Any]] = []
        issues: list[str] = []
        for container in containers:
            container_id = _container_id(container)
            inspection = inspections.get(container_id, {})
            state = _mapping(inspection.get("State"))
            status = str(state.get("Status") or container.get("State") or "").lower()
            health = str(_mapping(state.get("Health")).get("Status") or "").lower()
            restart_count = int(inspection.get("RestartCount") or 0)
            current.append(
                {"container_id": container_id, "status": status, "health": health, "restart_count": restart_count}
            )
            if (
                status != "running"
                or health == "unhealthy"
                or "unhealthy" in str(container.get("Status") or "").lower()
            ):
                issues.append(f"container {container_id} is not healthy and running")
            initial = baseline_restarts.setdefault(container_id, restart_count)
            if restart_count > initial:
                issues.append(f"container {container_id} restart count increased from {initial} to {restart_count}")
        samples.append({"containers": current})
        if issues:
            return {"state": "failed", "issues": issues, "samples": samples}
        if clock() >= deadline:
            return {"state": "stable", "issues": [], "samples": samples}
        await sleep(min(settings.stability_poll_seconds, max(0.0, deadline - clock())))


async def _inspect_containers(
    runtime: DockerRuntime,
    target: RuntimeTarget,
    containers: list[dict[str, Any]],
) -> dict[str, dict[str, Any]]:
    inspections: dict[str, dict[str, Any]] = {}
    for container in containers:
        container_id = _container_id(container)
        if not container_id:
            raise DockerRuntimeError("docker compose ps result did not contain a container id")
        result = await runtime.container_inspect(target, container_id)
        if not result["inspect"]:
            raise DockerRuntimeError(f"docker inspect did not return {container_id}")
        inspections[container_id] = result["inspect"][0]
    return inspections


def _compare_runtime(
    preview_yaml: str,
    config_yaml: str,
    containers: list[dict[str, Any]],
    inspections: Mapping[str, dict[str, Any]],
    target: RuntimeTarget,
) -> list[str]:
    try:
        preview = yaml.safe_load(preview_yaml) or {}
        config = yaml.safe_load(config_yaml) or {}
    except yaml.YAMLError as error:
        raise DockerRuntimeError("Compose configuration could not be parsed") from error
    if not isinstance(preview, dict) or not isinstance(config, dict):
        return ["Compose YAML did not contain an object"]
    preview_services = _mapping(preview.get("services"))
    config_services = _mapping(config.get("services"))
    running_services = {str(item.get("Service") or "") for item in containers}
    differences: list[str] = []
    for name, expected_value in preview_services.items():
        if not isinstance(name, str) or not isinstance(expected_value, Mapping):
            continue
        expected = _mapping(expected_value)
        actual_value = config_services.get(name)
        if not isinstance(actual_value, Mapping):
            differences.append(f"Compose config is missing expected service {name}")
            continue
        actual_config = _mapping(actual_value)
        if name not in running_services:
            differences.append(f"Docker has no running container for expected service {name}")
        expected_image = expected.get("image")
        actual_image = actual_config.get("image")
        if expected_image and actual_image and expected_image != actual_image:
            differences.append(f"service {name} image differs between preview and Compose config")
        _compare_expected_ports(name, actual_config, inspections, differences)
        _compare_expected_networks(name, actual_config, inspections, differences)
        _compare_labels(name, expected, inspections, target, differences)
    return differences


def _compare_expected_ports(
    service_name: str,
    service: Mapping[str, Any],
    inspections: Mapping[str, dict[str, Any]],
    differences: list[str],
) -> None:
    expected_ports = _compose_container_ports(service.get("ports"))
    if not expected_ports:
        return
    actual_ports: set[str] = set()
    for inspection in _inspections_for_service(service_name, inspections):
        ports = inspection.get("NetworkSettings", {}).get("Ports", {})
        if isinstance(ports, dict):
            actual_ports.update(str(port) for port in ports)
    missing = expected_ports - actual_ports
    if missing:
        differences.append(f"service {service_name} is missing expected container ports: {', '.join(sorted(missing))}")


def _compare_expected_networks(
    service_name: str,
    service: Mapping[str, Any],
    inspections: Mapping[str, dict[str, Any]],
    differences: list[str],
) -> None:
    networks = service.get("networks")
    if isinstance(networks, dict):
        expected = set(networks)
    elif isinstance(networks, list):
        expected = {str(value) for value in networks}
    else:
        return
    actual: set[str] = set()
    for inspection in _inspections_for_service(service_name, inspections):
        configured = inspection.get("NetworkSettings", {}).get("Networks", {})
        if isinstance(configured, dict):
            actual.update(str(name) for name in configured)
    missing = {
        name
        for name in expected
        if not any(
            network == name or network.endswith(f"_{name}") or network.endswith(f"-{name}") for network in actual
        )
    }
    if missing:
        differences.append(f"service {service_name} is missing expected networks: {', '.join(sorted(missing))}")


def _compare_labels(
    service_name: str,
    expected: Mapping[str, Any],
    inspections: Mapping[str, dict[str, Any]],
    target: RuntimeTarget,
    differences: list[str],
) -> None:
    expected_labels = _mapping(expected.get("labels"))
    for inspection in _inspections_for_service(service_name, inspections):
        labels_value = _mapping(inspection.get("Config")).get("Labels")
        if not isinstance(labels_value, Mapping):
            differences.append(f"service {service_name} container has no labels")
            continue
        labels = _mapping(labels_value)
        if labels.get("com.docker.compose.project") != target.compose_project:
            differences.append(f"service {service_name} container Compose project label differs")
        if labels.get("com.docker.compose.service") != service_name:
            differences.append(f"service {service_name} container Compose service label differs")
        for key, value in expected_labels.items():
            if labels.get(key) != str(value):
                differences.append(f"service {service_name} label {key} differs from preview")


def _compose_container_ports(value: Any) -> set[str]:
    ports: set[str] = set()
    if not isinstance(value, list):
        return ports
    for port in value:
        if isinstance(port, int):
            ports.add(f"{port}/tcp")
        elif isinstance(port, str):
            target = port.rsplit(":", 1)[-1]
            ports.add(target if "/" in target else f"{target}/tcp")
        elif isinstance(port, dict) and port.get("target"):
            protocol = str(port.get("protocol") or "tcp")
            ports.add(f"{port['target']}/{protocol}")
    return ports


def _inspections_for_service(service_name: str, inspections: Mapping[str, dict[str, Any]]) -> list[dict[str, Any]]:
    matched: list[dict[str, Any]] = []
    for inspection in inspections.values():
        labels = _mapping(_mapping(inspection.get("Config")).get("Labels"))
        if labels.get("com.docker.compose.service") == service_name:
            matched.append(inspection)
    return matched


def _container_id(container: Mapping[str, Any]) -> str:
    for key in ("ID", "Id", "id"):
        if container.get(key):
            return str(container[key])
    return ""


def _mapping(value: Any) -> Mapping[str, Any]:
    return cast(Mapping[str, Any], value) if isinstance(value, Mapping) else {}


def _nonempty_string(value: Any) -> str | None:
    return value if isinstance(value, str) and value else None


def _verification_summary(evidence: Mapping[str, Any], differences: list[str]) -> dict[str, Any]:
    containers = evidence.get("containers")
    inspections = evidence.get("inspections")
    checked = isinstance(containers, list) and isinstance(inspections, Mapping)
    container_items = (
        [container for container in containers if isinstance(container, dict)] if isinstance(containers, list) else []
    )
    inspection_items = (
        {
            str(container_id): inspection
            for container_id, inspection in inspections.items()
            if isinstance(container_id, str) and isinstance(inspection, dict)
        }
        if isinstance(inspections, Mapping)
        else {}
    )
    stability = evidence.get("stability")
    stability_issues = stability.get("issues") if isinstance(stability, Mapping) else []
    stability_summary = (
        {
            "state": _nonempty_string(stability.get("state")),
            "issues": [issue for issue in stability_issues if isinstance(issue, str)]
            if isinstance(stability_issues, list)
            else [],
        }
        if isinstance(stability, Mapping)
        else None
    )
    return {
        "components": compose_container_summaries(container_items, inspection_items),
        "port_constraints": _constraint_summary(differences, "port", checked),
        "network_constraints": _constraint_summary(differences, "network", checked),
        "stability": stability_summary,
    }


def _constraint_summary(differences: list[str], keyword: str, checked: bool) -> dict[str, Any]:
    issues = [difference for difference in differences if keyword in difference.lower()]
    if issues:
        status = "failed"
    elif checked:
        status = "passed"
    else:
        status = "not_checked"
    return {"status": status, "issues": issues}
