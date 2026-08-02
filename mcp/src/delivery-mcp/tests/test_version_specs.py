import pytest
from pydantic import ValidationError

from pomelo_delivery_mcp.version_specs import (
    ComponentDependency,
    ComponentEndpoint,
    DeviceRequestSpec,
    EnvironmentVariable,
    Healthcheck,
    LogicalMount,
    ResourceSpec,
    ServiceComponentEndpointOverlay,
    ServiceComponentMountOverlay,
    ServiceComponentOverlayUpdate,
    TmpfsSpec,
    UlimitSpec,
    VersionComponent,
    VersionComponentAdvancedUpdate,
    VersionComponentBasicUpdate,
    VersionComponentCreate,
    VersionComponentDependenciesUpdate,
    VersionComponentDevicesUpdate,
    VersionComponentEndpointsUpdate,
    VersionComponentEnvUpdate,
    VersionComponentMountsUpdate,
    VersionComponentResourcesUpdate,
    VersionComponentTmpfsUpdate,
    VersionComponentUlimitsUpdate,
    version_component_create_payload,
    version_component_payload,
)


def test_version_component_payload_serializes_protocol_json_fields() -> None:
    component = VersionComponent(
        name="mysql",
        image="mysql:8.0.39",
        command="--max_connections=1000",
        env=[EnvironmentVariable(key="MYSQL_ROOT_PASSWORD", value="${MYSQL_PASSWORD}")],
        mounts=[LogicalMount(source_type="directory", source="mysql", target="/var/lib/mysql")],
        dependencies=[ComponentDependency(name="database", condition="service_healthy")],
        devices=[DeviceRequestSpec(driver="nvidia", count="all", capabilities=["gpu"])],
        pull_policy="missing",
        restart_policy="unless-stopped",
    )

    assert version_component_payload(component) == {
        "name": "mysql",
        "image": "mysql:8.0.39",
        "command": "--max_connections=1000",
        "env": [{"key": "MYSQL_ROOT_PASSWORD", "value": "${MYSQL_PASSWORD}"}],
        "mounts": [
            {
                "source_type": "directory",
                "source": "mysql",
                "target": "/var/lib/mysql",
                "read_only": False,
                "source_is_host_path": False,
                "ignore_if_exists": False,
            }
        ],
        "dependencies": [{"name": "database", "condition": "service_healthy"}],
        "devices": [{"driver": "nvidia", "count": "all", "capabilities": ["gpu"]}],
        "pull_policy": "missing",
        "restart_policy": "unless-stopped",
    }


def test_version_component_create_payload_contains_only_basic_fields() -> None:
    component = VersionComponentCreate(
        name="web",
        image="nginx:1.27",
        pull_policy="missing",
        restart_policy="unless-stopped",
    )

    assert version_component_create_payload(component) == {
        "name": "web",
        "image": "nginx:1.27",
        "command": "",
        "pull_policy": "missing",
        "restart_policy": "unless-stopped",
    }


def test_service_component_overlay_uses_sparse_api_fields() -> None:
    overlay = ServiceComponentOverlayUpdate(
        mounts=[ServiceComponentMountOverlay(target="/data", state="deleted")],
        endpoints=[ServiceComponentEndpointOverlay(name="http", mode="host", listen_port=9380, state="override")]
    )

    assert overlay.model_dump(exclude_none=True) == {
        "env": [],
        "mounts": [{"target": "/data", "state": "deleted"}],
        "endpoints": [{"name": "http", "mode": "host", "listen_port": 9380, "state": "override"}],
    }
    with pytest.raises(ValidationError):
        ServiceComponentOverlayUpdate.model_validate(
            {"mounts": [{"position": 0, "state": "deleted"}]}
        )


def test_healthcheck_payload_uses_one_command_text() -> None:
    component = VersionComponent(
        name="redis",
        image="redis:7.4",
        pull_policy="missing",
        healthcheck=Healthcheck(
            test_mode="CMD-SHELL",
            test="redis-cli ping || exit 1",
        ),
    )

    assert version_component_payload(component)["healthcheck"] == {
        "test_mode": "CMD-SHELL",
        "test": "redis-cli ping || exit 1",
        "disabled": False,
    }


def test_component_group_payloads_match_the_split_json_contracts() -> None:
    basic = VersionComponentBasicUpdate(
        name="web",
        image="nginx:1.27",
        command="nginx -g 'daemon off;'",
        pull_policy="missing",
        restart_policy="unless-stopped",
    )
    advanced = VersionComponentAdvancedUpdate(
        resources=ResourceSpec(limit_memory="512m"),
        tmpfs=[TmpfsSpec(target="/tmp", size_bytes=67_108_864, mode="1777")],
        ulimits=[UlimitSpec(name="nofile", soft=65_535, hard=65_535)],
    )
    endpoints = VersionComponentEndpointsUpdate(
        endpoints=[ComponentEndpoint(name="http", protocol="http", container_port=80, mode="host", listen_port=8080)]
    )
    env = VersionComponentEnvUpdate(env=[EnvironmentVariable(key="MODE", value="production")])
    mounts = VersionComponentMountsUpdate(
        mounts=[
            LogicalMount(
                source_type="controlled_file",
                source="gateway/acme.json",
                target="/letsencrypt/acme.json",
                content="{}",
                mode="0600",
                ignore_if_exists=True,
            )
        ]
    )
    dependencies = VersionComponentDependenciesUpdate(
        dependencies=[ComponentDependency(name="database", condition="service_healthy")]
    )
    devices = VersionComponentDevicesUpdate(
        devices=[DeviceRequestSpec(driver="nvidia", count="1", capabilities=["gpu"])]
    )
    resources = VersionComponentResourcesUpdate(resources=ResourceSpec(limit_memory="1g"))
    tmpfs = VersionComponentTmpfsUpdate(tmpfs=[TmpfsSpec(target="/run", size_bytes=16_777_216, mode="0755")])
    ulimits = VersionComponentUlimitsUpdate(ulimits=[UlimitSpec(name="memlock", soft=1_024, hard=2_048)])

    assert basic.model_dump(exclude_none=True) == {
        "name": "web",
        "image": "nginx:1.27",
        "command": "nginx -g 'daemon off;'",
        "pull_policy": "missing",
        "restart_policy": "unless-stopped",
    }
    assert advanced.model_dump(exclude_none=True) == {
        "resources": {"limit_memory": "512m"},
        "tmpfs": [{"target": "/tmp", "size_bytes": 67_108_864, "mode": "1777"}],
        "ulimits": [{"name": "nofile", "soft": 65_535, "hard": 65_535}],
    }
    assert endpoints.model_dump() == {"endpoints": [{"name": "http", "protocol": "http", "container_port": 80, "mode": "host", "bind_address": None, "listen_port": 8080, "entrypoint": None, "path_prefix": None}]}
    assert env.model_dump() == {"env": [{"key": "MODE", "value": "production"}]}
    assert mounts.model_dump(exclude_none=True) == {
        "mounts": [
            {
                "source_type": "controlled_file",
                "source": "gateway/acme.json",
                "target": "/letsencrypt/acme.json",
                "read_only": False,
                "source_is_host_path": False,
                "content": "{}",
                "mode": "0600",
                "ignore_if_exists": True,
            }
        ]
    }
    assert dependencies.model_dump() == {"dependencies": [{"name": "database", "condition": "service_healthy"}]}
    assert devices.model_dump() == {"devices": [{"driver": "nvidia", "count": "1", "capabilities": ["gpu"]}]}
    assert resources.model_dump(exclude_none=True) == {"resources": {"limit_memory": "1g"}}
    assert tmpfs.model_dump() == {"tmpfs": [{"target": "/run", "size_bytes": 16_777_216, "mode": "0755"}]}
    assert ulimits.model_dump() == {"ulimits": [{"name": "memlock", "soft": 1_024, "hard": 2_048}]}


@pytest.mark.parametrize(
    "component_type, value",
    (
        (VersionComponent, {"name": "web", "image": "nginx:1.27"}),
        (VersionComponentCreate, {"name": "web", "image": "nginx:1.27"}),
        (VersionComponentBasicUpdate, {"name": "web", "image": "nginx:1.27", "command": ""}),
    ),
)
def test_pull_policy_is_required(component_type: type[object], value: dict[str, str]) -> None:
    with pytest.raises(ValidationError):
        component_type.model_validate(value)  # type: ignore[attr-defined]


@pytest.mark.parametrize(
    "component_type, value",
    (
        (VersionComponent, {"name": "web", "image": "nginx:1.27", "pull_policy": "on-demand"}),
        (VersionComponentCreate, {"name": "web", "image": "nginx:1.27", "pull_policy": "on-demand"}),
        (
            VersionComponentBasicUpdate,
            {"name": "web", "image": "nginx:1.27", "command": "", "pull_policy": "on-demand"},
        ),
    ),
)
def test_pull_policy_rejects_unsupported_values(component_type: type[object], value: dict[str, str]) -> None:
    with pytest.raises(ValidationError):
        component_type.model_validate(value)  # type: ignore[attr-defined]


@pytest.mark.parametrize(
    "value",
    (
        {"driver": "nvidia", "count": "all", "capabilities": []},
        {"driver": "nvidia", "count": "all", "capabilities": ["compute"]},
        {"driver": "nvidia", "count": "01", "capabilities": ["gpu"]},
        {"driver": "nvidia gpu", "count": "1", "capabilities": ["gpu"]},
        {"driver": "nvidia", "count": "1", "capabilities": ["gpu", "gpu"]},
        {"driver": "nvidia", "count": 1, "capabilities": ["gpu"]},
    ),
)
def test_device_request_rejects_invalid_or_coerced_values(value: object) -> None:
    with pytest.raises(ValidationError):
        DeviceRequestSpec.model_validate(value)
