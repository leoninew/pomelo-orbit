"""工作目录管理"""

import shutil
from pathlib import Path

from pomelo_orbit.infrastructure.config import get_project_root
from pomelo_orbit.infrastructure.docker import detect_container_id, get_container_mount_source


def _ci_root() -> Path:
    return get_project_root() / "data" / "ci"


def _get_physical_data_dir() -> Path:
    """获取 data 目录的宿主机物理路径"""
    container_id = detect_container_id()
    if container_id:
        try:
            # 容器模式：获取 /app/data 的宿主机挂载源
            mount_source = get_container_mount_source(container_id, "/app/data")
            return Path(mount_source)
        except RuntimeError as e:
            raise RuntimeError(f"容器模式下无法解析物理数据目录: {e}。请确保容器运行时已挂载 /app/data 目录。") from e
    # 本地模式：直接返回项目 data 目录
    return get_project_root() / "data"


def get_workspace_path(project_code: str) -> Path:
    """获取 workspace 路径（固化到项目，挂载到容器 /workspace）"""
    return _ci_root() / project_code / "workspace"


def get_workspace_physical_path(project_code: str) -> Path:
    """获取 workspace 的宿主机物理路径（用于 Docker 挂载）"""
    return _get_physical_data_dir() / "ci" / project_code / "workspace"


def get_artifacts_path(run_id: str) -> Path:
    """获取 artifacts 路径（按 run 隔离，挂载到容器 /artifacts）"""
    return _ci_root() / "runs" / run_id / "artifacts"


def get_artifacts_physical_path(run_id: str) -> Path:
    """获取 artifacts 的宿主机物理路径（用于 Docker 挂载）"""
    return _get_physical_data_dir() / "ci" / "runs" / run_id / "artifacts"


def get_stage_log_path(run_id: str, stage_run_id: str) -> Path:
    """获取 stage 日志文件路径。

    路径：data/ci/runs/{run_id}/stages/{stage_run_id}.log
    用 stage_run_id 而非 stage_name，避免同名 stage 重试时日志互相覆盖。
    """
    return _ci_root() / "runs" / run_id / "stages" / f"{stage_run_id}.log"


def get_secrets_path(run_id: str) -> Path:
    """获取 secrets 路径（按 run 隔离，挂载到容器 /run/secrets）"""
    return _ci_root() / "runs" / run_id / "secrets"


def create_workspace(project_code: str, run_id: str) -> tuple[Path, Path]:
    """
    创建工作目录

    Args:
        project_code: 项目编码（固化 workspace 路径）
        run_id: Pipeline Run ID（隔离 artifacts）

    Returns:
        (workspace_path, artifacts_path)
    """
    workspace_path = get_workspace_path(project_code)
    artifacts_path = get_artifacts_path(run_id)

    workspace_path.mkdir(parents=True, exist_ok=True)
    artifacts_path.mkdir(parents=True, exist_ok=True)

    return workspace_path, artifacts_path


def cleanup_run_secrets(run_id: str) -> None:
    """清理单次 run 执行产生的临时文件。

    只删除 secrets（SSH 私钥等敏感文件），artifacts 和 stage 日志保留供查阅。
    将来可引入按时间的清理策略统一处理 runs 目录。
    """
    secrets_path = get_secrets_path(run_id)
    if secrets_path.exists():
        shutil.rmtree(secrets_path, ignore_errors=True)


def cleanup_run(run_id: str) -> None:
    """清理单次 run 的全部数据目录（artifacts、secrets、stage 日志）。

    由外部清理策略（如按时间）调用，不在执行完成时自动触发。
    """
    run_dir = _ci_root() / "runs" / run_id
    if run_dir.exists():
        shutil.rmtree(run_dir, ignore_errors=True)


def cleanup_project(project_code: str) -> None:
    """清理项目全部数据目录（删除项目时调用）"""
    project_dir = _ci_root() / project_code
    if project_dir.exists():
        shutil.rmtree(project_dir, ignore_errors=True)
