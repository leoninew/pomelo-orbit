#!/usr/bin/env python3
"""Transfer one Orbit service deployment closure through dbtalk JSONL."""

from __future__ import annotations

import argparse
import contextlib
import json
import os
import shutil
import subprocess
import sys
import tempfile
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Sequence


TRANSFER_FORMAT = "dbtalk.database-transfer/v1"
# Keep this order aligned with dbtalk's target-schema foreign-key ordering.
SERVICE_TABLES = (
    "project",
    "application",
    "gateway_config",
    "route",
    "version",
    "service",
    "service_env",
    "version_component",
    "service_component",
    "service_component_endpoint",
    "service_component_env",
    "service_component_mount",
    "service_component_resource",
    "version_component_dependency",
    "version_component_device",
    "version_component_endpoint",
    "version_component_env",
    "version_component_healthcheck",
    "version_component_mount",
    "version_component_resource",
    "version_component_tmpfs",
    "version_component_ulimit",
)
VERSION_COMPONENT_CHILD_TABLES = (
    "version_component_env",
    "version_component_endpoint",
    "version_component_mount",
    "version_component_dependency",
    "version_component_healthcheck",
    "version_component_resource",
    "version_component_tmpfs",
    "version_component_ulimit",
    "version_component_device",
)
SERVICE_COMPONENT_CHILD_TABLES = (
    "service_component_env",
    "service_component_mount",
    "service_component_resource",
    "service_component_endpoint",
)


class ServiceTransferError(RuntimeError):
    """Raised when a service transfer cannot be safely prepared."""


@dataclass(frozen=True)
class TableBlock:
    name: str
    columns: tuple[dict[str, Any], ...]
    primary_key: tuple[str, ...]
    rows: tuple[tuple[Any, ...], ...]

    @property
    def column_names(self) -> tuple[str, ...]:
        return tuple(str(column["name"]) for column in self.columns)

    def row_maps(self) -> tuple[dict[str, Any], ...]:
        names = self.column_names
        return tuple(dict(zip(names, row, strict=True)) for row in self.rows)


@dataclass(frozen=True)
class TransferFile:
    header: dict[str, Any]
    tables: tuple[TableBlock, ...]


def parse_args(argv: Sequence[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    subcommands = parser.add_subparsers(dest="command", required=True)

    exporter = subcommands.add_parser(
        "export", help="export one service deployment closure"
    )
    exporter.add_argument("--source", choices=("sqlite", "mysql"), required=True)
    exporter.add_argument("--service-code", required=True)
    exporter.add_argument("--output", type=Path, required=True)
    connection = exporter.add_mutually_exclusive_group(required=True)
    connection.add_argument("--dsn")
    connection.add_argument("--dsn-env")
    exporter.add_argument("--tz", default="UTC")
    exporter.add_argument("--dbtalk-command", default="dbtalk")

    importer = subcommands.add_parser(
        "import", help="import one service deployment closure"
    )
    importer.add_argument("--target", choices=("sqlite", "mysql"), required=True)
    importer.add_argument("--input", type=Path, required=True)
    importer.add_argument("--mode", choices=("insert", "upsert"), required=True)
    connection = importer.add_mutually_exclusive_group(required=True)
    connection.add_argument("--dsn")
    connection.add_argument("--dsn-env")
    importer.add_argument("--tz", default="UTC")
    importer.add_argument("--dbtalk-command", default="dbtalk")

    return parser.parse_args(argv)


def load_transfer(path: Path) -> TransferFile:
    if not path.is_file():
        raise ServiceTransferError(f"transfer file does not exist: {path}")

    header: dict[str, Any] | None = None
    tables: list[TableBlock] = []
    current: dict[str, Any] | None = None
    current_rows: list[tuple[Any, ...]] = []

    try:
        lines = path.read_text(encoding="utf-8").splitlines()
        for line_number, line in enumerate(lines, 1):
            if not line.strip():
                raise ServiceTransferError(f"blank JSONL record at line {line_number}")
            record = json.loads(line)
            if not isinstance(record, dict) or not isinstance(record.get("kind"), str):
                raise ServiceTransferError(
                    f"invalid JSONL record at line {line_number}"
                )
            kind = record["kind"]
            if kind == "header":
                if header is not None or tables or current is not None:
                    raise ServiceTransferError("JSONL header must be the first record")
                if record.get("format") != TRANSFER_FORMAT:
                    raise ServiceTransferError("unsupported dbtalk JSONL format")
                if record.get("source") not in ("sqlite", "mysql"):
                    raise ServiceTransferError("JSONL header has an invalid source")
                header = record
            elif kind == "table":
                if header is None or current is not None:
                    raise ServiceTransferError(
                        f"unexpected table record at line {line_number}"
                    )
                current = parse_table_header(record, line_number)
                current_rows = []
            elif kind == "row":
                if current is None:
                    raise ServiceTransferError(
                        f"row outside table at line {line_number}"
                    )
                values = record.get("values")
                if not isinstance(values, list) or len(values) != len(
                    current["columns"]
                ):
                    raise ServiceTransferError(f"invalid row at line {line_number}")
                current_rows.append(tuple(values))
            elif kind == "end":
                if current is None or record.get("rows") != len(current_rows):
                    raise ServiceTransferError(
                        f"invalid table end at line {line_number}"
                    )
                tables.append(
                    TableBlock(
                        name=current["name"],
                        columns=tuple(current["columns"]),
                        primary_key=tuple(current["primary_key"]),
                        rows=tuple(current_rows),
                    )
                )
                current = None
                current_rows = []
            else:
                raise ServiceTransferError(f"unknown JSONL record kind: {kind}")
    except json.JSONDecodeError as error:
        raise ServiceTransferError(f"invalid JSONL at line {error.lineno}") from error

    if header is None:
        raise ServiceTransferError("JSONL header is missing")
    if current is not None:
        raise ServiceTransferError("JSONL table is missing its end record")
    if not tables:
        raise ServiceTransferError("JSONL contains no table blocks")
    if len({table.name for table in tables}) != len(tables):
        raise ServiceTransferError("JSONL contains duplicate tables")
    return TransferFile(header=header, tables=tuple(tables))


def parse_table_header(record: dict[str, Any], line_number: int) -> dict[str, Any]:
    name = record.get("name")
    columns = record.get("columns")
    primary_key = record.get("primary_key")
    if (
        not isinstance(name, str)
        or not isinstance(columns, list)
        or not isinstance(primary_key, list)
        or not columns
        or not all(isinstance(value, str) for value in primary_key)
    ):
        raise ServiceTransferError(f"invalid table header at line {line_number}")
    parsed_columns: list[dict[str, Any]] = []
    for column in columns:
        if (
            not isinstance(column, dict)
            or not isinstance(column.get("name"), str)
            or not isinstance(column.get("declared_type"), str)
        ):
            raise ServiceTransferError(
                f"invalid column definition at line {line_number}"
            )
        parsed_columns.append(column)
    names = [column["name"] for column in parsed_columns]
    if len(set(names)) != len(names) or any(key not in names for key in primary_key):
        raise ServiceTransferError(
            f"invalid primary key in table header at line {line_number}"
        )
    return {"name": name, "columns": parsed_columns, "primary_key": primary_key}


def write_transfer(path: Path, transfer: TransferFile) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary: Path | None = None
    try:
        with tempfile.NamedTemporaryFile(
            "w", encoding="utf-8", newline="\n", dir=path.parent, delete=False
        ) as output:
            temporary = Path(output.name)
            output.write(
                json.dumps(transfer.header, ensure_ascii=False, separators=(",", ":"))
                + "\n"
            )
            for table in transfer.tables:
                output.write(
                    json.dumps(
                        {
                            "kind": "table",
                            "name": table.name,
                            "columns": list(table.columns),
                            "primary_key": list(table.primary_key),
                        },
                        ensure_ascii=False,
                        separators=(",", ":"),
                    )
                    + "\n"
                )
                for row in table.rows:
                    output.write(
                        json.dumps(
                            {"kind": "row", "values": list(row)},
                            ensure_ascii=False,
                            separators=(",", ":"),
                        )
                        + "\n"
                    )
                output.write(
                    json.dumps({"kind": "end", "rows": len(table.rows)}) + "\n"
                )
        temporary.replace(path)
        temporary = None
    finally:
        if temporary is not None:
            with contextlib.suppress(OSError):
                temporary.unlink()


def table_map(transfer: TransferFile) -> dict[str, TableBlock]:
    return {table.name: table for table in transfer.tables}


def require_table(tables: dict[str, TableBlock], name: str) -> TableBlock:
    try:
        return tables[name]
    except KeyError as error:
        raise ServiceTransferError(
            f"service transfer is missing table: {name}"
        ) from error


def require_column(table: TableBlock, name: str) -> int:
    try:
        return table.column_names.index(name)
    except ValueError as error:
        raise ServiceTransferError(
            f"table {table.name} is missing column: {name}"
        ) from error


def value_key(value: Any) -> str | None:
    return None if value is None else str(value)


def rows_where(
    table: TableBlock, column: str, values: set[str | None]
) -> list[tuple[Any, ...]]:
    index = require_column(table, column)
    return [row for row in table.rows if value_key(row[index]) in values]


def one_row_where(table: TableBlock, column: str, value: str) -> tuple[Any, ...]:
    rows = rows_where(table, column, {value})
    if not rows:
        raise ServiceTransferError(
            f"service transfer references missing {table.name}.{column}: {value}"
        )
    if len(rows) > 1:
        raise ServiceTransferError(
            f"service transfer expects one {table.name} row for {column}: {value}"
        )
    return rows[0]


def service_tables(transfer: TransferFile, service_code: str) -> TransferFile:
    available = table_map(transfer)
    missing = [name for name in SERVICE_TABLES if name not in available]
    if missing:
        raise ServiceTransferError(
            "service export is missing required tables: " + ", ".join(missing)
        )

    service_table = require_table(available, "service")
    service_code_index = require_column(service_table, "code")
    service_rows = [
        row
        for row in service_table.rows
        if value_key(row[service_code_index]) == service_code
    ]
    if not service_rows:
        raise ServiceTransferError(f"service does not exist: {service_code}")
    if len(service_rows) > 1:
        raise ServiceTransferError(f"service code is not unique: {service_code}")

    service = dict(zip(service_table.column_names, service_rows[0], strict=True))
    service_id = value_key(service.get("id"))
    application_id = value_key(service.get("application_id"))
    version_id = value_key(service.get("version_id"))
    if not service_id or not application_id or not version_id:
        raise ServiceTransferError(
            "service row has incomplete application/version references"
        )

    application_table = require_table(available, "application")
    application = dict(
        zip(
            application_table.column_names,
            one_row_where(application_table, "id", application_id),
            strict=True,
        )
    )
    project_id = value_key(application.get("project_id"))
    if not project_id:
        raise ServiceTransferError("application row has no project reference")
    one_row_where(require_table(available, "project"), "id", project_id)

    version_table = require_table(available, "version")
    version_ids: list[str] = []
    current_version: str | None = version_id
    while current_version is not None:
        if current_version in version_ids:
            raise ServiceTransferError(
                f"service {service_code} has cyclic version lineage"
            )
        version_row = dict(
            zip(
                version_table.column_names,
                one_row_where(version_table, "id", current_version),
                strict=True,
            )
        )
        if value_key(version_row.get("application_id")) != application_id:
            raise ServiceTransferError(
                f"service {service_code} version lineage crosses applications"
            )
        version_ids.append(current_version)
        current_version = value_key(version_row.get("created_from_version_id"))
    version_ids.reverse()

    version_component_table = require_table(available, "version_component")
    version_component_rows = rows_where(
        version_component_table, "version_id", set(version_ids)
    )
    version_component_ids = {
        value_key(row[require_column(version_component_table, "id")])
        for row in version_component_rows
    }
    version_component_ids.discard(None)

    service_component_table = require_table(available, "service_component")
    service_component_rows = rows_where(
        service_component_table, "service_id", {service_id}
    )
    service_component_ids = {
        value_key(row[require_column(service_component_table, "id")])
        for row in service_component_rows
    }
    service_component_ids.discard(None)

    source_component_index = require_column(
        service_component_table, "source_version_component_id"
    )
    for row in service_component_rows:
        source_component_id = value_key(row[source_component_index])
        if source_component_id not in version_component_ids:
            raise ServiceTransferError(
                "service component references missing version component: "
                f"{source_component_id}"
            )

    selected: list[TableBlock] = []
    for table_name in SERVICE_TABLES:
        table = require_table(available, table_name)
        if table_name == "project":
            rows = rows_where(table, "id", {project_id})
        elif table_name == "application":
            rows = rows_where(table, "id", {application_id})
        elif table_name == "version":
            rows = rows_where(table, "id", set(version_ids))
        elif table_name == "version_component":
            rows = rows_where(table, "id", version_component_ids)
        elif table_name in VERSION_COMPONENT_CHILD_TABLES:
            rows = rows_where(table, "component_id", version_component_ids)
        elif table_name == "gateway_config":
            rows = rows_where(table, "application_id", {application_id})
        elif table_name == "service":
            rows = rows_where(table, "id", {service_id})
        elif table_name == "service_env":
            rows = rows_where(table, "service_id", {service_id})
        elif table_name == "service_component":
            rows = rows_where(table, "id", service_component_ids)
        elif table_name in SERVICE_COMPONENT_CHILD_TABLES:
            rows = rows_where(table, "service_component_id", service_component_ids)
        elif table_name == "route":
            rows = rows_where(table, "service_id", {service_id})
            project_index = require_column(table, "project_id")
            if any(value_key(row[project_index]) != project_id for row in rows):
                raise ServiceTransferError(
                    f"service {service_code} route references another project"
                )
        else:
            raise ServiceTransferError(
                f"no service selection rule for table: {table_name}"
            )
        selected.append(
            TableBlock(
                name=table.name,
                columns=table.columns,
                primary_key=table.primary_key,
                rows=tuple(rows),
            )
        )
    return TransferFile(header=transfer.header, tables=tuple(selected))


def validate_service_transfer(transfer: TransferFile) -> str:
    if tuple(table.name for table in transfer.tables) != SERVICE_TABLES:
        raise ServiceTransferError(
            "service transfer has an unexpected table set or order"
        )
    service_table = require_table(table_map(transfer), "service")
    code_index = require_column(service_table, "code")
    if len(service_table.rows) != 1:
        raise ServiceTransferError("service transfer must contain exactly one service")
    service_code = value_key(service_table.rows[0][code_index])
    if not service_code:
        raise ServiceTransferError("service transfer service code is empty")
    selected = service_tables(transfer, service_code)
    if selected.tables != transfer.tables:
        raise ServiceTransferError(
            "service transfer contains rows outside its deployment closure"
        )
    return service_code


def dbtalk_command(command: str) -> str:
    resolved = shutil.which(command)
    if resolved:
        return resolved
    if Path(command).is_file():
        return str(Path(command).resolve())
    raise ServiceTransferError(
        "dbtalk CLI was not found; install dbtalk and ensure the command is on PATH"
    )


def connection_arguments(dsn: str | None, dsn_env: str | None) -> list[str]:
    if (dsn is None) == (dsn_env is None):
        raise ServiceTransferError("provide exactly one of --dsn or --dsn-env")
    if dsn_env is not None:
        if not dsn_env or dsn_env not in os.environ:
            raise ServiceTransferError(
                f"DSN environment variable is not set: {dsn_env}"
            )
        return ["--dsn-env", dsn_env]
    if not dsn:
        raise ServiceTransferError("--dsn must not be empty")
    return ["--dsn", dsn]


def run_dbtalk(command: str, arguments: Sequence[str], *, operation: str) -> None:
    executable = dbtalk_command(command)
    result = subprocess.run(
        [executable, *arguments],
        check=False,
        capture_output=True,
        text=True,
    )
    if result.returncode:
        raise ServiceTransferError(
            f"dbtalk {operation} failed with exit code {result.returncode}"
        )


def export_service(args: argparse.Namespace) -> Path:
    output = args.output.resolve()
    connection = connection_arguments(args.dsn, args.dsn_env)
    with tempfile.TemporaryDirectory(prefix="orbit-service-transfer-") as directory:
        service_export = Path(directory) / "service.jsonl"
        run_dbtalk(
            args.dbtalk_command,
            [
                "export",
                "--source",
                args.source,
                "--output",
                str(service_export),
                *connection,
                *(
                    argument
                    for table_name in SERVICE_TABLES
                    for argument in ("--include-table", table_name)
                ),
                "--tz",
                args.tz,
            ],
            operation="export",
        )
        selected = service_tables(load_transfer(service_export), args.service_code)
        write_transfer(output, selected)
    return output


def import_service(args: argparse.Namespace) -> str:
    input_path = args.input.resolve()
    transfer = load_transfer(input_path)
    service_code = validate_service_transfer(transfer)
    connection = connection_arguments(args.dsn, args.dsn_env)
    run_dbtalk(
        args.dbtalk_command,
        [
            "import",
            "--target",
            args.target,
            "--input",
            str(input_path),
            "--mode",
            args.mode,
            *connection,
            "--tz",
            args.tz,
        ],
        operation="import",
    )
    return service_code


def main(argv: Sequence[str]) -> int:
    args = parse_args(argv)
    try:
        if args.command == "export":
            output = export_service(args)
            print(f"service transfer written to {output}")
        else:
            service_code = import_service(args)
            print(f"service transfer imported for {service_code}")
    except (OSError, ServiceTransferError) as error:
        print(f"service transfer blocked: {error}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
