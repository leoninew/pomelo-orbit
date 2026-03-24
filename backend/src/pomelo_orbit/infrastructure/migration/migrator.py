"""Database migration manager."""

import hashlib
import json
import logging
import re
import time
from collections.abc import Generator
from contextlib import contextmanager
from datetime import datetime
from pathlib import Path
from uuid import uuid4

import sqlparse
from sqlalchemy import Engine, Integer, String, Text, func, text
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column, sessionmaker
from sqlalchemy.sql import compiler

from pomelo_orbit.infrastructure.config import get_project_root

logger = logging.getLogger(__name__)


class MigrationError(Exception):
    """Migration execution error."""


class MigrationChecksumError(MigrationError):
    """Migration checksum mismatch error."""


class _Base(DeclarativeBase):
    """Base class for migration models."""


class MigrationHistory(_Base):
    """Migration history record."""

    __tablename__ = "__migration_history"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    filename: Mapped[str] = mapped_column(Text, nullable=False, unique=True)
    checksum: Mapped[str] = mapped_column(String(32), nullable=False)
    executed_at: Mapped[datetime] = mapped_column(default=func.now(), nullable=False)
    execution_time_ms: Mapped[int] = mapped_column(Integer, nullable=False)


@contextmanager
def disable_param_handling_if_needed(params: dict[str, object] | None) -> Generator[None, None, None]:
    """Temporarily disable SQLAlchemy parameter binding detection when executing SQL without parameters.

    This context manager prevents SQLAlchemy from trying to interpret :param or %(param)s
    patterns in SQL strings as parameter placeholders when no parameters are provided.

    Args:
        params: Dictionary of parameters for the SQL statement. If empty or None,
               parameter binding detection will be disabled.

    Yields:
        None
    """
    original_bind_params = compiler.BIND_PARAMS
    original_bind_params_esc = compiler.BIND_PARAMS_ESC
    try:
        if not params or len(params) == 0:
            compiler.BIND_PARAMS = re.compile(uuid4().__str__())
            compiler.BIND_PARAMS_ESC = re.compile(uuid4().__str__())
        yield
    finally:
        if not params or len(params) == 0:
            compiler.BIND_PARAMS = original_bind_params
            compiler.BIND_PARAMS_ESC = original_bind_params_esc


def _calculate_md5(file_path: Path) -> str:
    """Calculate MD5 checksum of a file.

    Args:
        file_path: Path to the file

    Returns:
        MD5 checksum as hex string
    """
    md5_hash = hashlib.md5()
    with file_path.open("rb") as f:
        for chunk in iter(lambda: f.read(4096), b""):
            md5_hash.update(chunk)
    return md5_hash.hexdigest()


def _is_empty_sql_file(file_path: Path) -> bool:
    """Check if SQL file is empty or contains only whitespace/comments.

    Args:
        file_path: Path to SQL file

    Returns:
        True if file is empty or contains only whitespace/comments
    """
    content = file_path.read_text(encoding="utf-8").strip()
    if not content:
        return True
    # Check if all lines are comments or whitespace
    lines = [line.strip() for line in content.split("\n")]
    return all(not line or line.startswith("--") for line in lines)


def _parse_sql_file(file_path: Path) -> list[str]:
    """Parse SQL file and return list of statements."""
    content = file_path.read_text(encoding="utf-8")
    parsed = sqlparse.parse(content)
    statements = []
    for stmt in parsed:
        cleaned_stmt = sqlparse.format(
            stmt.value,
            strip_comments=True,
            strip_whitespace=True,
        ).strip()
        if cleaned_stmt:
            statements.append(cleaned_stmt)
    return statements


def _qi(name: str) -> str:
    """Quote an identifier (table or column name) with double quotes.

    TODO: Currently only safe for SQLite and PostgreSQL (both use double-quote quoting).
          MySQL uses backtick quoting — extend this if MySQL support is needed.
    """
    return '"' + name.replace('"', '""') + '"'


def _build_where(where: dict) -> tuple[str, dict]:
    """Build WHERE clause from a flat equality dict. Returns (clause, params)."""
    params = {f"where_{col}": val for col, val in where.items()}
    clause = " AND ".join(f"{_qi(col)} = :where_{col}" for col in where)
    return clause, params


def _parse_single_op(op_data: dict, filename: str, op_index: int = 0) -> tuple[str, dict, str | None]:
    """Parse a single operation. Returns (sql, params, comment)."""
    op = op_data.get("type")
    table = op_data.get("table")
    comment = op_data.get("comment")

    if op not in ("insert", "update", "delete"):
        raise MigrationError(f"{filename}: 'type' must be insert | update | delete, got {op!r}")
    if not table or not isinstance(table, str):
        raise MigrationError(f"{filename}: 'table' must be a non-empty string")

    prefix = f"op{op_index}_" if op_index > 0 else ""

    if op == "insert":
        rows = op_data.get("data")
        if not isinstance(rows, list) or not rows:
            raise MigrationError(f"{filename}: insert requires 'data' as a non-empty list")
        if not all(isinstance(r, dict) for r in rows):
            raise MigrationError(f"{filename}: insert 'data' items must be objects")

        columns = list(rows[0].keys())
        col_list = ", ".join(_qi(col) for col in columns)
        value_rows = []
        params: dict = {}
        for i, row in enumerate(rows):
            placeholders = ", ".join(f":{prefix}r{i}_{col}" for col in columns)
            value_rows.append(f"({placeholders})")
            for col in columns:
                params[f"{prefix}r{i}_{col}"] = row[col]
        sql = f"INSERT INTO {_qi(table)} ({col_list}) VALUES {', '.join(value_rows)}"
        return sql, params, comment

    if op == "update":
        data = op_data.get("data")
        where = op_data.get("where")
        if not isinstance(data, dict) or not data:
            raise MigrationError(f"{filename}: update requires 'data' as a non-empty object")
        if not isinstance(where, dict) or not where:
            raise MigrationError(f"{filename}: update requires 'where' as a non-empty object")

        params = {f"{prefix}set_{col}": val for col, val in data.items()}
        set_clause = ", ".join(f"{_qi(col)} = :{prefix}set_{col}" for col in data)
        where_parts = []
        for col, val in where.items():
            params[f"{prefix}where_{col}"] = val
            where_parts.append(f"{_qi(col)} = :{prefix}where_{col}")
        sql = f"UPDATE {_qi(table)} SET {set_clause} WHERE {' AND '.join(where_parts)}"
        return sql, params, comment

    if op == "delete":
        where = op_data.get("where")
        if not isinstance(where, dict) or not where:
            raise MigrationError(f"{filename}: delete requires 'where' as a non-empty object")

        params = {}
        where_parts = []
        for col, val in where.items():
            params[f"{prefix}where_{col}"] = val
            where_parts.append(f"{_qi(col)} = :{prefix}where_{col}")
        sql = f"DELETE FROM {_qi(table)} WHERE {' AND '.join(where_parts)}"
        return sql, params, comment

    raise MigrationError(f"{filename}: unhandled type {op!r}")


def _parse_data_json_file(file_path: Path) -> list[tuple[str, dict, str | None]]:
    """Parse a .data.json migration file and return list of (sql, params, comment).

    File must contain an array of operations:
      [{"type": "insert", "table": "t", "data": [...], "comment": "..."}, ...]
    """
    raw = json.loads(file_path.read_text(encoding="utf-8"))
    filename = file_path.name

    if not isinstance(raw, list):
        raise MigrationError(f"{filename}: content must be an array of operations")

    return [_parse_single_op(op, filename, i) for i, op in enumerate(raw)]


def run_migrations(
    engine: Engine,
    migrations_dir: Path | str | None = None,
) -> None:
    """Run all pending database migrations in a single transaction.

    Args:
        engine: SQLAlchemy engine
        migrations_dir: Path to directory containing migration SQL files. If None, uses project root.

    Raises:
        MigrationError: If migration execution fails
        MigrationChecksumError: If a migration file's checksum doesn't match the recorded one

    Note:
        Migration history is always rolled back on failure.
        DDL changes rollback depends on database:
        - PostgreSQL: Full DDL rollback support
        - SQLite: DDL rollback supported but may have implicit commits
        - MySQL: DDL causes implicit commit, cannot rollback
    """

    migrations_dir = get_project_root() / "migrations" if migrations_dir is None else Path(migrations_dir)
    if not migrations_dir.exists():
        raise MigrationError(f"Migrations directory not found: {migrations_dir}")

    logger.info(f"Running migrations from: {migrations_dir}")

    # Step 1: Ensure migration history table exists (separate transaction)
    MigrationHistory.metadata.create_all(engine)

    # Step 2: Get executed migrations and validate checksums
    session = sessionmaker(bind=engine)()
    try:
        executed = session.execute(text("SELECT filename, checksum FROM __migration_history ORDER BY executed_at"))
        executed_migrations = {row[0]: row[1] for row in executed.fetchall()}
    finally:
        session.close()

    # Step 3: Collect pending migrations with validation
    all_files = sorted(list(migrations_dir.glob("*.sql")) + list(migrations_dir.glob("*.json")))

    pending_migrations = []
    for migration_file in all_files:
        filename = migration_file.name
        is_json = filename.endswith(".json")

        if not is_json and _is_empty_sql_file(migration_file):
            logger.info(f"Migration skipped: filename={filename}, reason=empty")
            continue

        checksum = _calculate_md5(migration_file)

        if filename in executed_migrations:
            if checksum != executed_migrations[filename]:
                error_msg = (
                    f"Checksum mismatch for migration '{filename}'.\n"
                    f"Stored: {executed_migrations[filename]}\n"
                    f"Current: {checksum}\n"
                    f"Migration files that have been executed should not be modified."
                )
                raise MigrationChecksumError(error_msg)
            logger.info(f"Migration skipped: filename={filename}, reason=already_executed")
            continue

        pending_migrations.append((migration_file, filename, checksum, is_json))

    if not pending_migrations:
        logger.info("No pending migrations")
        return

    # Step 4: Execute all pending migrations in a single transaction
    logger.info(f"Executing {len(pending_migrations)} pending migrations in single transaction")

    session = sessionmaker(bind=engine)()
    try:
        for migration_file, filename, checksum, is_json in pending_migrations:
            start_time = time.time()

            if is_json:
                operations = _parse_data_json_file(migration_file)
                for sql, params, _ in operations:
                    logger.debug(f"SQL: {sql}")
                    logger.debug(f"params: {params}")
                    session.execute(text(sql), params)
            else:
                with disable_param_handling_if_needed({}):
                    for statement in _parse_sql_file(migration_file):
                        logger.debug(f"SQL: {statement}")
                        session.execute(text(statement))

            execution_time_ms = int((time.time() - start_time) * 1000)
            session.add(MigrationHistory(filename=filename, checksum=checksum, execution_time_ms=execution_time_ms))
            logger.info(f"Migration executed: filename={filename}, duration_ms={execution_time_ms}")

        session.commit()
        logger.info("All migrations completed successfully")
    except Exception as e:
        session.rollback()
        logger.error(f"Migration failed, rolled back: error={e}", exc_info=True)
        raise MigrationError(f"Migration failed, rolled back: {e}") from e
    finally:
        session.close()
