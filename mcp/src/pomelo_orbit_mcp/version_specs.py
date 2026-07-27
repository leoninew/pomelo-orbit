"""Structured MCP inputs for Orbit Version components and exposes."""

from __future__ import annotations

import json
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field


class _Spec(BaseModel):
    model_config = ConfigDict(extra="forbid")


class EnvironmentVariable(_Spec):
    key: str
    value: str


class LogicalMount(_Spec):
    source_type: Literal["logical", "volume", "special"]
    source: str
    target: str
    read_only: bool = False
    content: str | None = None
    content_mode: Literal["seed", "sync"] | None = None


class NetworkAttachment(_Spec):
    aliases: list[str] = Field(default_factory=list)


class ComponentDependency(_Spec):
    component: str
    condition: Literal["service_started", "service_healthy", "service_completed_successfully"] = "service_started"


class Healthcheck(_Spec):
    test: list[str]
    interval: str | None = None
    timeout: str | None = None
    retries: int | None = None
    start_period: str | None = None


class ResourceLimits(_Spec):
    memory: str | None = None
    cpus: str | None = None


class ResourceSpec(_Spec):
    limits: ResourceLimits | None = None


class TmpfsSpec(_Spec):
    target: str
    size_bytes: int
    mode: str


class UlimitSpec(_Spec):
    name: Literal["memlock", "nofile"]
    soft: int
    hard: int


class VersionComponent(_Spec):
    name: str
    image: str
    command: list[str] | None = None
    args: list[str] | None = None
    environment: list[EnvironmentVariable] | None = None
    ports: list[str] | None = None
    mounts: list[LogicalMount] | None = None
    networks: dict[str, NetworkAttachment] | list[str] | None = None
    depends_on: list[ComponentDependency] | None = None
    healthcheck: Healthcheck | None = None
    resources: ResourceSpec | None = None
    pull_policy: str | None = None
    restart_policy: Literal["no", "unless-stopped"] | None = None
    tmpfs: list[TmpfsSpec] | None = None
    ulimits: list[UlimitSpec] | None = None


class VersionExpose(_Spec):
    component_name: str
    protocol: Literal["http", "tcp"]
    container_port: int
    path_prefix: str | None = None
    access: Literal["local", "public"] = "public"
    listen_port: int | None = None


def version_component_payload(component: VersionComponent) -> dict[str, str]:
    values: dict[str, str | None] = {
        "name": component.name,
        "image": component.image,
        "command_json": _json_text(component.command),
        "args_json": _json_text(component.args),
        "env_json": _json_text(component.environment),
        "ports_json": _json_text(component.ports),
        "mounts_json": _json_text(component.mounts),
        "networks_json": _json_text(component.networks),
        "depends_on_json": _depends_on_json(component.depends_on),
        "healthcheck_json": _json_text(component.healthcheck),
        "resources_json": _json_text(component.resources),
        "pull_policy": component.pull_policy,
        "restart_policy": component.restart_policy,
        "tmpfs_json": _json_text(component.tmpfs),
        "ulimits_json": _json_text(component.ulimits),
    }
    return {key: value for key, value in values.items() if value is not None}


def version_expose_payload(expose: VersionExpose) -> dict[str, str | int]:
    values: dict[str, str | int | None] = {
        "component_name": expose.component_name,
        "protocol": expose.protocol,
        "container_port": expose.container_port,
        "path_prefix": expose.path_prefix,
        "access": expose.access,
        "listen_port": expose.listen_port,
    }
    return {key: value for key, value in values.items() if value is not None}


def _depends_on_json(dependencies: list[ComponentDependency] | None) -> str | None:
    if dependencies is None:
        return None
    return _json_text({item.component: {"condition": item.condition} for item in dependencies})


def _json_text(value: object | None) -> str | None:
    if value is None:
        return None
    if isinstance(value, BaseModel):
        value = value.model_dump(exclude_none=True)
    elif isinstance(value, list):
        value = [item.model_dump(exclude_none=True) if isinstance(item, BaseModel) else item for item in value]
    elif isinstance(value, dict):
        value = {
            key: item.model_dump(exclude_none=True) if isinstance(item, BaseModel) else item
            for key, item in value.items()
        }
    return json.dumps(value, ensure_ascii=True, separators=(",", ":"))
