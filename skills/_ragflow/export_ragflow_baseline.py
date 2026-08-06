#!/usr/bin/env python3
"""Export non-Gateway RAGFlow SQLite control-plane data as SQL INSERT statements."""

from __future__ import annotations

import argparse
import re
import sqlite3
import sys
import tempfile
from pathlib import Path
from typing import Any


TABLES = (
    "project",
    "application",
    "version",
    "version_component",
    "version_component_dependency",
    "version_component_env",
    "version_component_healthcheck",
    "version_component_mount",
    "version_component_endpoint",
    "version_component_resource",
    "version_component_tmpfs",
    "version_component_ulimit",
    "version_component_device",
    "service",
    "service_env",
    "service_component",
    "service_component_env",
    "service_component_mount",
    "service_component_resource",
    "service_component_endpoint",
)
IDENTIFIER = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*$")


class ExportError(RuntimeError):
    pass


def parse_args(argv: list[str]) -> argparse.Namespace:
    root = Path(__file__).resolve().parents[2]
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--database", default=str(root / "data" / "db" / "pomelo-orbit.db"))
    parser.add_argument("--output", required=True, help="SQL file to replace after a successful export")
    parser.add_argument(
        "--replace",
        action="store_true",
        help="replace an existing output file after all export checks pass",
    )
    parser.add_argument("--project-id", help="Accepted for command compatibility; the baseline exports all configured tables")
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


def quote_identifier(identifier: str) -> str:
    if not IDENTIFIER.fullmatch(identifier):
        raise ExportError(f"invalid SQLite identifier: {identifier}")
    return f'"{identifier}"'


def table_columns(connection: sqlite3.Connection, table: str) -> list[str]:
    rows = list(connection.execute(f"PRAGMA table_info({quote_identifier(table)})"))
    if not rows:
        raise ExportError(f"required table is missing: {table}")
    return [str(row["name"]) for row in rows]


def table_order_columns(connection: sqlite3.Connection, table: str) -> list[str]:
    rows = list(connection.execute(f"PRAGMA table_info({quote_identifier(table)})"))
    primary_key = sorted((row for row in rows if int(row["pk"]) > 0), key=lambda row: int(row["pk"]))
    return [str(row["name"]) for row in primary_key] or [str(row["name"]) for row in rows]


def table_rows(connection: sqlite3.Connection, table: str) -> list[dict[str, Any]]:
    columns = table_columns(connection, table)
    order_by = ", ".join(quote_identifier(column) for column in table_order_columns(connection, table))
    query = f"SELECT * FROM {quote_identifier(table)} ORDER BY {order_by}"
    return [dict(row) for row in connection.execute(query)]


def rows_with_foreign_key(rows: list[dict[str, Any]], key: str, values: set[str]) -> list[dict[str, Any]]:
    return [row for row in rows if str(row[key]) in values]


def selected_baseline(connection: sqlite3.Connection, _args: argparse.Namespace) -> dict[str, list[dict[str, Any]]]:
    data = {table: table_rows(connection, table) for table in TABLES}

    applications = [row for row in data["application"] if row["kind"] != "gateway"]
    application_ids = {str(row["id"]) for row in applications}
    project_ids = {str(row["project_id"]) for row in applications if row["project_id"] is not None}
    versions = rows_with_foreign_key(data["version"], "application_id", application_ids)
    version_ids = {str(row["id"]) for row in versions}
    components = rows_with_foreign_key(data["version_component"], "version_id", version_ids)
    component_ids = {str(row["id"]) for row in components}
    services = rows_with_foreign_key(data["service"], "application_id", application_ids)
    service_ids = {str(row["id"]) for row in services}
    service_components = rows_with_foreign_key(data["service_component"], "service_id", service_ids)
    service_component_ids = {str(row["id"]) for row in service_components}

    data["project"] = rows_with_foreign_key(data["project"], "id", project_ids)
    data["application"] = applications
    data["version"] = versions
    data["version_component"] = components
    data["version_component_dependency"] = rows_with_foreign_key(
        data["version_component_dependency"], "component_id", component_ids
    )
    data["version_component_env"] = rows_with_foreign_key(data["version_component_env"], "component_id", component_ids)
    data["version_component_healthcheck"] = rows_with_foreign_key(
        data["version_component_healthcheck"], "component_id", component_ids
    )
    data["version_component_mount"] = rows_with_foreign_key(data["version_component_mount"], "component_id", component_ids)
    data["version_component_endpoint"] = rows_with_foreign_key(
        data["version_component_endpoint"], "component_id", component_ids
    )
    data["version_component_resource"] = rows_with_foreign_key(
        data["version_component_resource"], "component_id", component_ids
    )
    data["version_component_tmpfs"] = rows_with_foreign_key(data["version_component_tmpfs"], "component_id", component_ids)
    data["version_component_ulimit"] = rows_with_foreign_key(
        data["version_component_ulimit"], "component_id", component_ids
    )
    data["version_component_device"] = rows_with_foreign_key(
        data["version_component_device"], "component_id", component_ids
    )
    data["service"] = services
    data["service_env"] = rows_with_foreign_key(data["service_env"], "service_id", service_ids)
    data["service_component"] = service_components
    data["service_component_env"] = rows_with_foreign_key(
        data["service_component_env"], "service_component_id", service_component_ids
    )
    data["service_component_mount"] = rows_with_foreign_key(
        data["service_component_mount"], "service_component_id", service_component_ids
    )
    data["service_component_resource"] = rows_with_foreign_key(
        data["service_component_resource"], "service_component_id", service_component_ids
    )
    data["service_component_endpoint"] = rows_with_foreign_key(
        data["service_component_endpoint"], "service_component_id", service_component_ids
    )
    return data


def sql_value(connection: sqlite3.Connection, value: Any) -> str:
    return str(connection.execute("SELECT quote(?)", (value,)).fetchone()[0])


def sql_insert(connection: sqlite3.Connection, table: str, row: dict[str, Any]) -> str:
    columns = list(row)
    if not columns:
        raise ExportError(f"table {table} has no columns")
    values = ", ".join(sql_value(connection, row[column]) for column in columns)
    names = ", ".join(quote_identifier(column) for column in columns)
    return f"INSERT OR IGNORE INTO {quote_identifier(table)} ({names}) VALUES ({values});"


def render_sql(connection: sqlite3.Connection, data: dict[str, list[dict[str, Any]]]) -> str:
    lines = [
        "-- RAGFlow data export. Generated by skills/_ragflow/export_ragflow_baseline.py.",
        "PRAGMA foreign_keys = ON;",
        "BEGIN;",
    ]
    for table in TABLES:
        for row in data[table]:
            lines.append(sql_insert(connection, table, row))
    lines.extend(("COMMIT;", "PRAGMA foreign_keys = ON;", ""))
    return "\n".join(lines)


def write_output(output: Path, rendered: str, *, replace: bool) -> None:
    if output.exists() and not replace:
        raise ExportError(f"output already exists; pass --replace to overwrite: {output}")
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
    try:
        with connect_read_only(Path(args.database).resolve()) as connection:
            ensure_database_integrity(connection)
            rendered = render_sql(connection, selected_baseline(connection, args))
        write_output(Path(args.output).resolve(), rendered, replace=args.replace)
    except (ExportError, OSError, sqlite3.Error) as error:
        print(f"baseline export blocked: {error}", file=sys.stderr)
        return 2
    print(f"baseline SQL written: {Path(args.output).resolve()}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
