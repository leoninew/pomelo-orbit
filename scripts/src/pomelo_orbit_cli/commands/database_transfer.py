"""Transfer Orbit service closures and selected Environments through dbtalk."""

from __future__ import annotations

import contextlib
import tempfile
from dataclasses import dataclass, replace
from pathlib import Path
from typing import Any, Sequence, TextIO, cast
from zoneinfo import ZoneInfo, ZoneInfoNotFoundError

import click
from cryptography.fernet import Fernet, InvalidToken
from dbtalk.database import DatabaseClient, create_client
from dbtalk.database.transfer import (
    ColumnDefinition,
    DatabaseOperationError,
    DatabaseTransferError,
    ExportOptions,
    ImportOptions,
    TableBlock as DbtalkTableBlock,
    TableBlockHeader,
    TransferConnection,
    TransferHeader,
    TransferMode,
    export_database,
    import_database,
    read_jsonl,
    write_jsonl,
)

from pomelo_orbit_cli.context import app_context
from pomelo_orbit_cli.settings import DatabaseConnection, Settings


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


class ServiceTransferError(DatabaseTransferError):
    """Raised when a service transfer cannot be safely prepared."""


@dataclass(frozen=True)
class TableBlock:
    name: str
    columns: tuple[ColumnDefinition, ...]
    primary_key: tuple[str, ...]
    rows: tuple[tuple[Any, ...], ...]

    @property
    def column_names(self) -> tuple[str, ...]:
        return tuple(column.name for column in self.columns)

    def row_maps(self) -> tuple[dict[str, Any], ...]:
        names = self.column_names
        return tuple(dict(zip(names, row, strict=True)) for row in self.rows)


@dataclass(frozen=True)
class TransferFile:
    header: TransferHeader
    tables: tuple[TableBlock, ...]


@dataclass(frozen=True)
class ServiceTransferScope:
    project_id: str
    service_code: str


@dataclass(frozen=True)
class EnvironmentTransferScope:
    project_id: str
    environment_id: str


def load_transfer(path: Path) -> TransferFile:
    if not path.is_file():
        raise ServiceTransferError(f"transfer file does not exist: {path}")
    try:
        with path.open(encoding="utf-8") as input_file:
            header, tables = read_jsonl(input_file)
    except (OSError, DatabaseTransferError) as error:
        raise ServiceTransferError(f"could not read transfer file: {error}") from error
    if not tables:
        raise ServiceTransferError("JSONL contains no table blocks")
    return TransferFile(
        header=header,
        tables=tuple(
            TableBlock(
                name=table.header.name,
                columns=table.header.columns,
                primary_key=table.header.primary_key,
                rows=table.rows,
            )
            for table in tables
        ),
    )


def write_transfer(path: Path, transfer: TransferFile) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary: Path | None = None
    try:
        with tempfile.NamedTemporaryFile(
            "w", encoding="utf-8", newline="\n", dir=path.parent, delete=False
        ) as output:
            temporary = Path(output.name)
            write_jsonl(
                cast(TextIO, output.file),
                transfer.header,
                tuple(
                    DbtalkTableBlock(
                        header=TableBlockHeader(
                            name=table.name,
                            columns=table.columns,
                            primary_key=table.primary_key,
                        ),
                        rows=table.rows,
                    )
                    for table in transfer.tables
                ),
            )
        temporary.replace(path)
        temporary = None
    except (DatabaseTransferError, OSError) as error:
        raise ServiceTransferError("could not write transfer file") from error
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
    columns[source_index] = replace(columns[source_index], name=target_name)
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
    project_reference = empty_table(project_table)

    if target_type == "local":
        if credential_id is not None or credential_revision is not None:
            raise ServiceTransferError(
                "local environment must not bind an SSH credential"
            )
        return TransferFile(transfer.header, (project_reference, selected_environment))
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
        transfer.header, (project_reference, selected_credential, selected_environment)
    )


def environment_transfer_scope(transfer: TransferFile) -> EnvironmentTransferScope:
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


def timezone(value: str) -> ZoneInfo:
    """Resolve a user-facing IANA timezone name without leaking implementation data."""
    try:
        return ZoneInfo(value)
    except ZoneInfoNotFoundError as error:
        raise ServiceTransferError(f"unknown timezone: {value}") from error


def transfer_connection(connection: DatabaseConnection) -> TransferConnection:
    """Adapt shared Orbit settings to the dbtalk transfer API."""
    return TransferConnection(driver=connection.driver, dsn=connection.dsn)


def export_from_database(
    connection: DatabaseConnection,
    output: Path,
    table_names: tuple[str, ...],
    tz: ZoneInfo,
    *,
    operation: str,
) -> None:
    try:
        export_database(
            ExportOptions(
                connection=transfer_connection(connection),
                output=output,
                include_tables=table_names,
                timezone=tz,
            )
        )
    except (DatabaseOperationError, DatabaseTransferError, OSError) as error:
        raise ServiceTransferError(f"database {operation} failed") from error


def import_into_database(
    connection: DatabaseConnection,
    input_path: Path,
    mode: TransferMode,
    tz: ZoneInfo,
    *,
    operation: str,
) -> None:
    try:
        import_database(
            ImportOptions(
                connection=transfer_connection(connection),
                input=input_path,
                mode=mode,
                timezone=tz,
            )
        )
    except (DatabaseOperationError, DatabaseTransferError, OSError) as error:
        raise ServiceTransferError(f"database {operation} failed") from error


@contextlib.contextmanager
def database_client(connection: DatabaseConnection):
    """Create and close the dbtalk client used for target preflight queries."""
    client = create_client(connection.dsn)
    try:
        client.connect()
        yield client
    except DatabaseOperationError as error:
        raise ServiceTransferError("database query failed") from error
    finally:
        client.close()


def query_rows(
    client: DatabaseClient, statement: str, parameters: dict[str, str]
) -> list[dict[str, Any]]:
    try:
        result = client.query(statement, parameters)
    except DatabaseOperationError as error:
        raise ServiceTransferError("database query failed") from error
    return [dict(zip(result.columns, row, strict=True)) for row in result.rows]


def target_project(client: DatabaseClient, project_id: str) -> dict[str, Any] | None:
    rows = query_rows(
        client,
        "SELECT id, code FROM project WHERE id = :project_id",
        {"project_id": project_id},
    )
    if len(rows) != 1 or value_key(rows[0].get("id")) != project_id:
        return None
    return rows[0]


def target_project_has_environment(client: DatabaseClient, project_id: str) -> bool:
    return bool(
        query_rows(
            client,
            "SELECT id FROM environment WHERE project_id = :project_id",
            {"project_id": project_id},
        )
    )


def export_service(
    settings: Settings,
    *,
    project_id: str,
    service_code: str,
    output: Path,
    tz: str,
) -> Path:
    """Export the selected service closure using the configured source database."""
    output = output.resolve()
    connection = settings.connection
    with tempfile.TemporaryDirectory(prefix="orbit-service-transfer-") as directory:
        source_export = Path(directory) / "service.jsonl"
        export_from_database(
            connection,
            source_export,
            SERVICE_TABLES,
            timezone(tz),
            operation="export",
        )
        write_transfer(
            output,
            service_tables(load_transfer(source_export), project_id, service_code),
        )
    return output


def export_environment(
    settings: Settings,
    *,
    project_id: str,
    environment_id: str,
    output: Path,
    tz: str,
) -> Path:
    """Export the selected Environment and reformat its SSH key for transport."""
    output = output.resolve()
    connection = settings.connection
    source_fernet = settings.fernet
    with tempfile.TemporaryDirectory(prefix="orbit-environment-transfer-") as directory:
        source_export = Path(directory) / "environment.jsonl"
        export_from_database(
            connection,
            source_export,
            ENVIRONMENT_TABLES,
            timezone(tz),
            operation="environment export",
        )
        write_transfer(
            output,
            select_environment_transfer(
                load_transfer(source_export),
                project_id,
                environment_id,
                source_fernet,
            ),
        )
    return output


def import_service(
    settings: Settings,
    *,
    project_id: str,
    input_path: Path,
    mode: TransferMode,
    tz: str,
) -> str:
    """Validate, retarget, and import a service closure into the configured database."""
    transfer = load_transfer(input_path.resolve())
    scope = service_transfer_scope(transfer)
    connection = settings.connection
    with database_client(connection) as client:
        if target_project(client, project_id) is None:
            raise ServiceTransferError(f"target project does not exist: {project_id}")
    target_transfer = retarget_service_transfer(transfer, project_id)
    with tempfile.TemporaryDirectory(prefix="orbit-service-import-") as directory:
        target_input = Path(directory) / "service.jsonl"
        write_transfer(target_input, target_transfer)
        import_into_database(
            connection,
            target_input,
            mode,
            timezone(tz),
            operation="import",
        )
    return scope.service_code


def import_environment(
    settings: Settings,
    *,
    project_id: str,
    input_path: Path,
    tz: str,
) -> str:
    """Restore a selected Environment into the configured target project."""
    transfer = load_transfer(input_path.resolve())
    scope = environment_transfer_scope(transfer)
    connection = settings.connection
    target_fernet = settings.fernet
    with database_client(connection) as client:
        target = target_project(client, project_id)
        if target is None:
            raise ServiceTransferError(f"target project does not exist: {project_id}")
        target_project_code = value_key(target.get("code"))
        if not target_project_code:
            raise ServiceTransferError(f"target project has no code: {project_id}")
        if target_project_has_environment(client, project_id):
            raise ServiceTransferError(
                f"target project already has an environment: {project_id}"
            )
    target_transfer = retarget_environment_transfer(
        transfer, project_id, target_project_code, target_fernet
    )
    with tempfile.TemporaryDirectory(prefix="orbit-environment-import-") as directory:
        target_input = Path(directory) / "environment.jsonl"
        write_transfer(target_input, target_transfer)
        import_into_database(
            connection,
            target_input,
            "insert",
            timezone(tz),
            operation="environment import",
        )
    return scope.environment_id


@click.group(name="database-transfer")
def database_transfer() -> None:
    """Export and import Orbit services and Environments."""


@database_transfer.command("export")
@click.option("--project-id", required=True, help="Source Orbit Project ID.")
@click.option("--service-code", required=True, help="Service code within the Project.")
@click.option(
    "--output", type=click.Path(path_type=Path, dir_okay=False), required=True
)
@click.option("--tz", default="UTC", show_default=True, help="IANA timezone name.")
@click.pass_context
def export_command(
    ctx: click.Context,
    project_id: str,
    service_code: str,
    output: Path,
    tz: str,
) -> None:
    """Export one service deployment closure."""
    settings = app_context(ctx).settings
    try:
        written = export_service(
            settings,
            project_id=project_id,
            service_code=service_code,
            output=output,
            tz=tz,
        )
    except ServiceTransferError as error:
        raise click.ClickException(f"database transfer blocked: {error}") from error
    click.echo(f"service transfer written to {written}")


@database_transfer.command("import")
@click.option("--project-id", required=True, help="Target Orbit Project ID.")
@click.option(
    "--input",
    "input_path",
    type=click.Path(path_type=Path, dir_okay=False),
    required=True,
)
@click.option("--mode", type=click.Choice(("insert", "upsert")), required=True)
@click.option("--tz", default="UTC", show_default=True, help="IANA timezone name.")
@click.pass_context
def import_command(
    ctx: click.Context,
    project_id: str,
    input_path: Path,
    mode: str,
    tz: str,
) -> None:
    """Import one service deployment closure."""
    settings = app_context(ctx).settings
    try:
        service_code = import_service(
            settings,
            project_id=project_id,
            input_path=input_path,
            mode=cast(TransferMode, mode),
            tz=tz,
        )
    except ServiceTransferError as error:
        raise click.ClickException(f"database transfer blocked: {error}") from error
    click.echo(f"service transfer imported for {service_code}")


@database_transfer.command("export-environment")
@click.option("--project-id", required=True, help="Source Orbit Project ID.")
@click.option("--environment-id", required=True, help="Source Environment ID.")
@click.option(
    "--output", type=click.Path(path_type=Path, dir_okay=False), required=True
)
@click.option("--tz", default="UTC", show_default=True, help="IANA timezone name.")
@click.pass_context
def export_environment_command(
    ctx: click.Context,
    project_id: str,
    environment_id: str,
    output: Path,
    tz: str,
) -> None:
    """Export one Environment and its SSH key."""
    settings = app_context(ctx).settings
    try:
        written = export_environment(
            settings,
            project_id=project_id,
            environment_id=environment_id,
            output=output,
            tz=tz,
        )
    except ServiceTransferError as error:
        raise click.ClickException(f"database transfer blocked: {error}") from error
    click.echo(f"environment transfer written to {written}")


@database_transfer.command("import-environment")
@click.option("--project-id", required=True, help="Target Orbit Project ID.")
@click.option(
    "--input",
    "input_path",
    type=click.Path(path_type=Path, dir_okay=False),
    required=True,
)
@click.option("--tz", default="UTC", show_default=True, help="IANA timezone name.")
@click.pass_context
def import_environment_command(
    ctx: click.Context,
    project_id: str,
    input_path: Path,
    tz: str,
) -> None:
    """Import one Environment and its SSH key."""
    settings = app_context(ctx).settings
    try:
        environment_id = import_environment(
            settings,
            project_id=project_id,
            input_path=input_path,
            tz=tz,
        )
    except ServiceTransferError as error:
        raise click.ClickException(f"database transfer blocked: {error}") from error
    click.echo(f"environment transfer imported for {environment_id}")
