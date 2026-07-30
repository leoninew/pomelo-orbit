#!/usr/bin/env python3
"""Reconcile Docker Compose proxy settings for hand-maintained CD projects."""

from __future__ import annotations

import argparse
import ipaddress
import json
import logging
import os
import re
import shutil
import subprocess
import sys
import tempfile
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path
from typing import Any, Callable, Sequence

import yaml

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger(__name__)

DEFAULT_ROOT = Path("/opt/pomelo-orbit/data/cd")
DEFAULT_HOST = "host.docker.internal"
DEFAULT_PROBE_URL = "https://www.gstatic.com/generate_204"
TARGET_ENV_KEYS = ("HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY")
LISTENER_PATTERN = re.compile(
    r"(?P<address>(?:\d{1,3}\.){3}\d{1,3}):(?P<port>\d+)\s+.*users:\(\(\"mihomo\",",
)


class ReconcileError(RuntimeError):
    """Raised when a Compose file cannot be safely reconciled."""


@dataclass(frozen=True)
class ProxyEndpoint:
    address: str
    port: int
    network_name: str
    subnet: str


Runner = Callable[..., subprocess.CompletedProcess[str]]


def run_command(
    command: Sequence[str], *, cwd: Path | None = None, check: bool = True,
    runner: Runner = subprocess.run,
) -> subprocess.CompletedProcess[str]:
    try:
        return runner(
            list(command),
            cwd=cwd,
            check=check,
            text=True,
            capture_output=True,
        )
    except subprocess.CalledProcessError as error:
        detail = error.stderr.strip() or error.stdout.strip() or "command failed"
        raise ReconcileError(f"command failed: {' '.join(command)}: {detail}") from error
    except OSError as error:
        raise ReconcileError(f"unable to execute {' '.join(command)}: {error}") from error


def parse_args(argv: Sequence[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Reconcile proxy configuration into CD Docker Compose manifests.",
    )
    parser.add_argument("--root", type=Path, default=DEFAULT_ROOT)
    parser.add_argument("--host", default=DEFAULT_HOST)
    parser.add_argument("--network", help="Only reconcile services using this Docker bridge network.")
    parser.add_argument("--proxy-listen", help="Require this discovered Mihomo listen address.")
    parser.add_argument("--proxy-port", type=int, help="Require this discovered Mihomo listen port.")
    parser.add_argument("--no-proxy", action="append", default=[], help="Additional comma-separated NO_PROXY entries.")
    parser.add_argument("--include", action="append", default=[], metavar="DIRECTORY")
    parser.add_argument("--backup-dir", type=Path)
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--no-deploy", action="store_true")
    parser.add_argument("--continue-on-error", action="store_true")
    parser.add_argument("--compose-bin", default="docker")
    args = parser.parse_args(argv)
    if not args.host or any(character.isspace() for character in args.host):
        parser.error("--host must be a non-empty hostname without whitespace")
    if args.proxy_port is not None and not 1 <= args.proxy_port <= 65535:
        parser.error("--proxy-port must be between 1 and 65535")
    if args.proxy_listen is not None:
        try:
            ipaddress.ip_address(args.proxy_listen)
        except ValueError as error:
            parser.error(f"--proxy-listen must be an IP address: {error}")
    return args


def discover_compose_files(root: Path, includes: Sequence[str]) -> list[Path]:
    root = root.resolve(strict=True)
    if not root.is_dir():
        raise ReconcileError(f"Compose root is not a directory: {root}")
    include_paths = {(root / name).resolve(strict=False) for name in includes}
    for include in include_paths:
        if root not in include.parents and include != root:
            raise ReconcileError(f"include path escapes root: {include}")
    files: list[Path] = []
    for candidate in sorted(root.rglob("docker-compose.yml")):
        if not candidate.is_file() or candidate.is_symlink():
            continue
        resolved = candidate.resolve()
        if root not in resolved.parents:
            raise ReconcileError(f"Compose path escapes root through symlink: {candidate}")
        if include_paths and not any(include in resolved.parents for include in include_paths):
            continue
        files.append(resolved)
    if not files:
        raise ReconcileError(f"no docker-compose.yml files found below {root}")
    return files


def load_compose(path: Path) -> tuple[bytes, dict[str, Any]]:
    raw = path.read_bytes()
    try:
        document = yaml.safe_load(raw)
    except yaml.YAMLError as error:
        raise ReconcileError(f"invalid YAML: {path}: {error}") from error
    if not isinstance(document, dict):
        raise ReconcileError(f"Compose root must be a mapping: {path}")
    services = document.get("services")
    if not isinstance(services, dict):
        raise ReconcileError(f"Compose services must be a mapping: {path}")
    if any(not isinstance(service, dict) for service in services.values()):
        raise ReconcileError(f"Compose services must contain mappings: {path}")
    return raw, document


def service_network_names(service: dict[str, Any]) -> set[str]:
    networks = service.get("networks")
    if networks is None:
        return set()
    if isinstance(networks, list):
        result: set[str] = set()
        for item in networks:
            if isinstance(item, str):
                result.add(item)
            elif isinstance(item, dict) and len(item) == 1:
                result.update(item)
            else:
                raise ReconcileError("service networks list contains unsupported entry")
        return result
    if isinstance(networks, dict):
        return set(networks)
    raise ReconcileError("service networks must be a list or mapping")


def docker_network(network_name: str, compose_bin: str, runner: Runner) -> tuple[str, str]:
    result = run_command(
        [compose_bin, "network", "inspect", network_name], runner=runner,
    )
    try:
        networks = json.loads(result.stdout)
        network = networks[0]
        if network.get("Driver") != "bridge" or network.get("Scope") != "local":
            raise ReconcileError(f"network {network_name} is not a local bridge network")
        config = network.get("IPAM", {}).get("Config", [])
        ipv4 = [entry for entry in config if entry.get("Subnet") and entry.get("Gateway")]
        if len(ipv4) != 1:
            raise ReconcileError(f"network {network_name} must have exactly one IPv4 subnet and gateway")
        subnet = str(ipaddress.ip_network(ipv4[0]["Subnet"], strict=False))
        gateway = str(ipaddress.ip_address(ipv4[0]["Gateway"]))
        return gateway, subnet
    except (IndexError, KeyError, TypeError, ValueError, json.JSONDecodeError) as error:
        if isinstance(error, ReconcileError):
            raise
        raise ReconcileError(f"unable to inspect Docker network {network_name}") from error


def parse_mihomo_listeners(ss_output: str) -> set[tuple[str, int]]:
    listeners: set[tuple[str, int]] = set()
    for line in ss_output.splitlines():
        match = LISTENER_PATTERN.search(line.strip())
        if match:
            address = match.group("address")
            if not ipaddress.ip_address(address).is_loopback:
                listeners.add((address, int(match.group("port"))))
    return listeners


def discover_endpoint(
    *, network_name: str, compose_bin: str, proxy_listen: str | None,
    proxy_port: int | None, runner: Runner,
) -> ProxyEndpoint:
    gateway, subnet = docker_network(network_name, compose_bin, runner)
    result = run_command(["ss", "-H", "-ltnp"], runner=runner)
    candidates = [
        (address, port)
        for address, port in parse_mihomo_listeners(result.stdout)
        if address == gateway
        and (proxy_listen is None or address == proxy_listen)
        and (proxy_port is None or port == proxy_port)
    ]
    if len(candidates) != 1:
        qualifiers = []
        if proxy_listen:
            qualifiers.append(f"listen={proxy_listen}")
        if proxy_port:
            qualifiers.append(f"port={proxy_port}")
        suffix = f" ({', '.join(qualifiers)})" if qualifiers else ""
        raise ReconcileError(
            f"expected exactly one Mihomo listener bound to Docker network {network_name} gateway {gateway}{suffix}; found {len(candidates)}",
        )
    address, port = candidates[0]
    return ProxyEndpoint(address=address, port=port, network_name=network_name, subnet=subnet)


def probe_endpoint(endpoint: ProxyEndpoint, runner: Runner) -> None:
    run_command(
        [
            "curl", "--fail", "--silent", "--show-error",
            "--proxy", f"http://{endpoint.address}:{endpoint.port}",
            "--connect-timeout", "10", "--max-time", "30", "-o", "/dev/null",
            DEFAULT_PROBE_URL,
        ],
        runner=runner,
    )


def split_no_proxy(value: str) -> list[str]:
    return [part.strip() for part in value.split(",") if part.strip()]


def ordered_unique(values: Sequence[str]) -> list[str]:
    result: list[str] = []
    seen: set[str] = set()
    for value in values:
        if value not in seen:
            seen.add(value)
            result.append(value)
    return result


def desired_environment(
    host: str,
    endpoint: ProxyEndpoint,
    internal_service_names: Sequence[str],
    additions: Sequence[str],
    existing_no_proxy: str = "",
) -> dict[str, str]:
    no_proxy = ordered_unique(
        split_no_proxy(existing_no_proxy)
        + ["localhost", "127.0.0.1", "::1", host, endpoint.subnet]
        + list(internal_service_names)
        + [entry for value in additions for entry in split_no_proxy(value)],
    )
    http_proxy = f"http://{host}:{endpoint.port}"
    return {
        "HTTP_PROXY": http_proxy,
        "HTTPS_PROXY": http_proxy,
        "ALL_PROXY": f"socks5h://{host}:{endpoint.port}",
        "NO_PROXY": ",".join(no_proxy),
    }


def environment_as_mapping(value: Any) -> tuple[dict[str, Any], str]:
    if value is None:
        return {}, "mapping"
    if isinstance(value, dict):
        return dict(value), "mapping"
    if isinstance(value, list):
        parsed: dict[str, Any] = {}
        for item in value:
            if not isinstance(item, str) or "=" not in item:
                raise ReconcileError("environment list must contain KEY=value strings")
            key, item_value = item.split("=", 1)
            parsed[key] = item_value
        return parsed, "list"
    raise ReconcileError("environment must be a mapping or KEY=value list")


def reconcile_environment(service: dict[str, Any], values: dict[str, str]) -> bool:
    current, style = environment_as_mapping(service.get("environment"))
    updated = dict(current)
    updated.update(values)
    if style == "list":
        existing_items = service.get("environment") or []
        preserved = [
            item for item in existing_items
            if item.split("=", 1)[0] not in TARGET_ENV_KEYS
        ]
        canonical = preserved + [f"{key}={values[key]}" for key in TARGET_ENV_KEYS]
        if canonical == existing_items:
            return False
        service["environment"] = canonical
        return True
    if updated == current and service.get("environment") is not None:
        return False
    service["environment"] = updated
    return True


def hosts_as_list(value: Any) -> list[str]:
    if value is None:
        return []
    if isinstance(value, list):
        if not all(isinstance(item, str) for item in value):
            raise ReconcileError("extra_hosts list must contain strings")
        return list(value)
    if isinstance(value, dict):
        return [f"{host}:{address}" for host, address in value.items()]
    raise ReconcileError("extra_hosts must be a mapping or list")


def reconcile_hosts(service: dict[str, Any], host: str, address: str) -> bool:
    existing = hosts_as_list(service.get("extra_hosts"))
    target = f"{host}:{address}"
    preserved = [item for item in existing if item.split(":", 1)[0] != host]
    updated = preserved + [target]
    if updated == existing:
        return False
    service["extra_hosts"] = updated
    return True


def reconcile_compose(
    document: dict[str, Any], endpoint: ProxyEndpoint, host: str, additions: Sequence[str],
    network_name: str | None,
) -> list[str]:
    changed: list[str] = []
    services = document["services"]
    internal_service_names = [
        str(name)
        for name, service in services.items()
        if endpoint.network_name in service_network_names(service)
    ]
    for service_name, service in services.items():
        networks = service_network_names(service)
        if network_name is not None and network_name not in networks:
            continue
        if not networks:
            raise ReconcileError(f"service {service_name} has no declared network")
        if network_name is None and endpoint.network_name not in networks:
            continue
        values = desired_environment(
            host, endpoint, internal_service_names, additions,
            str(environment_as_mapping(service.get("environment"))[0].get("NO_PROXY", "")),
        )
        if reconcile_environment(service, values) | reconcile_hosts(service, host, endpoint.address):
            changed.append(str(service_name))
    return changed


def dump_compose(document: dict[str, Any]) -> bytes:
    return yaml.safe_dump(document, sort_keys=False, allow_unicode=True, default_flow_style=False).encode()


def compose_command(compose_bin: str, path: Path, args: Sequence[str]) -> list[str]:
    return [compose_bin, "compose", "--project-directory", str(path.parent), "-f", str(path), *args]


def validate_compose(path: Path, compose_bin: str, runner: Runner) -> None:
    run_command(compose_command(compose_bin, path, ["config", "-q"]), cwd=path.parent, runner=runner)


def deploy_compose(path: Path, compose_bin: str, runner: Runner) -> None:
    run_command(compose_command(compose_bin, path, ["up", "-d"]), cwd=path.parent, runner=runner)
    status = run_command(compose_command(compose_bin, path, ["ps", "--all", "--format", "json"]), cwd=path.parent, runner=runner)
    if status.stdout.strip():
        try:
            entries = json.loads(status.stdout)
        except json.JSONDecodeError as error:
            raise ReconcileError(f"invalid Docker Compose status JSON: {path}") from error
        for entry in entries if isinstance(entries, list) else [entries]:
            state = str(entry.get("State", ""))
            health = str(entry.get("Health", ""))
            if state != "running" or health == "unhealthy":
                raise ReconcileError(f"Compose service failed after deploy: {path}: state={state} health={health}")


def backup_path(root: Path, backup_dir: Path, source: Path) -> Path:
    target = backup_dir / source.relative_to(root)
    target.parent.mkdir(parents=True, exist_ok=True)
    return target


def atomic_write(path: Path, data: bytes) -> None:
    descriptor, temporary_name = tempfile.mkstemp(prefix=f".{path.name}.", dir=path.parent)
    temporary = Path(temporary_name)
    try:
        with os.fdopen(descriptor, "wb") as file:
            file.write(data)
            file.flush()
            os.fsync(file.fileno())
        shutil.copystat(path, temporary)
        os.replace(temporary, path)
        directory_fd = os.open(path.parent, os.O_DIRECTORY)
        try:
            os.fsync(directory_fd)
        finally:
            os.close(directory_fd)
    finally:
        temporary.unlink(missing_ok=True)


def process_file(
    path: Path, root: Path, endpoint: ProxyEndpoint, args: argparse.Namespace,
    backup_dir: Path, runner: Runner,
) -> bool:
    raw, document = load_compose(path)
    changed_services = reconcile_compose(document, endpoint, args.host, args.no_proxy, args.network)
    if not changed_services:
        logger.info("unchanged: %s", path)
        return False
    candidate = dump_compose(document)
    temporary = path.with_name(f".{path.name}.candidate")
    try:
        temporary.write_bytes(candidate)
        validate_compose(temporary, args.compose_bin, runner)
    finally:
        temporary.unlink(missing_ok=True)
    logger.info("validated: %s services=%s", path, ",".join(changed_services))
    if args.dry_run:
        logger.info("dry-run: would update and deploy %s", path)
        return True
    backup = backup_path(root, backup_dir, path)
    if backup.exists():
        raise ReconcileError(f"backup already exists: {backup}")
    backup.write_bytes(raw)
    shutil.copystat(path, backup)
    atomic_write(path, candidate)
    try:
        validate_compose(path, args.compose_bin, runner)
        if not args.no_deploy:
            logger.info("deploying: %s", path)
            deploy_compose(path, args.compose_bin, runner)
    except ReconcileError:
        logger.error("restoring after failure: %s", path)
        atomic_write(path, raw)
        validate_compose(path, args.compose_bin, runner)
        if not args.no_deploy:
            deploy_compose(path, args.compose_bin, runner)
        raise
    logger.info("updated: %s backup=%s", path, backup)
    return True


def main(argv: Sequence[str] | None = None, runner: Runner = subprocess.run) -> int:
    args = parse_args(argv)
    try:
        root = args.root.resolve(strict=True)
        files = discover_compose_files(root, args.include)
        network_names: set[str] = set()
        for path in files:
            _, document = load_compose(path)
            for service in document["services"].values():
                networks = service_network_names(service)
                if args.network:
                    if args.network in networks:
                        network_names.add(args.network)
                else:
                    network_names.update(networks)
        if len(network_names) != 1:
            raise ReconcileError(f"expected exactly one target Docker network; found {sorted(network_names)}; use --network")
        network_name = next(iter(network_names))
        endpoint = discover_endpoint(
            network_name=network_name,
            compose_bin=args.compose_bin,
            proxy_listen=args.proxy_listen,
            proxy_port=args.proxy_port,
            runner=runner,
        )
        probe_endpoint(endpoint, runner)
        logger.info("discovered Mihomo endpoint: network=%s address=%s port=%d subnet=%s", endpoint.network_name, endpoint.address, endpoint.port, endpoint.subnet)
        backup_dir = args.backup_dir or root / ".proxy-reconcile-backups" / datetime.now(UTC).strftime("%Y%m%dT%H%M%SZ")
        changed = 0
        errors = 0
        for path in files:
            try:
                changed += process_file(path, root, endpoint, args, backup_dir, runner)
            except ReconcileError as error:
                errors += 1
                logger.error("failed: %s: %s", path, error)
                if not args.continue_on_error:
                    break
        logger.info("completed: files=%d changed=%d failed=%d", len(files), changed, errors)
        return 1 if errors else 0
    except ReconcileError as error:
        logger.error("reconciliation aborted: %s", error)
        return 1


if __name__ == "__main__":
    sys.exit(main())
