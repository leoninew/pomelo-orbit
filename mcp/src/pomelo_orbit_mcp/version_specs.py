"""Structured MCP inputs for Orbit Version components and exposes."""

from __future__ import annotations

from typing import Any, Literal, TypeVar

from pydantic import BaseModel, ConfigDict


class _Spec(BaseModel):
    model_config = ConfigDict(extra="forbid")


SpecT = TypeVar("SpecT", bound=_Spec)


class EnvironmentVariable(_Spec):
    key: str
    value: str


class LogicalMount(_Spec):
    source_type: Literal["directory", "file", "named_volume", "special"]
    source: str
    target: str
    read_only: bool = False
    content: str | None = None
    content_mode: Literal["seed", "sync"] | None = None


class ComponentDependency(_Spec):
    name: str
    condition: Literal["service_started", "service_healthy", "service_completed_successfully"] = "service_started"


class Healthcheck(_Spec):
    test_mode: Literal["CMD", "CMD-SHELL"] = "CMD"
    test: list[str]
    interval: str | None = None
    timeout: str | None = None
    retries: int | None = None
    start_period: str | None = None
    start_interval: str | None = None
    disabled: bool = False


class ResourceSpec(_Spec):
    limit_cpus: str | None = None
    limit_memory: str | None = None
    reservation_cpus: str | None = None
    reservation_memory: str | None = None


class ComponentPort(_Spec):
    host_port: int
    container_port: int


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
    env: list[EnvironmentVariable] | None = None
    ports: list[ComponentPort] | None = None
    mounts: list[LogicalMount] | None = None
    networks: list[str] | None = None
    dependencies: list[ComponentDependency] | None = None
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


def version_component_payload(component: VersionComponent) -> dict[str, Any]:
    values: dict[str, Any] = {
        "name": component.name,
        "image": component.image,
        "command": component.command,
        "args": component.args,
        "env": _model_items(component.env),
        "ports": _model_items(component.ports),
        "mounts": _model_items(component.mounts),
        "networks": component.networks,
        "dependencies": _model_items(component.dependencies),
        "healthcheck": _model_value(component.healthcheck),
        "resources": _model_value(component.resources),
        "pull_policy": component.pull_policy,
        "restart_policy": component.restart_policy,
        "tmpfs": _model_items(component.tmpfs),
        "ulimits": _model_items(component.ulimits),
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


def _model_items(value: list[SpecT] | None) -> list[dict[str, Any]] | None:
    if value is None:
        return None
    return [item.model_dump(exclude_none=True) for item in value]


def _model_value(value: _Spec | None) -> dict[str, Any] | None:
    if value is None:
        return None
    return value.model_dump(exclude_none=True)
