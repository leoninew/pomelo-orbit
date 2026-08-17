#!/usr/bin/env python3
"""Export, import, and convert complete SQLite and MySQL table-data files."""

from __future__ import annotations

import argparse
import base64
import json
import re
import sqlite3
import sys
import tempfile
from datetime import date, datetime, time
from decimal import Decimal
from pathlib import Path
from typing import Any, Sequence


ARCHIVE_FORMAT = "database-transfer/v1"
ARCHIVE_PREFIX = "-- database-transfer-payload: "
IDENTIFIER = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*$")

SERVICE_TABLES = (
    "project",
    "application",
    "version",
    "version_component",
    "version_component_env",
    "version_component_endpoint",
    "version_component_mount",
    "version_component_dependency",
    "version_component_healthcheck",
    "version_component_resource",
    "version_component_tmpfs",
    "version_component_ulimit",
    "version_component_device",
    "gateway_config",
    "service",
    "service_env",
    "service_component",
    "service_component_env",
    "service_component_mount",
    "service_component_resource",
    "service_component_endpoint",
    "route",
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


class TransferError(RuntimeError):
    pass


def parse_args(argv: Sequence[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    subcommands = parser.add_subparsers(dest="command", required=True)

    export = subcommands.add_parser("export", help="export all table data from SQLite or MySQL")
    export.add_argument("--source", choices=("sqlite", "mysql"), required=True)
    export.add_argument("--output", type=Path, required=True)
    add_connection_arguments(export)

    importer = subcommands.add_parser("import", help="import a generated table-data file into SQLite or MySQL")
    importer.add_argument("--target", choices=("sqlite", "mysql"), required=True)
    importer.add_argument("--input", type=Path, required=True)
    importer.add_argument("--replace", action="store_true", help="delete the exported tables before inserting data")
    add_connection_arguments(importer)

    service_export = subcommands.add_parser("export-service", help="export one SQLite service deployment closure")
    service_export.add_argument("--sqlite-path", type=Path, required=True)
    service_export.add_argument("--service-code", required=True)
    service_export.add_argument("--output", type=Path, required=True)

    service_import = subcommands.add_parser("import-service", help="import one SQLite service deployment closure without deleting data")
    service_import.add_argument("--sqlite-path", type=Path, required=True)
    service_import.add_argument("--input", type=Path, required=True)

    converter = subcommands.add_parser("convert", help="convert a generated SQLite or MySQL file to the other SQL dialect")
    converter.add_argument("--from", dest="source_format", choices=("sqlite", "mysql"), required=True)
    converter.add_argument("--to", dest="target_format", choices=("sqlite", "mysql"), required=True)
    converter.add_argument("--input", type=Path, required=True)
    converter.add_argument("--output", type=Path, required=True)

    return parser.parse_args(argv)


def add_connection_arguments(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("--sqlite-path", type=Path)
    parser.add_argument("--mysql-host")
    parser.add_argument("--mysql-port", type=int)
    parser.add_argument("--mysql-user")
    parser.add_argument("--mysql-password")
    parser.add_argument("--mysql-database")


def quote_identifier(identifier: str, driver: str) -> str:
    if not IDENTIFIER.fullmatch(identifier):
        raise TransferError(f"invalid database identifier: {identifier}")
    return f'"{identifier}"' if driver == "sqlite" else f"`{identifier}`"


def connect_sqlite_read_only(path: Path) -> sqlite3.Connection:
    if not path.is_file():
        raise TransferError(f"SQLite database does not exist: {path}")
    connection = sqlite3.connect(f"file:{path.as_posix()}?mode=ro", uri=True)
    connection.row_factory = sqlite3.Row
    return connection


def connect_sqlite_read_write(path: Path) -> sqlite3.Connection:
    if not path.is_file():
        raise TransferError(f"SQLite database does not exist: {path}")
    connection = sqlite3.connect(path)
    connection.execute("PRAGMA foreign_keys = ON")
    return connection


def mysql_connection(args: argparse.Namespace) -> Any:
    required = ("mysql_host", "mysql_port", "mysql_user", "mysql_password", "mysql_database")
    missing = [name for name in required if getattr(args, name, None) in (None, "")]
    if missing:
        raise TransferError("MySQL connection arguments are required: " + ", ".join("--" + name.replace("_", "-") for name in missing))
    try:
        import pymysql  # type: ignore[import-untyped]
    except ImportError as error:
        raise TransferError("PyMySQL is required for MySQL operations") from error
    try:
        return pymysql.connect(
            host=args.mysql_host,
            port=args.mysql_port,
            user=args.mysql_user,
            password=args.mysql_password,
            database=args.mysql_database,
            charset="utf8mb4",
            autocommit=False,
        )
    except pymysql.MySQLError as error:
        raise TransferError(f"failed to connect to MySQL: {error}") from error


def ensure_sqlite_integrity(connection: sqlite3.Connection) -> None:
    integrity = connection.execute("PRAGMA integrity_check").fetchone()[0]
    if integrity != "ok":
        raise TransferError(f"integrity_check failed: {integrity}")
    foreign_keys = list(connection.execute("PRAGMA foreign_key_check"))
    if foreign_keys:
        raise TransferError(f"foreign_key_check returned {len(foreign_keys)} row(s)")


def encode_value(value: Any) -> Any:
    if value is None or isinstance(value, str | int | float | bool):
        return value
    if isinstance(value, bytes):
        return {"type": "blob", "base64": base64.b64encode(value).decode("ascii")}
    if isinstance(value, Decimal):
        return {"type": "decimal", "value": str(value)}
    if isinstance(value, datetime):
        return {"type": "datetime", "value": value.isoformat(sep=" ")}
    if isinstance(value, date):
        return {"type": "date", "value": value.isoformat()}
    if isinstance(value, time):
        return {"type": "time", "value": value.isoformat()}
    raise TransferError(f"unsupported database value type: {type(value).__name__}")


def decode_value(value: Any) -> Any:
    if not isinstance(value, dict):
        return value
    value_type = value.get("type")
    if value_type == "blob" and isinstance(value.get("base64"), str):
        return base64.b64decode(value["base64"])
    if value_type in ("decimal", "datetime", "date", "time") and isinstance(value.get("value"), str):
        return value["value"]
    raise TransferError("invalid typed value in database transfer file")


def service_table_names(table_names: Sequence[str]) -> tuple[str, ...]:
    available = set(table_names)
    missing = [name for name in SERVICE_TABLES if name not in available]
    if missing:
        raise TransferError("service export is missing required tables: " + ", ".join(missing))
    return SERVICE_TABLES


def sqlite_table_archive(
    connection: sqlite3.Connection,
    table_name: str,
    *,
    where_column: str | None = None,
    values: Sequence[str] = (),
) -> dict[str, Any]:
    columns = tuple(str(column[1]) for column in connection.execute(f"PRAGMA table_info({quote_identifier(table_name, 'sqlite')})"))
    if not columns:
        raise TransferError(f"SQLite table has no columns: {table_name}")
    query = f"SELECT {', '.join(quote_identifier(column, 'sqlite') for column in columns)} FROM {quote_identifier(table_name, 'sqlite')}"
    parameters: tuple[str, ...] = ()
    if where_column is not None:
        if where_column not in columns:
            raise TransferError(f"SQLite table {table_name} has no {where_column} column")
        unique_values = tuple(dict.fromkeys(values))
        if unique_values:
            query += f" WHERE {quote_identifier(where_column, 'sqlite')} IN ({', '.join('?' for _ in unique_values)})"
            parameters = unique_values
        else:
            query += " WHERE 1 = 0"
    data = [[encode_value(record[column]) for column in columns] for record in connection.execute(query, parameters)]
    return {"name": table_name, "columns": list(columns), "rows": data}


def sqlite_service_archive(
    connection: sqlite3.Connection,
    table_names: Sequence[str],
    service_code: str,
) -> list[dict[str, Any]]:
    service_table_names(table_names)
    service_rows = list(
        connection.execute(
            'SELECT "id", "application_id", "version_id" FROM "service" WHERE "code" = ?',
            (service_code,),
        )
    )
    if not service_rows:
        raise TransferError(f"service does not exist: {service_code}")
    if len(service_rows) > 1:
        raise TransferError(f"service code is not unique: {service_code}")
    service = service_rows[0]
    application_id = str(service["application_id"])
    application = connection.execute('SELECT "project_id" FROM "application" WHERE "id" = ?', (application_id,)).fetchone()
    if application is None:
        raise TransferError(f"service {service_code} references a missing application")
    project_id = str(application["project_id"])

    version_ids: list[str] = []
    version_id: str | None = str(service["version_id"])
    while version_id is not None:
        if version_id in version_ids:
            raise TransferError(f"service {service_code} has a cyclic version lineage")
        version = connection.execute(
            'SELECT "application_id", "created_from_version_id" FROM "version" WHERE "id" = ?',
            (version_id,),
        ).fetchone()
        if version is None:
            raise TransferError(f"service {service_code} references a missing version")
        if str(version["application_id"]) != application_id:
            raise TransferError(f"service {service_code} version lineage crosses applications")
        version_ids.append(version_id)
        parent_id = version["created_from_version_id"]
        version_id = str(parent_id) if parent_id else None
    version_ids.reverse()

    version_component_ids = [
        str(row[0])
        for row in connection.execute(
            f'SELECT "id" FROM "version_component" WHERE "version_id" IN ({", ".join("?" for _ in version_ids)})',
            tuple(version_ids),
        )
    ]
    service_id = str(service["id"])
    service_component_ids = [
        str(row[0])
        for row in connection.execute('SELECT "id" FROM "service_component" WHERE "service_id" = ?', (service_id,))
    ]

    archives: list[dict[str, Any]] = []
    for table_name in SERVICE_TABLES:
        if table_name == "project":
            archives.append(sqlite_table_archive(connection, table_name, where_column="id", values=(project_id,)))
        elif table_name == "application":
            archives.append(sqlite_table_archive(connection, table_name, where_column="id", values=(application_id,)))
        elif table_name == "version":
            archives.append(sqlite_table_archive(connection, table_name, where_column="id", values=version_ids))
        elif table_name == "version_component":
            archives.append(sqlite_table_archive(connection, table_name, where_column="id", values=version_component_ids))
        elif table_name in VERSION_COMPONENT_CHILD_TABLES:
            archives.append(sqlite_table_archive(connection, table_name, where_column="component_id", values=version_component_ids))
        elif table_name == "gateway_config":
            archives.append(sqlite_table_archive(connection, table_name, where_column="application_id", values=(application_id,)))
        elif table_name == "service":
            archives.append(sqlite_table_archive(connection, table_name, where_column="id", values=(service_id,)))
        elif table_name == "service_env":
            archives.append(sqlite_table_archive(connection, table_name, where_column="service_id", values=(service_id,)))
        elif table_name == "service_component":
            archives.append(sqlite_table_archive(connection, table_name, where_column="id", values=service_component_ids))
        elif table_name in SERVICE_COMPONENT_CHILD_TABLES:
            archives.append(sqlite_table_archive(connection, table_name, where_column="service_component_id", values=service_component_ids))
        elif table_name == "route":
            archives.append(sqlite_table_archive(connection, table_name, where_column="service_id", values=(service_id,)))
        else:
            raise TransferError(f"service deployment table has no export rule: {table_name}")
    return archives


def archive_from_sqlite(path: Path) -> dict[str, Any]:
    connection = connect_sqlite_read_only(path)
    try:
        connection.execute("BEGIN")
        ensure_sqlite_integrity(connection)
        rows = connection.execute(
            "SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name"
        )
        table_names = tuple(str(row[0]) for row in rows)
        tables = [sqlite_table_archive(connection, name) for name in sorted(table_names)]
    finally:
        connection.close()
    return {"format": ARCHIVE_FORMAT, "driver": "sqlite", "tables": tables}


def archive_service_from_sqlite(path: Path, service_code: str) -> dict[str, Any]:
    connection = connect_sqlite_read_only(path)
    try:
        connection.execute("BEGIN")
        ensure_sqlite_integrity(connection)
        rows = connection.execute(
            "SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name"
        )
        table_names = tuple(str(row[0]) for row in rows)
        tables = sqlite_service_archive(connection, table_names, service_code)
    finally:
        connection.close()
    return {"format": ARCHIVE_FORMAT, "driver": "sqlite", "tables": tables}


def archive_from_mysql(args: argparse.Namespace) -> dict[str, Any]:
    connection = mysql_connection(args)
    try:
        with connection.cursor() as cursor:
            cursor.execute(
                "SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE' ORDER BY table_name"
            )
            table_names = tuple(str(row[0]) for row in cursor.fetchall())
            tables = []
            for name in table_names:
                cursor.execute(
                    "SELECT column_name FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = %s AND extra NOT LIKE '%% GENERATED' ORDER BY ordinal_position",
                    (name,),
                )
                columns = tuple(str(row[0]) for row in cursor.fetchall())
                if not columns:
                    raise TransferError(f"MySQL table has no columns: {name}")
                query = f"SELECT {', '.join(quote_identifier(column, 'mysql') for column in columns)} FROM {quote_identifier(name, 'mysql')}"
                cursor.execute(query)
                tables.append({"name": name, "columns": list(columns), "rows": [[encode_value(value) for value in row] for row in cursor.fetchall()]})
    finally:
        connection.close()
    return {"format": ARCHIVE_FORMAT, "driver": "mysql", "tables": tables}


def validate_archive(archive: Any) -> dict[str, Any]:
    if not isinstance(archive, dict) or archive.get("format") != ARCHIVE_FORMAT:
        raise TransferError("unsupported database transfer file")
    if archive.get("driver") not in ("sqlite", "mysql"):
        raise TransferError("database transfer file has an invalid driver")
    tables = archive.get("tables")
    if not isinstance(tables, list):
        raise TransferError("database transfer file has no tables")
    for table in tables:
        if not isinstance(table, dict) or not isinstance(table.get("name"), str) or not isinstance(table.get("columns"), list) or not isinstance(table.get("rows"), list):
            raise TransferError("database transfer file has an invalid table")
        quote_identifier(table["name"], "sqlite")
        for column in table["columns"]:
            if not isinstance(column, str):
                raise TransferError("database transfer file has an invalid column")
            quote_identifier(column, "sqlite")
        for row in table["rows"]:
            if not isinstance(row, list) or len(row) != len(table["columns"]):
                raise TransferError("database transfer file has an invalid row")
    return archive


def validate_service_archive(archive: Any) -> dict[str, Any]:
    archive = validate_archive(archive)
    if archive["driver"] != "sqlite":
        raise TransferError("service transfer file must use the SQLite format")
    tables = archive["tables"]
    if [table["name"] for table in tables] != list(SERVICE_TABLES):
        raise TransferError("service transfer file has an unexpected table set")
    service_table = tables[SERVICE_TABLES.index("service")]
    if "code" not in service_table["columns"] or len(service_table["rows"]) != 1:
        raise TransferError("service transfer file must contain exactly one service")
    return archive


def archive_payload_lines(archive: dict[str, Any]) -> list[str]:
    payload = base64.b64encode(json.dumps(archive, ensure_ascii=False, separators=(",", ":")).encode("utf-8")).decode("ascii")
    return [ARCHIVE_PREFIX + payload[index : index + 76] for index in range(0, len(payload), 76)]


def sqlite_sql_value(value: Any) -> str:
    value = decode_value(value)
    if value is None:
        return "NULL"
    if isinstance(value, bytes):
        return "X'" + value.hex().upper() + "'"
    if isinstance(value, int | float):
        return str(value)
    text = str(value)
    if "\x00" not in text:
        return "'" + text.replace("'", "''") + "'"
    return "CAST(X'" + text.encode("utf-8").hex().upper() + "' AS TEXT)"


def mysql_sql_value(value: Any) -> str:
    value = decode_value(value)
    if value is None:
        return "NULL"
    if isinstance(value, bytes):
        return "X'" + value.hex().upper() + "'"
    if isinstance(value, int | float):
        return str(value)
    text = str(value)
    escaped = (
        text.replace("\\", "\\\\")
        .replace("\x00", "\\0")
        .replace("\n", "\\n")
        .replace("\r", "\\r")
        .replace("\x1a", "\\Z")
        .replace("'", "''")
    )
    return "'" + escaped + "'"


def render_sql_file(archive: dict[str, Any], driver: str, *, include_deletes: bool = True) -> str:
    archive = dict(validate_archive(archive))
    archive["driver"] = driver
    tables = archive["tables"]
    lines = [
        f"-- Database transfer SQL for {driver}. Generated by scripts/database_transfer.py.",
        *archive_payload_lines(archive),
    ]
    if driver == "sqlite":
        lines.extend(("PRAGMA foreign_keys = OFF;", "BEGIN;"))
    else:
        lines.append("SET FOREIGN_KEY_CHECKS = 0;")
    if include_deletes:
        lines.extend(f"DELETE FROM {quote_identifier(table['name'], driver)};" for table in reversed(tables))
    value_renderer = sqlite_sql_value if driver == "sqlite" else mysql_sql_value
    for table in tables:
        columns = table["columns"]
        names = ", ".join(quote_identifier(column, driver) for column in columns)
        for row in table["rows"]:
            values = ", ".join(value_renderer(value) for value in row)
            lines.append(f"INSERT INTO {quote_identifier(table['name'], driver)} ({names}) VALUES ({values});")
    if driver == "sqlite":
        lines.extend(("COMMIT;", "PRAGMA foreign_keys = ON;"))
    else:
        lines.append("SET FOREIGN_KEY_CHECKS = 1;")
    lines.append("")
    return "\n".join(lines)


def write_output(output: Path, rendered: str) -> None:
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


def load_archive(input_path: Path) -> dict[str, Any]:
    if not input_path.is_file():
        raise TransferError(f"database transfer file does not exist: {input_path}")
    parts = []
    for line in input_path.read_text(encoding="utf-8").splitlines():
        if line.startswith(ARCHIVE_PREFIX):
            parts.append(line.removeprefix(ARCHIVE_PREFIX))
    if not parts:
        raise TransferError("database transfer payload is missing")
    try:
        archive = json.loads(base64.b64decode("".join(parts), validate=True).decode("utf-8"))
    except (UnicodeDecodeError, ValueError, json.JSONDecodeError) as error:
        raise TransferError("database transfer payload is invalid") from error
    return validate_archive(archive)


def import_into_sqlite(archive: dict[str, Any], path: Path, replace: bool) -> None:
    connection = connect_sqlite_read_write(path)
    try:
        connection.execute("PRAGMA foreign_keys = OFF")
        if replace:
            for table in reversed(archive["tables"]):
                connection.execute(f"DELETE FROM {quote_identifier(table['name'], 'sqlite')}")
        for table in archive["tables"]:
            columns = table["columns"]
            query = f"INSERT INTO {quote_identifier(table['name'], 'sqlite')} ({', '.join(quote_identifier(column, 'sqlite') for column in columns)}) VALUES ({', '.join('?' for _ in columns)})"
            connection.executemany(query, [tuple(decode_value(value) for value in row) for row in table["rows"]])
        connection.commit()
        connection.execute("PRAGMA foreign_keys = ON")
        ensure_sqlite_integrity(connection)
    except Exception as error:
        connection.rollback()
        connection.execute("PRAGMA foreign_keys = ON")
        if isinstance(error, TransferError):
            raise
        raise TransferError(f"SQLite data import failed: {error}") from error
    finally:
        connection.close()


def import_into_mysql(archive: dict[str, Any], args: argparse.Namespace, replace: bool) -> None:
    connection = mysql_connection(args)
    try:
        with connection.cursor() as cursor:
            cursor.execute("SET FOREIGN_KEY_CHECKS = 0")
            if replace:
                for table in reversed(archive["tables"]):
                    cursor.execute(f"DELETE FROM {quote_identifier(table['name'], 'mysql')}")
            for table in archive["tables"]:
                columns = table["columns"]
                query = f"INSERT INTO {quote_identifier(table['name'], 'mysql')} ({', '.join(quote_identifier(column, 'mysql') for column in columns)}) VALUES ({', '.join('%s' for _ in columns)})"
                cursor.executemany(query, [tuple(decode_value(value) for value in row) for row in table["rows"]])
            cursor.execute("SET FOREIGN_KEY_CHECKS = 1")
        connection.commit()
    except Exception as error:
        connection.rollback()
        raise TransferError(f"MySQL data import failed: {error}") from error
    finally:
        connection.close()


def export_command(args: argparse.Namespace) -> Path:
    if args.source == "sqlite":
        if args.sqlite_path is None:
            raise TransferError("--sqlite-path is required for SQLite operations")
        archive = archive_from_sqlite(args.sqlite_path.resolve())
    else:
        archive = archive_from_mysql(args)
    output = args.output.resolve()
    write_output(output, render_sql_file(archive, args.source))
    return output


def export_service_command(args: argparse.Namespace) -> Path:
    archive = archive_service_from_sqlite(args.sqlite_path.resolve(), args.service_code)
    output = args.output.resolve()
    write_output(output, render_sql_file(archive, "sqlite", include_deletes=False))
    return output


def import_command(args: argparse.Namespace) -> None:
    archive = load_archive(args.input.resolve())
    if archive["driver"] != args.target:
        raise TransferError(f"database transfer file is {archive['driver']}; convert it before importing into {args.target}")
    if args.target == "sqlite":
        if args.sqlite_path is None:
            raise TransferError("--sqlite-path is required for SQLite operations")
        import_into_sqlite(archive, args.sqlite_path.resolve(), args.replace)
    else:
        import_into_mysql(archive, args, args.replace)


def import_service_command(args: argparse.Namespace) -> None:
    archive = validate_service_archive(load_archive(args.input.resolve()))
    import_into_sqlite(archive, args.sqlite_path.resolve(), replace=False)


def convert_command(args: argparse.Namespace) -> Path:
    if args.source_format == args.target_format:
        raise TransferError("source and target formats must differ")
    archive = load_archive(args.input.resolve())
    if archive["driver"] != args.source_format:
        raise TransferError(f"database transfer file is {archive['driver']}, not {args.source_format}")
    output = args.output.resolve()
    write_output(output, render_sql_file(archive, args.target_format))
    return output


def main(argv: Sequence[str]) -> int:
    args = parse_args(argv)
    try:
        if args.command == "export":
            output = export_command(args)
            print(f"database transfer file written: {output}")
        elif args.command == "export-service":
            output = export_service_command(args)
            print(f"service transfer file written: {output}")
        elif args.command == "import":
            import_command(args)
            print("database transfer file imported")
        elif args.command == "import-service":
            import_service_command(args)
            print("service transfer file imported")
        else:
            output = convert_command(args)
            print(f"database transfer file converted: {output}")
    except (TransferError, OSError, sqlite3.Error) as error:
        print(f"database transfer blocked: {error}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
