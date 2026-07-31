#!/usr/bin/env python3
"""Export a stopped RAGFlow+TEI control-plane SQLite initialization baseline."""

from __future__ import annotations

import argparse
import json
import secrets
import sqlite3
import sys
import tempfile
from pathlib import Path
from typing import Any, Iterable


RAGFLOW_CODE = "ragflow"
GATEWAY_CODE = "traefik"
CPU_LABEL = "ragflow-tei-cpu"
GPU_LABEL = "ragflow-tei-gpu"
COMPONENT_NAMES = ("es01", "minio", "mysql", "ragflow-cpu", "redis", "tei")
COMPONENT_CHILD_TABLES = (
    "version_component_dependency",
    "version_component_env",
    "version_component_healthcheck",
    "version_component_mount",
    "version_component_port",
    "version_component_resource",
    "version_component_tmpfs",
    "version_component_ulimit",
    "version_component_device",
)
COMPONENT_CHILD_ORDER_BY = {
    "version_component_dependency": "component_id, position",
    "version_component_env": "component_id, position",
    "version_component_healthcheck": "component_id",
    "version_component_mount": "component_id, position",
    "version_component_port": "component_id, position",
    "version_component_resource": "component_id",
    "version_component_tmpfs": "component_id, position",
    "version_component_ulimit": "component_id, name",
    "version_component_device": "component_id, position",
}
GPU_RUNTIME_UNVERIFIED_MARKER = "GPU runtime unverified"
CPU_TEI_IMAGE = "ghcr.io/huggingface/text-embeddings-inference:cpu-1.9.3@sha256:ad950d30878eceb72aaf32024d26fa2b1d04a75304fa0b4776b49aa1941fea07"
GPU_TEI_IMAGE = "ghcr.io/huggingface/text-embeddings-inference:cuda-1.9.3@sha256:249a0bc87522bfe2f1012b4d194f0225878f47079115ada3aeb0b1ef257b402a"
RAGFLOW_RUNTIME_CONFIG_KEYS = (
    "MYSQL_PASSWORD",
    "REDIS_PASSWORD",
    "MINIO_USER",
    "MINIO_PASSWORD",
    "ELASTIC_PASSWORD",
)


class ExportError(RuntimeError):
    pass


def parse_args(argv: list[str]) -> argparse.Namespace:
    root = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--database", default=str(root / "data" / "db" / "pomelo-orbit.db"))
    parser.add_argument("--output", required=True, help="Candidate SQL file to create")
    parser.add_argument("--replace", action="store_true", help="Replace an existing output after a successful export")
    parser.add_argument("--project-id", help="Optional explicit Project ID")
    return parser.parse_args(argv)


def connect_read_only(path: Path) -> sqlite3.Connection:
    if not path.is_file():
        raise ExportError(f"SQLite database does not exist: {path}")
    connection = sqlite3.connect(f"file:{path.as_posix()}?mode=ro", uri=True)
    connection.row_factory = sqlite3.Row
    return connection


def ensure_database_integrity(connection: sqlite3.Connection) -> None:
    integrity = connection.execute("PRAGMA integrity_check").fetchone()[0]
    if integrity != "ok":
        raise ExportError(f"integrity_check failed: {integrity}")
    foreign_keys = list(connection.execute("PRAGMA foreign_key_check"))
    if foreign_keys:
        raise ExportError(f"foreign_key_check returned {len(foreign_keys)} row(s)")


def one(rows: Iterable[sqlite3.Row], description: str) -> sqlite3.Row:
    values = list(rows)
    if len(values) != 1:
        raise ExportError(f"expected exactly one {description}, found {len(values)}")
    return values[0]


def placeholders(values: Iterable[str]) -> str:
    return ", ".join("?" for _ in values)


def table_columns(connection: sqlite3.Connection, table: str) -> list[str]:
    columns = [str(row["name"]) for row in connection.execute(f"PRAGMA table_info({table})")]
    if not columns:
        raise ExportError(f"required table is missing: {table}")
    return columns


def rows_by_values(
    connection: sqlite3.Connection, table: str, column: str, values: list[str], order_by: str
) -> list[sqlite3.Row]:
    if not values:
        return []
    return list(
        connection.execute(
            f"SELECT * FROM {table} WHERE {column} IN ({placeholders(values)}) ORDER BY {order_by}", values
        )
    )


def sql_value(connection: sqlite3.Connection, value: Any) -> str:
    return str(connection.execute("SELECT quote(?)", (value,)).fetchone()[0])


def sql_insert(
    connection: sqlite3.Connection,
    table: str,
    row: sqlite3.Row,
    overrides: dict[str, Any] | None = None,
    *,
    or_ignore: bool = False,
) -> str:
    overrides = overrides or {}
    columns = table_columns(connection, table)
    values = [overrides.get(column, row[column]) for column in columns]
    insert = "INSERT OR IGNORE" if or_ignore else "INSERT"
    return f"{insert} INTO {table} ({', '.join(columns)}) VALUES ({', '.join(sql_value(connection, value) for value in values)});"


def component_rows(connection: sqlite3.Connection, version_ids: list[str]) -> list[sqlite3.Row]:
    rows = rows_by_values(connection, "version_component", "version_id", version_ids, "version_id, name")
    by_version: dict[str, list[str]] = {version_id: [] for version_id in version_ids}
    for row in rows:
        by_version[str(row["version_id"])].append(str(row["name"]))
    expected = list(COMPONENT_NAMES)
    for version_id, names in by_version.items():
        if names != expected:
            raise ExportError(f"version {version_id} must contain exactly {', '.join(expected)}; found {', '.join(names)}")
    return rows


def component_rows_by_name(rows: list[sqlite3.Row], version_id: str) -> dict[str, sqlite3.Row]:
    return {str(row["name"]): row for row in rows if row["version_id"] == version_id}


def child_payloads(connection: sqlite3.Connection, table: str, component_id: str) -> list[dict[str, Any]]:
    columns = [column for column in table_columns(connection, table) if column != "component_id"]
    rows = rows_by_values(connection, table, "component_id", [component_id], COMPONENT_CHILD_ORDER_BY[table])
    return [{column: row[column] for column in columns} for row in rows]


def ensure_variant_specification(
    connection: sqlite3.Connection,
    cpu_version: sqlite3.Row,
    gpu_version: sqlite3.Row,
    rows: list[sqlite3.Row],
) -> None:
    cpu_components = component_rows_by_name(rows, str(cpu_version["id"]))
    gpu_components = component_rows_by_name(rows, str(gpu_version["id"]))
    if cpu_components["tei"]["image"] != CPU_TEI_IMAGE:
        raise ExportError("CPU TEI component must use the pinned CPU 1.9.3 image digest")
    if gpu_components["tei"]["image"] != GPU_TEI_IMAGE:
        raise ExportError("GPU TEI component must use the pinned CUDA 1.9.3 image digest")

    ignored_component_columns = {"id", "version_id", "image", "created_at", "updated_at"}
    for name in COMPONENT_NAMES:
        cpu_component = cpu_components[name]
        gpu_component = gpu_components[name]
        cpu_payload = {key: cpu_component[key] for key in cpu_component.keys() if key not in ignored_component_columns}
        gpu_payload = {key: gpu_component[key] for key in gpu_component.keys() if key not in ignored_component_columns}
        if cpu_payload != gpu_payload:
            raise ExportError(f"CPU and GPU component {name!r} differ outside the TEI image/device variant")
        for table in COMPONENT_CHILD_TABLES:
            if table == "version_component_device":
                continue
            if child_payloads(connection, table, str(cpu_component["id"])) != child_payloads(
                connection, table, str(gpu_component["id"])
            ):
                raise ExportError(f"CPU and GPU component {name!r} differ in {table}")

    cpu_devices = rows_by_values(
        connection,
        "version_component_device",
        "component_id",
        [str(row["id"]) for row in cpu_components.values()],
        COMPONENT_CHILD_ORDER_BY["version_component_device"],
    )
    if cpu_devices:
        raise ExportError("CPU Version must not contain device requests")
    gpu_devices = rows_by_values(
        connection,
        "version_component_device",
        "component_id",
        [str(row["id"]) for row in gpu_components.values()],
        COMPONENT_CHILD_ORDER_BY["version_component_device"],
    )
    if len(gpu_devices) != 1:
        raise ExportError("GPU Version must contain exactly one device request")
    gpu_device = gpu_devices[0]
    try:
        capabilities = json.loads(str(gpu_device["capabilities_json"]))
    except json.JSONDecodeError as error:
        raise ExportError("GPU TEI device request has invalid capabilities JSON") from error
    if not (
        gpu_device["component_id"] == gpu_components["tei"]["id"]
        and gpu_device["driver"] == "nvidia"
        and gpu_device["device_count"] == "all"
        and capabilities == ["gpu"]
    ):
        raise ExportError("GPU Version must request all NVIDIA GPUs for the TEI component")


def selected_baseline(connection: sqlite3.Connection, args: argparse.Namespace) -> dict[str, list[sqlite3.Row]]:
    ragflow = one(connection.execute("SELECT * FROM application WHERE code = ?", (RAGFLOW_CODE,)), "RAGFlow Application")
    project_id = args.project_id or ragflow["project_id"]
    if not project_id:
        raise ExportError("RAGFlow Application must belong to a Project")
    project = one(connection.execute("SELECT * FROM project WHERE id = ?", (project_id,)), "Project")
    gateway = one(
        connection.execute(
            "SELECT * FROM application WHERE code = ? AND project_id = ?", (GATEWAY_CODE, project_id)
        ),
        "managed Gateway Application",
    )
    gateway_config = one(
        connection.execute("SELECT * FROM gateway_config WHERE application_id = ?", (gateway["id"],)),
        "Gateway configuration",
    )
    versions = list(connection.execute("SELECT * FROM version WHERE application_id = ? ORDER BY label", (ragflow["id"],)))
    if [row["label"] for row in versions] != [CPU_LABEL, GPU_LABEL]:
        raise ExportError(f"RAGFlow must contain exactly {CPU_LABEL} and {GPU_LABEL} Versions")
    cpu_version = versions[0]
    gpu_version = versions[1]
    gpu_note = str(gpu_version["note"] or "")
    if GPU_RUNTIME_UNVERIFIED_MARKER not in gpu_note:
        raise ExportError(
            "GPU Version note must include the marker: "
            f"{GPU_RUNTIME_UNVERIFIED_MARKER!r}"
        )
    ragflow_components = component_rows(connection, [str(cpu_version["id"]), str(gpu_version["id"])])
    ensure_variant_specification(connection, cpu_version, gpu_version, ragflow_components)
    ragflow_service = one(
        connection.execute(
            "SELECT * FROM service WHERE application_id = ? AND instance_key = 'default'", (ragflow["id"],)
        ),
        "RAGFlow default Service",
    )
    if ragflow_service["version_id"] != cpu_version["id"]:
        raise ExportError("RAGFlow default Service must initially select the CPU Version")
    gateway_service = one(
        connection.execute(
            "SELECT * FROM service WHERE application_id = ? AND instance_key = 'default'", (gateway["id"],)
        ),
        "Gateway default Service",
    )
    gateway_version = one(
        connection.execute("SELECT * FROM version WHERE id = ?", (gateway_service["version_id"],)), "Gateway selected Version"
    )
    gateway_components = component_rows_for_gateway(connection, str(gateway_version["id"]))
    applications = [gateway, ragflow]
    all_versions = [gateway_version, cpu_version, gpu_version]
    all_components = [*gateway_components, *ragflow_components]
    services = [gateway_service, ragflow_service]
    service_exposes = rows_by_values(connection, "service_expose", "service_id", [str(row["id"]) for row in services], "service_id, component_name, protocol, container_port")
    ragflow_exposes = [row for row in service_exposes if row["service_id"] == ragflow_service["id"]]
    if len(ragflow_exposes) != 1:
        raise ExportError("RAGFlow default Service must have exactly one local expose")
    expose = ragflow_exposes[0]
    if not (
        expose["component_name"] == "ragflow-cpu"
        and expose["protocol"] == "http"
        and expose["container_port"] == 80
        and expose["access"] == "local"
        and expose["listen_port"] == 9380
    ):
        raise ExportError("RAGFlow local expose must be ragflow-cpu HTTP port 80 on 9380")
    return {
        "project": [project],
        "application": applications,
        "gateway_config": [gateway_config],
        "version": all_versions,
        "version_component": all_components,
        "service": services,
        "service_expose": service_exposes,
    }


def component_rows_for_gateway(connection: sqlite3.Connection, version_id: str) -> list[sqlite3.Row]:
    rows = rows_by_values(connection, "version_component", "version_id", [version_id], "version_id, name")
    if len(rows) != 1 or rows[0]["name"] != "traefik":
        raise ExportError("Gateway selected Version must contain exactly one traefik component")
    return rows


def child_rows(connection: sqlite3.Connection, component_ids: list[str]) -> dict[str, list[sqlite3.Row]]:
    result: dict[str, list[sqlite3.Row]] = {}
    for table in COMPONENT_CHILD_TABLES:
        result[table] = rows_by_values(
            connection,
            table,
            "component_id",
            component_ids,
            COMPONENT_CHILD_ORDER_BY[table],
        )
    return result


def generated_ragflow_runtime_config() -> dict[str, str]:
    return {
        "MYSQL_PASSWORD": secrets.token_urlsafe(24),
        "REDIS_PASSWORD": secrets.token_urlsafe(24),
        "MINIO_USER": "minio" + secrets.token_hex(8),
        "MINIO_PASSWORD": secrets.token_urlsafe(24),
        "ELASTIC_PASSWORD": secrets.token_urlsafe(24),
    }


def render_sql(connection: sqlite3.Connection, data: dict[str, list[sqlite3.Row]]) -> str:
    component_ids = [str(row["id"]) for row in data["version_component"]]
    children = child_rows(connection, component_ids)
    lines = [
        "-- RAGFlow + TEI stopped initialization baseline. Generated by scripts/export_ragflow_tei_baseline.py.",
        "PRAGMA foreign_keys = ON;",
        "BEGIN;",
    ]
    table_order = ("project", "application", "gateway_config", "version", "version_component")
    for table in table_order:
        for row in sorted(data[table], key=lambda item: str(item["id"] if "id" in item.keys() else item["application_id"])):
            lines.append(sql_insert(connection, table, row, or_ignore=table == "project"))
    for table in COMPONENT_CHILD_TABLES:
        for row in children[table]:
            lines.append(sql_insert(connection, table, row))
    ragflow_application_id = one(
        (row for row in data["application"] if row["code"] == RAGFLOW_CODE), "RAGFlow Application"
    )["id"]
    ragflow_runtime_config = json.dumps(generated_ragflow_runtime_config(), separators=(",", ":"), sort_keys=True)
    for row in sorted(data["service"], key=lambda item: str(item["application_id"])):
        runtime_config_json = ragflow_runtime_config if row["application_id"] == ragflow_application_id else "{}"
        lines.append(sql_insert(connection, "service", row, {"runtime_config_json": runtime_config_json, "status": "stopped"}))
    for row in data["service_expose"]:
        lines.append(sql_insert(connection, "service_expose", row))
    lines.extend(("COMMIT;", "PRAGMA foreign_keys = ON;", ""))
    return "\n".join(lines)


def write_output(output: Path, rendered: str, replace: bool) -> None:
    if output.exists() and not replace:
        raise ExportError(f"refusing to overwrite existing output: {output}")
    output.parent.mkdir(parents=True, exist_ok=True)
    temporary_path: Path | None = None
    try:
        with tempfile.NamedTemporaryFile("w", encoding="utf-8", newline="\n", dir=output.parent, delete=False) as temporary:
            temporary.write(rendered)
            temporary_path = Path(temporary.name)
        temporary_path.replace(output)
    finally:
        if temporary_path and temporary_path.exists():
            temporary_path.unlink()


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    database = Path(args.database).resolve()
    output = Path(args.output).resolve()
    try:
        with connect_read_only(database) as connection:
            ensure_database_integrity(connection)
            data = selected_baseline(connection, args)
            rendered = render_sql(connection, data)
        write_output(output, rendered, args.replace)
    except (ExportError, OSError, sqlite3.Error) as error:
        print(f"baseline export blocked: {error}", file=sys.stderr)
        return 2
    print(f"baseline SQL written: {output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
