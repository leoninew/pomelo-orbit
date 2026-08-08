"""Reconcile legacy service.status='deploying' records from an approved mapping.

The application does not invoke this script. Review the JSON mapping, run the
default dry-run, then use --apply only against an explicitly named SQLite file.
"""

from __future__ import annotations

import argparse
import json
import sqlite3
import sys
from pathlib import Path
from typing import Sequence


ALLOWED_TARGET_STATUSES = frozenset({"running", "stopped", "faulted"})


class ReconcileError(Exception):
    """Raised when the reviewed mapping cannot be safely applied."""


def parse_args(argv: Sequence[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--database", required=True, type=Path, help="SQLite database to inspect")
    parser.add_argument("--mapping", required=True, type=Path, help="Reviewed JSON object: service_id -> target status")
    parser.add_argument("--apply", action="store_true", help="Create the backup and apply the reviewed mapping")
    parser.add_argument("--backup", type=Path, help="SQLite backup destination; required with --apply")
    return parser.parse_args(argv)


def load_mapping(path: Path) -> dict[str, str]:
    try:
        document = json.loads(path.read_text(encoding="utf-8"))
    except OSError as error:
        raise ReconcileError(f"read mapping {path}: {error}") from error
    except json.JSONDecodeError as error:
        raise ReconcileError(f"parse mapping {path}: {error}") from error
    if not isinstance(document, dict):
        raise ReconcileError("mapping must be a JSON object of service_id to target status")

    mapping: dict[str, str] = {}
    for raw_id, raw_status in document.items():
        service_id = raw_id.strip() if isinstance(raw_id, str) else ""
        target_status = raw_status.strip() if isinstance(raw_status, str) else ""
        if not service_id:
            raise ReconcileError("mapping contains an empty service_id")
        if target_status not in ALLOWED_TARGET_STATUSES:
            allowed = ", ".join(sorted(ALLOWED_TARGET_STATUSES))
            raise ReconcileError(f"mapping service {service_id} has invalid target status {raw_status!r}; expected one of {allowed}")
        mapping[service_id] = target_status
    return mapping


def connect(path: Path) -> sqlite3.Connection:
    if not path.is_file():
        raise ReconcileError(f"database does not exist: {path}")
    connection = sqlite3.connect(path)
    connection.row_factory = sqlite3.Row
    return connection


def planned_changes(connection: sqlite3.Connection, mapping: dict[str, str]) -> list[tuple[str, str, str]]:
    try:
        rows = connection.execute("SELECT id, status FROM service WHERE status = 'deploying' ORDER BY id").fetchall()
    except sqlite3.Error as error:
        raise ReconcileError(f"query legacy services: {error}") from error

    legacy_ids = {str(row["id"]) for row in rows}
    missing = sorted(legacy_ids - mapping.keys())
    if missing:
        raise ReconcileError("mapping is missing legacy deploying service(s): " + ", ".join(missing))

    for service_id in sorted(mapping):
        row = connection.execute("SELECT status FROM service WHERE id = ?", (service_id,)).fetchone()
        if row is None:
            raise ReconcileError(f"mapping references unknown service: {service_id}")
        if row["status"] != "deploying":
            raise ReconcileError(f"mapping service {service_id} is {row['status']!r}, not 'deploying'")

    return [(str(row["id"]), "deploying", mapping[str(row["id"])]) for row in rows]


def backup_database(source: sqlite3.Connection, backup_path: Path, database_path: Path) -> None:
    if backup_path.resolve() == database_path.resolve():
        raise ReconcileError("backup path must differ from database path")
    if backup_path.exists():
        raise ReconcileError(f"backup path already exists: {backup_path}")
    try:
        backup_path.parent.mkdir(parents=True, exist_ok=True)
        destination = sqlite3.connect(backup_path)
        try:
            source.backup(destination)
        finally:
            destination.close()
    except (OSError, sqlite3.Error) as error:
        raise ReconcileError(f"create backup {backup_path}: {error}") from error


def apply_changes(connection: sqlite3.Connection, changes: list[tuple[str, str, str]]) -> None:
    try:
        with connection:
            for service_id, _, target_status in changes:
                result = connection.execute(
                    "UPDATE service SET status = ?, updated_at = datetime('now') WHERE id = ? AND status = 'deploying'",
                    (target_status, service_id),
                )
                if result.rowcount != 1:
                    raise ReconcileError(f"service {service_id} was no longer 'deploying'; no update was applied")
    except sqlite3.Error as error:
        raise ReconcileError(f"apply mapping: {error}") from error


def report(changes: list[tuple[str, str, str]], applied: bool) -> None:
    for service_id, old_status, target_status in changes:
        print(json.dumps({"service_id": service_id, "from": old_status, "to": target_status, "applied": applied}, sort_keys=True))
    print(json.dumps({"count": len(changes), "applied": applied}, sort_keys=True))


def run(args: argparse.Namespace) -> int:
    if args.apply and args.backup is None:
        raise ReconcileError("--backup is required with --apply")
    if not args.apply and args.backup is not None:
        raise ReconcileError("--backup is only valid with --apply")

    mapping = load_mapping(args.mapping)
    connection = connect(args.database)
    try:
        changes = planned_changes(connection, mapping)
        if args.apply:
            backup_database(connection, args.backup, args.database)
            apply_changes(connection, changes)
        report(changes, args.apply)
    finally:
        connection.close()
    return 0


def main(argv: Sequence[str] | None = None) -> int:
    try:
        return run(parse_args(argv))
    except ReconcileError as error:
        print(f"error: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
