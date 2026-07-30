from pomelo_orbit_mcp.version_specs import (
    ComponentDependency,
    ComponentPort,
    EnvironmentVariable,
    Healthcheck,
    LogicalMount,
    ResourceSpec,
    TmpfsSpec,
    UlimitSpec,
    VersionComponent,
    VersionComponentAdvancedUpdate,
    VersionComponentBasicUpdate,
    VersionComponentCreate,
    VersionComponentDependenciesUpdate,
    VersionComponentEnvUpdate,
    VersionComponentMountsUpdate,
    VersionComponentPortsUpdate,
    VersionComponentResourcesUpdate,
    VersionComponentTmpfsUpdate,
    VersionComponentUlimitsUpdate,
    VersionExpose,
    version_component_create_payload,
    version_component_payload,
    version_expose_payload,
)


def test_version_component_payload_serializes_protocol_json_fields() -> None:
    component = VersionComponent(
        name="mysql",
        image="mysql:8.0.39",
        command="--max_connections=1000",
        env=[EnvironmentVariable(key="MYSQL_ROOT_PASSWORD", value="${MYSQL_PASSWORD}")],
        mounts=[LogicalMount(source_type="directory", source="mysql", target="/var/lib/mysql")],
        dependencies=[ComponentDependency(name="database", condition="service_healthy")],
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


def test_version_expose_payload_uses_api_field_names() -> None:
    expose = VersionExpose(
        component_name="ragflow-cpu",
        protocol="http",
        container_port=80,
        access="local",
        listen_port=9380,
    )

    assert version_expose_payload(expose) == {
        "component_name": "ragflow-cpu",
        "protocol": "http",
        "container_port": 80,
        "access": "local",
        "listen_port": 9380,
    }


def test_healthcheck_payload_uses_one_command_text() -> None:
    component = VersionComponent(
        name="redis",
        image="redis:7.4",
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
    ports = VersionComponentPortsUpdate(ports=[ComponentPort(host_port=8080, container_port=80)])
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
    assert ports.model_dump() == {"ports": [{"host_port": 8080, "container_port": 80}]}
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
    assert resources.model_dump(exclude_none=True) == {"resources": {"limit_memory": "1g"}}
    assert tmpfs.model_dump() == {"tmpfs": [{"target": "/run", "size_bytes": 16_777_216, "mode": "0755"}]}
    assert ulimits.model_dump() == {"ulimits": [{"name": "memlock", "soft": 1_024, "hard": 2_048}]}
