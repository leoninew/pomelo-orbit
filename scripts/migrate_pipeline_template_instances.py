#!/usr/bin/env python3
"""Cut over the legacy SQLite pipeline model to template-only Pipelines.

The target application code uses Pipeline(kind=template|application).  Legacy
templates can be converted to template Pipelines, but legacy run history has
no reliable Application Pipeline identity and is therefore explicitly removed.
"""

from __future__ import annotations

import argparse
import json
import sqlite3
import sys
from collections import defaultdict
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable


LEGACY_MIGRATION_VERSION = 30

LEGACY_TABLE_COLUMNS: dict[str, tuple[str, ...]] = {
    "pipeline_template": (
        "id",
        "name",
        "description",
        "variable_declarations",
        "version",
        "created_at",
        "updated_at",
        "project_id",
    ),
    "pipeline_stage": (
        "id",
        "name",
        "image",
        "script",
        "artifacts",
        "description",
        "version",
        "created_at",
        "updated_at",
        "project_id",
    ),
    "pipeline_template_stage": (
        "id",
        "template_id",
        "stage_id",
        "stage_name",
        "stage_version",
        "depends_on",
        "sort_order",
    ),
    "pipeline_snapshot": (
        "id",
        "template_id",
        "version",
        "stages_snapshot",
        "variables_snapshot",
        "created_at",
        "project_id",
    ),
    "pipeline_run": (
        "id",
        "repository_id",
        "repository_name",
        "snapshot_id",
        "template_id",
        "template_name",
        "template_version",
        "trigger",
        "trigger_ref",
        "variables_snapshot",
        "status",
        "retry_of",
        "started_at",
        "finished_at",
        "error_message",
        "created_at",
        "project_id",
    ),
    "pipeline_stage_run": (
        "id",
        "pipeline_run_id",
        "stage_id",
        "stage_name",
        "status",
        "started_at",
        "finished_at",
        "exit_code",
        "error_message",
    ),
    "artifact": (
        "id",
        "pipeline_run_id",
        "pipeline_stage_id",
        "stage_name",
        "collector",
        "name",
        "location",
        "value",
        "value_format",
        "image_ref",
        "local_image_sha256",
        "source_artifact_id",
        "created_at",
        "repository_id",
        "repository_name",
        "template_id",
        "template_name",
        "project_id",
    ),
    "pipeline_stage_build_version_binding": (
        "pipeline_stage_id",
        "application_id",
        "application_name",
        "component_name",
        "fork_strategy",
        "fixed_version_id",
    ),
    "pipeline_run_build_version_binding": (
        "pipeline_run_id",
        "pipeline_stage_id",
        "application_id",
        "application_name",
        "component_name",
        "source_version_id",
        "source_version_label",
        "generated_version_id",
        "generated_version_label",
        "artifact_id",
    ),
    "repository_webhook": (
        "id",
        "repository_id",
        "name",
        "template_id",
        "branch_filter",
        "encrypted_secret",
        "enabled",
        "created_at",
        "updated_at",
    ),
    "version_component": (
        "id",
        "version_id",
        "name",
        "image",
        "artifact_id",
        "command_json",
        "pull_policy",
        "restart_policy",
        "created_at",
        "updated_at",
    ),
}

LEGACY_HISTORY_TABLES = (
    "repository_webhook",
    "pipeline_snapshot",
    "pipeline_run",
    "pipeline_stage_run",
    "artifact",
    "pipeline_stage_build_version_binding",
    "pipeline_run_build_version_binding",
)

TARGET_TABLE_COLUMNS: dict[str, tuple[str, ...]] = {
    "pipeline": (
        "id",
        "project_id",
        "kind",
        "source_pipeline_id",
        "source_template_name",
        "source_template_version",
        "application_id",
        "application_name",
        "repository_id",
        "repository_name",
        "version_fork_strategy",
        "fixed_version_id",
        "fixed_version_label",
        "name",
        "description",
        "variable_declarations",
        "version",
        "created_at",
        "updated_at",
    ),
    "pipeline_stage": (
        "id",
        "pipeline_id",
        "name",
        "image",
        "script",
        "artifacts",
        "depends_on",
        "sort_order",
        "description",
        "created_at",
        "updated_at",
    ),
    "pipeline_snapshot": (
        "id",
        "project_id",
        "pipeline_id",
        "pipeline_name",
        "pipeline_version",
        "source_pipeline_id",
        "source_template_name",
        "source_template_version",
        "application_id",
        "application_name",
        "repository_id",
        "repository_name",
        "version_fork_strategy",
        "fixed_version_id",
        "fixed_version_label",
        "stages_snapshot",
        "variables_snapshot",
        "created_at",
    ),
    "pipeline_run": (
        "id",
        "project_id",
        "repository_id",
        "repository_name",
        "snapshot_id",
        "pipeline_id",
        "pipeline_name",
        "pipeline_version",
        "trigger",
        "trigger_ref",
        "variables_snapshot",
        "status",
        "retry_of",
        "started_at",
        "finished_at",
        "error_message",
        "created_at",
    ),
    "pipeline_stage_run": LEGACY_TABLE_COLUMNS["pipeline_stage_run"],
    "pipeline_run_version_binding": (
        "pipeline_run_id",
        "application_id",
        "application_name",
        "source_version_id",
        "source_version_label",
        "generated_version_id",
        "generated_version_label",
    ),
    "artifact": (
        "id",
        "project_id",
        "pipeline_run_id",
        "repository_id",
        "repository_name",
        "pipeline_id",
        "pipeline_name",
        "pipeline_stage_id",
        "stage_name",
        "collector",
        "name",
        "location",
        "value",
        "value_format",
        "image_ref",
        "local_image_sha256",
        "source_artifact_id",
        "created_at",
    ),
    "version_component": (
        "id",
        "version_id",
        "name",
        "image",
        "artifact_id",
        "command_json",
        "pull_policy",
        "restart_policy",
        "created_at",
        "updated_at",
        "artifact_name",
        "artifact_image_ref",
        "artifact_local_image_sha256",
        "artifact_source_commit_sha",
    ),
}


class MigrationError(RuntimeError):
    pass


@dataclass(frozen=True)
class MigrationSummary:
    templates: int
    template_stages: int
    version_components: int
    legacy_history: dict[str, int]

    @property
    def legacy_history_count(self) -> int:
        return sum(self.legacy_history.values())


@dataclass(frozen=True)
class StageCopy:
    id: str
    pipeline_id: str
    legacy_stage_id: str
    name: str
    image: str
    script: str
    artifacts: str | None
    depends_on: tuple[str, ...]
    sort_order: int
    description: str
    created_at: str
    updated_at: str


def parse_args(argv: list[str]) -> argparse.Namespace:
    root = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--database",
        default=str(root / "data" / "db" / "pomelo-orbit.db"),
        help="legacy SQLite database to inspect or cut over",
    )
    backup_group = parser.add_mutually_exclusive_group()
    backup_group.add_argument("--backup", help="new backup path; required with --apply unless --no-backup is explicit")
    backup_group.add_argument(
        "--no-backup",
        action="store_true",
        help="explicitly skip backup creation; only for a disposable or already-backed-up database",
    )
    parser.add_argument("--apply", action="store_true", help="perform the cutover instead of preflight only")
    parser.add_argument(
        "--drop-legacy-history",
        action="store_true",
        help="acknowledge deletion of incompatible legacy webhook, snapshot, run, artifact, and binding records",
    )
    return parser.parse_args(argv)


def connect_read_only(path: Path) -> sqlite3.Connection:
    connection = sqlite3.connect(f"file:{path.as_posix()}?mode=ro", uri=True)
    connection.row_factory = sqlite3.Row
    connection.execute("PRAGMA query_only = ON")
    return connection


def connect_read_write(path: Path) -> sqlite3.Connection:
    connection = sqlite3.connect(path, timeout=1)
    connection.row_factory = sqlite3.Row
    connection.execute("PRAGMA busy_timeout = 1000")
    return connection


def table_exists(connection: sqlite3.Connection, table: str) -> bool:
    return (
        connection.execute(
            "SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?",
            (table,),
        ).fetchone()
        is not None
    )


def table_columns(connection: sqlite3.Connection, table: str) -> tuple[str, ...]:
    return tuple(str(row["name"]) for row in connection.execute(f'PRAGMA table_info("{table}")'))


def count_rows(connection: sqlite3.Connection, table: str) -> int:
    return int(connection.execute(f'SELECT COUNT(*) FROM "{table}"').fetchone()[0])


def ensure_integrity(connection: sqlite3.Connection) -> None:
    integrity = connection.execute("PRAGMA integrity_check").fetchone()[0]
    if integrity != "ok":
        raise MigrationError(f"integrity_check failed: {integrity}")
    foreign_keys = list(connection.execute("PRAGMA foreign_key_check"))
    if foreign_keys:
        raise MigrationError(f"foreign_key_check returned {len(foreign_keys)} row(s)")


def ensure_legacy_migration_version(connection: sqlite3.Connection) -> None:
    if not table_exists(connection, "schema_migrations"):
        raise MigrationError("schema_migrations is missing; refuse an unversioned database")
    columns = set(table_columns(connection, "schema_migrations"))
    if not {"version", "dirty"}.issubset(columns):
        raise MigrationError("schema_migrations does not expose version and dirty columns")
    rows = list(connection.execute("SELECT version, dirty FROM schema_migrations"))
    if len(rows) != 1:
        raise MigrationError("schema_migrations must contain exactly one current version row")
    version, dirty = int(rows[0]["version"]), int(rows[0]["dirty"])
    if version != LEGACY_MIGRATION_VERSION or dirty != 0:
        raise MigrationError(
            f"expected clean legacy migration version {LEGACY_MIGRATION_VERSION}; got version={version} dirty={dirty}"
        )


def ensure_legacy_schema(connection: sqlite3.Connection) -> None:
    if table_exists(connection, "pipeline"):
        raise MigrationError("target pipeline table already exists; this database has already been cut over or diverged")
    for table, expected_columns in LEGACY_TABLE_COLUMNS.items():
        if not table_exists(connection, table):
            raise MigrationError(f"legacy table is missing: {table}")
        actual_columns = table_columns(connection, table)
        if set(actual_columns) != set(expected_columns):
            raise MigrationError(
                f"legacy table columns differ for {table}: expected {expected_columns}, got {actual_columns}"
            )


def ensure_unique_template_names(connection: sqlite3.Connection) -> None:
    duplicate = connection.execute(
        """
        SELECT COALESCE(project_id, '<NULL>') AS project_id, name, COUNT(*) AS count
        FROM pipeline_template
        GROUP BY project_id, name
        HAVING COUNT(*) > 1
        LIMIT 1
        """
    ).fetchone()
    if duplicate:
        raise MigrationError(
            "legacy template names are not unique within a project: "
            f"project={duplicate['project_id']!r} name={duplicate['name']!r} count={duplicate['count']}"
        )


def ensure_template_stage_rows_are_complete(connection: sqlite3.Connection) -> None:
    invalid = connection.execute(
        """
        SELECT pts.id
        FROM pipeline_template_stage AS pts
        LEFT JOIN pipeline_template AS pt ON pt.id = pts.template_id
        LEFT JOIN pipeline_stage AS ps ON ps.id = pts.stage_id
        WHERE pt.id IS NULL OR ps.id IS NULL
        LIMIT 1
        """
    ).fetchone()
    if invalid:
        raise MigrationError(f"pipeline_template_stage {invalid['id']} has a missing template or stage")


def normalize_artifacts(raw_artifacts: str | None, stage_id: str) -> str | None:
    if raw_artifacts is None:
        return None
    try:
        artifacts = json.loads(raw_artifacts)
    except json.JSONDecodeError as error:
        raise MigrationError(f"stage {stage_id} has invalid artifacts JSON: {error.msg}") from error
    if not isinstance(artifacts, list) or not all(isinstance(item, dict) for item in artifacts):
        raise MigrationError(f"stage {stage_id} artifacts must be a JSON array of objects")
    for artifact in artifacts:
        artifact.pop("component_name", None)
    return json.dumps(artifacts, ensure_ascii=False, separators=(",", ":"))


def parse_dependencies(raw_dependencies: str, stage_id: str) -> tuple[str, ...]:
    try:
        dependencies = json.loads(raw_dependencies)
    except json.JSONDecodeError as error:
        raise MigrationError(f"template stage {stage_id} has invalid depends_on JSON: {error.msg}") from error
    if not isinstance(dependencies, list) or not all(isinstance(item, str) for item in dependencies):
        raise MigrationError(f"template stage {stage_id} depends_on must be a JSON array of stage IDs")
    if len(dependencies) != len(set(dependencies)):
        raise MigrationError(f"template stage {stage_id} declares a dependency more than once")
    return tuple(dependencies)


def load_stage_copies(connection: sqlite3.Connection) -> list[StageCopy]:
    rows = connection.execute(
        """
        SELECT
            pts.id AS id,
            pts.template_id AS pipeline_id,
            pts.stage_id AS legacy_stage_id,
            pts.stage_name AS name,
            pts.depends_on AS depends_on,
            pts.sort_order AS sort_order,
            ps.image AS image,
            ps.script AS script,
            ps.artifacts AS artifacts,
            ps.description AS description,
            ps.created_at AS created_at,
            ps.updated_at AS updated_at
        FROM pipeline_template_stage AS pts
        JOIN pipeline_stage AS ps ON ps.id = pts.stage_id
        ORDER BY pts.template_id, pts.sort_order, pts.id
        """
    )
    return [
        StageCopy(
            id=str(row["id"]),
            pipeline_id=str(row["pipeline_id"]),
            legacy_stage_id=str(row["legacy_stage_id"]),
            name=str(row["name"]),
            image=str(row["image"]),
            script=str(row["script"]),
            artifacts=normalize_artifacts(row["artifacts"], str(row["id"])),
            depends_on=parse_dependencies(str(row["depends_on"]), str(row["id"])),
            sort_order=int(row["sort_order"]),
            description=str(row["description"]),
            created_at=str(row["created_at"]),
            updated_at=str(row["updated_at"]),
        )
        for row in rows
    ]


def remap_stage_dependencies(stage_copies: Iterable[StageCopy]) -> list[StageCopy]:
    by_pipeline: dict[str, list[StageCopy]] = defaultdict(list)
    for stage in stage_copies:
        by_pipeline[stage.pipeline_id].append(stage)

    remapped: list[StageCopy] = []
    for pipeline_id, stages in by_pipeline.items():
        legacy_to_new: dict[str, str] = {}
        for stage in stages:
            if stage.legacy_stage_id in legacy_to_new:
                raise MigrationError(
                    f"template {pipeline_id} references legacy stage {stage.legacy_stage_id} more than once; "
                    "dependency remapping is ambiguous"
                )
            legacy_to_new[stage.legacy_stage_id] = stage.id

        graph: dict[str, tuple[str, ...]] = {}
        for stage in stages:
            mapped_dependencies: list[str] = []
            for dependency in stage.depends_on:
                mapped_dependency = legacy_to_new.get(dependency)
                if mapped_dependency is None:
                    raise MigrationError(
                        f"template stage {stage.id} depends on {dependency}, which is not owned by template {pipeline_id}"
                    )
                if mapped_dependency == stage.id:
                    raise MigrationError(f"template stage {stage.id} cannot depend on itself")
                mapped_dependencies.append(mapped_dependency)
            graph[stage.id] = tuple(mapped_dependencies)
            remapped.append(
                StageCopy(
                    id=stage.id,
                    pipeline_id=stage.pipeline_id,
                    legacy_stage_id=stage.legacy_stage_id,
                    name=stage.name,
                    image=stage.image,
                    script=stage.script,
                    artifacts=stage.artifacts,
                    depends_on=tuple(mapped_dependencies),
                    sort_order=stage.sort_order,
                    description=stage.description,
                    created_at=stage.created_at,
                    updated_at=stage.updated_at,
                )
            )
        ensure_acyclic_pipeline_graph(pipeline_id, graph)
    return remapped


def ensure_acyclic_pipeline_graph(pipeline_id: str, graph: dict[str, tuple[str, ...]]) -> None:
    visiting: set[str] = set()
    visited: set[str] = set()

    def visit(stage_id: str) -> None:
        if stage_id in visiting:
            raise MigrationError(f"template {pipeline_id} has a cyclic stage dependency at {stage_id}")
        if stage_id in visited:
            return
        visiting.add(stage_id)
        for dependency in graph[stage_id]:
            visit(dependency)
        visiting.remove(stage_id)
        visited.add(stage_id)

    for stage_id in graph:
        visit(stage_id)


def preflight(connection: sqlite3.Connection) -> tuple[MigrationSummary, list[StageCopy]]:
    ensure_integrity(connection)
    ensure_legacy_migration_version(connection)
    ensure_legacy_schema(connection)
    ensure_unique_template_names(connection)
    ensure_template_stage_rows_are_complete(connection)
    stage_copies = remap_stage_dependencies(load_stage_copies(connection))
    summary = MigrationSummary(
        templates=count_rows(connection, "pipeline_template"),
        template_stages=count_rows(connection, "pipeline_template_stage"),
        version_components=count_rows(connection, "version_component"),
        legacy_history={table: count_rows(connection, table) for table in LEGACY_HISTORY_TABLES},
    )
    return summary, stage_copies


def print_summary(summary: MigrationSummary) -> None:
    print("preflight passed")
    print(f"  templates to convert: {summary.templates}")
    print(f"  template stages to materialize: {summary.template_stages}")
    print(f"  version components to retain: {summary.version_components}")
    print(f"  legacy records to delete: {summary.legacy_history_count}")
    for table, count in summary.legacy_history.items():
        if count:
            print(f"    {table}: {count}")


def validate_backup_path(database: Path, backup: Path | None) -> Path:
    if backup is None:
        raise MigrationError("--backup is required with --apply")
    resolved_backup = backup.resolve()
    if resolved_backup == database:
        raise MigrationError("--backup must not point to the database being migrated")
    if resolved_backup.exists():
        raise MigrationError(f"backup already exists and will not be overwritten: {resolved_backup}")
    if not resolved_backup.parent.is_dir():
        raise MigrationError(f"backup directory does not exist: {resolved_backup.parent}")
    return resolved_backup


def create_backup(connection: sqlite3.Connection, backup: Path) -> None:
    destination: sqlite3.Connection | None = None
    try:
        destination = sqlite3.connect(backup)
        connection.backup(destination)
        integrity = destination.execute("PRAGMA integrity_check").fetchone()[0]
        if integrity != "ok":
            raise MigrationError(f"backup integrity_check failed: {integrity}")
    except (OSError, sqlite3.Error) as error:
        raise MigrationError(f"create SQLite backup: {error}") from error
    finally:
        if destination is not None:
            destination.close()
    print(f"backup created: {backup}")


def create_target_tables(connection: sqlite3.Connection) -> None:
    connection.executescript(
        """
        CREATE TABLE pipeline_new (
            id TEXT PRIMARY KEY,
            project_id TEXT,
            kind TEXT NOT NULL,
            source_pipeline_id TEXT,
            source_template_name TEXT,
            source_template_version INTEGER,
            application_id TEXT,
            application_name TEXT,
            repository_id TEXT,
            repository_name TEXT,
            version_fork_strategy TEXT,
            fixed_version_id TEXT,
            fixed_version_label TEXT,
            name TEXT NOT NULL,
            description TEXT NOT NULL,
            variable_declarations TEXT NOT NULL,
            version INTEGER NOT NULL,
            created_at DATETIME NOT NULL,
            updated_at DATETIME NOT NULL,
            UNIQUE (project_id, name)
        );

        CREATE TABLE pipeline_stage_new (
            id TEXT PRIMARY KEY,
            pipeline_id TEXT NOT NULL,
            name TEXT NOT NULL,
            image TEXT NOT NULL,
            script TEXT NOT NULL,
            artifacts TEXT,
            depends_on TEXT NOT NULL,
            sort_order INTEGER NOT NULL,
            description TEXT NOT NULL,
            created_at DATETIME NOT NULL,
            updated_at DATETIME NOT NULL,
            FOREIGN KEY (pipeline_id) REFERENCES pipeline_new(id) ON DELETE CASCADE,
            UNIQUE (pipeline_id, name)
        );

        CREATE TABLE pipeline_snapshot_new (
            id TEXT PRIMARY KEY,
            project_id TEXT,
            pipeline_id TEXT NOT NULL,
            pipeline_name TEXT NOT NULL,
            pipeline_version INTEGER NOT NULL,
            source_pipeline_id TEXT NOT NULL,
            source_template_name TEXT NOT NULL,
            source_template_version INTEGER NOT NULL,
            application_id TEXT,
            application_name TEXT,
            repository_id TEXT NOT NULL,
            repository_name TEXT NOT NULL,
            version_fork_strategy TEXT,
            fixed_version_id TEXT,
            fixed_version_label TEXT,
            stages_snapshot TEXT NOT NULL,
            variables_snapshot TEXT NOT NULL,
            created_at DATETIME NOT NULL,
            UNIQUE (pipeline_id, pipeline_version)
        );

        CREATE TABLE pipeline_run_new (
            id TEXT PRIMARY KEY,
            project_id TEXT,
            repository_id TEXT NOT NULL,
            repository_name TEXT NOT NULL,
            snapshot_id TEXT NOT NULL,
            pipeline_id TEXT NOT NULL,
            pipeline_name TEXT NOT NULL,
            pipeline_version INTEGER NOT NULL,
            trigger TEXT NOT NULL,
            trigger_ref TEXT NOT NULL,
            variables_snapshot TEXT NOT NULL,
            status TEXT NOT NULL,
            retry_of TEXT,
            started_at DATETIME,
            finished_at DATETIME,
            error_message TEXT,
            created_at DATETIME NOT NULL,
            FOREIGN KEY (snapshot_id) REFERENCES pipeline_snapshot_new(id)
        );

        CREATE TABLE pipeline_stage_run_new (
            id TEXT PRIMARY KEY,
            pipeline_run_id TEXT NOT NULL,
            stage_id TEXT NOT NULL,
            stage_name TEXT NOT NULL,
            status TEXT NOT NULL,
            started_at DATETIME,
            finished_at DATETIME,
            exit_code INTEGER,
            error_message TEXT,
            FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_run_new(id) ON DELETE CASCADE
        );

        CREATE TABLE pipeline_run_version_binding_new (
            pipeline_run_id TEXT PRIMARY KEY,
            application_id TEXT NOT NULL,
            application_name TEXT NOT NULL,
            source_version_id TEXT NOT NULL,
            source_version_label TEXT NOT NULL,
            generated_version_id TEXT,
            generated_version_label TEXT,
            FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_run_new(id) ON DELETE CASCADE
        );

        CREATE TABLE artifact_new (
            id TEXT PRIMARY KEY,
            project_id TEXT,
            pipeline_run_id TEXT NOT NULL,
            repository_id TEXT NOT NULL,
            repository_name TEXT NOT NULL,
            pipeline_id TEXT NOT NULL,
            pipeline_name TEXT NOT NULL,
            pipeline_stage_id TEXT NOT NULL,
            stage_name TEXT NOT NULL,
            collector TEXT NOT NULL,
            name TEXT NOT NULL,
            location TEXT,
            value TEXT,
            value_format TEXT,
            image_ref TEXT,
            local_image_sha256 TEXT,
            source_artifact_id TEXT,
            created_at DATETIME NOT NULL,
            FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_run_new(id) ON DELETE CASCADE
        );

        CREATE TABLE version_component_new (
            id TEXT PRIMARY KEY,
            version_id TEXT NOT NULL,
            name TEXT NOT NULL,
            image TEXT NOT NULL,
            artifact_id TEXT,
            command_json TEXT NOT NULL DEFAULT '[]',
            pull_policy TEXT NOT NULL CHECK (pull_policy IN ('always', 'missing', 'never')),
            restart_policy TEXT,
            created_at DATETIME NOT NULL DEFAULT (datetime('now')),
            updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
            artifact_name TEXT,
            artifact_image_ref TEXT,
            artifact_local_image_sha256 TEXT,
            artifact_source_commit_sha TEXT,
            FOREIGN KEY (version_id) REFERENCES version(id) ON DELETE CASCADE,
            UNIQUE(version_id, name)
        );
        """
    )


def copy_templates(connection: sqlite3.Connection) -> None:
    connection.execute(
        """
        INSERT INTO pipeline_new (
            id, project_id, kind, source_pipeline_id, source_template_name, source_template_version,
            application_id, application_name, repository_id, repository_name, version_fork_strategy,
            fixed_version_id, fixed_version_label, name, description, variable_declarations, version,
            created_at, updated_at
        )
        SELECT
            id, project_id, 'template', NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL,
            name, description, variable_declarations, version, created_at, updated_at
        FROM pipeline_template
        """
    )


def copy_template_stages(connection: sqlite3.Connection, stages: Iterable[StageCopy]) -> None:
    connection.executemany(
        """
        INSERT INTO pipeline_stage_new (
            id, pipeline_id, name, image, script, artifacts, depends_on, sort_order, description, created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """,
        (
            (
                stage.id,
                stage.pipeline_id,
                stage.name,
                stage.image,
                stage.script,
                stage.artifacts,
                json.dumps(stage.depends_on, separators=(",", ":")),
                stage.sort_order,
                stage.description,
                stage.created_at,
                stage.updated_at,
            )
            for stage in stages
        ),
    )


def copy_version_components(connection: sqlite3.Connection) -> None:
    connection.execute(
        """
        INSERT INTO version_component_new (
            id, version_id, name, image, artifact_id, command_json, pull_policy, restart_policy,
            created_at, updated_at, artifact_name, artifact_image_ref, artifact_local_image_sha256,
            artifact_source_commit_sha
        )
        SELECT
            vc.id, vc.version_id, vc.name, vc.image, vc.artifact_id, vc.command_json, vc.pull_policy,
            vc.restart_policy, vc.created_at, vc.updated_at,
            artifact.name, artifact.image_ref, artifact.local_image_sha256,
            CASE
                WHEN source.collector = 'command' AND source.value_format = 'git_object_id' THEN source.value
                ELSE NULL
            END
        FROM version_component AS vc
        LEFT JOIN artifact ON artifact.id = vc.artifact_id
        LEFT JOIN artifact AS source ON source.id = artifact.source_artifact_id
        """
    )


def drop_legacy_tables(connection: sqlite3.Connection) -> None:
    for table in (
        "repository_webhook",
        "pipeline_run_build_version_binding",
        "pipeline_stage_build_version_binding",
        "artifact",
        "pipeline_stage_run",
        "pipeline_run",
        "pipeline_snapshot",
        "pipeline_template_stage",
        "pipeline_stage",
        "pipeline_template",
        "version_component",
    ):
        connection.execute(f'DROP TABLE "{table}"')


def rename_target_tables(connection: sqlite3.Connection) -> None:
    for table in (
        "pipeline",
        "pipeline_stage",
        "pipeline_snapshot",
        "pipeline_run",
        "pipeline_stage_run",
        "pipeline_run_version_binding",
        "artifact",
        "version_component",
    ):
        connection.execute(f'ALTER TABLE "{table}_new" RENAME TO "{table}"')


def create_target_indexes(connection: sqlite3.Connection) -> None:
    connection.executescript(
        """
        CREATE INDEX idx_pipeline_project ON pipeline(project_id);
        CREATE INDEX idx_pipeline_kind ON pipeline(kind);
        CREATE INDEX idx_pipeline_source_pipeline ON pipeline(source_pipeline_id);
        CREATE INDEX idx_pipeline_application ON pipeline(application_id);
        CREATE INDEX idx_pipeline_repository ON pipeline(repository_id);

        CREATE INDEX idx_pipeline_stage_pipeline ON pipeline_stage(pipeline_id);
        CREATE INDEX idx_pipeline_stage_pipeline_sort ON pipeline_stage(pipeline_id, sort_order);

        CREATE INDEX idx_pipeline_snapshot_pipeline ON pipeline_snapshot(pipeline_id);
        CREATE INDEX idx_pipeline_snapshot_project ON pipeline_snapshot(project_id);

        CREATE INDEX idx_pipeline_run_pipeline_created ON pipeline_run(pipeline_id, created_at DESC);
        CREATE INDEX idx_pipeline_run_repository_status ON pipeline_run(repository_id, status);
        CREATE INDEX idx_pipeline_run_status ON pipeline_run(status);
        CREATE INDEX idx_pipeline_run_snapshot ON pipeline_run(snapshot_id);
        CREATE INDEX idx_pipeline_run_retry_of ON pipeline_run(retry_of);
        CREATE INDEX idx_pipeline_run_project ON pipeline_run(project_id);

        CREATE INDEX idx_pipeline_stage_run_run ON pipeline_stage_run(pipeline_run_id);
        CREATE INDEX idx_pipeline_stage_run_status ON pipeline_stage_run(status);

        CREATE INDEX idx_pipeline_run_version_binding_source_version
            ON pipeline_run_version_binding(source_version_id);
        CREATE INDEX idx_pipeline_run_version_binding_generated_version
            ON pipeline_run_version_binding(generated_version_id);

        CREATE INDEX idx_artifact_run ON artifact(pipeline_run_id);
        CREATE INDEX idx_artifact_run_stage ON artifact(pipeline_run_id, pipeline_stage_id);
        CREATE INDEX idx_artifact_repository ON artifact(repository_id);
        CREATE INDEX idx_artifact_project ON artifact(project_id);
        CREATE INDEX idx_artifact_pipeline ON artifact(pipeline_id);

        CREATE INDEX idx_version_component_version ON version_component(version_id);
        CREATE INDEX idx_version_component_artifact_id ON version_component(artifact_id);
        """
    )


def ensure_target_schema(connection: sqlite3.Connection, summary: MigrationSummary) -> None:
    for table, expected_columns in TARGET_TABLE_COLUMNS.items():
        if not table_exists(connection, table):
            raise MigrationError(f"target table is missing: {table}")
        actual_columns = table_columns(connection, table)
        if set(actual_columns) != set(expected_columns):
            raise MigrationError(
                f"target table columns differ for {table}: expected {expected_columns}, got {actual_columns}"
            )
    for legacy_table in LEGACY_TABLE_COLUMNS:
        if legacy_table not in TARGET_TABLE_COLUMNS and table_exists(connection, legacy_table):
            raise MigrationError(f"legacy table still exists: {legacy_table}")
    if count_rows(connection, "pipeline") != summary.templates:
        raise MigrationError("pipeline count does not match converted legacy template count")
    if count_rows(connection, "pipeline_stage") != summary.template_stages:
        raise MigrationError("pipeline_stage count does not match converted legacy template stage count")
    if count_rows(connection, "version_component") != summary.version_components:
        raise MigrationError("version_component count changed during rebuild")
    non_templates = count_rows(connection, "pipeline") - int(
        connection.execute("SELECT COUNT(*) FROM pipeline WHERE kind = 'template'").fetchone()[0]
    )
    if non_templates:
        raise MigrationError("legacy cutover must create template Pipelines only")
    artifact_foreign_keys = list(connection.execute("PRAGMA foreign_key_list(version_component)"))
    if any(str(row["from"]) == "artifact_id" for row in artifact_foreign_keys):
        raise MigrationError("version_component.artifact_id remains a physical foreign key")
    for row in connection.execute("SELECT id, artifacts FROM pipeline_stage WHERE artifacts IS NOT NULL"):
        artifacts = json.loads(str(row["artifacts"]))
        if any("component_name" in artifact for artifact in artifacts):
            raise MigrationError(f"template pipeline stage {row['id']} still contains component_name")
    ensure_integrity(connection)


def apply_cutover(database: Path, backup: Path | None, allow_history_drop: bool) -> MigrationSummary:
    connection = connect_read_write(database)
    began_transaction = False
    try:
        connection.execute("PRAGMA foreign_keys = OFF")
        connection.execute("BEGIN EXCLUSIVE")
        began_transaction = True
        summary, stages = preflight(connection)
        if summary.legacy_history_count and not allow_history_drop:
            raise MigrationError(
                "legacy history exists; rerun with --drop-legacy-history to acknowledge its deletion"
            )
        # SQLite's backup API cannot copy from this connection while it owns an
        # exclusive transaction. Prove exclusive access first, then release the
        # read-only transaction for the consistent SQLite backup and lock again.
        connection.rollback()
        began_transaction = False
        if backup is None:
            print("backup skipped by explicit --no-backup")
        else:
            create_backup(connection, backup)
        connection.execute("BEGIN EXCLUSIVE")
        began_transaction = True
        summary, stages = preflight(connection)
        if summary.legacy_history_count and not allow_history_drop:
            raise MigrationError(
                "legacy history exists; rerun with --drop-legacy-history to acknowledge its deletion"
            )
        create_target_tables(connection)
        copy_templates(connection)
        copy_template_stages(connection, stages)
        copy_version_components(connection)
        drop_legacy_tables(connection)
        rename_target_tables(connection)
        create_target_indexes(connection)
        ensure_target_schema(connection, summary)
        connection.commit()
        began_transaction = False
        connection.execute("PRAGMA foreign_keys = ON")
        ensure_integrity(connection)
        return summary
    except (MigrationError, OSError, sqlite3.Error):
        if began_transaction:
            connection.rollback()
        raise
    finally:
        connection.close()


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    try:
        database = Path(args.database).resolve()
        if not database.is_file():
            raise MigrationError(f"SQLite database does not exist: {database}")
        if not args.apply:
            if args.no_backup:
                raise MigrationError("--no-backup requires --apply")
            connection = connect_read_only(database)
            try:
                summary, _ = preflight(connection)
            finally:
                connection.close()
            print_summary(summary)
            if summary.legacy_history_count and not args.drop_legacy_history:
                raise MigrationError(
                    "legacy history would be deleted; inspect the counts above, then use --apply "
                    "--backup <new-path> --drop-legacy-history"
                )
            print("dry run only; no database changes were made")
            return 0

        if args.no_backup:
            backup = None
        else:
            backup = validate_backup_path(database, Path(args.backup) if args.backup else None)
        summary = apply_cutover(database, backup, args.drop_legacy_history)
        print_summary(summary)
        print(f"pipeline template-instance cutover completed: {database}")
        return 0
    except (MigrationError, OSError, sqlite3.Error) as error:
        print(f"pipeline template-instance cutover blocked: {error}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
