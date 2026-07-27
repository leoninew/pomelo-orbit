from pomelo_orbit_mcp.version_specs import (
    ComponentDependency,
    EnvironmentVariable,
    LogicalMount,
    VersionComponent,
    VersionExpose,
    version_component_payload,
    version_expose_payload,
)


def test_version_component_payload_serializes_protocol_json_fields() -> None:
    component = VersionComponent(
        name="mysql",
        image="mysql:8.0.39",
        command=["--max_connections=1000"],
        environment=[EnvironmentVariable(key="MYSQL_ROOT_PASSWORD", value="${MYSQL_PASSWORD}")],
        mounts=[LogicalMount(source_type="logical", source="mysql", target="/var/lib/mysql")],
        depends_on=[ComponentDependency(component="database", condition="service_healthy")],
        restart_policy="unless-stopped",
    )

    assert version_component_payload(component) == {
        "name": "mysql",
        "image": "mysql:8.0.39",
        "command_json": '["--max_connections=1000"]',
        "env_json": '[{"key":"MYSQL_ROOT_PASSWORD","value":"${MYSQL_PASSWORD}"}]',
        "mounts_json": '[{"source_type":"logical","source":"mysql","target":"/var/lib/mysql","read_only":false}]',
        "depends_on_json": '{"database":{"condition":"service_healthy"}}',
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
