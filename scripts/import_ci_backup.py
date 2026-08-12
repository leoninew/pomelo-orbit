#!/usr/bin/env python3
"""Import legacy CI data into the current SQLite Pipeline model.

The source database uses the pre-v30 pipeline_template schema.  The target
database must already have been migrated and seeded through v30.  Imports are
transactional, preserve CI history, and re-encrypt repository credentials for
the target JWT key.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import sqlite3
import sys
from collections import defaultdict
from pathlib import Path
from typing import Any, Iterable, Sequence

from cryptography.fernet import Fernet, InvalidToken


SOURCE_TABLES = (
    "project",
    "credential",
    "repository",
    "pipeline_template",
    "pipeline_template_stage",
    "pipeline_stage",
    "pipeline_snapshot",
    "pipeline_run",
    "pipeline_stage_run",
    "pipeline_stage_build_version_binding",
    "pipeline_run_build_version_binding",
    "artifact",
    "repository_webhook",
)
TARGET_TABLES = (
    "project",
    "credential",
    "repository",
    "pipeline",
    "pipeline_stage",
    "pipeline_stage_reference",
    "pipeline_snapshot",
    "pipeline_run",
    "pipeline_stage_run",
    "pipeline_run_version_binding",
    "artifact",
)
JWT_SECRET_KEY = "POMELO_ORBIT_JWT__SECRET_KEY"
CROCKFORD_BASE32 = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"


class ImportError(RuntimeError):
    pass


def parse_args(argv: Sequence[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)

    clone = commands.add_parser("clone", help="create a consistent SQLite copy using the backup API")
    clone.add_argument("--database", type=Path, required=True, help="database to copy")
    clone.add_argument("--output", type=Path, required=True, help="new database path")

    for name, help_text in (
        ("import", "transform and import the legacy CI records"),
        ("verify", "verify a completed legacy CI import"),
    ):
        command = commands.add_parser(name, help=help_text)
        command.add_argument("--source", type=Path, required=True, help="legacy backup SQLite database")
        command.add_argument("--database", type=Path, required=True, help="current-schema target SQLite database")
        command.add_argument("--source-env", type=Path, required=True, help="backup .env containing the source JWT key")
        command.add_argument("--target-env", type=Path, required=True, help="development .env containing the target JWT key")

    return parser.parse_args(argv)


def connect_read_only(path: Path) -> sqlite3.Connection:
    if not path.is_file():
        raise ImportError(f"SQLite database does not exist: {path}")
    connection = sqlite3.connect(f"file:{path.resolve().as_posix()}?mode=ro", uri=True)
    connection.row_factory = sqlite3.Row
    return connection


def connect_read_write(path: Path) -> sqlite3.Connection:
    if not path.is_file():
        raise ImportError(f"SQLite database does not exist: {path}")
    connection = sqlite3.connect(path)
    connection.row_factory = sqlite3.Row
    connection.execute("PRAGMA foreign_keys = ON")
    connection.execute("PRAGMA busy_timeout = 10000")
    return connection


def ensure_integrity(connection: sqlite3.Connection, label: str) -> None:
    integrity = connection.execute("PRAGMA integrity_check").fetchone()[0]
    if integrity != "ok":
        raise ImportError(f"{label} integrity_check failed: {integrity}")
    foreign_keys = list(connection.execute("PRAGMA foreign_key_check"))
    if foreign_keys:
        raise ImportError(f"{label} foreign_key_check returned {len(foreign_keys)} row(s)")


def ensure_tables(connection: sqlite3.Connection, tables: Iterable[str], label: str) -> None:
    available = {row[0] for row in connection.execute("SELECT name FROM sqlite_master WHERE type = 'table'")}
    missing = sorted(set(tables) - available)
    if missing:
        raise ImportError(f"{label} is missing required table(s): {', '.join(missing)}")


def read_rows(connection: sqlite3.Connection, table: str) -> list[dict[str, Any]]:
    return [dict(row) for row in connection.execute(f'SELECT * FROM "{table}" ORDER BY 1')]


def read_env_value(path: Path, key: str) -> str:
    if not path.is_file():
        raise ImportError(f"environment file does not exist: {path}")
    for raw_line in path.read_text(encoding="utf-8").splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        name, value = line.split("=", 1)
        if name.strip() == key and value.strip():
            return value.strip()
    raise ImportError(f"environment file does not define {key}: {path}")


def fernet_from_env(path: Path) -> Fernet:
    try:
        return Fernet(read_env_value(path, JWT_SECRET_KEY).encode("ascii"))
    except (ValueError, UnicodeEncodeError) as error:
        raise ImportError(f"invalid {JWT_SECRET_KEY} in {path}") from error


def stable_id(*parts: str) -> str:
    digest = hashlib.sha256("\x00".join(parts).encode("utf-8")).digest()[:16]
    value = int.from_bytes(digest, "big")
    result = ["0"] * 26
    for index in range(25, -1, -1):
        result[index] = CROCKFORD_BASE32[value & 31]
        value >>= 5
    return "".join(result)


def json_array(value: Any, field: str) -> list[Any]:
    if value is None or value == "":
        return []
    if isinstance(value, list):
        return value
    try:
        decoded = json.loads(value)
    except (TypeError, json.JSONDecodeError) as error:
        raise ImportError(f"invalid JSON array in {field}") from error
    if not isinstance(decoded, list):
        raise ImportError(f"{field} must be a JSON array")
    return decoded


def encode_json(value: Any) -> str:
    return json.dumps(value, ensure_ascii=False, separators=(",", ":"))


def convert_artifacts(value: Any, field: str) -> str:
    converted: list[dict[str, Any]] = []
    for artifact in json_array(value, field):
        if not isinstance(artifact, dict):
            raise ImportError(f"{field} contains a non-object artifact")
        if "component_name" in artifact:
            raise ImportError(f"{field} contains unsupported legacy component mapping")
        collector = artifact.get("collector")
        if collector is None:
            collector = artifact.get("type")
        name = artifact.get("name")
        if not isinstance(name, str) or not name.strip() or not isinstance(collector, str):
            raise ImportError(f"{field} contains an invalid artifact")
        if collector == "docker_image":
            reference = artifact.get("reference", artifact.get("path"))
            if not isinstance(reference, str) or not reference.strip():
                raise ImportError(f"{field} docker_image artifact has no reference")
            converted.append({"name": name, "collector": collector, "reference": reference})
        elif collector == "command":
            command, value_format = artifact.get("command"), artifact.get("format")
            if not isinstance(command, str) or not command.strip() or value_format not in {"text", "git_object_id"}:
                raise ImportError(f"{field} command artifact is invalid")
            converted.append({"name": name, "collector": collector, "command": command, "format": value_format})
        else:
            raise ImportError(f"{field} has unsupported artifact collector {collector!r}")
    return encode_json(converted)


def convert_variables(value: Any, field: str) -> str:
    converted: list[dict[str, Any]] = []
    for variable in json_array(value, field):
        if not isinstance(variable, dict) or not isinstance(variable.get("name"), str) or not variable["name"].strip():
            raise ImportError(f"{field} contains an invalid variable declaration")
        item = dict(variable)
        if item.get("source") == "template_custom":
            item["source"] = "pipeline_custom"
        elif item.get("source") == "template":
            item["source"] = "default"
        item.setdefault("description", "")
        item.setdefault("default", None)
        item.setdefault("value", None)
        item.setdefault("secret", False)
        item.setdefault("editable", True)
        converted.append(item)
    return encode_json(converted)


def unique_name(existing: set[str], name: str, identifier: str) -> str:
    if name not in existing:
        existing.add(name)
        return name
    candidate = f"{name} (backup {identifier[-6:]})"
    suffix = 2
    while candidate in existing:
        candidate = f"{name} (backup {identifier[-6:]} {suffix})"
        suffix += 1
    existing.add(candidate)
    return candidate


def index_rows(rows: Iterable[dict[str, Any]], table: str) -> dict[str, dict[str, Any]]:
    indexed: dict[str, dict[str, Any]] = {}
    for row in rows:
        identifier = row.get("id")
        if not isinstance(identifier, str) or not identifier:
            raise ImportError(f"{table} contains a missing id")
        if identifier in indexed:
            raise ImportError(f"{table} contains duplicate id {identifier}")
        indexed[identifier] = row
    return indexed


def assert_source_is_migratable(data: dict[str, list[dict[str, Any]]]) -> None:
    for table in ("pipeline_stage_build_version_binding", "pipeline_run_build_version_binding", "repository_webhook"):
        if data[table]:
            raise ImportError(f"{table} is not empty; this legacy relation cannot be represented safely")

    projects = index_rows(data["project"], "project")
    credentials = index_rows(data["credential"], "credential")
    repositories = index_rows(data["repository"], "repository")
    templates = index_rows(data["pipeline_template"], "pipeline_template")
    stages = index_rows(data["pipeline_stage"], "pipeline_stage")
    snapshots = index_rows(data["pipeline_snapshot"], "pipeline_snapshot")
    runs = index_rows(data["pipeline_run"], "pipeline_run")
    artifacts = index_rows(data["artifact"], "artifact")

    ci_rows = (
        data["credential"]
        + data["repository"]
        + data["pipeline_template"]
        + data["pipeline_stage"]
        + data["pipeline_snapshot"]
        + data["pipeline_run"]
        + data["artifact"]
    )
    project_ids = {row.get("project_id") for row in ci_rows if row.get("project_id") is not None}
    if not project_ids or not project_ids <= set(projects):
        raise ImportError("legacy CI data refers to a missing project")

    for repository in repositories.values():
        credential_id = repository.get("git_credential_id")
        if credential_id is not None and credential_id not in credentials:
            raise ImportError(f"repository {repository['id']} refers to a missing credential")

    references_by_template: dict[str, list[dict[str, Any]]] = defaultdict(list)
    for reference in data["pipeline_template_stage"]:
        template_id, stage_id = reference.get("template_id"), reference.get("stage_id")
        if template_id not in templates or stage_id not in stages:
            raise ImportError("legacy pipeline_template_stage has a missing template or stage")
        json_array(reference.get("depends_on"), "pipeline_template_stage.depends_on")
        references_by_template[template_id].append(reference)
    for template_id, references in references_by_template.items():
        stage_ids = {reference["stage_id"] for reference in references}
        for reference in references:
            if any(dependency not in stage_ids for dependency in json_array(reference["depends_on"], "pipeline_template_stage.depends_on")):
                raise ImportError(f"template {template_id} has an external stage dependency")

    for snapshot in snapshots.values():
        if snapshot.get("template_id") not in templates:
            raise ImportError(f"snapshot {snapshot['id']} refers to a missing template")
        json_array(snapshot.get("stages_snapshot"), "pipeline_snapshot.stages_snapshot")
        json_array(snapshot.get("variables_snapshot"), "pipeline_snapshot.variables_snapshot")

    snapshot_users: set[str] = set()
    for run in runs.values():
        if run.get("repository_id") not in repositories or run.get("template_id") not in templates or run.get("snapshot_id") not in snapshots:
            raise ImportError(f"pipeline run {run['id']} has a missing repository, template, or snapshot")
        if snapshots[run["snapshot_id"]]["template_id"] != run["template_id"]:
            raise ImportError(f"pipeline run {run['id']} does not match its snapshot template")
        retry_of = run.get("retry_of")
        if retry_of is not None and retry_of not in runs:
            raise ImportError(f"pipeline run {run['id']} refers to a missing retry target")
        snapshot_users.add(run["snapshot_id"])
        json_array(run.get("variables_snapshot"), "pipeline_run.variables_snapshot")
    unused_snapshots = set(snapshots) - snapshot_users
    if unused_snapshots:
        raise ImportError("legacy snapshots without runs cannot be assigned to an application pipeline")

    for stage_run in data["pipeline_stage_run"]:
        if stage_run.get("pipeline_run_id") not in runs or stage_run.get("stage_id") not in stages:
            raise ImportError("legacy pipeline_stage_run has a missing run or stage")
    for artifact in artifacts.values():
        if artifact.get("pipeline_run_id") not in runs:
            raise ImportError(f"artifact {artifact['id']} refers to a missing pipeline run")
        source_artifact_id = artifact.get("source_artifact_id")
        if source_artifact_id is not None and source_artifact_id not in artifacts:
            raise ImportError(f"artifact {artifact['id']} refers to a missing source artifact")


def source_data(connection: sqlite3.Connection) -> dict[str, list[dict[str, Any]]]:
    ensure_integrity(connection, "source database")
    ensure_tables(connection, SOURCE_TABLES, "source database")
    data = {table: read_rows(connection, table) for table in SOURCE_TABLES}
    assert_source_is_migratable(data)
    return data


def destination_has_import(connection: sqlite3.Connection, data: dict[str, list[dict[str, Any]]]) -> bool:
    template_ids = [row["id"] for row in data["pipeline_template"]]
    placeholders = ", ".join("?" for _ in template_ids)
    found = {row[0] for row in connection.execute(f"SELECT id FROM pipeline WHERE id IN ({placeholders})", template_ids)}
    if found and found != set(template_ids):
        raise ImportError("target contains only part of this backup import; restore the target and retry")
    if found:
        return True

    checks = (
        ("credential", data["credential"]),
        ("repository", data["repository"]),
        ("pipeline_stage", data["pipeline_stage"]),
        ("pipeline_stage_reference", data["pipeline_template_stage"]),
        ("pipeline_run", data["pipeline_run"]),
        ("pipeline_stage_run", data["pipeline_stage_run"]),
        ("artifact", data["artifact"]),
    )
    for table, rows in checks:
        identifiers = [row["id"] for row in rows]
        if not identifiers:
            continue
        placeholders = ", ".join("?" for _ in identifiers)
        if connection.execute(f"SELECT 1 FROM {table} WHERE id IN ({placeholders}) LIMIT 1", identifiers).fetchone():
            raise ImportError(f"target already contains a legacy {table} identity; refusing to merge ambiguously")
    return False


def prepare_import(
    source: dict[str, list[dict[str, Any]]], target: sqlite3.Connection, source_cipher: Fernet, target_cipher: Fernet
) -> dict[str, Any]:
    templates = index_rows(source["pipeline_template"], "pipeline_template")
    stages = index_rows(source["pipeline_stage"], "pipeline_stage")
    repositories = index_rows(source["repository"], "repository")
    snapshots = index_rows(source["pipeline_snapshot"], "pipeline_snapshot")
    runs = index_rows(source["pipeline_run"], "pipeline_run")

    existing_stage_names = {
        row[0]
        for row in target.execute("SELECT name FROM pipeline_stage WHERE kind = 'template'")
    }
    existing_pipeline_names = {row[0] for row in target.execute("SELECT name FROM pipeline")}
    stage_names = {
        stage_id: unique_name(existing_stage_names, stage["name"], stage_id)
        for stage_id, stage in stages.items()
    }
    template_names = {
        template_id: unique_name(existing_pipeline_names, template["name"], template_id)
        for template_id, template in templates.items()
    }

    references_by_template: dict[str, list[dict[str, Any]]] = defaultdict(list)
    reference_by_template_and_stage: dict[tuple[str, str], dict[str, Any]] = {}
    for reference in source["pipeline_template_stage"]:
        references_by_template[reference["template_id"]].append(reference)
        reference_by_template_and_stage[(reference["template_id"], reference["stage_id"])] = reference
    for references in references_by_template.values():
        references.sort(key=lambda row: (row["sort_order"], row["id"]))

    app_pairs = sorted({(run["template_id"], run["repository_id"]) for run in runs.values()})
    app_pipelines: dict[tuple[str, str], dict[str, Any]] = {}
    for template_id, repository_id in app_pairs:
        pipeline_id = stable_id("legacy-ci-application-pipeline", template_id, repository_id)
        pipeline_name = unique_name(
            existing_pipeline_names,
            f"{template_names[template_id]} / {repositories[repository_id]['code']}",
            pipeline_id,
        )
        used_snapshots = [snapshots[run["snapshot_id"]] for run in runs.values() if run["template_id"] == template_id and run["repository_id"] == repository_id]
        app_pipelines[(template_id, repository_id)] = {
            "id": pipeline_id,
            "name": pipeline_name,
            "version": max(snapshot["version"] for snapshot in used_snapshots),
        }

    app_stage_ids: dict[tuple[str, str], str] = {}
    for pair, app_pipeline in app_pipelines.items():
        for reference in references_by_template[pair[0]]:
            app_stage_ids[(app_pipeline["id"], reference["id"])] = stable_id(
                "legacy-ci-application-stage", app_pipeline["id"], reference["id"]
            )

    snapshot_ids: dict[tuple[str, str], str] = {}
    for run in runs.values():
        app_pipeline = app_pipelines[(run["template_id"], run["repository_id"])]
        snapshot_ids[(app_pipeline["id"], run["snapshot_id"])] = stable_id(
            "legacy-ci-snapshot", app_pipeline["id"], run["snapshot_id"]
        )

    credentials: list[dict[str, Any]] = []
    for credential in source["credential"]:
        try:
            plain = source_cipher.decrypt(credential["encrypted_data"].encode("ascii"))
        except (InvalidToken, UnicodeEncodeError) as error:
            raise ImportError(f"credential {credential['id']} cannot be decrypted with the source JWT key") from error
        credentials.append({**credential, "encrypted_data": target_cipher.encrypt(plain).decode("ascii")})

    return {
        "credentials": credentials,
        "templates": templates,
        "stages": stages,
        "repositories": repositories,
        "snapshots": snapshots,
        "runs": runs,
        "stage_names": stage_names,
        "template_names": template_names,
        "references_by_template": references_by_template,
        "reference_by_template_and_stage": reference_by_template_and_stage,
        "app_pipelines": app_pipelines,
        "app_stage_ids": app_stage_ids,
        "snapshot_ids": snapshot_ids,
    }


def insert_row(connection: sqlite3.Connection, table: str, row: dict[str, Any]) -> None:
    columns = list(row)
    names = ", ".join(f'"{column}"' for column in columns)
    placeholders = ", ".join("?" for _ in columns)
    connection.execute(f"INSERT INTO {table} ({names}) VALUES ({placeholders})", [row[column] for column in columns])


def translated_stage_snapshot(
    snapshot: dict[str, Any], app_pipeline: dict[str, Any], plan: dict[str, Any]
) -> str:
    stages = plan["stages"]
    source_stages = json_array(snapshot["stages_snapshot"], "pipeline_snapshot.stages_snapshot")
    result: list[dict[str, Any]] = []
    for stage in source_stages:
        if not isinstance(stage, dict) or not isinstance(stage.get("id"), str):
            raise ImportError(f"snapshot {snapshot['id']} contains an invalid stage")
        reference = plan["reference_by_template_and_stage"].get((snapshot["template_id"], stage["id"]))
        if reference is None:
            raise ImportError(f"snapshot {snapshot['id']} contains a stage outside its template")
        dependencies = json_array(stage.get("depends_on"), "pipeline_snapshot.stages_snapshot.depends_on")
        dependency_ids: list[str] = []
        for dependency in dependencies:
            dependency_reference = plan["reference_by_template_and_stage"].get((snapshot["template_id"], dependency))
            if dependency_reference is None:
                raise ImportError(f"snapshot {snapshot['id']} contains an external stage dependency")
            dependency_ids.append(plan["app_stage_ids"][(app_pipeline["id"], dependency_reference["id"])])
        source_stage = stages[stage["id"]]
        stage_version = stage.get("version")
        if not isinstance(stage_version, int) or stage_version <= 0:
            raise ImportError(f"snapshot {snapshot['id']} stage has an invalid version")
        result.append(
            {
                "id": plan["app_stage_ids"][(app_pipeline["id"], reference["id"])],
                "name": stage.get("name", reference["stage_name"]),
                "image": stage.get("image", source_stage["image"]),
                "depends_on": dependency_ids,
                "script": stage.get("script", source_stage["script"]),
                "artifacts": json.loads(convert_artifacts(stage.get("artifacts"), "pipeline_snapshot.stages_snapshot.artifacts")),
                "sort_order": reference["sort_order"],
                "description": source_stage["description"],
                "source_template_stage_id": source_stage["id"],
                "source_template_stage_name": plan["stage_names"][source_stage["id"]],
                "source_template_stage_version": stage_version,
            }
        )
    return encode_json(result)


def translated_run_variables(run: dict[str, Any]) -> str:
    return convert_variables(run["variables_snapshot"], "pipeline_run.variables_snapshot")


def apply_import(connection: sqlite3.Connection, source: dict[str, list[dict[str, Any]]], plan: dict[str, Any]) -> None:
    templates = plan["templates"]
    stages = plan["stages"]
    repositories = plan["repositories"]
    runs = plan["runs"]

    for credential in plan["credentials"]:
        insert_row(connection, "credential", credential)
    for repository in source["repository"]:
        insert_row(connection, "repository", repository)
    for stage in stages.values():
        insert_row(
            connection,
            "pipeline_stage",
            {
                "id": stage["id"], "project_id": stage["project_id"], "kind": "template", "pipeline_id": None,
                "name": plan["stage_names"][stage["id"]], "image": stage["image"], "script": stage["script"],
                "description": stage["description"], "version": stage["version"],
                "source_template_stage_id": None, "source_template_stage_name": None,
                "source_template_stage_version": None, "source_template_stage_description": None,
                "artifacts": convert_artifacts(stage["artifacts"], "pipeline_stage.artifacts"),
                "depends_on": None, "sort_order": None, "created_at": stage["created_at"], "updated_at": stage["updated_at"],
            },
        )
    for template in templates.values():
        insert_row(
            connection,
            "pipeline",
            {
                "id": template["id"], "project_id": template["project_id"], "kind": "template",
                "source_pipeline_id": None, "source_template_name": None, "source_template_version": None,
                "application_id": None, "application_name": None, "repository_id": None, "repository_name": None,
                "version_fork_strategy": None, "fixed_version_id": None, "fixed_version_label": None,
                "name": plan["template_names"][template["id"]], "description": template["description"],
                "variable_declarations": convert_variables(template["variable_declarations"], "pipeline_template.variable_declarations"),
                "version": template["version"], "created_at": template["created_at"], "updated_at": template["updated_at"],
            },
        )
    for template_id, references in plan["references_by_template"].items():
        stage_to_reference = {reference["stage_id"]: reference["id"] for reference in references}
        for reference in references:
            stage = stages[reference["stage_id"]]
            dependencies = [stage_to_reference[item] for item in json_array(reference["depends_on"], "pipeline_template_stage.depends_on")]
            insert_row(
                connection,
                "pipeline_stage_reference",
                {
                    "id": reference["id"], "pipeline_id": template_id,
                    "source_template_stage_id": stage["id"], "source_template_stage_name": plan["stage_names"][stage["id"]],
                    "source_template_stage_version": reference["stage_version"],
                    "source_template_stage_description": stage["description"], "name": reference["stage_name"],
                    "image": stage["image"], "script": stage["script"], "description": stage["description"],
                    "artifacts": convert_artifacts(stage["artifacts"], "pipeline_stage.artifacts"),
                    "depends_on": encode_json(dependencies), "sort_order": reference["sort_order"],
                    "created_at": stage["created_at"], "updated_at": stage["updated_at"],
                },
            )
    for (template_id, repository_id), app_pipeline in plan["app_pipelines"].items():
        template, repository = templates[template_id], repositories[repository_id]
        insert_row(
            connection,
            "pipeline",
            {
                "id": app_pipeline["id"], "project_id": template["project_id"], "kind": "application",
                "source_pipeline_id": template_id, "source_template_name": plan["template_names"][template_id],
                "source_template_version": template["version"], "application_id": None, "application_name": None,
                "repository_id": repository_id, "repository_name": repository["name"], "version_fork_strategy": None,
                "fixed_version_id": None, "fixed_version_label": None, "name": app_pipeline["name"],
                "description": template["description"],
                "variable_declarations": convert_variables(template["variable_declarations"], "pipeline_template.variable_declarations"),
                "version": app_pipeline["version"], "created_at": template["created_at"], "updated_at": template["updated_at"],
            },
        )
        for reference in plan["references_by_template"][template_id]:
            stage = stages[reference["stage_id"]]
            dependencies = [
                plan["app_stage_ids"][(
                    app_pipeline["id"],
                    plan["reference_by_template_and_stage"][(template_id, dependency)]["id"],
                )]
                for dependency in json_array(reference["depends_on"], "pipeline_template_stage.depends_on")
            ]
            insert_row(
                connection,
                "pipeline_stage",
                {
                    "id": plan["app_stage_ids"][(app_pipeline["id"], reference["id"])],
                    "project_id": template["project_id"], "kind": "application", "pipeline_id": app_pipeline["id"],
                    "name": reference["stage_name"], "image": stage["image"], "script": stage["script"],
                    "description": stage["description"], "version": None, "source_template_stage_id": stage["id"],
                    "source_template_stage_name": plan["stage_names"][stage["id"]],
                    "source_template_stage_version": reference["stage_version"],
                    "source_template_stage_description": stage["description"],
                    "artifacts": convert_artifacts(stage["artifacts"], "pipeline_stage.artifacts"),
                    "depends_on": encode_json(dependencies), "sort_order": reference["sort_order"],
                    "created_at": stage["created_at"], "updated_at": stage["updated_at"],
                },
            )

    inserted_snapshots: set[tuple[str, str]] = set()
    for run in runs.values():
        app_pipeline = plan["app_pipelines"][(run["template_id"], run["repository_id"])]
        snapshot_key = (app_pipeline["id"], run["snapshot_id"])
        if snapshot_key in inserted_snapshots:
            continue
        inserted_snapshots.add(snapshot_key)
        snapshot = plan["snapshots"][run["snapshot_id"]]
        template, repository = templates[run["template_id"]], repositories[run["repository_id"]]
        insert_row(
            connection,
            "pipeline_snapshot",
            {
                "id": plan["snapshot_ids"][snapshot_key], "project_id": snapshot["project_id"],
                "pipeline_id": app_pipeline["id"], "pipeline_name": app_pipeline["name"], "pipeline_version": snapshot["version"],
                "source_pipeline_id": template["id"], "source_template_name": plan["template_names"][template["id"]],
                "source_template_version": snapshot["version"], "application_id": None, "application_name": None,
                "repository_id": repository["id"], "repository_name": repository["name"], "version_fork_strategy": None,
                "fixed_version_id": None, "fixed_version_label": None,
                "stages_snapshot": translated_stage_snapshot(snapshot, app_pipeline, plan),
                "variables_snapshot": convert_variables(snapshot["variables_snapshot"], "pipeline_snapshot.variables_snapshot"),
                "created_at": snapshot["created_at"],
            },
        )

    pending_runs = dict(runs)
    while pending_runs:
        ready = [run for run in pending_runs.values() if run.get("retry_of") is None or run["retry_of"] not in pending_runs]
        if not ready:
            raise ImportError("pipeline_run retry lineage contains a cycle")
        for run in ready:
            app_pipeline = plan["app_pipelines"][(run["template_id"], run["repository_id"])]
            insert_row(
                connection,
                "pipeline_run",
                {
                    "id": run["id"], "project_id": run["project_id"], "repository_id": run["repository_id"],
                    "repository_name": run["repository_name"], "snapshot_id": plan["snapshot_ids"][(app_pipeline["id"], run["snapshot_id"])],
                    "pipeline_id": app_pipeline["id"], "pipeline_name": app_pipeline["name"],
                    "pipeline_version": run["template_version"], "trigger": run["trigger"], "repository_ref": run["trigger_ref"],
                    "variables_snapshot": translated_run_variables(run), "status": run["status"], "retry_of": run["retry_of"],
                    "started_at": run["started_at"], "finished_at": run["finished_at"], "error_message": run["error_message"],
                    "created_at": run["created_at"],
                },
            )
            del pending_runs[run["id"]]

    for stage_run in source["pipeline_stage_run"]:
        run = runs[stage_run["pipeline_run_id"]]
        app_pipeline = plan["app_pipelines"][(run["template_id"], run["repository_id"])]
        reference = plan["reference_by_template_and_stage"][(run["template_id"], stage_run["stage_id"])]
        insert_row(
            connection,
            "pipeline_stage_run",
            {**stage_run, "stage_id": plan["app_stage_ids"][(app_pipeline["id"], reference["id"])]},
        )

    pending_artifacts = {artifact["id"]: artifact for artifact in source["artifact"]}
    while pending_artifacts:
        ready = [artifact for artifact in pending_artifacts.values() if artifact.get("source_artifact_id") is None or artifact["source_artifact_id"] not in pending_artifacts]
        if not ready:
            raise ImportError("artifact source lineage contains a cycle")
        for artifact in ready:
            run = runs[artifact["pipeline_run_id"]]
            app_pipeline = plan["app_pipelines"][(run["template_id"], run["repository_id"])]
            reference = next(
                (item for item in plan["references_by_template"][run["template_id"]] if item["stage_name"] == artifact["stage_name"]),
                None,
            )
            if reference is None:
                raise ImportError(f"artifact {artifact['id']} cannot be assigned to a template stage")
            insert_row(
                connection,
                "artifact",
                {
                    "id": artifact["id"], "project_id": artifact["project_id"], "pipeline_run_id": artifact["pipeline_run_id"],
                    "repository_id": artifact["repository_id"], "repository_name": artifact["repository_name"],
                    "pipeline_id": app_pipeline["id"], "pipeline_name": app_pipeline["name"],
                    "pipeline_stage_id": plan["app_stage_ids"][(app_pipeline["id"], reference["id"])],
                    "stage_name": artifact["stage_name"], "collector": artifact["collector"], "name": artifact["name"],
                    "location": artifact["location"], "value": artifact["value"], "value_format": artifact["value_format"],
                    "image_ref": artifact["image_ref"], "local_image_sha256": artifact["local_image_sha256"],
                    "source_artifact_id": artifact["source_artifact_id"], "created_at": artifact["created_at"],
                },
            )
            del pending_artifacts[artifact["id"]]


def verify_import(connection: sqlite3.Connection, source: dict[str, list[dict[str, Any]]], plan: dict[str, Any], target_cipher: Fernet) -> dict[str, int]:
    ensure_integrity(connection, "target database")
    for credential in plan["credentials"]:
        row = connection.execute("SELECT encrypted_data FROM credential WHERE id = ?", (credential["id"],)).fetchone()
        if row is None:
            raise ImportError(f"credential {credential['id']} is missing after import")
        try:
            target_cipher.decrypt(row[0].encode("ascii"))
        except (InvalidToken, UnicodeEncodeError) as error:
            raise ImportError(f"credential {credential['id']} cannot be decrypted with the target JWT key") from error

    expected_application_ids = [pipeline["id"] for pipeline in plan["app_pipelines"].values()]
    expected_snapshot_ids = list(plan["snapshot_ids"].values())
    expected = {
        "credentials": len(plan["credentials"]), "repositories": len(source["repository"]),
        "template_pipelines": len(source["pipeline_template"]), "application_pipelines": len(expected_application_ids),
        "template_stages": len(source["pipeline_stage"]),
        "template_stage_references": len(source["pipeline_template_stage"]),
        "pipeline_snapshots": len(expected_snapshot_ids), "pipeline_runs": len(source["pipeline_run"]),
        "pipeline_stage_runs": len(source["pipeline_stage_run"]), "artifacts": len(source["artifact"]),
    }
    queries = {
        "credentials": ("credential", [row["id"] for row in source["credential"]]),
        "repositories": ("repository", [row["id"] for row in source["repository"]]),
        "template_pipelines": ("pipeline", [row["id"] for row in source["pipeline_template"]]),
        "application_pipelines": ("pipeline", expected_application_ids),
        "template_stages": ("pipeline_stage", [row["id"] for row in source["pipeline_stage"]]),
        "template_stage_references": ("pipeline_stage_reference", [row["id"] for row in source["pipeline_template_stage"]]),
        "pipeline_snapshots": ("pipeline_snapshot", expected_snapshot_ids),
        "pipeline_runs": ("pipeline_run", [row["id"] for row in source["pipeline_run"]]),
        "pipeline_stage_runs": ("pipeline_stage_run", [row["id"] for row in source["pipeline_stage_run"]]),
        "artifacts": ("artifact", [row["id"] for row in source["artifact"]]),
    }
    for label, (table, identifiers) in queries.items():
        if not identifiers:
            continue
        placeholders = ", ".join("?" for _ in identifiers)
        count = connection.execute(f"SELECT COUNT(*) FROM {table} WHERE id IN ({placeholders})", identifiers).fetchone()[0]
        if count != expected[label]:
            raise ImportError(f"verification failed for {label}: found {count}, expected {expected[label]}")
    return expected


def clone_command(args: argparse.Namespace) -> Path:
    source_path, output_path = args.database.resolve(), args.output.resolve()
    if output_path.exists():
        raise ImportError(f"output database already exists: {output_path}")
    output_path.parent.mkdir(parents=True, exist_ok=True)
    source = connect_read_only(source_path)
    target = sqlite3.connect(output_path)
    try:
        ensure_integrity(source, "source database")
        source.backup(target)
    finally:
        target.close()
        source.close()
    return output_path


def import_command(args: argparse.Namespace) -> tuple[str, dict[str, int]]:
    source_connection = connect_read_only(args.source.resolve())
    target_connection = connect_read_write(args.database.resolve())
    try:
        source = source_data(source_connection)
        ensure_integrity(target_connection, "target database")
        ensure_tables(target_connection, TARGET_TABLES, "target database")
        source_cipher, target_cipher = fernet_from_env(args.source_env.resolve()), fernet_from_env(args.target_env.resolve())
        plan = prepare_import(source, target_connection, source_cipher, target_cipher)
        if destination_has_import(target_connection, source):
            return "already imported", verify_import(target_connection, source, plan, target_cipher)
        target_connection.execute("BEGIN IMMEDIATE")
        try:
            apply_import(target_connection, source, plan)
            counts = verify_import(target_connection, source, plan, target_cipher)
        except BaseException:
            target_connection.rollback()
            raise
        target_connection.commit()
        return "imported", counts
    finally:
        target_connection.close()
        source_connection.close()


def verify_command(args: argparse.Namespace) -> dict[str, int]:
    source_connection = connect_read_only(args.source.resolve())
    target_connection = connect_read_only(args.database.resolve())
    try:
        source = source_data(source_connection)
        ensure_integrity(target_connection, "target database")
        ensure_tables(target_connection, TARGET_TABLES, "target database")
        plan = prepare_import(source, target_connection, fernet_from_env(args.source_env.resolve()), fernet_from_env(args.target_env.resolve()))
        if not destination_has_import(target_connection, source):
            raise ImportError("target does not contain this backup import")
        return verify_import(target_connection, source, plan, fernet_from_env(args.target_env.resolve()))
    finally:
        target_connection.close()
        source_connection.close()


def format_counts(counts: dict[str, int]) -> str:
    return ", ".join(f"{name}={value}" for name, value in counts.items())


def main(argv: Sequence[str]) -> int:
    args = parse_args(argv)
    try:
        if args.command == "clone":
            print(f"SQLite copy written: {clone_command(args)}")
        elif args.command == "import":
            status, counts = import_command(args)
            print(f"legacy CI data {status}: {format_counts(counts)}")
        else:
            print(f"legacy CI import verified: {format_counts(verify_command(args))}")
    except (ImportError, OSError, sqlite3.Error) as error:
        print(f"legacy CI import blocked: {error}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
