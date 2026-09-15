#!/usr/bin/env python3
"""Transfer Orbit service closures and selected Environments through dbtalk JSONL."""

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

from cryptography.fernet import Fernet, InvalidToken


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
ENVIRONMENT_TABLES = ("project", "environment_credential", "environment")
# dbtalk topologically sorts independent tables by name. Environment has no
# database-level foreign key, so it must precede project in the import file.
ENVIRONMENT_DBTALK_IMPORT_TABLES = (
    "environment",
    "project",
    "environment_credential",
)
ENVIRONMENT_TRANSFER_SCOPE = "environment"
ENVIRONMENT_REQUIRED_COLUMNS = (
    "id",
    "project_id",
    "code",
    "target_type",
    "ssh_credential_id",
    "ssh_credential_revision",
    "gateway_application_id",
)
ENVIRONMENT_CREDENTIAL_REQUIRED_COLUMNS = ("id", "project_id", "revision")


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


@dataclass(frozen=True)
class ServiceTransferScope:
    project_id: str
    service_code: str


@dataclass(frozen=True)
class EnvironmentTransferScope:
    project_id: str
    environment_id: str


def add_source_database_argument(parser: argparse.ArgumentParser) -> None:
    parser.add_argument(
        "--source", choices=("sqlite", "mysql", "postgresql"), required=True
    )


def add_target_database_argument(parser: argparse.ArgumentParser) -> None:
    parser.add_argument(
        "--target", choices=("sqlite", "mysql", "postgresql"), required=True
    )


def add_connection_arguments(parser: argparse.ArgumentParser) -> None:
    connection = parser.add_mutually_exclusive_group(required=True)
    connection.add_argument("--dsn")
    connection.add_argument("--dsn-env")


def add_dbtalk_runtime_arguments(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("--tz", default="UTC")
    parser.add_argument("--dbtalk-command", default="dbtalk")


def parse_args(argv: Sequence[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    subcommands = parser.add_subparsers(dest="command", required=True)

    exporter = subcommands.add_parser(
        "export", help="export one service deployment closure"
    )
    add_source_database_argument(exporter)
    exporter.add_argument("--project-id", required=True)
    exporter.add_argument("--service-code", required=True)
    exporter.add_argument("--output", type=Path, required=True)
    add_connection_arguments(exporter)
    add_dbtalk_runtime_arguments(exporter)

    importer = subcommands.add_parser(
        "import", help="import one service deployment closure"
    )
    add_target_database_argument(importer)
    importer.add_argument("--project-id", required=True)
    importer.add_argument("--input", type=Path, required=True)
    importer.add_argument("--mode", choices=("insert", "upsert"), required=True)
    add_connection_arguments(importer)
    add_dbtalk_runtime_arguments(importer)

    environment_exporter = subcommands.add_parser(
        "export-environment", help="export one Environment and its SSH key"
    )
    add_source_database_argument(environment_exporter)
    environment_exporter.add_argument("--project-id", required=True)
    environment_exporter.add_argument("--environment-id", required=True)
    environment_exporter.add_argument(
        "--source-secret-env",
        required=True,
        help="environment variable containing the source jwt.secret_key Fernet key",
    )
    environment_exporter.add_argument("--output", type=Path, required=True)
    add_connection_arguments(environment_exporter)
    add_dbtalk_runtime_arguments(environment_exporter)

    environment_importer = subcommands.add_parser(
        "import-environment", help="import one Environment and its SSH key"
    )
    add_target_database_argument(environment_importer)
    environment_importer.add_argument("--project-id", required=True)
    environment_importer.add_argument("--input", type=Path, required=True)
    environment_importer.add_argument(
        "--target-secret-env",
        required=True,
        help="environment variable containing the target jwt.secret_key Fernet key",
    )
    add_connection_arguments(environment_importer)
    add_dbtalk_runtime_arguments(environment_importer)

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
                if record.get("source") not in ("sqlite", "mysql", "postgresql"):
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


def require_columns(table: TableBlock, names: Sequence[str]) -> None:
    for name in names:
        require_column(table, name)


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


def service_tables(
    transfer: TransferFile, project_id: str, service_code: str
) -> TransferFile:
    available = table_map(transfer)
    missing = [name for name in SERVICE_TABLES if name not in available]
    if missing:
        raise ServiceTransferError(
            "service export is missing required tables: " + ", ".join(missing)
        )

    service_table = require_table(available, "service")
    service_code_index = require_column(service_table, "code")
    service_project_index = require_column(service_table, "project_id")
    service_rows = [
        row
        for row in service_table.rows
        if value_key(row[service_code_index]) == service_code
        and value_key(row[service_project_index]) == project_id
    ]
    if not service_rows:
        raise ServiceTransferError(
            f"service does not exist in project {project_id}: {service_code}"
        )
    if len(service_rows) > 1:
        raise ServiceTransferError(
            f"service code is not unique in project {project_id}: {service_code}"
        )

    service = dict(zip(service_table.column_names, service_rows[0], strict=True))
    service_id = value_key(service.get("id"))
    service_project_id = value_key(service.get("project_id"))
    application_id = value_key(service.get("application_id"))
    version_id = value_key(service.get("version_id"))
    if not service_id or not service_project_id or not application_id or not version_id:
        raise ServiceTransferError(
            "service row has incomplete project/application/version references"
        )

    application_table = require_table(available, "application")
    application = dict(
        zip(
            application_table.column_names,
            one_row_where(application_table, "id", application_id),
            strict=True,
        )
    )
    application_project_id = value_key(application.get("project_id"))
    if not application_project_id:
        raise ServiceTransferError("application row has no project reference")
    if service_project_id != project_id or application_project_id != project_id:
        raise ServiceTransferError(
            f"service {service_code} has inconsistent project references"
        )
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


def service_transfer_scope(transfer: TransferFile) -> ServiceTransferScope:
    if tuple(table.name for table in transfer.tables) != SERVICE_TABLES:
        raise ServiceTransferError(
            "service transfer has an unexpected table set or order"
        )
    service_table = require_table(table_map(transfer), "service")
    code_index = require_column(service_table, "code")
    project_index = require_column(service_table, "project_id")
    if len(service_table.rows) != 1:
        raise ServiceTransferError("service transfer must contain exactly one service")
    service_code = value_key(service_table.rows[0][code_index])
    if not service_code:
        raise ServiceTransferError("service transfer service code is empty")
    project_id = value_key(service_table.rows[0][project_index])
    if not project_id:
        raise ServiceTransferError("service transfer service project is empty")
    selected = service_tables(transfer, project_id, service_code)
    if selected.tables != transfer.tables:
        raise ServiceTransferError(
            "service transfer contains rows outside its deployment closure"
        )
    return ServiceTransferScope(project_id=project_id, service_code=service_code)


def validate_service_transfer(transfer: TransferFile) -> str:
    return service_transfer_scope(transfer).service_code


def retarget_service_transfer(
    transfer: TransferFile, target_project_id: str
) -> TransferFile:
    retargeted: list[TableBlock] = []
    for table in transfer.tables:
        if table.name == "project":
            # dbtalk needs the referenced table in the file, but the target
            # project's own row must never be imported or updated.
            retargeted.append(
                TableBlock(
                    name=table.name,
                    columns=table.columns,
                    primary_key=table.primary_key,
                    rows=(),
                )
            )
            continue
        if table.name not in ("application", "service", "route"):
            retargeted.append(table)
            continue
        project_index = require_column(table, "project_id")
        rows = replace_column_values(table.rows, project_index, target_project_id)
        retargeted.append(table_with_rows(table, rows))
    return TransferFile(header=transfer.header, tables=tuple(retargeted))


def environment_export_header(header: dict[str, Any]) -> dict[str, Any]:
    exported = dict(header)
    exported["orbit_scope"] = ENVIRONMENT_TRANSFER_SCOPE
    return exported


def environment_row(
    table: TableBlock, project_id: str, environment_id: str
) -> dict[str, Any]:
    rows = rows_where(table, "id", {environment_id})
    if not rows:
        raise ServiceTransferError(
            f"environment does not exist in project {project_id}: {environment_id}"
        )
    if len(rows) != 1:
        raise ServiceTransferError(
            f"environment export expects exactly one environment: {environment_id}"
        )
    environment = dict(zip(table.column_names, rows[0], strict=True))
    if value_key(environment.get("project_id")) != project_id:
        raise ServiceTransferError(
            f"environment does not exist in project {project_id}: {environment_id}"
        )
    return environment


def fernet_from_environment(variable_name: str, purpose: str) -> Fernet:
    secret = os.environ.get(variable_name)
    if secret is None or not secret.strip():
        raise ServiceTransferError(
            f"{purpose} secret environment variable is not set: {variable_name}"
        )
    try:
        encoded = secret.strip().encode("ascii")
        encoded += b"=" * (-len(encoded) % 4)
        return Fernet(encoded)
    except (UnicodeEncodeError, ValueError):
        raise ServiceTransferError(
            f"{purpose} secret environment variable is not a valid Fernet key: "
            f"{variable_name}"
        ) from None


def decrypt_environment_private_key(fernet: Fernet, encrypted_value: Any) -> str:
    if not isinstance(encrypted_value, str) or not encrypted_value:
        raise ServiceTransferError(
            "environment credential encrypted private key is empty"
        )
    try:
        plaintext = fernet.decrypt(encrypted_value.encode("ascii"))
        private_key = plaintext.decode("utf-8")
    except (InvalidToken, UnicodeError, ValueError):
        raise ServiceTransferError(
            "environment credential private key could not be decrypted"
        ) from None
    if not private_key:
        raise ServiceTransferError("environment credential private key is empty")
    return private_key


def encrypt_environment_private_key(fernet: Fernet, private_key: Any) -> str:
    if not isinstance(private_key, str) or not private_key:
        raise ServiceTransferError("environment transfer private key is empty")
    try:
        return fernet.encrypt(private_key.encode("utf-8")).decode("ascii")
    except UnicodeError:
        raise ServiceTransferError(
            "environment transfer private key could not be encrypted"
        ) from None


def renamed_column_table(
    table: TableBlock,
    source_name: str,
    target_name: str,
    rows: Sequence[tuple[Any, ...]],
) -> TableBlock:
    source_index = require_column(table, source_name)
    if target_name in table.column_names:
        raise ServiceTransferError(
            f"table {table.name} contains both {source_name} and {target_name}"
        )
    columns = list(table.columns)
    columns[source_index] = {**columns[source_index], "name": target_name}
    return TableBlock(
        name=table.name,
        columns=tuple(columns),
        primary_key=table.primary_key,
        rows=tuple(rows),
    )


def table_with_rows(table: TableBlock, rows: Sequence[tuple[Any, ...]]) -> TableBlock:
    return TableBlock(table.name, table.columns, table.primary_key, tuple(rows))


def empty_table(table: TableBlock) -> TableBlock:
    return table_with_rows(table, ())


def replace_value(row: tuple[Any, ...], index: int, value: Any) -> tuple[Any, ...]:
    return row[:index] + (value,) + row[index + 1 :]


def replace_column_values(
    rows: Sequence[tuple[Any, ...]], index: int, value: Any
) -> tuple[tuple[Any, ...], ...]:
    return tuple(replace_value(row, index, value) for row in rows)


def select_environment_transfer(
    transfer: TransferFile,
    project_id: str,
    environment_id: str,
    source_fernet: Fernet,
) -> TransferFile:
    available = table_map(transfer)
    missing = [name for name in ENVIRONMENT_TABLES if name not in available]
    if missing:
        raise ServiceTransferError(
            "environment export is missing required tables: " + ", ".join(missing)
        )

    project_table = require_table(available, "project")
    environment_table = require_table(available, "environment")
    require_columns(environment_table, ENVIRONMENT_REQUIRED_COLUMNS)
    environment = environment_row(environment_table, project_id, environment_id)
    target_type = value_key(environment.get("target_type"))
    credential_id = value_key(environment.get("ssh_credential_id"))
    credential_revision = value_key(environment.get("ssh_credential_revision"))
    gateway_index = require_column(environment_table, "gateway_application_id")
    environment_rows = rows_where(environment_table, "id", {environment_id})
    exported_environment = replace_value(environment_rows[0], gateway_index, None)
    selected_environment = table_with_rows(environment_table, (exported_environment,))
    header = environment_export_header(transfer.header)
    project_reference = empty_table(project_table)

    if target_type == "local":
        if credential_id is not None or credential_revision is not None:
            raise ServiceTransferError(
                "local environment must not bind an SSH credential"
            )
        return TransferFile(header, (project_reference, selected_environment))
    if target_type != "ssh":
        raise ServiceTransferError("environment target_type must be local or ssh")
    if not credential_id or not credential_revision:
        raise ServiceTransferError("SSH environment credential binding is incomplete")

    credential_table = require_table(available, "environment_credential")
    require_columns(
        credential_table,
        (*ENVIRONMENT_CREDENTIAL_REQUIRED_COLUMNS, "encrypted_private_key"),
    )
    credential_rows = rows_where(credential_table, "id", {credential_id})
    if len(credential_rows) != 1:
        raise ServiceTransferError(
            "SSH environment credential binding does not identify one credential"
        )
    credential = dict(
        zip(credential_table.column_names, credential_rows[0], strict=True)
    )
    if value_key(credential.get("project_id")) != project_id:
        raise ServiceTransferError(
            "SSH environment credential belongs to a different project"
        )
    if value_key(credential.get("revision")) != credential_revision:
        raise ServiceTransferError("SSH environment credential revision does not match")
    encrypted_index = require_column(credential_table, "encrypted_private_key")
    private_key = decrypt_environment_private_key(
        source_fernet, credential_rows[0][encrypted_index]
    )
    exported_credential = replace_value(
        credential_rows[0], encrypted_index, private_key
    )
    selected_credential = renamed_column_table(
        credential_table,
        "encrypted_private_key",
        "private_key",
        (exported_credential,),
    )
    return TransferFile(
        header, (project_reference, selected_credential, selected_environment)
    )


def environment_transfer_scope(transfer: TransferFile) -> EnvironmentTransferScope:
    if transfer.header.get("orbit_scope") != ENVIRONMENT_TRANSFER_SCOPE:
        raise ServiceTransferError("transfer file is not an Orbit environment export")
    table_names = tuple(table.name for table in transfer.tables)
    if table_names not in (("project", "environment"), ENVIRONMENT_TABLES):
        raise ServiceTransferError(
            "environment transfer has an unexpected table set or order"
        )

    tables = table_map(transfer)
    project_table = require_table(tables, "project")
    if project_table.rows:
        raise ServiceTransferError("environment transfer must not contain Project rows")
    environment_table = require_table(tables, "environment")
    require_columns(environment_table, ENVIRONMENT_REQUIRED_COLUMNS)
    if len(environment_table.rows) != 1:
        raise ServiceTransferError(
            "environment transfer must contain exactly one environment"
        )
    environment = dict(
        zip(environment_table.column_names, environment_table.rows[0], strict=True)
    )
    project_id = value_key(environment.get("project_id"))
    environment_id = value_key(environment.get("id"))
    target_type = value_key(environment.get("target_type"))
    if not project_id or not environment_id:
        raise ServiceTransferError("environment transfer identity is incomplete")
    if environment.get("gateway_application_id") is not None:
        raise ServiceTransferError(
            "environment transfer must not contain a Gateway binding"
        )

    credential_id = value_key(environment.get("ssh_credential_id"))
    credential_revision = value_key(environment.get("ssh_credential_revision"))
    if target_type == "local":
        if table_names != ("project", "environment"):
            raise ServiceTransferError(
                "local environment transfer must not contain an SSH credential"
            )
        if credential_id is not None or credential_revision is not None:
            raise ServiceTransferError(
                "local environment must not bind an SSH credential"
            )
    elif target_type == "ssh":
        if table_names != ENVIRONMENT_TABLES:
            raise ServiceTransferError(
                "SSH environment transfer must contain its credential"
            )
        if not credential_id or not credential_revision:
            raise ServiceTransferError(
                "SSH environment credential binding is incomplete"
            )
        credential_table = require_table(tables, "environment_credential")
        require_columns(
            credential_table,
            (*ENVIRONMENT_CREDENTIAL_REQUIRED_COLUMNS, "private_key"),
        )
        if "encrypted_private_key" in credential_table.column_names:
            raise ServiceTransferError(
                "environment transfer credential must contain a plaintext private_key"
            )
        private_key_index = require_column(credential_table, "private_key")
        if len(credential_table.rows) != 1:
            raise ServiceTransferError(
                "SSH environment transfer must contain exactly one credential"
            )
        credential = dict(
            zip(credential_table.column_names, credential_table.rows[0], strict=True)
        )
        if value_key(credential.get("id")) != credential_id:
            raise ServiceTransferError("SSH environment credential ID does not match")
        if value_key(credential.get("project_id")) != project_id:
            raise ServiceTransferError(
                "SSH environment credential belongs to a different project"
            )
        if value_key(credential.get("revision")) != credential_revision:
            raise ServiceTransferError(
                "SSH environment credential revision does not match"
            )
        if (
            not isinstance(credential_table.rows[0][private_key_index], str)
            or not credential_table.rows[0][private_key_index]
        ):
            raise ServiceTransferError("environment transfer private key is empty")
    else:
        raise ServiceTransferError("environment target_type must be local or ssh")
    return EnvironmentTransferScope(
        project_id=project_id, environment_id=environment_id
    )


def retarget_environment_transfer(
    transfer: TransferFile,
    target_project_id: str,
    target_project_code: str,
    target_fernet: Fernet,
) -> TransferFile:
    retargeted: dict[str, TableBlock] = {}
    for table in transfer.tables:
        if table.name == "project":
            if table.rows:
                raise ServiceTransferError(
                    "environment transfer must not contain Project rows"
                )
            retargeted[table.name] = table
            continue
        project_index = require_column(table, "project_id")
        if table.name == "environment":
            code_index = require_column(table, "code")
            gateway_index = require_column(table, "gateway_application_id")
            rows = replace_column_values(table.rows, project_index, target_project_id)
            rows = replace_column_values(rows, code_index, target_project_code)
            rows = replace_column_values(rows, gateway_index, None)
            retargeted[table.name] = table_with_rows(table, rows)
            continue
        if table.name == "environment_credential":
            private_key_index = require_column(table, "private_key")
            project_rows = replace_column_values(
                table.rows, project_index, target_project_id
            )
            rows = tuple(
                replace_value(
                    row,
                    private_key_index,
                    encrypt_environment_private_key(
                        target_fernet, row[private_key_index]
                    ),
                )
                for row in project_rows
            )
            retargeted[table.name] = renamed_column_table(
                table,
                "private_key",
                "encrypted_private_key",
                rows,
            )
            continue
        raise ServiceTransferError(
            f"environment transfer has an unexpected table: {table.name}"
        )
    return TransferFile(
        header=transfer.header,
        tables=tuple(
            retargeted[table_name]
            for table_name in ENVIRONMENT_DBTALK_IMPORT_TABLES
            if table_name in retargeted
        ),
    )


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
        if not dsn_env:
            raise ServiceTransferError("--dsn-env must not be empty")
        return ["--dsn-env", dsn_env]
    if not dsn:
        raise ServiceTransferError("--dsn must not be empty")
    return ["--dsn", dsn]


def dbtalk_result(
    command: str, arguments: Sequence[str], *, operation: str
) -> subprocess.CompletedProcess[str]:
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
    return result


def run_dbtalk(command: str, arguments: Sequence[str], *, operation: str) -> None:
    dbtalk_result(command, arguments, operation=operation)


def dbtalk_query_rows(
    command: str,
    connection: Sequence[str],
    sql: str,
    parameter_name: str,
    parameter_value: str,
    operation: str,
) -> list[dict[str, Any]]:
    result = dbtalk_result(
        command,
        [
            "query",
            *connection,
            "--sql",
            sql,
            "--param",
            parameter_name + "=" + json.dumps(parameter_value),
            "--format",
            "json",
        ],
        operation=operation,
    )
    try:
        payload = json.loads(result.stdout)
    except (AttributeError, json.JSONDecodeError) as error:
        raise ServiceTransferError(
            f"dbtalk {operation} returned invalid JSON"
        ) from error
    rows = payload.get("rows") if isinstance(payload, dict) else None
    if not isinstance(rows, list) or not all(isinstance(row, dict) for row in rows):
        raise ServiceTransferError(f"dbtalk {operation} returned invalid JSON")
    return rows


def target_project(
    command: str, connection: Sequence[str], project_id: str
) -> dict[str, Any] | None:
    rows = dbtalk_query_rows(
        command,
        connection,
        "SELECT id, code FROM project WHERE id = :project_id",
        "project_id",
        project_id,
        "project lookup",
    )
    if len(rows) != 1 or value_key(rows[0].get("id")) != project_id:
        return None
    return rows[0]


def target_project_exists(
    command: str, connection: Sequence[str], project_id: str
) -> bool:
    return target_project(command, connection, project_id) is not None


def target_project_has_environment(
    command: str, connection: Sequence[str], project_id: str
) -> bool:
    rows = dbtalk_query_rows(
        command,
        connection,
        "SELECT id FROM environment WHERE project_id = :project_id",
        "project_id",
        project_id,
        "environment lookup",
    )
    return bool(rows)


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
        selected = service_tables(
            load_transfer(service_export), args.project_id, args.service_code
        )
        write_transfer(output, selected)
    return output


def export_environment(args: argparse.Namespace) -> Path:
    output = args.output.resolve()
    connection = connection_arguments(args.dsn, args.dsn_env)
    source_fernet = fernet_from_environment(args.source_secret_env, "source")
    with tempfile.TemporaryDirectory(prefix="orbit-environment-transfer-") as directory:
        environment_export = Path(directory) / "environment.jsonl"
        run_dbtalk(
            args.dbtalk_command,
            [
                "export",
                "--source",
                args.source,
                "--output",
                str(environment_export),
                *connection,
                *(
                    argument
                    for table_name in ENVIRONMENT_TABLES
                    for argument in ("--include-table", table_name)
                ),
                "--tz",
                args.tz,
            ],
            operation="environment export",
        )
        selected = select_environment_transfer(
            load_transfer(environment_export),
            args.project_id,
            args.environment_id,
            source_fernet,
        )
        write_transfer(output, selected)
    return output


def import_service(args: argparse.Namespace) -> str:
    transfer = load_transfer(args.input.resolve())
    scope = service_transfer_scope(transfer)
    connection = connection_arguments(args.dsn, args.dsn_env)
    if not target_project_exists(args.dbtalk_command, connection, args.project_id):
        raise ServiceTransferError(f"target project does not exist: {args.project_id}")
    target_transfer = retarget_service_transfer(transfer, args.project_id)
    with tempfile.TemporaryDirectory(prefix="orbit-service-import-") as directory:
        target_input = Path(directory) / "service.jsonl"
        write_transfer(target_input, target_transfer)
        run_dbtalk(
            args.dbtalk_command,
            [
                "import",
                "--target",
                args.target,
                "--input",
                str(target_input),
                "--mode",
                args.mode,
                *connection,
                "--tz",
                args.tz,
            ],
            operation="import",
        )
    return scope.service_code


def import_environment(args: argparse.Namespace) -> str:
    transfer = load_transfer(args.input.resolve())
    scope = environment_transfer_scope(transfer)
    connection = connection_arguments(args.dsn, args.dsn_env)
    target_fernet = fernet_from_environment(args.target_secret_env, "target")
    target = target_project(args.dbtalk_command, connection, args.project_id)
    if target is None:
        raise ServiceTransferError(f"target project does not exist: {args.project_id}")
    target_project_code = value_key(target.get("code"))
    if not target_project_code:
        raise ServiceTransferError(f"target project has no code: {args.project_id}")
    if target_project_has_environment(args.dbtalk_command, connection, args.project_id):
        raise ServiceTransferError(
            f"target project already has an environment: {args.project_id}"
        )
    target_transfer = retarget_environment_transfer(
        transfer, args.project_id, target_project_code, target_fernet
    )
    with tempfile.TemporaryDirectory(prefix="orbit-environment-import-") as directory:
        target_input = Path(directory) / "environment.jsonl"
        write_transfer(target_input, target_transfer)
        run_dbtalk(
            args.dbtalk_command,
            [
                "import",
                "--target",
                args.target,
                "--input",
                str(target_input),
                "--mode",
                "insert",
                *connection,
                "--tz",
                args.tz,
            ],
            operation="environment import",
        )
    return scope.environment_id


def main(argv: Sequence[str]) -> int:
    args = parse_args(argv)
    try:
        if args.command == "export":
            output = export_service(args)
            print(f"service transfer written to {output}")
        elif args.command == "import":
            service_code = import_service(args)
            print(f"service transfer imported for {service_code}")
        elif args.command == "export-environment":
            output = export_environment(args)
            print(f"environment transfer written to {output}")
        elif args.command == "import-environment":
            environment_id = import_environment(args)
            print(f"environment transfer imported for {environment_id}")
        else:
            raise ServiceTransferError(f"unsupported transfer command: {args.command}")
    except (OSError, ServiceTransferError) as error:
        print(f"database transfer blocked: {error}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
