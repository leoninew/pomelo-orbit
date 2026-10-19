"""Relocate the current Orbit workspace. apply --target previews; --no-dry-run executes."""

from __future__ import annotations

import argparse
import base64
import copy
import contextlib
import hashlib
import json
import logging
import os
from pathlib import Path, PurePosixPath, PureWindowsPath
import re
import secrets
import shutil
import socket
import stat
import subprocess
import sys
import tempfile
import time
from collections.abc import Callable, Iterator
from typing import Any
import urllib.parse

TERMINAL = {"ran_to_completion", "faulted", "canceled"}
WORKSPACE_CHILDREN = ("pipeline", "deployment", "deployment-logs")
CONFIG_DSN_ENV = "DBTALK_DSN_ORBIT_MIGRATION"


class MigrationError(Exception):
    """An actionable migration failure whose message contains no credentials."""


def configure_database(config: dict[str, Any]) -> None:
    value = config["database_url"]
    scheme, separator, tail = value.partition("://")
    if not separator or scheme not in {"postgres", "postgresql"}:
        raise MigrationError("The configured Orbit database is not PostgreSQL")
    os.environ[CONFIG_DSN_ENV] = "postgresql+psycopg://" + tail


def run(argv: list[str], cwd: Path, timeout: int = 120) -> str:
    try:
        result = subprocess.run(
            argv, cwd=cwd, capture_output=True, text=True, encoding="utf-8",
            errors="replace", timeout=timeout, check=False,
        )
    except (OSError, subprocess.TimeoutExpired):
        raise MigrationError(f"{Path(argv[0]).name} could not finish; check the tool and its configuration") from None
    if result.returncode:
        # Subcommands may print Compose environment or a DSN. Never echo them.
        raise MigrationError(f"{Path(argv[0]).name} failed (exit {result.returncode}); inspect it separately")
    return result.stdout


def digest(value: Any) -> str:
    return hashlib.sha256(json.dumps(value, sort_keys=True).encode()).hexdigest()


def file_hash(path: Path) -> str:
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def write_private(path: Path, content: str) -> None:
    no_links(path)
    descriptor, name = tempfile.mkstemp(prefix="." + path.name + ".", dir=path.parent)
    temporary = Path(name)
    try:
        with os.fdopen(descriptor, "w", encoding="utf-8", newline="\n") as stream:
            stream.write(content)
            stream.flush()
            os.fsync(stream.fileno())
        temporary.replace(path)
    finally:
        temporary.unlink(missing_ok=True)


def write_json(path: Path, value: Any) -> None:
    write_private(path, json.dumps(value, indent=2, ensure_ascii=False) + "\n")


def no_links(path: Path) -> None:
    for part in (path, *path.parents):
        try:
            reparse = getattr(part.lstat(), "st_reparse_tag", 0)
        except FileNotFoundError:
            reparse = 0
        if part.is_symlink() or part.is_junction() or reparse:
            raise MigrationError(f"Symlink/junction is outside the relocation contract: {part}")


def absolute_path(path: Path) -> Path:
    path = path.absolute()
    no_links(path)
    return path.resolve()


def runtime_socket(path: Path) -> bool:
    """Do not copy sockets or MySQL's known symlink to its ephemeral Unix socket."""
    metadata = path.lstat()
    if stat.S_ISSOCK(metadata.st_mode):
        return True
    if path.name != "mysql.sock":
        return False
    if stat.S_ISLNK(metadata.st_mode):
        destination = os.readlink(path)
    elif os.name == "nt" and getattr(metadata, "st_reparse_tag", 0) == 0xA000001D:
        # WSL LX_SYMLINK is not recognized by pathlib on native Windows.
        import ctypes
        from ctypes import wintypes
        kernel = ctypes.WinDLL("kernel32", use_last_error=True)
        kernel.CreateFileW.argtypes = [wintypes.LPCWSTR, wintypes.DWORD, wintypes.DWORD,
                                      wintypes.LPVOID, wintypes.DWORD, wintypes.DWORD, wintypes.HANDLE]
        kernel.CreateFileW.restype = wintypes.HANDLE
        kernel.DeviceIoControl.argtypes = [wintypes.HANDLE, wintypes.DWORD, wintypes.LPVOID,
                                          wintypes.DWORD, wintypes.LPVOID, wintypes.DWORD,
                                          ctypes.POINTER(wintypes.DWORD), wintypes.LPVOID]
        kernel.CloseHandle.argtypes = [wintypes.HANDLE]
        handle = kernel.CreateFileW(str(path), 0, 7, None, 3, 0x02200000, None)
        if handle == wintypes.HANDLE(-1).value:
            raise OSError(ctypes.get_last_error(), "Cannot inspect WSL socket link")
        try:
            buffer = ctypes.create_string_buffer(16384)
            length = wintypes.DWORD()
            if not kernel.DeviceIoControl(handle, 0x000900A8, None, 0, buffer, len(buffer), ctypes.byref(length), None):
                raise OSError(ctypes.get_last_error(), "Cannot read WSL socket link")
            destination = buffer.raw[12:length.value].rstrip(b"\x00").decode("utf-8")
        finally:
            kernel.CloseHandle(handle)
    else:
        return False
    return destination in {"/var/run/mysqld/mysqld.sock", "/run/mysqld/mysqld.sock"}


def native_slashes(value: str | Path) -> str:
    value = str(value).replace("\\", "/").rstrip("/")
    for prefix in ("/run/desktop/mnt/host/", "/host_mnt/"):
        if value.startswith(prefix):
            tail = value[len(prefix):]
            if re.match(r"^[a-zA-Z]/", tail):
                value = tail[0] + ":" + tail[1:]
    return value


def canonical(value: str | Path) -> str:
    value = native_slashes(value)
    return value.casefold() if re.match(r"^[a-zA-Z]:/", value) else value


def within(value: str | Path, root: str | Path) -> bool:
    value, root = canonical(value), canonical(root)
    return value == root or value.startswith(root + "/")


def relocate(value: str | None, source: str, target: str) -> str | None:
    if not value or not within(value, source):
        return value
    tail = native_slashes(value)[len(native_slashes(source)):]
    return target.replace("\\", "/").rstrip("/") + tail


def relocate_data_path(value: str | None, source: str, target: str) -> str | None:
    if value and within(value, source) and not any(within(value, join_target(source, child)) for child in WORKSPACE_CHILDREN):
        raise MigrationError("Workspace path is outside the copied pipeline/deployment/deployment-logs directories: " + value)
    return relocate(value, source, target)


def validate_paths(source: Path, target: Path, bundle: Path) -> None:
    for path in (source, target, bundle):
        if not path.is_absolute():
            raise MigrationError("Source, target and plan directory must be absolute paths")
        no_links(path)
    if not source.is_dir() or source == Path(source.anchor) or target == Path(target.anchor):
        raise MigrationError("Source must be a directory; source/target cannot be a filesystem root")
    if within(source, target) or within(target, source):
        raise MigrationError("Source and target must be separate directory trees")
    copied_roots = [source / child for child in WORKSPACE_CHILDREN]
    if (within(source, bundle) or within(bundle, target) or within(target, bundle)
            or any(within(bundle, path) or within(path, bundle) for path in copied_roots)):
        raise MigrationError("Keep the plan directory outside the copied workspace subdirectories and target")


class Database:
    def __init__(self, cwd: Path, variable: str):
        if not re.fullmatch(r"DBTALK_DSN_[A-Z0-9_]+", variable):
            raise MigrationError("Use a DBTALK_DSN_* variable from the process or cwd/.env")
        self.cwd, self.variable = cwd, variable

    def query(self, sql: str) -> list[dict[str, Any]]:
        output = run(["dbtalk", "query", "--dsn-env", self.variable,
                      "--sql", sql, "--format", "json"], self.cwd)
        try:
            return json.loads(output)["rows"]
        except (ValueError, KeyError):
            raise MigrationError("dbtalk did not return a JSON query result") from None

    def execute(self, path: Path) -> None:
        run(["dbtalk", "exec", "--dsn-env", self.variable, "--file", str(path)], self.cwd)

    def backup(self, path: Path) -> None:
        if path.exists():
            return  # Final name is published only after a successful dump.
        pending = path.with_name(path.name + "." + secrets.token_hex(8) + ".pending")
        run(["dbtalk", "postgres", "dump", "--dsn-env", self.variable,
             "--output", str(pending)], self.cwd, 600)
        if not pending.is_file() or pending.stat().st_size == 0:
            raise MigrationError("Database dump did not produce a nonempty archive")
        pending.replace(path)


def sql_value(value: Any) -> str:
    if value is None:
        return "NULL"
    if isinstance(value, bool):
        return "TRUE" if value else "FALSE"
    if isinstance(value, (int, float)):
        return str(value)
    return "'" + str(value).replace("'", "''") + "'"


def predicate(fields: dict[str, Any]) -> str:
    return " AND ".join(f'"{key}" IS NOT DISTINCT FROM {sql_value(value)}' for key, value in fields.items())


def mutation(table: str, key: dict[str, Any], before: dict[str, Any] | None,
             after: dict[str, Any]) -> dict[str, Any]:
    return {"table": table, "key": key, "before": before, "after": after}


def migration_sql(operations: list[dict[str, Any]]) -> str:
    """One PostgreSQL statement: guard every input, then commit all changes together."""
    guards, writes, finished = [], [], []
    for op in operations:
        table, key = op["table"], op["key"]
        if not re.fullmatch(r"[a-z_]+", table):
            raise MigrationError("Invalid mutation table")
        for fields in (key, op["before"] or {}, op["after"]):
            if any(not re.fullmatch(r"[a-z_]+", field) for field in fields):
                raise MigrationError("Invalid mutation column")
        before, after = op["before"], {**key, **op["after"]}
        exists = f'SELECT 1 FROM "{table}" WHERE {predicate(key)}'
        if before is None:
            guards.append(f"IF EXISTS ({exists}) THEN RAISE EXCEPTION 'migration input changed'; END IF;")
            writes.append(f'INSERT INTO "{table}" ({", ".join(after)}) VALUES ({", ".join(map(sql_value, after.values()))});')
        else:
            guards.append(f'IF NOT EXISTS (SELECT 1 FROM "{table}" WHERE {predicate({**key, **before})}) THEN RAISE EXCEPTION \'migration input changed\'; END IF;')
            assignments = ", ".join(f'"{k}" = {sql_value(v)}' for k, v in op["after"].items())
            writes.append(f'UPDATE "{table}" SET {assignments} WHERE {predicate(key)};')
        finished.append(f'EXISTS (SELECT 1 FROM "{table}" WHERE {predicate(after)})')
    if not operations:
        raise MigrationError("No relocation mutations were planned")
    # Locks prevent a stale plan from racing another business writer.
    body = "LOCK TABLE environment, service, pipeline_run, artifact, service_component, service_component_mount IN EXCLUSIVE MODE;\n"
    body += "IF " + " AND ".join(finished) + " THEN RETURN; END IF;\n"
    body += "\n".join([*guards, *writes])
    delimiter = "$orbit_" + secrets.token_hex(16) + "$"
    while delimiter in body:
        delimiter = "$orbit_" + secrets.token_hex(16) + "$"
    return f"DO {delimiter}\nBEGIN\n{body}\nEND;\n{delimiter};\n"


def inventory(db: Database, environment: str) -> dict[str, Any]:
    environments = db.query(
        "SELECT e.*,p.name AS project_name FROM environment e JOIN project p ON p.id=e.project_id "
        f"WHERE e.id={sql_value(environment)} OR e.code={sql_value(environment)}"
    )
    if len(environments) != 1:
        raise MigrationError("Environment ID or code must identify exactly one Environment")
    env = environments[0]
    project = sql_value(env["project_id"])
    services = db.query(
        "SELECT s.*,a.kind AS application_kind FROM service s JOIN application a ON a.id=s.application_id "
        f"WHERE s.project_id={project} ORDER BY s.code"
    )
    mounts = db.query(
        "SELECT s.id AS service_id,vc.id AS component_id,vc.name AS component_name,"
        "m.target,m.source,m.source_is_host_path,sc.id AS overlay_id,sm.id AS mount_id,"
        "sm.source AS overlay_source,sm.source_is_host_path AS overlay_host_path,sm.state AS overlay_state "
        "FROM service s JOIN version_component vc ON vc.version_id=s.version_id "
        "JOIN version_component_mount m ON m.component_id=vc.id "
        "LEFT JOIN service_component sc ON sc.service_id=s.id AND sc.source_version_component_id=vc.id "
        "LEFT JOIN service_component_mount sm ON sm.service_component_id=sc.id AND sm.target=m.target "
        f"WHERE s.project_id={project} ORDER BY s.id,vc.id,m.target"
    )
    runs = db.query(
        "SELECT id,environment_id,environment_target_type,environment_target_revision,"
        f"ssh_credential_id,ssh_credential_revision,status FROM pipeline_run WHERE project_id={project} ORDER BY id"
    )
    artifacts = db.query(f"SELECT id,location FROM artifact WHERE project_id={project} AND location IS NOT NULL ORDER BY id")
    routes = db.query(f"SELECT id,enabled,updated_at FROM route WHERE project_id={project} ORDER BY id")
    return {"environment": env, "services": services, "mounts": mounts, "runs": runs,
            "artifacts": artifacts, "routes": routes}


def active_jobs(db: Database) -> bool:
    return bool(db.query("SELECT id FROM background_task WHERE status NOT IN ('succeeded','failed','canceled') LIMIT 1"))


def docker_inventory(cwd: Path) -> tuple[str, list[dict[str, Any]]]:
    endpoint = os.environ.get("DOCKER_HOST") or run(
        ["docker", "context", "inspect", "--format", '{{.Endpoints.docker.Host}}'], cwd).strip()
    if not endpoint.startswith(("unix://", "npipe://")):
        raise MigrationError("This host-side script requires a local Docker socket, not a remote Docker endpoint")
    daemon = run(["docker", "info", "--format", "{{.ID}}"], cwd).strip()
    ids = run(["docker", "ps", "-aq"], cwd).split()
    # Do not persist Config.Env or arbitrary labels, which can contain secrets.
    containers = json.loads(run(["docker", "inspect", *ids], cwd)) if ids else []
    result = []
    for container in containers:
        labels = container["Config"].get("Labels") or {}
        result.append({
            "id": container["Id"], "running": container["State"]["Running"],
            "project": labels.get("com.docker.compose.project", ""),
            "component": labels.get("com.docker.compose.service", ""),
            "oneoff": labels.get("com.docker.compose.oneoff", "False").casefold() == "true",
            "image": container["Config"].get("Image", ""),
            "working_dir": labels.get("com.docker.compose.project.working_dir", ""),
            "addresses": [network.get("IPAddress") for network in
                          container.get("NetworkSettings", {}).get("Networks", {}).values()],
            "mounts": [{key: mount.get(key) for key in ("Type", "Source", "Destination")}
                       for mount in container.get("Mounts", [])],
        })
    return daemon, result


def local_target(env: dict[str, Any]) -> bool:
    if env["target_type"] == "local":
        return True
    local_names = {"localhost", socket.gethostname().casefold(), socket.getfqdn().casefold()}
    host = str(env["host"]).casefold()
    if host in local_names or host in {"127.0.0.1", "::1"}:
        return True
    try:
        addresses = {entry[4][0] for name in local_names for entry in socket.getaddrinfo(name, None)}
        target = {entry[4][0] for entry in socket.getaddrinfo(host, None)}
        return bool(addresses & target)
    except socket.gaierror:
        return False


def assess_files(source: Path, target: Path) -> dict[str, Any]:
    count, size, problems, sockets = 0, 0, [], []
    for child in WORKSPACE_CHILDREN:
        root = source / child
        try:
            no_links(root)
        except MigrationError:
            problems.append("link: " + str(root))
            continue
        if not root.exists():
            continue
        for base, directories, files in os.walk(root, onerror=lambda error: problems.append(str(error.filename))):
            for name in [*directories, *files]:
                path = Path(base) / name
                if name in files:
                    try:
                        if runtime_socket(path):
                            sockets.append(str(path))
                            continue
                    except FileNotFoundError:
                        continue
                    except OSError:
                        problems.append(str(path))
                        continue
                try:
                    linked = path.is_symlink() or path.is_junction() or getattr(path.lstat(), "st_reparse_tag", 0)
                except OSError:
                    problems.append(str(path))
                    if name in directories:
                        directories.remove(name)
                    continue
                if linked:
                    problems.append("link: " + str(path))
                    if name in directories:
                        directories.remove(name)
                    continue
                if name in files:
                    try:
                        size += path.stat().st_size
                        count += 1
                    except OSError:
                        problems.append(str(path))
    ancestor = target
    while not ancestor.exists():
        ancestor = ancestor.parent
    return {"files": count, "bytes": size, "free_bytes": shutil.disk_usage(ancestor).free,
            "unreadable_or_linked": problems, "runtime_sockets_skipped": sockets}


def new_id() -> str:
    number = (int(time.time() * 1000) << 80) | int.from_bytes(secrets.token_bytes(10))
    alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
    return "".join(alphabet[(number >> shift) & 31] for shift in range(125, -1, -5))


def build_mutations(snapshot: dict[str, Any], source: str, target: str,
                    timestamp: str, expanded_paths: dict[str, str]) -> list[dict[str, Any]]:
    def move(value: str | None) -> str | None:
        expanded = expanded_paths.get(value, value)
        relocated = relocate_data_path(expanded, source, target)
        return value if expanded == relocated else relocated

    env = snapshot["environment"]
    revision = env["target_revision"] + 1
    before = {key: value for key, value in env.items() if key != "project_name"}
    after = {"workspace_root": target, "target_revision": revision, "updated_at": timestamp}
    operations = [mutation("environment", {"id": env["id"]}, before, after)]
    for service in snapshot["services"]:
        changed = {}
        for field, revision_field in (("deployment_directory", "directory_target_revision"),
                                      ("runtime_directory", "runtime_target_revision")):
            value = move(service[field])
            if value != service[field]:
                changed[field], changed[revision_field] = value, revision
        if changed:
            changed["updated_at"] = timestamp
            before = {key: value for key, value in service.items() if key != "application_kind"}
            operations.append(mutation("service", {"id": service["id"]}, before, changed))
    overlays = {}
    for mount in snapshot["mounts"]:
        if mount["overlay_state"] == "deleted":
            continue
        old = mount["overlay_source"] if mount["overlay_state"] == "override" else mount["source"]
        new = move(old)
        if old == new:
            continue
        component_key = (mount["service_id"], mount["component_id"])
        overlay = mount["overlay_id"] or overlays.get(component_key)
        if not overlay:
            overlay = new_id()
            overlays[component_key] = overlay
            operations.append(mutation("service_component", {"id": overlay}, None, {
                "service_id": mount["service_id"], "source_version_component_id": mount["component_id"],
                "component_name": mount["component_name"], "status": "active",
                "created_at": timestamp, "updated_at": timestamp,
            }))
        key = {"id": mount["mount_id"] or new_id()}
        before = None if not mount["mount_id"] else {
            "source": old, "state": "override", "source_is_host_path": mount["overlay_host_path"],
        }
        operations.append(mutation("service_component_mount", key, before, {
            "service_component_id": overlay, "target": mount["target"], "source": new,
            "state": "override", "source_is_host_path": mount["overlay_host_path"]
            if mount["overlay_host_path"] is not None else mount["source_is_host_path"],
        }))
    for artifact in snapshot["artifacts"]:
        location = move(artifact["location"])
        if location != artifact["location"]:
            operations.append(mutation("artifact", {"id": artifact["id"]},
                                       {"location": artifact["location"]}, {"location": location}))
    for pipeline_run in snapshot["runs"]:
        # Only an already-valid binding can be carried across a path-only move.
        if (pipeline_run["environment_id"] == env["id"]
                and pipeline_run["environment_target_type"] == env["target_type"]
                and pipeline_run["environment_target_revision"] == env["target_revision"]
                and pipeline_run["ssh_credential_id"] == env["ssh_credential_id"]
                and pipeline_run["ssh_credential_revision"] == env["ssh_credential_revision"]):
            operations.append(mutation("pipeline_run", {"id": pipeline_run["id"]}, pipeline_run,
                                       {"environment_target_revision": revision}))
    return operations


@contextlib.contextmanager
def execution_lock(path: Path) -> Iterator[None]:
    """OS locks are released on process exit, including after an interrupted run."""
    with path.open("a+b") as lock:
        lock.seek(0)
        if os.name == "nt":
            import msvcrt
            if path.stat().st_size == 0:
                lock.write(b"0")
                lock.flush()
            lock.seek(0)
            try:
                msvcrt.locking(lock.fileno(), msvcrt.LK_NBLCK, 1)
            except OSError:
                raise MigrationError("Another process is applying this plan") from None
        else:
            import fcntl
            try:
                fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
            except OSError:
                raise MigrationError("Another process is applying this plan") from None
        yield


def copy_workspace(source: Path, target: Path) -> dict[str, int]:
    count, total = 0, 0
    def preserve_directory(original: Path, copied: Path) -> None:
        if os.name != "nt":
            metadata = original.stat()
            os.chown(copied, metadata.st_uid, metadata.st_gid)
            copied.chmod(stat.S_IMODE(metadata.st_mode))

    target.mkdir(parents=True, exist_ok=True)
    preserve_directory(source, target)
    for child in WORKSPACE_CHILDREN:
        root = source / child
        no_links(root)
        if not root.exists():
            continue
        def copy_error(error: OSError) -> None:
            raise error

        for base, directories, files in os.walk(root, onerror=copy_error):
            base_path = Path(base)
            no_links(base_path)
            destination = target / base_path.relative_to(source)
            no_links(destination)
            destination.mkdir(parents=True, exist_ok=True)
            preserve_directory(base_path, destination)
            for name in directories:
                no_links(base_path / name)
            for name in files:
                original, copied = base_path / name, destination / name
                if runtime_socket(original):
                    continue
                no_links(original)
                no_links(copied)
                if not original.is_file():
                    raise MigrationError(f"Cannot copy a non-regular file: {original}")
                if not copied.exists() or file_hash(original) != file_hash(copied):
                    shutil.copy2(original, copied)
                if os.name != "nt":
                    metadata = original.stat()
                    os.chown(copied, metadata.st_uid, metadata.st_gid)
                if file_hash(original) != file_hash(copied):
                    raise MigrationError(f"Copied file checksum differs: {copied}")
                count += 1
                total += original.stat().st_size
    logging.info("Workspace copied and verified files=%d bytes=%d", count, total)
    return {"files": count, "bytes": total}



def secure_directory(directory: Path) -> None:
    directory.mkdir(parents=True, exist_ok=True, mode=0o700)
    no_links(directory)
    if os.name == "nt":
        identity = run(["whoami", "/user", "/fo", "csv", "/nh"], directory)
        sid = re.search(r"S-1-5-[0-9-]+", identity)
        if sid is None:
            raise MigrationError("Cannot resolve the current Windows SID")
        run(["icacls", str(directory), "/inheritance:r", "/grant:r",
             "*" + sid[0] + ":(OI)(CI)F", "*S-1-5-18:(OI)(CI)F"], directory)
    else:
        directory.chmod(0o700)


class Runtime:
    """Small, automatically built bridge to the project's existing Go adapters."""

    def __init__(self, cwd: Path):
        self.cwd = cwd
        directory = cwd / "data" / "workspace-migration" / "runtime"
        secure_directory(directory)
        self.binary = directory / ("helper.exe" if os.name == "nt" else "helper")
        started = time.monotonic()
        logging.info("Building migration runtime")
        run(["go", "build", "-o", str(self.binary), str(cwd / "scripts" / "workspace_migration_runtime.go")], cwd, 300)
        logging.info("Migration runtime ready elapsed_s=%.1f", time.monotonic() - started)

    def call(self, action: str, **values: Any) -> Any:
        try:
            result = subprocess.run(
                [str(self.binary)], input=json.dumps({"action": action, **values}), cwd=self.cwd,
                capture_output=True, text=True, encoding="utf-8", errors="replace",
                timeout=int(values.get("timeout", 120)) + 30, check=False,
            )
            payload = json.loads(result.stdout)
        except (OSError, ValueError, subprocess.TimeoutExpired):
            raise MigrationError("Migration runtime could not finish action=" + action) from None
        if result.returncode or isinstance(payload, dict) and "error" in payload:
            raise MigrationError(payload.get("error", "Migration runtime failed action=" + action))
        return payload


def target_path(value: str, platform: str) -> str:
    kind = PureWindowsPath if platform == "windows" else PurePosixPath
    return str(kind(value))


def join_target(root: str, name: str) -> str:
    kind = PureWindowsPath if re.match(r"^[A-Za-z]:", root) or root.startswith("\\\\") else PurePosixPath
    return str(kind(root) / name)


def _host_action(request: dict[str, Any]) -> Any:
    """These operations run on the data host, including across SSH."""
    action = request["action"]
    if action == "resolve-target":
        return str(absolute_path(Path(os.path.expanduser(request["target"]))))
    if action == "expand-home-paths":
        return {value: str(absolute_path(Path(os.path.expanduser(native_slashes(value)))))
                for value in request["paths"] if value and (value == "~" or value.startswith(("~/", "~\\")))}
    if action == "paths":
        if sys.version_info < (3, 12):
            raise MigrationError("The data host needs Python 3.12 or later")
        source = absolute_path(Path(os.path.expanduser(request["source"])))
        if not request.get("target") or not request["target"].strip():
            raise MigrationError("Target directory is required")
        target = absolute_path(Path(os.path.expanduser(request["target"])))
        if source != target:
            validate_paths(source, target, source / "workspace-migration" / "validation")
        return {"source": str(source), "target": str(target), "host": socket.gethostname(),
                "platform": "windows" if os.name == "nt" else "linux", "python": sys.executable}
    if action == "containers":
        daemon, containers = docker_inventory(Path.home())
        return {"daemon": daemon, "containers": containers}
    if action == "assessment":
        source, target = Path(request["source"]), Path(request["target"])
        if target.exists() and (not target.is_dir() or any(target.iterdir())):
            raise MigrationError("The migration destination is not empty: " + str(target))
        return assess_files(source, target)
    if action == "files":
        return {path: file_hash(Path(path)) if Path(path).is_file() else None for path in request["paths"]}
    if action == "read":
        path = Path(request["path"])
        no_links(path)
        return path.read_text(encoding="utf-8-sig") if path.exists() else None
    if action == "write":
        path = Path(request["path"])
        no_links(path)
        path.parent.mkdir(parents=True, exist_ok=True)
        write_private(path, request["content"])
        return None
    if action == "command":
        return run(request["argv"], Path.home(), request.get("timeout", 120))
    if action == "copy":
        source, target = Path(request["source"]), Path(request["target"])
        _, containers = docker_inventory(Path.home())
        if any(container["running"] and any(mount["Type"] == "bind" and any(
                within(mount["Source"], root) or within(root, mount["Source"])
                for root in (source, target)) for mount in container["mounts"]) for container in containers):
            raise MigrationError("A running container still mounts the source or destination; refusing a live data copy")
        return copy_workspace(source, target)
    if action == "same-host-environments":
        local_control = request["control_host"].casefold() == socket.gethostname().casefold()
        return [item["id"] for item in request["environments"] if
                (item["target_type"] == "local" and local_control) or
                (item["target_type"] == "ssh" and local_target(item))]
    if action in {"claim", "release"}:
        path = Path(request["source"]) / ".orbit-workspace-migration"
        no_links(path)
        if action == "release":
            if path.exists() and path.read_text(encoding="utf-8") == request["owner"]:
                path.unlink()
            return None
        try:
            descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        except FileExistsError:
            if path.read_text(encoding="utf-8") != request["owner"]:
                raise MigrationError("A different migration already owns this workspace")
        else:
            with os.fdopen(descriptor, "w", encoding="utf-8") as output:
                output.write(request["owner"])
        return None
    raise MigrationError("Unknown data-host action")


def host_entry(request: dict[str, Any]) -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s", stream=sys.stderr)
    try:
        result = {"result": _host_action(request)}
    except (MigrationError, OSError, RuntimeError) as error:
        result = {"error": str(error)}
    print(json.dumps(result, ensure_ascii=False))


class Host:
    def __init__(self, runtime: Runtime, connection: dict[str, Any], python: str | None = None):
        self.runtime, self.connection = runtime, copy.deepcopy(connection)
        self.python = python
        self.remote = connection["environment"]["target_type"] == "ssh"

    def python_candidates(self) -> list[str]:
        if self.python:
            return [self.python]
        environment = self.connection["environment"]
        if local_target(environment) and (environment["platform"] == "windows") == (os.name == "nt"):
            return [str(Path(sys.executable).resolve())]
        if environment["platform"] == "windows":
            script = (
                "$ErrorActionPreference='Stop'; [Console]::OutputEncoding=[Text.UTF8Encoding]::new($false); "
                "$profiles=@($env:USERPROFILE,[Environment]::GetFolderPath('UserProfile')); "
                "$sid=[Security.Principal.WindowsIdentity]::GetCurrent().User.Value; "
                "$profileKey='HKLM:\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\ProfileList\\'+$sid; "
                "if(Test-Path -LiteralPath $profileKey) { $profiles += (Get-ItemProperty -LiteralPath $profileKey).ProfileImagePath }; "
                "$paths=@($profiles | Where-Object { $_ } | Select-Object -Unique | ForEach-Object { Join-Path $_ '.version-fox/sdks/python/python.exe' }); "
                "foreach($base in @('HKCU:\\Software\\Python\\PythonCore','HKLM:\\Software\\Python\\PythonCore')) { "
                "if(Test-Path -LiteralPath $base) { Get-ChildItem -LiteralPath $base | ForEach-Object { "
                "$key=Join-Path $_.PSPath 'InstallPath'; if(Test-Path -LiteralPath $key) { "
                "$install=(Get-Item -LiteralPath $key).GetValue(''); if($install) { $paths += Join-Path $install 'python.exe' } } } } }; "
                "$paths += @(Get-Command python3,python -CommandType Application -All -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source); "
                "ConvertTo-Json -InputObject @($paths | Where-Object { Test-Path -LiteralPath $_ -PathType Leaf } | Select-Object -Unique) -Compress"
            )
            argv = ["powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script]
        else:
            argv = ["sh", "-c", "command -v python3 || command -v python"]
        response = self.runtime.call("exec", connection=self.connection, argv=argv, workdir="~", timeout=30)
        if response["exit_code"]:
            raise MigrationError("No Python executable could be discovered on the SSH data host")
        if response.get("fingerprint"):
            self.connection["environment"]["host_key_fingerprint"] = response["fingerprint"]
        candidates = json.loads(response["stdout"]) if environment["platform"] == "windows" else response["stdout"].splitlines()
        return list(dict.fromkeys(candidates))

    def call(self, action: str, **values: Any) -> Any:
        if not self.remote:
            return _host_action({"action": action, **values})
        # Send the same tested implementation over SSH; no remote checkout or
        # dbtalk installation is needed, and encrypted bindings survive DB downtime.
        source = Path(__file__).read_text(encoding="utf-8").rsplit('\nif __name__ == "__main__":', 1)[0]
        payload = source + "\nhost_entry(" + repr({"action": action, **values}) + ")\n"
        programs = self.python_candidates()
        last_error = None
        for program in programs:
            try:
                response = self.runtime.call(
                    "exec", connection=self.connection,
                    argv=[program, "-c", "import sys; exec(compile(sys.stdin.buffer.read(), '<orbit-migration>', 'exec'))"],
                    workdir="~", stdin=base64.b64encode(payload.encode()).decode(), timeout=values.get("timeout", 120),
                )
                if response["exit_code"] != 0:
                    raise MigrationError("Python could not start on the SSH data host")
                result = json.loads(response["stdout"])
                if "error" in result:
                    raise MigrationError(result["error"])
                self.python = program
                if response.get("fingerprint"):
                    self.connection["environment"]["host_key_fingerprint"] = response["fingerprint"]
                return result["result"]
            except (MigrationError, ValueError) as error:
                last_error = error
                if self.python or action != "paths":
                    raise
        raise MigrationError(str(last_error))


def select_environment(db: Database, config: dict[str, Any], requested: str | None) -> dict[str, Any]:
    rows = db.query("SELECT e.*,p.name AS project_name FROM environment e JOIN project p ON p.id=e.project_id WHERE p.is_active=TRUE ORDER BY e.code")
    if requested:
        matches = [row for row in rows if requested in {row["id"], row["code"]}]
    else:
        matches = [row for row in rows if canonical(row["workspace_root"] or "") == canonical(config["workspace_root"]) and local_target(row)]
        if not matches:
            matches = [row for row in rows if canonical(row["workspace_root"] or "") == canonical(config["workspace_root"])]
        if not matches and len(rows) == 1:
            matches = rows
    if len(matches) != 1:
        raise MigrationError("Cannot identify one Environment from this workspace; available codes: " + ", ".join(row["code"] for row in rows))
    return matches[0]


def connection_for(db: Database, environment: dict[str, Any]) -> dict[str, Any]:
    result = {"environment": copy.deepcopy(environment)}
    if environment["target_type"] == "ssh":
        credentials = db.query("SELECT id,project_id,revision,public_key,encrypted_private_key FROM environment_credential WHERE id=" + sql_value(environment["ssh_credential_id"]))
        if len(credentials) != 1:
            raise MigrationError("The Environment's existing managed SSH credential could not be read")
        result["credential"] = credentials[0]
    return result


def normalize_target_argument(value: str, environment: dict[str, Any]) -> str:
    windows = environment["platform"] == "windows" if environment["target_type"] == "ssh" else os.name == "nt"
    if not windows:
        return value
    if value.startswith("/") and os.environ.get("ORBIT_MIGRATION_CYGPATH"):
        return run([os.environ["ORBIT_MIGRATION_CYGPATH"], "-am", "--", value], Path.cwd()).strip()
    if not PureWindowsPath(value).is_absolute() and os.environ.get("ORBIT_MIGRATION_CALLER_CWD"):
        return str(PureWindowsPath(os.environ["ORBIT_MIGRATION_CALLER_CWD"]) / value)
    return value


def service_priority(service: dict[str, Any]) -> tuple:
    return (not service["bootstrap"], service["kind"] != "gateway", service["code"])


def plan_migration(args: argparse.Namespace, runtime: Runtime, config: dict[str, Any], db: Database, cwd: Path) -> tuple[Path, dict[str, Any]]:
    started = time.monotonic()
    logging.info("Selecting Environment from current workspace root=%s", config["workspace_root"])
    environment = select_environment(db, config, args.environment)
    logging.info("Environment selected environment=%s transport=%s revision=%s", environment["code"],
                 environment["target_type"], environment["target_revision"])
    logging.info("Reading business snapshot environment=%s", environment["code"])
    snapshot = inventory(db, environment["id"])
    logging.info("Business snapshot loaded services=%d mounts=%d pipeline_runs=%d artifacts=%d routes=%d",
                 len(snapshot["services"]), len(snapshot["mounts"]), len(snapshot["runs"]), len(snapshot["artifacts"]), len(snapshot["routes"]))
    logging.info("Checking data host and workspace paths transport=%s host=%s", environment["target_type"],
                 environment.get("host") or socket.gethostname())
    host = Host(runtime, connection_for(db, environment))
    paths = host.call("paths", source=environment["workspace_root"], target=normalize_target_argument(args.target, environment))
    logging.info("Data host ready host=%s platform=%s python=%s", paths["host"], paths["platform"], paths["python"])
    source, target = paths["source"], paths["target"]
    if canonical(source) == canonical(target):
        logging.info("Workspace already uses the destination environment=%s target=%s", environment["code"], target)
        return cwd / "data" / "workspace-migration" / "complete.json", {"already_migrated": True, "blockers": []}
    if paths["platform"] != (environment["platform"] if environment["target_type"] == "ssh" else ("windows" if os.name == "nt" else "linux")):
        raise MigrationError("Data-host Python platform differs from the saved Environment")
    logging.info("Assessing migration environment=%s transport=%s source=%s target=%s", environment["code"], environment["target_type"], source, target)
    root = cwd / "data" / "workspace-migration"
    bundle = root / (time.strftime("%Y%m%d-%H%M%S") + "-" + secrets.token_hex(3))
    secure_directory(bundle)
    local = Host(runtime, {"environment": {"target_type": "local"}})
    if paths["host"].casefold() == socket.gethostname().casefold():
        validate_paths(Path(source), Path(target), bundle)
    logging.info("Scanning workspace files and destination capacity")
    files = host.call("assessment", source=source, target=target)
    logging.info("Workspace scan completed files=%d size_mib=%.1f free_gib=%.1f skipped_sockets=%d problems=%d",
                 files["files"], files["bytes"] / 1024 ** 2, files["free_bytes"] / 1024 ** 3,
                 len(files["runtime_sockets_skipped"]), len(files["unreadable_or_linked"]))
    logging.info("Inspecting Docker daemon and containers host=%s", paths["host"])
    docker = host.call("containers")
    codes = {item["code"] for item in snapshot["services"]}
    if any(not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9_.-]{0,127}", code) for code in codes):
        raise MigrationError("A service code is unsuitable for a Compose filename")
    selected = [item for item in docker["containers"] if item["project"] in codes and
                (within(item["working_dir"], source) or any(m["Type"] == "bind" and within(m["Source"], source) for m in item["mounts"]))]
    logging.info("Docker inventory loaded containers=%d selected=%d selected_running=%d", len(docker["containers"]),
                 len(selected), sum(item["running"] for item in selected))
    logging.info("Checking active CI/CD work and workspace conflicts")
    blockers = []
    if files["unreadable_or_linked"]:
        blockers.append("Unreadable files or non-runtime links exist in the copied directories")
    if files["bytes"] > files["free_bytes"]:
        blockers.append("The destination has insufficient free space")
    for container in docker["containers"]:
        if not container["running"]:
            continue
        if container in selected:
            if container["oneoff"]:
                blockers.append("A Compose one-off task is still running: " + container["project"])
            continue
        if container["project"] in codes or any(m["Type"] == "bind" and (within(m["Source"], source) or within(source, m["Source"])) for m in container["mounts"]):
            blockers.append("A running container outside this migration overlaps a selected service or its data: " + container["id"][:12])
    if active_jobs(db) or any(item["status"] not in TERMINAL for item in snapshot["runs"]):
        blockers.append("CI/CD work is still queued or running")
    others = db.query("SELECT id,target_type,platform,host,workspace_root FROM environment WHERE id<>" + sql_value(environment["id"]))
    same_host = host.call("same-host-environments", control_host=socket.gethostname(), environments=others)
    other_roots = [item["workspace_root"] for item in others if item["id"] in same_host and item["workspace_root"]]
    expanded_roots = host.call("expand-home-paths", paths=other_roots)
    if any(within(root, source) or within(source, root) for root in
           (expanded_roots.get(value, value) for value in other_roots)):
        blockers.append("Another Environment on this data host overlaps the source")
    logging.info("Activity and workspace conflict checks completed blockers=%d", len(blockers))
    expanded_paths = host.call("expand-home-paths", paths=[
        *[row[field] for row in snapshot["services"] for field in ("deployment_directory", "runtime_directory")],
        *[row[field] for row in snapshot["mounts"] for field in ("source", "overlay_source")],
        *[row["location"] for row in snapshot["artifacts"]],
    ])
    services = []
    for service in snapshot["services"]:
        actual = [item for item in selected if item["project"] == service["code"]]
        directory = service["runtime_directory"] or service["deployment_directory"]
        if not directory:
            directories = {item["working_dir"] for item in actual}
            directory = next(iter(directories)) if len(directories) == 1 else join_target(join_target(source, "deployment"), service["code"])
        directory = target_path(expanded_paths.get(directory, directory), paths["platform"])
        if not within(directory, join_target(source, "deployment")):
            blockers.append("A service directory lies outside source/deployment: " + service["code"])
        if any(item["running"] and item["working_dir"] and canonical(item["working_dir"]) != canonical(directory) for item in actual):
            blockers.append("Saved and observed directories disagree: " + service["code"])
        components = {}
        for item in actual:
            if item["running"] and not item["oneoff"]:
                if not item["component"]:
                    blockers.append("A running container has no Compose component: " + service["code"])
                components[item["component"]] = components.get(item["component"], 0) + 1
        bootstrap = bool(components) and any(
            any(m["Destination"].startswith(("/var/lib/postgresql", "/var/lib/mysql")) for m in item["mounts"])
            or any(kind in item["image"].split("/")[-1].lower() for kind in ("postgres", "mysql", "mariadb", "redis", "rabbitmq"))
            for item in actual)
        services.append({"id": service["id"], "code": service["code"], "kind": service["application_kind"],
                         "running": bool(components), "components": components, "bootstrap": bootstrap,
                         "source_directory": directory, "target_directory": relocate(directory, source, target)})
    logging.info("Checking service Compose files services=%d", len(services))
    hashes = host.call("files", paths=[join_target(item["source_directory"], "docker-compose.yml") for item in services])
    for index, service in enumerate(services, 1):
        compose = join_target(service["source_directory"], "docker-compose.yml")
        service["compose_hash"] = hashes[compose]
        if service["running"] and not hashes[compose]:
            blockers.append("A running service has no readable Compose file: " + service["code"])
        logging.info("Checking service index=%d/%d service=%s running_containers=%d bootstrap=%s compose=%s", index,
                     len(services), service["code"], sum(service["components"].values()), service["bootstrap"],
                     "present" if hashes[compose] else "missing")
        if hashes[compose]:
            rendered = json.loads(host.call("command", argv=["docker", "compose", "-p", service["code"], "-f", compose, "config", "--format", "json"]))
            relocate_compose(rendered, source, target, host)
    services.sort(key=service_priority)
    logging.info("Service checks completed recreate=%d keep_stopped=%d bootstrap=%s", sum(item["running"] for item in services),
                 sum(not item["running"] for item in services), ",".join(item["code"] for item in services if item["bootstrap"]) or "none")
    logging.info("Planning business path updates and existing configuration overrides")
    operations = build_mutations(snapshot, source, target, time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), expanded_paths)
    for service in services:
        if not service["running"]:
            continue
        op = next((op for op in operations if op["table"] == "service" and op["key"]["id"] == service["id"]), None)
        if op is None:
            row = next(row for row in snapshot["services"] if row["id"] == service["id"])
            op = mutation("service", {"id": service["id"]}, {k: v for k, v in row.items() if k != "application_kind"}, {})
            operations.append(op)
        op["after"].update(runtime_directory=service["target_directory"], runtime_target_revision=environment["target_revision"] + 1)
    overrides = []
    if paths["host"].casefold() == socket.gethostname().casefold():
        settings = {}
        for key, field in (("POMELO_ORBIT_WORKSPACE__ROOT", "workspace_root"), ("POMELO_ORBIT_LOGGING__DEPLOYMENT_ROOT", "deployment_root")):
            replacement = relocate(config[field], source, target)
            if replacement != config[field]:
                settings[key] = replacement
        if settings:
            path = cwd / "overwrite.env"
            before = local.call("read", path=str(path))
            overrides.append({"location": "control", "path": str(path),
                              "before_hash": hashlib.sha256(before.encode()).hexdigest() if before is not None else None,
                              "settings": settings})
    server = db.query("SELECT current_database() AS name,host(inet_server_addr()) AS address,inet_server_port() AS port")[0]
    logging.info("Business updates planned mutations=%d override_files=%d revision=%s->%s", len(operations), len(overrides),
                 environment["target_revision"], environment["target_revision"] + 1)
    plan = {"format": 2, "host": socket.gethostname(), "profile": config["profile"], "source": source, "target": target,
            "data_host": paths["host"], "python": host.python, "connection": host.connection,
            "environment_id": environment["id"], "project_code": environment["code"], "docker_daemon": docker["daemon"],
            "snapshot": snapshot, "services": services, "containers": selected, "filesystem": files,
            "overrides": overrides, "mutations": operations, "database": server,
            "new_revision": environment["target_revision"] + 1, "blockers": blockers}
    logging.info("Saving assessment files directory=%s", bundle)
    write_json(bundle / "plan.json", plan)
    write_private(bundle / "plan.sha256", digest(plan) + "\n")
    write_private(bundle / "update.sql", migration_sql(operations))
    lines = ["# Orbit workspace migration", "", f"Environment: {environment['code']}", f"Transport: {environment['target_type']}",
             f"Source: {source}", f"Destination: {target}", f"Files: {files['files']}; bytes: {files['bytes']}",
             "Stop Orbit API/workers before execution; start Orbit manually after migration.", "",
             "| Service | Action | Destination |", "| --- | --- | --- |"]
    for service in services:
        lines.append(f"| {service['code']} | {'recreate running components' if service['running'] else 'keep stopped'} | {service['target_directory']} |")
    lines += ["", "Blockers:", *["- " + item for item in blockers], "",
              "Copy and SHA-256 verify pipeline/deployment/deployment-logs; retain source; recreate the existing Compose topology.",
              "Published Versions and historical Deployments remain unchanged. Migration does not submit queue tasks or use HTTP login.",
              "Gateway dynamic/certs/acme files are retained with their container paths; no Route business change is submitted."]
    write_private(bundle / "plan.md", "\n".join(lines) + "\n")
    write_json(root / "active.json", {"plan": str(bundle / "plan.json")})
    logging.info("Assessment saved services=%d blockers=%d elapsed_s=%.1f path=%s",
                 len(services), len(blockers), time.monotonic() - started, bundle / "plan.json")
    for item in blockers:
        logging.warning("Migration blocker: %s", item)
    return bundle / "plan.json", plan


def relocate_compose(config: dict[str, Any], source: str, target: str, host: Host) -> None:
    fields = []
    for component in config.get("services", {}).values():
        if isinstance(component.get("build"), dict) and "context" in component["build"]:
            fields.append((component["build"], "context"))
        fields.extend((volume, "source") for volume in component.get("volumes", []) if volume.get("type") == "bind")
    for kind in ("configs", "secrets"):
        fields.extend((item, "file") for item in config.get(kind, {}).values() if "file" in item)
    expanded_paths = host.call("expand-home-paths", paths=[item[key] for item, key in fields])
    for item, key in fields:
        value = item[key]
        expanded = expanded_paths.get(value, value)
        relocated = relocate_data_path(expanded, source, target)
        if relocated != expanded:
            item[key] = relocated


def freeze_compose(plan: dict[str, Any], bundle: Path, host: Host) -> None:
    for service in plan["services"]:
        if not service["compose_hash"]:
            continue
        logging.info("Freezing Compose service=%s", service["code"])
        original = join_target(service["source_directory"], "docker-compose.yml")
        config = json.loads(host.call("command", argv=["docker", "compose", "-p", service["code"], "-f", original, "config", "--format", "json"]))
        relocate_compose(config, plan["source"], plan["target"], host)

        def escape(value: Any) -> Any:
            if isinstance(value, str):
                return value.replace("$", "$$")
            if isinstance(value, list):
                return [escape(item) for item in value]
            if isinstance(value, dict):
                return {key: escape(item) for key, item in value.items()}
            return value
        write_json(bundle / (service["code"] + ".compose.json"), escape(config))


def recreate_service(plan: dict[str, Any], service: dict[str, Any], host: Host) -> None:
    observed = [item for item in host.call("containers")["containers"] if item["project"] == service["code"] and item["running"]]
    if observed and all(canonical(item["working_dir"]) == canonical(service["target_directory"]) and
                        not any(m["Type"] == "bind" and within(m["Source"], plan["source"]) for m in item["mounts"]) for item in observed):
        counts = {}
        for item in observed:
            counts[item["component"]] = counts.get(item["component"], 0) + 1
        if counts == service["components"]:
            logging.info("Service already restored service=%s running_containers=%d", service["code"], len(observed))
            return
    logging.info("Recreating service service=%s components=%s directory=%s", service["code"],
                 ",".join(name + ":" + str(count) for name, count in service["components"].items()), service["target_directory"])
    argv = ["docker", "compose", "-p", service["code"], "--project-directory", service["target_directory"],
            "-f", join_target(service["target_directory"], "docker-compose.yml"), "up", "-d",
            "--force-recreate", "--no-deps", "--no-build", "--pull", "never"]
    for component, count in service["components"].items():
        argv += ["--scale", component + "=" + str(count)]
    argv += list(service["components"])
    host.call("command", argv=argv, timeout=180)


def update_overrides(plan: dict[str, Any], bundle: Path, hosts: dict[str, Host]) -> None:
    for index, item in enumerate(plan["overrides"]):
        host = hosts[item["location"]]
        backup = bundle / ("override-" + str(index) + ".before")
        if not backup.exists():
            before = host.call("read", path=item["path"])
            write_private(backup, before or "")
        content = backup.read_text(encoding="utf-8")
        for key, value in item["settings"].items():
            line = key + "=" + json.dumps(value, ensure_ascii=False)
            expression = re.compile(r"^" + re.escape(key) + r"=.*$", re.MULTILINE)
            content = expression.sub(lambda _: line, content) if expression.search(content) else content.rstrip() + "\n" + line + "\n"
        host.call("write", path=item["path"], content=content)


def await_database(db: Database) -> None:
    deadline = time.monotonic() + 120
    while True:
        try:
            db.query("SELECT 1")
            return
        except MigrationError:
            if time.monotonic() >= deadline:
                raise MigrationError("Database did not become reachable after bootstrap") from None
            time.sleep(2)


def release_active_plan(path: Path) -> None:
    pointer = path.parent.parent / "active.json"
    no_links(pointer)
    if pointer.exists() and canonical(json.loads(pointer.read_text(encoding="utf-8"))["plan"]) == canonical(path):
        pointer.unlink()


def apply_migration(path: Path, runtime: Runtime, config: dict[str, Any], db: Database, cwd: Path) -> None:
    logging.info("Loading migration execution plan path=%s", path)
    no_links(path)
    plan = json.loads(path.read_text(encoding="utf-8"))
    bundle = path.parent
    if plan.get("format") != 2 or plan["blockers"]:
        raise MigrationError("The current assessment is blocked or belongs to the previous script; run apply to reassess")
    if (bundle / "plan.sha256").read_text(encoding="utf-8").strip() != digest(plan):
        raise MigrationError("The saved migration assessment changed")
    if plan["host"] != socket.gethostname() or plan["profile"] != config["profile"]:
        raise MigrationError("The saved migration belongs to a different host or startup profile")
    host = Host(runtime, plan["connection"], plan["python"])
    local = Host(runtime, {"environment": {"target_type": "local"}})
    hosts = {"control": local, "data": host}
    with execution_lock(bundle / "apply.lock"):
        state_path = bundle / "state.json"
        state = json.loads(state_path.read_text(encoding="utf-8")) if state_path.exists() else {"plan_hash": digest(plan), "steps": {}}
        if state["plan_hash"] != digest(plan):
            raise MigrationError("The assessment changed after execution started")
        logging.info("Migration execution ready environment=%s completed_steps=%d source=%s target=%s", plan["project_code"],
                     sum(status == "done" for status in state["steps"].values()), plan["source"], plan["target"])
        if state["steps"].get("verify") == "done":
            if state["steps"].get("release-workspace") != "done":
                host.call("release", source=plan["source"], owner=digest(plan))
                state["steps"]["release-workspace"] = "done"
                write_json(state_path, state)
            release_active_plan(path)
            logging.info("Migration already completed environment=%s target=%s", plan["project_code"], plan["target"])
            logging.info("Start Orbit manually using your usual startup command profile=%s cwd=%s", plan["profile"], cwd)
            return

        def save() -> None:
            write_json(state_path, state)

        def step(name: str, action: Callable[[], Any]) -> None:
            if state["steps"].get(name) == "done":
                logging.info("Migration step skipped step=%s reason=already-completed", name)
                return
            started = time.monotonic()
            logging.info("Migration step started step=%s", name)
            state["steps"][name] = "started"
            save()
            action()
            state["steps"][name] = "done"
            save()
            logging.info("Migration step completed step=%s elapsed_s=%.1f", name, time.monotonic() - started)

        def preflight() -> None:
            if digest(inventory(db, plan["environment_id"])) != digest(plan["snapshot"]) or active_jobs(db):
                raise MigrationError("Business data or queued work changed; run apply to create a fresh assessment")
            current = host.call("containers")
            if current["daemon"] != plan["docker_daemon"]:
                raise MigrationError("The data-host Docker daemon changed")
            codes = {item["code"] for item in plan["services"]}
            relevant = [item for item in current["containers"] if item["project"] in codes]
            if {item["id"]: item for item in relevant} != {item["id"]: item for item in plan["containers"]}:
                raise MigrationError("Selected container state changed; run apply to reassess")
            host.call("assessment", source=plan["source"], target=plan["target"])
            actual = host.call("files", paths=[join_target(item["source_directory"], "docker-compose.yml") for item in plan["services"]])
            if any(actual[join_target(item["source_directory"], "docker-compose.yml")] != item["compose_hash"] for item in plan["services"]):
                raise MigrationError("Saved Compose changed; run apply to reassess")
            for item in plan["overrides"]:
                text = hosts[item["location"]].call("read", path=item["path"])
                actual_hash = hashlib.sha256(text.encode()).hexdigest() if text is not None else None
                if actual_hash != item["before_hash"]:
                    raise MigrationError("An existing override file changed; run apply to reassess")

        step("preflight", preflight)
        step("claim-workspace", lambda: host.call("claim", source=plan["source"], owner=digest(plan)))
        step("backup-database", lambda: db.backup(bundle / "database.dump"))
        step("freeze-compose", lambda: freeze_compose(plan, bundle, host))
        order = {service["code"]: index for index, service in enumerate(plan["services"])}
        for container in sorted(plan["containers"], key=lambda item: order[item["project"]], reverse=True):
            if container["running"]:
                step("stop:" + container["id"], lambda container=container: host.call("command", argv=["docker", "stop", "--time", "60", container["id"]]))
        step("copy-workspace", lambda: host.call("copy", source=plan["source"], target=plan["target"], timeout=3600))
        for service in plan["services"]:
            if service["compose_hash"]:
                step("materialize:" + service["code"], lambda service=service: host.call(
                    "write", path=join_target(service["target_directory"], "docker-compose.yml"),
                    content=(bundle / (service["code"] + ".compose.json")).read_text(encoding="utf-8")))
        for service in plan["services"]:
            if service["bootstrap"]:
                step("bootstrap:" + service["code"], lambda service=service: recreate_service(plan, service, host))
        step("database-ready", lambda: await_database(db))
        write_private(bundle / "update.sql", migration_sql(plan["mutations"]))
        step("update-business-data", lambda: db.execute(bundle / "update.sql"))
        step("update-existing-overrides", lambda: update_overrides(plan, bundle, hosts))
        for service in plan["services"]:
            if service["running"] and not service["bootstrap"]:
                step("redeploy:" + service["code"], lambda service=service: recreate_service(plan, service, host))

        def verify() -> None:
            rows = db.query("SELECT workspace_root,target_revision FROM environment WHERE id=" + sql_value(plan["environment_id"]))
            if (len(rows) != 1 or canonical(rows[0]["workspace_root"]) != canonical(plan["target"])
                    or rows[0]["target_revision"] != plan["new_revision"]):
                raise MigrationError("The saved destination or target revision does not match the migration")
            current = host.call("containers")
            if current["daemon"] != plan["docker_daemon"]:
                raise MigrationError("The Docker daemon changed during migration")
            for service in plan["services"]:
                running = [item for item in current["containers"] if item["project"] == service["code"] and item["running"]]
                counts = {}
                for item in running:
                    counts[item["component"]] = counts.get(item["component"], 0) + 1
                if counts != service["components"]:
                    raise MigrationError("The running component count differs from the original state: " + service["code"])
                if any(m["Type"] == "bind" and within(m["Source"], plan["source"]) for item in running for m in item["mounts"]):
                    raise MigrationError("A running service still mounts the old workspace: " + service["code"])
            write_json(bundle / "verification.json", {"environment_id": plan["environment_id"], "workspace_root": plan["target"],
                       "services_verified": len(plan["services"]), "source_retained": True})
        step("verify", verify)
        step("release-workspace", lambda: host.call("release", source=plan["source"], owner=digest(plan)))
        release_active_plan(path)
        logging.info("Migration completed environment=%s target=%s; original data retained", plan["project_code"], plan["target"])
        logging.info("Start Orbit manually using your usual startup command profile=%s cwd=%s", plan["profile"], cwd)


def pending_plan(root: Path, args: argparse.Namespace, runtime: Runtime) -> Path | None:
    pointer = root / "active.json"
    if not pointer.exists():
        return None
    no_links(pointer)
    path = absolute_path(Path(json.loads(pointer.read_text(encoding="utf-8"))["plan"]))
    if not within(path, root):
        raise MigrationError("The active migration path is outside data/workspace-migration")
    state_path = path.parent / "state.json"
    if not state_path.exists():
        return None
    state = json.loads(state_path.read_text(encoding="utf-8"))
    if state["steps"].get("verify") == "done" and state["steps"].get("release-workspace") == "done":
        release_active_plan(path)
        logging.info("Completed migration archived path=%s", path)
        return None
    if set(state["steps"]) <= {"preflight"}:
        return None  # No workspace claim or business write occurred; reassess automatically.
    plan = json.loads(path.read_text(encoding="utf-8"))
    if args.environment and args.environment not in {plan["environment_id"], plan["project_code"]}:
        raise MigrationError("Another Environment has an active migration in this workspace")
    host = Host(runtime, plan["connection"], python=plan["python"])
    requested_target = host.call("resolve-target", target=normalize_target_argument(args.target, plan["connection"]["environment"]))
    if canonical(requested_target) != canonical(plan["target"]):
        raise MigrationError("The active migration has a different destination")
    return path


def main(argv: list[str] | None = None) -> int:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s", stream=sys.stdout)
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    apply = commands.add_parser("apply", help="Assess by default; --no-dry-run migrates or resumes automatically")
    apply.add_argument("--no-dry-run", action="store_true", help="Execute the migration after assessment")
    apply.add_argument("--environment", help="Optional Environment ID or code when discovery is ambiguous")
    apply.add_argument("--target", required=True, help="Destination directory on the selected host")
    args = parser.parse_args(argv)
    if not args.target.strip():
        parser.error("--target must not be empty")
    os.environ.setdefault("POMELO_ORBIT_APP__ENV", "development")
    cwd = Path(__file__).resolve().parent.parent
    root = cwd / "data" / "workspace-migration"
    try:
        logging.info("Workspace migration started mode=%s profile=%s root=%s", "execute" if args.no_dry_run else "dry-run",
                     os.environ["POMELO_ORBIT_APP__ENV"], cwd)
        if args.no_dry_run:
            logging.warning("Keep Orbit API and workers stopped until migration finishes; start Orbit manually afterwards")
        secure_directory(root)
        with execution_lock(root / "migration.lock"):
            runtime = Runtime(cwd)
            logging.info("Loading existing Orbit startup configuration")
            config = runtime.call("config")
            configure_database(config)
            logging.info("Orbit configuration loaded profile=%s database=postgresql", config["profile"])
            db = Database(cwd, CONFIG_DSN_ENV)
            logging.info("Checking saved migration progress directory=%s", root)
            pending = pending_plan(root, args, runtime)
            if pending is not None:
                logging.info("Saved migration progress found path=%s", pending)
                if args.no_dry_run:
                    apply_migration(pending, runtime, config, db, cwd)
                else:
                    state = json.loads((pending.parent / "state.json").read_text(encoding="utf-8"))
                    if state["steps"].get("verify") == "done":
                        plan = json.loads(pending.read_text(encoding="utf-8"))
                        logging.info("Migration already completed environment=%s target=%s", plan["project_code"], plan["target"])
                        logging.info("Start Orbit manually using your usual startup command profile=%s cwd=%s", plan["profile"], cwd)
                    else:
                        logging.info('Dry run: existing migration progress retained; apply --target "%s" --no-dry-run resumes path=%s', args.target, pending)
                return 0
            path, plan = plan_migration(args, runtime, config, db, cwd)
            if plan.get("already_migrated"):
                logging.info("Start Orbit manually using your usual startup command profile=%s cwd=%s", config["profile"], cwd)
                return 0
            if plan["blockers"]:
                return 2
            if args.no_dry_run:
                apply_migration(path, runtime, config, db, cwd)
            else:
                logging.info("Before execution, stop Orbit API and workers; start Orbit manually after migration")
                logging.info('Dry run completed; apply --target "%s" --no-dry-run executes the migration', args.target)
        return 0
    except (MigrationError, OSError, ValueError, KeyError, RuntimeError) as error:
        if isinstance(error, (MigrationError, RuntimeError)):
            logging.error("Migration stopped: %s", error)
        else:
            logging.error("Migration stopped (%s); inspect the saved progress and filesystem permissions", type(error).__name__)
        return 1
    except KeyboardInterrupt:
        logging.warning("Migration interrupted; repeat apply with the same --target and --no-dry-run to resume saved progress")
        return 130


if __name__ == "__main__":
    raise SystemExit(main())
