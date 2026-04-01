"""工作目录管理"""

import shutil
from pathlib import Path

from pomelo_orbit.infrastructure.config import get_project_root


def get_workspace_path(run_id: str) -> Path:
    """获取 workspace 路径（挂载到容器 /workspace）"""
    return get_project_root() / "data" / "ci" / "runs" / run_id / "workspace"


def get_artifacts_path(run_id: str) -> Path:
    """获取 artifacts 路径（挂载到容器 /artifacts）"""
    return get_project_root() / "data" / "ci" / "runs" / run_id / "artifacts"


def get_secrets_path(run_id: str) -> Path:
    """获取 secrets 路径（挂载到容器 /run/secrets，不污染 workspace）"""
    return get_project_root() / "data" / "ci" / "runs" / run_id / "secrets"


def create_workspace(run_id: str) -> tuple[Path, Path]:
    """
    创建工作目录

    Args:
        run_id: Pipeline Run ID

    Returns:
        (workspace_path, artifacts_path)
    """
    workspace_path = get_workspace_path(run_id)
    artifacts_path = get_artifacts_path(run_id)

    workspace_path.mkdir(parents=True, exist_ok=True)
    artifacts_path.mkdir(parents=True, exist_ok=True)

    return workspace_path, artifacts_path


def cleanup_workspace(run_id: str) -> None:
    """清理 run 目录（workspace、artifacts、secrets）"""
    run_dir = get_project_root() / "data" / "ci" / "runs" / run_id
    if run_dir.exists():
        shutil.rmtree(run_dir, ignore_errors=True)
