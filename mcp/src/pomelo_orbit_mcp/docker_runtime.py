"""Restricted, read-only Docker and Docker Compose access."""

from __future__ import annotations

import asyncio
import json
import shlex
from collections.abc import Awaitable, Callable, Sequence
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from .settings import Settings
from .workspace import RuntimeTarget


class DockerRuntimeError(RuntimeError):
    """Raised for an unavailable Docker CLI or a rejected read-only command."""


@dataclass(frozen=True)
class CommandResult:
    command: tuple[str, ...]
    stdout: str
    stderr: str
    return_code: int

    @property
    def rendered_command(self) -> str:
        return shlex.join(self.command)


CommandRunner = Callable[[Sequence[str], Path | None], Awaitable[CommandResult]]


class DockerRuntime:
    """Executes restricted Docker diagnostics and fixed HTTP probes."""

    def __init__(self, settings: Settings, runner: CommandRunner | None = None) -> None:
        self.settings = settings
        self._runner = runner or self._run_subprocess

    async def doctor(
        self,
        target: RuntimeTarget | None = None,
        *,
        expected_external_networks: Sequence[str] = (),
    ) -> dict[str, Any]:
        context = await self._run(["docker", "context", "show"], None)
        compose = await self._run(["docker", "compose", "version"], None)
        actual_context = context.stdout.strip()
        issues: list[str] = []
        if self.settings.docker_context and actual_context != self.settings.docker_context:
            issues.append(
                f"Docker context mismatch: expected {self.settings.docker_context}, got {actual_context or '<empty>'}"
            )
        if not self.settings.data_root.exists():
            issues.append(f"configured data root does not exist: {self.settings.data_root}")
        if target and not target.working_directory.is_dir():
            issues.append(f"deployment workspace does not exist: {target.working_directory}")
        external_networks: list[str] = []
        external_network_details: list[dict[str, str]] = []
        for network_name in expected_external_networks:
            if target is None:
                issues.append(f"external network {network_name} requires a managed target")
                continue
            try:
                inspected = await self.network_inspect(target, network_name)
                detail = _external_network_detail(network_name, inspected)
            except DockerRuntimeError:
                issues.append(f"external network {network_name} is unavailable from the managed target")
                continue
            if detail["driver"] != "bridge":
                issues.append(f"external network {network_name} has unsupported driver")
                continue
            external_networks.append(network_name)
            external_network_details.append(detail)
        return {
            "healthy": not issues,
            "issues": issues,
            "docker_context": actual_context,
            "data_root": str(self.settings.data_root),
            "working_directory": str(target.working_directory) if target else None,
            "commands": [context.rendered_command, compose.rendered_command],
            "docker_compose_version": compose.stdout.strip(),
            "external_networks": external_networks,
            "external_network_details": external_network_details,
        }

    async def compose_config(self, target: RuntimeTarget) -> dict[str, Any]:
        result = await self._run(self._compose_command(target, "config"), target.working_directory)
        return {"command": result.rendered_command, "compose_yaml": result.stdout, "stderr": result.stderr}

    async def compose_ps(self, target: RuntimeTarget) -> dict[str, Any]:
        result = await self._run(
            self._compose_command(target, "ps", "--all", "--format", "json"), target.working_directory
        )
        return {
            "command": result.rendered_command,
            "containers": _parse_compose_ps(result.stdout),
            "stderr": result.stderr,
        }

    async def compose_logs(
        self,
        target: RuntimeTarget,
        tail: int,
        since: str | None = None,
        services: Sequence[str] | None = None,
    ) -> dict[str, Any]:
        if tail < 1 or tail > 10_000:
            raise DockerRuntimeError("tail must be between 1 and 10000")
        available = {str(item.get("Service") or "") for item in (await self.compose_ps(target))["containers"]}
        selected = [str(service).strip() for service in (services or []) if str(service).strip()]
        if any(service not in available for service in selected):
            raise DockerRuntimeError("requested service is not managed by the resolved Compose project")
        args: list[str] = ["logs", "--tail", str(tail)]
        if since:
            normalized_since = str(since).strip()
            if not normalized_since or len(normalized_since) > 128:
                raise DockerRuntimeError("invalid logs since value")
            args.extend(["--since", normalized_since])
        args.extend(selected)
        result = await self._run(self._compose_command(target, *args), target.working_directory)
        return {
            "command": result.rendered_command,
            "logs": result.stdout,
            "stderr": result.stderr,
            "services": selected,
        }

    async def container_inspect(self, target: RuntimeTarget, container_id: str) -> dict[str, Any]:
        container_id = str(container_id).strip()
        containers = (await self.compose_ps(target))["containers"]
        allowed_ids = {_container_id(item) for item in containers}
        if not container_id or container_id not in allowed_ids:
            raise DockerRuntimeError("container id was not returned by the resolved Compose project")
        result = await self._run(["docker", "inspect", container_id], None)
        return {"command": result.rendered_command, "inspect": _parse_inspect(result.stdout), "stderr": result.stderr}

    async def network_inspect(self, target: RuntimeTarget, network_name: str) -> dict[str, Any]:
        network_name = str(network_name).strip()
        if not network_name:
            raise DockerRuntimeError("network name is required")
        containers = (await self.compose_ps(target))["containers"]
        network_names: set[str] = set()
        for item in containers:
            container_id = _container_id(item)
            if not container_id:
                continue
            inspected = await self.container_inspect(target, container_id)
            for detail in inspected["inspect"]:
                networks = detail.get("NetworkSettings", {}).get("Networks", {})
                if isinstance(networks, dict):
                    network_names.update(name for name in networks if isinstance(name, str))
        if network_name not in network_names:
            raise DockerRuntimeError("network was not derived from the resolved Compose project")
        result = await self._run(["docker", "network", "inspect", network_name], None)
        return {"command": result.rendered_command, "inspect": _parse_inspect(result.stdout), "stderr": result.stderr}

    async def http_probe(
        self, target: RuntimeTarget, component_name: str, port: int, path: str = "/"
    ) -> dict[str, Any]:
        component_name = str(component_name).strip()
        if not component_name or len(component_name) > 128:
            raise DockerRuntimeError("component name is invalid")
        if port < 1 or port > 65_535:
            raise DockerRuntimeError("probe port must be between 1 and 65535")
        if not path.startswith("/") or len(path) > 512 or any(character.isspace() for character in path):
            raise DockerRuntimeError("probe path is invalid")
        containers = (await self.compose_ps(target))["containers"]
        matching = [item for item in containers if str(item.get("Service") or "") == component_name]
        if len(matching) != 1:
            raise DockerRuntimeError("probe component was not uniquely returned by the resolved Compose project")
        container = matching[0]
        if str(container.get("State") or "").lower() != "running":
            raise DockerRuntimeError("probe component is not running")
        container_id = _container_id(container)
        if not container_id:
            raise DockerRuntimeError("probe component did not contain a container id")
        endpoint = f"http://127.0.0.1:{port}{path}"
        result = await self._runner(
            ["docker", "exec", container_id, "curl", "-fsS", "--max-time", "10", "--output", "/dev/null", endpoint],
            None,
        )
        if result.return_code != 0:
            raise DockerRuntimeError("HTTP probe failed")
        return {"status": "reachable", "component_name": component_name, "port": port, "path": path}

    def _compose_command(self, target: RuntimeTarget, *args: str) -> list[str]:
        return ["docker", "compose", "-p", target.compose_project, "-f", "docker-compose.yml", *args]

    async def _run(self, args: Sequence[str], cwd: Path | None) -> CommandResult:
        result = await self._runner(args, cwd)
        if result.return_code != 0:
            message = result.stderr.strip() or result.stdout.strip() or "Docker command failed"
            raise DockerRuntimeError(message)
        return result

    async def _run_subprocess(self, args: Sequence[str], cwd: Path | None) -> CommandResult:
        try:
            process = await asyncio.create_subprocess_exec(
                *args,
                cwd=str(cwd) if cwd else None,
                env=self.settings.docker_cli_environment(),
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )
        except OSError as error:
            raise DockerRuntimeError("Docker CLI is unavailable") from error
        stdout, stderr = await process.communicate()
        return CommandResult(
            command=tuple(args),
            stdout=stdout.decode("utf-8", errors="replace"),
            stderr=stderr.decode("utf-8", errors="replace"),
            return_code=process.returncode if process.returncode is not None else 1,
        )


def _parse_compose_ps(output: str) -> list[dict[str, Any]]:
    output = output.strip()
    if not output:
        return []
    try:
        value = json.loads(output)
    except json.JSONDecodeError:
        values: list[dict[str, Any]] = []
        for line in output.splitlines():
            try:
                value = json.loads(line)
            except json.JSONDecodeError as error:
                raise DockerRuntimeError("docker compose ps returned invalid JSON") from error
            if isinstance(value, dict):
                values.append(value)
        return values
    if isinstance(value, list):
        return [item for item in value if isinstance(item, dict)]
    if isinstance(value, dict):
        return [value]
    raise DockerRuntimeError("docker compose ps returned invalid JSON")


def _parse_inspect(output: str) -> list[dict[str, Any]]:
    try:
        value = json.loads(output)
    except json.JSONDecodeError as error:
        raise DockerRuntimeError("docker inspect returned invalid JSON") from error
    if not isinstance(value, list) or not all(isinstance(item, dict) for item in value):
        raise DockerRuntimeError("docker inspect returned an unexpected JSON value")
    return value


def _external_network_detail(network_name: str, result: dict[str, Any]) -> dict[str, str]:
    inspected = result.get("inspect")
    if not isinstance(inspected, list) or len(inspected) != 1 or not isinstance(inspected[0], dict):
        raise DockerRuntimeError("docker network inspect returned an unexpected result")
    network = inspected[0]
    if str(network.get("Name") or "") != network_name:
        raise DockerRuntimeError("docker network inspect returned a different network")
    driver = str(network.get("Driver") or "")
    scope = str(network.get("Scope") or "")
    if not driver or not scope:
        raise DockerRuntimeError("docker network inspect did not include driver and scope")
    return {"name": network_name, "driver": driver, "scope": scope}


def _container_id(container: dict[str, Any]) -> str:
    for key in ("ID", "Id", "id"):
        value = container.get(key)
        if value:
            return str(value)
    return ""
