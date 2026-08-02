"""Structured MCP inputs for Orbit component declarations and Service overlays."""

from __future__ import annotations

from typing import Any, Literal, TypeVar

from pydantic import BaseModel, ConfigDict, field_validator, model_validator


class _Spec(BaseModel):
    model_config = ConfigDict(extra="forbid")


SpecT = TypeVar("SpecT", bound=_Spec)


class EnvironmentVariable(_Spec):
    key: str
    value: str


class LogicalMount(_Spec):
    source_type: Literal["directory", "file", "named_volume", "controlled_file"]
    source: str
    target: str
    read_only: bool = False
    source_is_host_path: bool = False
    content: str | None = None
    mode: str | None = None
    ignore_if_exists: bool = False


class ComponentDependency(_Spec):
    name: str
    condition: Literal["service_started", "service_healthy", "service_completed_successfully"] = "service_started"


class Healthcheck(_Spec):
    test_mode: Literal["CMD", "CMD-SHELL"] = "CMD"
    test: str
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


class ComponentEndpoint(_Spec):
    name: str
    protocol: Literal["http", "tcp"]
    container_port: int
    mode: Literal["internal", "local", "host", "gateway_http", "gateway_tcp"] = "internal"
    bind_address: str | None = None
    listen_port: int | None = None
    entrypoint: str | None = None
    path_prefix: str | None = None


class TmpfsSpec(_Spec):
    target: str
    size_bytes: int
    mode: str


class UlimitSpec(_Spec):
    name: Literal["memlock", "nofile"]
    soft: int
    hard: int


class DeviceRequestSpec(_Spec):
    model_config = ConfigDict(extra="forbid", strict=True)

    driver: str
    count: str
    capabilities: list[str]

    @field_validator("driver")
    @classmethod
    def validate_driver(cls, value: str) -> str:
        if not value or value != value.strip() or any(character.isspace() for character in value):
            raise ValueError("device driver must be a non-empty token")
        return value

    @field_validator("count")
    @classmethod
    def validate_count(cls, value: str) -> str:
        if value == "all":
            return value
        if not value.isdecimal() or value.startswith("0"):
            raise ValueError("device count must be all or a positive integer")
        return value

    @field_validator("capabilities")
    @classmethod
    def validate_capabilities(cls, value: list[str]) -> list[str]:
        if not value:
            raise ValueError("device capabilities are required")
        if any(not item or item != item.strip() or any(character.isspace() for character in item) for item in value):
            raise ValueError("device capabilities must be non-empty tokens")
        if len(set(value)) != len(value):
            raise ValueError("device capabilities must not contain duplicates")
        return value

    @model_validator(mode="after")
    def validate_nvidia_capabilities(self) -> DeviceRequestSpec:
        if self.driver == "nvidia" and "gpu" not in self.capabilities:
            raise ValueError("nvidia device requests require gpu capability")
        return self


class VersionComponent(_Spec):
    name: str
    image: str
    command: str | None = None
    env: list[EnvironmentVariable] | None = None
    endpoints: list[ComponentEndpoint] | None = None
    mounts: list[LogicalMount] | None = None
    dependencies: list[ComponentDependency] | None = None
    healthcheck: Healthcheck | None = None
    resources: ResourceSpec | None = None
    pull_policy: Literal["always", "missing", "never"]
    restart_policy: Literal["no", "unless-stopped"] | None = None
    tmpfs: list[TmpfsSpec] | None = None
    ulimits: list[UlimitSpec] | None = None
    devices: list[DeviceRequestSpec] | None = None


class VersionComponentCreate(_Spec):
    name: str
    image: str
    command: str = ""
    pull_policy: Literal["always", "missing", "never"]
    restart_policy: Literal["no", "unless-stopped"] | None = None


class VersionComponentBasicUpdate(_Spec):
    name: str
    image: str
    command: str
    pull_policy: Literal["always", "missing", "never"]
    restart_policy: Literal["no", "unless-stopped"] | None = None


class VersionComponentRuntimeUpdate(_Spec):
    healthcheck: Healthcheck | None = None


class VersionComponentEndpointsUpdate(_Spec):
    endpoints: list[ComponentEndpoint]


class VersionComponentEnvUpdate(_Spec):
    env: list[EnvironmentVariable]


class VersionComponentMountsUpdate(_Spec):
    mounts: list[LogicalMount]


class VersionComponentDependenciesUpdate(_Spec):
    dependencies: list[ComponentDependency]


class VersionComponentAdvancedUpdate(_Spec):
    resources: ResourceSpec | None = None
    tmpfs: list[TmpfsSpec]
    ulimits: list[UlimitSpec]


class VersionComponentResourcesUpdate(_Spec):
    resources: ResourceSpec | None


class VersionComponentTmpfsUpdate(_Spec):
    tmpfs: list[TmpfsSpec]


class VersionComponentUlimitsUpdate(_Spec):
    ulimits: list[UlimitSpec]


class VersionComponentDevicesUpdate(_Spec):
    devices: list[DeviceRequestSpec]


class ServiceComponentEnvOverlay(_Spec):
    key: str
    value: str | None = None
    state: Literal["override", "deleted"]


class ServiceComponentMountOverlay(_Spec):
    target: str
    source: str | None = None
    state: Literal["override", "deleted"]


class ServiceComponentResourceOverlay(_Spec):
    limit_cpus: str | None = None
    limit_memory: str | None = None
    reservation_cpus: str | None = None
    reservation_memory: str | None = None
    state: Literal["override", "deleted"]


class ServiceComponentEndpointOverlay(_Spec):
    name: str
    mode: Literal["internal", "local", "host", "gateway_http", "gateway_tcp"] | None = None
    bind_address: str | None = None
    listen_port: int | None = None
    entrypoint: str | None = None
    path_prefix: str | None = None
    state: Literal["override", "deleted"]


class ServiceComponentOverlayUpdate(_Spec):
    env: list[ServiceComponentEnvOverlay] = []
    mounts: list[ServiceComponentMountOverlay] = []
    resources: ServiceComponentResourceOverlay | None = None
    endpoints: list[ServiceComponentEndpointOverlay] = []


class ServiceEnvUpdate(_Spec):
    env: list[EnvironmentVariable]


def version_component_payload(component: VersionComponent) -> dict[str, Any]:
    values: dict[str, Any] = {
        "name": component.name,
        "image": component.image,
        "command": component.command,
        "env": _model_items(component.env),
        "endpoints": _model_items(component.endpoints),
        "mounts": _model_items(component.mounts),
        "dependencies": _model_items(component.dependencies),
        "healthcheck": _model_value(component.healthcheck),
        "resources": _model_value(component.resources),
        "pull_policy": component.pull_policy,
        "restart_policy": component.restart_policy,
        "tmpfs": _model_items(component.tmpfs),
        "ulimits": _model_items(component.ulimits),
        "devices": _model_items(component.devices),
    }
    return {key: value for key, value in values.items() if value is not None}


def version_component_create_payload(component: VersionComponentCreate) -> dict[str, Any]:
    return component.model_dump(exclude_none=True)


def _model_items(value: list[SpecT] | None) -> list[dict[str, Any]] | None:
    if value is None:
        return None
    return [item.model_dump(exclude_none=True) for item in value]


def _model_value(value: _Spec | None) -> dict[str, Any] | None:
    if value is None:
        return None
    return value.model_dump(exclude_none=True)
