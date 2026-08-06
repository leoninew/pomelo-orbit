#!/usr/bin/env python3
"""Prepare and verify the local BGE-M3 cache used by the RAGFlow TEI component."""

from __future__ import annotations

import argparse
from datetime import datetime
import hashlib
import json
import logging
import os
import re
import shutil
import subprocess
import sys
import tarfile
import tempfile
import time
import uuid
from dataclasses import dataclass, field
from pathlib import Path, PurePosixPath
from typing import Any, Iterable
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import Request, urlopen
from zoneinfo import ZoneInfo


MODEL_ID = "BAAI/bge-m3"
MODEL_REVISION = "5617a9f61b028005a4858fdac845db406aefb181"
TEI_IMAGES = {
    "cpu": "ghcr.io/huggingface/text-embeddings-inference:cpu-1.9.3@sha256:ad950d30878eceb72aaf32024d26fa2b1d04a75304fa0b4776b49aa1941fea07",
    "gpu": "ghcr.io/huggingface/text-embeddings-inference:cuda-1.9.3@sha256:249a0bc87522bfe2f1012b4d194f0225878f47079115ada3aeb0b1ef257b402a",
}
MANIFEST_NAME = ".pomelo-ragflow-tei-manifest.json"
MANIFEST_VERSION = 1
OCI_ACCEPT = ", ".join(
    (
        "application/vnd.oci.image.index.v1+json",
        "application/vnd.oci.image.manifest.v1+json",
        "application/vnd.docker.distribution.manifest.list.v2+json",
        "application/vnd.docker.distribution.manifest.v2+json",
    )
)
LOGGER = logging.getLogger(__name__)
LOG_TIMEZONE = ZoneInfo("Asia/Shanghai")


class PreparationError(RuntimeError):
    """A requested preparation action could not safely complete."""


class LocalTimeFormatter(logging.Formatter):
    def formatTime(self, record: logging.LogRecord, datefmt: str | None = None) -> str:
        value = datetime.fromtimestamp(record.created, LOG_TIMEZONE)
        return value.strftime(datefmt or "%Y-%m-%d %H:%M:%S")


@dataclass
class Result:
    action: str
    profile: str | None = None
    checks: list[dict[str, Any]] = field(default_factory=list)
    values: dict[str, Any] = field(default_factory=dict)

    def add(self, name: str, ok: bool, detail: str, **values: Any) -> bool:
        self.checks.append({"name": name, "ok": ok, "detail": detail, **values})
        return ok

    @property
    def ok(self) -> bool:
        return all(check["ok"] for check in self.checks)

    def document(self) -> dict[str, Any]:
        return {
            "action": self.action,
            "profile": self.profile,
            "ok": self.ok,
            "model_id": MODEL_ID,
            "model_revision": MODEL_REVISION,
            "checks": self.checks,
            **self.values,
        }


def repository_root() -> Path:
    return Path(__file__).resolve().parents[2]


def default_model_dir() -> Path:
    return repository_root() / "data" / "deployment" / "ragflow" / "default" / "tei" / "cache" / "bge-m3"


def default_archive_dir() -> Path:
    return repository_root() / "data" / "backup"


def run(command: list[str], *, cwd: Path | None = None, check: bool = True) -> subprocess.CompletedProcess[str]:
    LOGGER.info("Running command: %s", " ".join(command[:3]))
    try:
        completed = subprocess.run(
            command,
            cwd=cwd,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )
    except FileNotFoundError as error:
        raise PreparationError(f"command is unavailable: {command[0]}") from error
    if check and completed.returncode != 0:
        message = completed.stderr.strip() or completed.stdout.strip() or "unknown command failure"
        raise PreparationError(f"{' '.join(command[:3])} failed: {message}")
    return completed


def command_available(command: list[str]) -> tuple[bool, str]:
    try:
        completed = run(command, check=False)
    except PreparationError as error:
        return False, str(error)
    if completed.returncode != 0:
        return False, (completed.stderr.strip() or completed.stdout.strip() or "command returned non-zero")
    detail = completed.stdout.strip() or "available"
    return True, detail.splitlines()[0]


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as source:
        for block in iter(lambda: source.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def is_nonempty_directory(path: Path) -> bool:
    return path.exists() and any(path.iterdir())


def required_model_files(model_dir: Path) -> list[str]:
    missing: list[str] = []
    if not (model_dir / "config.json").is_file():
        missing.append("config.json")
    if not any((model_dir / name).is_file() for name in ("tokenizer.json", "tokenizer.model", "tokenizer_config.json")):
        missing.append("tokenizer file")
    if not any((model_dir / name).is_file() for name in ("pytorch_model.bin", "model.safetensors")):
        missing.append("model weight")
    if not (model_dir / "onnx").is_dir():
        missing.append("onnx/")
    if not any(path.is_file() for path in (model_dir / "onnx").glob("*.onnx")):
        missing.append("onnx model")
    if not (model_dir / "onnx" / "model.onnx_data").is_file():
        missing.append("onnx/model.onnx_data")
    return missing


def load_manifest(model_dir: Path) -> dict[str, Any] | None:
    path = model_dir / MANIFEST_NAME
    if not path.is_file():
        return None
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        raise PreparationError(f"invalid model manifest: {path}: {error}") from error
    if not isinstance(value, dict) or value.get("version") != MANIFEST_VERSION or not isinstance(value.get("files"), list):
        raise PreparationError(f"invalid model manifest structure: {path}")
    return value


def verify_manifest(model_dir: Path, manifest: dict[str, Any]) -> tuple[bool, str]:
    files = manifest["files"]
    expected: set[str] = set()
    for entry in files:
        if not isinstance(entry, dict):
            return False, "manifest file entry is invalid"
        relative = entry.get("path")
        digest = entry.get("sha256")
        size = entry.get("size")
        if not isinstance(relative, str) or not isinstance(digest, str) or not isinstance(size, int):
            return False, "manifest file entry is incomplete"
        path = model_dir / relative
        if not safe_relative_path(relative) or not path.is_file() or path.stat().st_size != size:
            return False, f"manifest file is missing or changed: {relative}"
        if sha256_file(path) != digest:
            return False, f"manifest hash mismatch: {relative}"
        expected.add(relative)
    actual = {path.relative_to(model_dir).as_posix() for path in iter_model_files(model_dir)}
    if actual != expected:
        return False, "manifest file list does not match restored cache"
    return True, f"verified {len(expected)} files"


def validate_model(model_dir: Path) -> tuple[bool, str]:
    missing = required_model_files(model_dir)
    if missing:
        return False, "missing " + ", ".join(missing)
    manifest = load_manifest(model_dir)
    if manifest is not None:
        return verify_manifest(model_dir, manifest)
    return True, "required model files are present"


def check_lfs(model_dir: Path) -> tuple[bool, str]:
    if not (model_dir / ".git").exists():
        return True, "not a Git checkout; manifest or file checks apply"
    try:
        completed = run(["git", "-C", str(model_dir), "lfs", "fsck"], check=False)
    except PreparationError as error:
        return False, str(error)
    if completed.returncode != 0:
        return False, completed.stderr.strip() or completed.stdout.strip() or "git lfs fsck failed"
    return True, "git lfs fsck passed"


def parse_image(image: str) -> tuple[str, str, str, str]:
    if "@sha256:" not in image:
        raise PreparationError("image must include an immutable sha256 digest")
    reference, digest = image.split("@", 1)
    parts = reference.split("/", 1)
    if len(parts) != 2:
        raise PreparationError("image must include a registry and repository")
    registry, repository = parts
    name, separator, tag = repository.rpartition(":")
    if not separator or "/" in tag:
        raise PreparationError("image must include a tag before the digest")
    return registry, name, tag, digest


def bearer_token(header: str) -> str | None:
    if not header.lower().startswith("bearer "):
        return None
    values = dict(re.findall(r'([a-zA-Z_]+)="([^"]*)"', header))
    realm = values.get("realm")
    if not realm:
        return None
    query = urlencode({key: value for key, value in values.items() if key != "realm"})
    with urlopen(f"{realm}?{query}", timeout=15) as response:
        document = json.loads(response.read().decode("utf-8"))
    token = document.get("token") or document.get("access_token")
    return token if isinstance(token, str) else None


def registry_manifest(image: str) -> tuple[bool, str]:
    LOGGER.info("Checking image manifest: %s", image)
    registry, repository, tag, expected_digest = parse_image(image)
    url = f"https://{registry}/v2/{repository}/manifests/{tag}"
    headers = {"Accept": OCI_ACCEPT, "User-Agent": "pomelo-orbit-ragflow-tei-preflight"}
    request = Request(url, headers=headers)
    try:
        response = urlopen(request, timeout=20)
    except HTTPError as error:
        if error.code != 401:
            return False, f"manifest request returned HTTP {error.code}"
        try:
            token = bearer_token(error.headers.get("WWW-Authenticate", ""))
        except (HTTPError, URLError, OSError, json.JSONDecodeError) as token_error:
            return False, f"registry authentication failed: {token_error}"
        if not token:
            return False, "registry requires unsupported authentication"
        headers["Authorization"] = f"Bearer {token}"
        try:
            response = urlopen(Request(url, headers=headers), timeout=20)
        except (HTTPError, URLError, OSError) as retry_error:
            return False, f"manifest retry failed: {retry_error}"
    except (URLError, OSError) as error:
        return False, f"manifest request failed: {error}"
    with response:
        content_type = response.headers.get("Content-Type", "")
        received_digest = response.headers.get("Docker-Content-Digest", "")
        body = response.read()
    if "json" not in content_type:
        return False, f"manifest response is not OCI JSON: {content_type or 'missing content type'}"
    try:
        json.loads(body.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError):
        return False, "manifest response body is not JSON"
    if received_digest != expected_digest:
        return False, f"manifest digest mismatch: expected {expected_digest}, got {received_digest or 'missing'}"
    return True, f"manifest digest verified: {received_digest}"


def check_gpu_host() -> tuple[bool, str]:
    gpu_ok, gpu_detail = command_available(["nvidia-smi", "--query-gpu=name,driver_version", "--format=csv,noheader"])
    if not gpu_ok:
        return False, f"nvidia-smi unavailable: {gpu_detail}"
    docker_ok, docker_detail = command_available(["docker", "info", "--format", "{{json .Runtimes}}"])
    if not docker_ok:
        return False, f"docker runtime inspection failed: {docker_detail}"
    if "nvidia" not in docker_detail.lower():
        return False, "Docker does not report an nvidia runtime"
    return True, gpu_detail


def check_action(args: argparse.Namespace) -> Result:
    model_dir = Path(args.model_dir).resolve()
    image = args.image or TEI_IMAGES[args.profile]
    LOGGER.info("Starting %s preflight for model cache: %s", args.profile, model_dir)
    result = Result("check", args.profile, values={"model_dir": str(model_dir), "image": image})
    git_ok, git_detail = command_available(["git", "--version"])
    result.add("git", git_ok, git_detail)
    lfs_ok, lfs_detail = command_available(["git", "lfs", "version"])
    result.add("git_lfs", lfs_ok, lfs_detail)
    hf_ok, hf_detail = command_available(["hf", "download", "--help"])
    result.add("hf_cli", hf_ok, hf_detail)
    model_ok, model_detail = validate_model(model_dir)
    result.add("model_cache", model_ok, model_detail)
    if model_ok:
        lfs_cache_ok, lfs_cache_detail = check_lfs(model_dir)
        result.add("model_lfs", lfs_cache_ok, lfs_cache_detail)
    manifest_ok, manifest_detail = registry_manifest(image)
    result.add("image_manifest", manifest_ok, manifest_detail)
    docker_ok, docker_detail = command_available(["docker", "version", "--format", "{{.Server.Version}}"])
    result.add("docker", docker_ok, docker_detail)
    if args.profile == "gpu":
        gpu_ok, gpu_detail = check_gpu_host()
        result.add("gpu_host", gpu_ok, gpu_detail)
    return result


def prepare_cli_action(args: argparse.Namespace) -> Result:
    LOGGER.info("Checking Git LFS and Hugging Face CLI")
    result = Result("prepare-cli")
    lfs_ok, lfs_detail = command_available(["git", "lfs", "version"])
    result.add("git_lfs", lfs_ok, lfs_detail)
    if not lfs_ok:
        result.values["git_lfs_installation"] = "Install Git LFS with the supported package manager for this host, then rerun prepare-cli."
    hf_ok, hf_detail = command_available(["hf", "download", "--help"])
    if not hf_ok and args.install_hf_cli:
        try:
            run([sys.executable, "-m", "pip", "install", "--upgrade", "huggingface_hub"])
            hf_ok, hf_detail = command_available(["hf", "download", "--help"])
        except PreparationError as error:
            hf_ok, hf_detail = False, str(error)
    result.add("hf_cli", hf_ok, hf_detail)
    return result


def ensure_target_for_download(model_dir: Path, resume: bool, source: str) -> bool:
    if not model_dir.exists():
        return False
    if not is_nonempty_directory(model_dir):
        return False
    if source == "git-lfs" and resume and (model_dir / ".git").exists():
        return True
    raise PreparationError("model target is non-empty; use --resume only for an existing Git LFS checkout, or choose an empty target")


def prepare_model_action(args: argparse.Namespace) -> Result:
    model_dir = Path(args.model_dir).resolve()
    result = Result("prepare-model", values={"model_dir": str(model_dir), "source": args.source})
    LOGGER.info("Preparing model cache from %s: %s", args.source, model_dir)
    resume = ensure_target_for_download(model_dir, args.resume, args.source)
    model_dir.parent.mkdir(parents=True, exist_ok=True)
    if args.source == "git-lfs":
        lfs_ok, lfs_detail = command_available(["git", "lfs", "version"])
        if not lfs_ok:
            raise PreparationError(f"Git LFS is required: {lfs_detail}")
        if resume:
            run(["git", "-C", str(model_dir), "lfs", "pull"])
        else:
            run(["git", "lfs", "clone", f"https://huggingface.co/{MODEL_ID}", str(model_dir)])
    else:
        hf_ok, hf_detail = command_available(["hf", "download", "--help"])
        if not hf_ok:
            raise PreparationError(f"Hugging Face CLI is required: {hf_detail}")
        run(["hf", "download", MODEL_ID, "--revision", MODEL_REVISION, "--local-dir", str(model_dir)])
    model_ok, model_detail = validate_model(model_dir)
    result.add("model_cache", model_ok, model_detail)
    lfs_ok, lfs_detail = check_lfs(model_dir)
    result.add("model_lfs", lfs_ok, lfs_detail)
    return result


def stage_model_action(args: argparse.Namespace) -> Result:
    source_dir = Path(args.source_dir).resolve()
    model_dir = Path(args.model_dir).resolve()
    if source_dir == model_dir:
        raise PreparationError("source and target model directories must differ")
    LOGGER.info("Staging verified model cache from %s to %s", source_dir, model_dir)
    source_ok, source_detail = validate_model(source_dir)
    if not source_ok:
        raise PreparationError(f"source model cache is not ready: {source_detail}")
    lfs_ok, lfs_detail = check_lfs(source_dir)
    if not lfs_ok:
        raise PreparationError(lfs_detail)
    if is_nonempty_directory(model_dir):
        raise PreparationError(f"model target is non-empty: {model_dir}")
    if model_dir.exists():
        model_dir.rmdir()
    model_dir.parent.mkdir(parents=True, exist_ok=True)
    temporary = model_dir.parent / f".{model_dir.name}.stage-{uuid.uuid4().hex}"
    try:
        temporary.mkdir()
        copied = 0
        for source in iter_model_files(source_dir):
            destination = temporary / source.relative_to(source_dir)
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, destination)
            copied += 1
        LOGGER.info("Copied %d model files; building SHA-256 manifest", copied)
        manifest = build_manifest(temporary)
        (temporary / MANIFEST_NAME).write_text(json.dumps(manifest, indent=2, sort_keys=True), encoding="utf-8")
        staged_ok, staged_detail = validate_model(temporary)
        if not staged_ok:
            raise PreparationError(f"staged model cache is invalid: {staged_detail}")
        os.replace(temporary, model_dir)
        LOGGER.info("Staged model cache verified: %s", model_dir)
    finally:
        if temporary.exists():
            shutil.rmtree(temporary)
    result = Result("stage-model", values={"source_dir": str(source_dir), "model_dir": str(model_dir)})
    result.add("source_lfs", lfs_ok, lfs_detail)
    result.add("model_cache", True, f"copied and verified {len(manifest['files'])} files")
    return result


def prepare_image_action(args: argparse.Namespace) -> Result:
    image = args.image or TEI_IMAGES[args.profile]
    result = Result("prepare-image", args.profile, values={"image": image})
    LOGGER.info("Preparing %s TEI image: %s", args.profile, image)
    manifest_ok, manifest_detail = registry_manifest(image)
    result.add("image_manifest", manifest_ok, manifest_detail)
    if not manifest_ok:
        return result
    LOGGER.info("Pulling verified TEI image")
    run(["docker", "pull", image])
    inspected = run(["docker", "image", "inspect", image, "--format", "{{json .RepoDigests}}"])
    _, _, _, digest = parse_image(image)
    result.add("image_local", digest in inspected.stdout, "image digest is present locally" if digest in inspected.stdout else "local image digest is missing")
    return result


def iter_model_files(model_dir: Path) -> Iterable[Path]:
    for path in sorted(model_dir.rglob("*")):
        if not path.is_file():
            continue
        relative = path.relative_to(model_dir)
        if relative.parts[0] in {".git", ".cache"} or relative.name == MANIFEST_NAME:
            continue
        if path.is_symlink():
            raise PreparationError(f"model cache contains a symbolic link: {relative.as_posix()}")
        yield path


def safe_relative_path(value: str) -> bool:
    path = PurePosixPath(value)
    return not path.is_absolute() and ".." not in path.parts and value != ""


def build_manifest(model_dir: Path) -> dict[str, Any]:
    files = []
    for path in iter_model_files(model_dir):
        files.append(
            {
                "path": path.relative_to(model_dir).as_posix(),
                "size": path.stat().st_size,
                "sha256": sha256_file(path),
            }
        )
    return {"version": MANIFEST_VERSION, "model_id": MODEL_ID, "revision": MODEL_REVISION, "files": files}


def backup_model_action(args: argparse.Namespace) -> Result:
    model_dir = Path(args.model_dir).resolve()
    model_ok, model_detail = validate_model(model_dir)
    if not model_ok:
        raise PreparationError(f"model cache is not ready: {model_detail}")
    lfs_ok, lfs_detail = check_lfs(model_dir)
    if not lfs_ok:
        raise PreparationError(lfs_detail)
    archive = Path(args.archive).resolve() if args.archive else default_archive_dir() / f"bge-m3-{MODEL_REVISION}.tar.gz"
    if archive.exists() and not args.replace:
        raise PreparationError(f"archive already exists: {archive}; use --replace to explicitly overwrite it")
    LOGGER.info("Creating tar.gz model archive: %s", archive)
    archive.parent.mkdir(parents=True, exist_ok=True)
    manifest = build_manifest(model_dir)
    started = time.monotonic()
    temporary = archive.with_name(f".{archive.name}.{uuid.uuid4().hex}.partial")
    try:
        with tarfile.open(temporary, "w:gz", format=tarfile.PAX_FORMAT) as output:
            for entry in manifest["files"]:
                source = model_dir / entry["path"]
                output.add(source, arcname=f"bge-m3/{entry['path']}", recursive=False)
            encoded_manifest = json.dumps(manifest, indent=2, sort_keys=True).encode("utf-8")
            tar_info = tarfile.TarInfo(f"bge-m3/{MANIFEST_NAME}")
            tar_info.size = len(encoded_manifest)
            tar_info.mtime = 0
            output.addfile(tar_info, fileobj=BytesReader(encoded_manifest))
        os.replace(temporary, archive)
        LOGGER.info("Model archive completed: %s", archive)
    finally:
        if temporary.exists():
            temporary.unlink()
    result = Result("backup-model", values={"model_dir": str(model_dir), "archive": str(archive)})
    result.add("model_lfs", lfs_ok, lfs_detail)
    result.add("archive", True, "tar.gz archive created", bytes=archive.stat().st_size, elapsed_seconds=round(time.monotonic() - started, 3), files=len(manifest["files"]))
    return result


class BytesReader:
    def __init__(self, value: bytes) -> None:
        self.value = value
        self.position = 0

    def read(self, size: int = -1) -> bytes:
        if size < 0:
            size = len(self.value) - self.position
        result = self.value[self.position : self.position + size]
        self.position += len(result)
        return result


def restore_model_action(args: argparse.Namespace) -> Result:
    archive = Path(args.archive).resolve()
    model_dir = Path(args.model_dir).resolve()
    if not archive.is_file():
        raise PreparationError(f"archive does not exist: {archive}")
    LOGGER.info("Restoring tar.gz model archive from %s to %s", archive, model_dir)
    if is_nonempty_directory(model_dir):
        raise PreparationError(f"restore target is non-empty: {model_dir}")
    if model_dir.exists():
        model_dir.rmdir()
    model_dir.parent.mkdir(parents=True, exist_ok=True)
    temporary = model_dir.parent / f".{model_dir.name}.restore-{uuid.uuid4().hex}"
    try:
        with tarfile.open(archive, "r:gz") as source:
            members = source.getmembers()
            for member in members:
                member_path = PurePosixPath(member.name)
                if not member.name.startswith("bge-m3/") or not safe_relative_path(member_path.as_posix()) or member.issym() or member.islnk() or member.isdev():
                    raise PreparationError(f"archive contains an unsafe member: {member.name}")
            for member in members:
                source.extract(member, temporary, filter="data")
        extracted = temporary / "bge-m3"
        manifest = load_manifest(extracted)
        if manifest is None:
            raise PreparationError("archive does not contain a model manifest")
        manifest_ok, manifest_detail = verify_manifest(extracted, manifest)
        if not manifest_ok:
            raise PreparationError(manifest_detail)
        model_ok, model_detail = validate_model(extracted)
        if not model_ok:
            raise PreparationError(f"restored model is incomplete: {model_detail}")
        os.replace(extracted, model_dir)
        LOGGER.info("Restored model cache verified: %s", model_dir)
    finally:
        if temporary.exists():
            shutil.rmtree(temporary)
    result = Result("restore-model", values={"archive": str(archive), "model_dir": str(model_dir)})
    result.add("restored_model", True, "tar.gz manifest and model files verified")
    return result


def prepare_action(args: argparse.Namespace) -> Result:
    cli_result = prepare_cli_action(args)
    if not cli_result.ok:
        return cli_result
    model_result = prepare_model_action(args)
    if not model_result.ok:
        return model_result
    image_result = prepare_image_action(args)
    if not image_result.ok:
        return image_result
    return check_action(args)


def add_common_arguments(parser: argparse.ArgumentParser, *, profile: bool = False) -> None:
    parser.add_argument("--model-dir", default=str(default_model_dir()))
    if profile:
        parser.add_argument("--profile", choices=sorted(TEI_IMAGES), default="cpu")
        parser.add_argument("--image", help="Full immutable image reference; use this to probe a mirror before pulling it")


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--json", action="store_true", help="Emit a machine-readable result")
    subparsers = parser.add_subparsers(dest="command", required=True)

    check_parser = subparsers.add_parser("check", help="Read-only tool, cache, registry, and host checks")
    add_common_arguments(check_parser, profile=True)

    cli_parser = subparsers.add_parser("prepare-cli", help="Check CLIs and optionally install Hugging Face CLI")
    cli_parser.add_argument("--install-hf-cli", action="store_true", help="Explicitly install or upgrade huggingface_hub with this Python")

    model_parser = subparsers.add_parser("prepare-model", help="Download or resume a fixed-revision model cache")
    add_common_arguments(model_parser)
    model_parser.add_argument("--source", choices=("git-lfs", "hf-cli"), default="git-lfs")
    model_parser.add_argument("--resume", action="store_true", help="Resume an existing Git LFS checkout only")

    stage_parser = subparsers.add_parser("stage-model", help="Copy a verified local model cache into an empty RAGFlow cache")
    add_common_arguments(stage_parser)
    stage_parser.add_argument("--source-dir", required=True, help="Existing verified Git LFS or manifest-backed model cache")

    image_parser = subparsers.add_parser("prepare-image", help="Verify and pull the profile image")
    add_common_arguments(image_parser, profile=True)

    backup_parser = subparsers.add_parser("backup-model", help="Create a tar.gz model archive with hashes")
    add_common_arguments(backup_parser)
    backup_parser.add_argument("--archive", help="Output tar.gz path")
    backup_parser.add_argument("--replace", action="store_true", help="Explicitly replace an existing archive")

    restore_parser = subparsers.add_parser("restore-model", help="Restore a tar.gz model archive to an empty target")
    add_common_arguments(restore_parser)
    restore_parser.add_argument("--archive", required=True, help="Input tar.gz archive")

    prepare_parser = subparsers.add_parser("prepare", help="Prepare CLI, model, image, then rerun check")
    add_common_arguments(prepare_parser, profile=True)
    prepare_parser.add_argument("--source", choices=("git-lfs", "hf-cli"), default="git-lfs")
    prepare_parser.add_argument("--resume", action="store_true", help="Resume an existing Git LFS checkout only")
    prepare_parser.add_argument("--install-hf-cli", action="store_true", help="Explicitly install Hugging Face CLI")
    return parser.parse_args(argv)


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    handler = logging.StreamHandler()
    handler.setFormatter(LocalTimeFormatter("%(asctime)s %(levelname)s %(message)s"))
    logging.basicConfig(level=logging.INFO, handlers=[handler])
    LOGGER.info("Starting action: %s", args.command)
    try:
        if args.command == "check":
            result = check_action(args)
        elif args.command == "prepare-cli":
            result = prepare_cli_action(args)
        elif args.command == "prepare-model":
            result = prepare_model_action(args)
        elif args.command == "stage-model":
            result = stage_model_action(args)
        elif args.command == "prepare-image":
            result = prepare_image_action(args)
        elif args.command == "backup-model":
            result = backup_model_action(args)
        elif args.command == "restore-model":
            result = restore_model_action(args)
        else:
            result = prepare_action(args)
    except PreparationError as error:
        result = Result(args.command, getattr(args, "profile", None), values={"error": str(error)})
        result.add("action", False, str(error))
    document = result.document()
    if args.json:
        print(json.dumps(document, indent=2, sort_keys=True))
    else:
        print(f"{document['action']}: {'ready' if document['ok'] else 'blocked'}")
        for check in document["checks"]:
            print(f"- {'ok' if check['ok'] else 'blocked'} {check['name']}: {check['detail']}")
        for key, value in document.items():
            if key not in {"action", "profile", "ok", "model_id", "model_revision", "checks"}:
                print(f"- {key}: {value}")
    return 0 if result.ok else 2


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
