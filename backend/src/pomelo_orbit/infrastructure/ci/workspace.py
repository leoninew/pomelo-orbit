"""工作目录管理"""

import shutil
from pathlib import Path

from pomelo_orbit.infrastructure.config import get_project_root


def _ci_root() -> Path:
    return get_project_root() / "data" / "ci"


def get_workspace_path(project_code: str) -> Path:
    """获取 workspace 路径（固化到项目，挂载到容器 /workspace）"""
    return _ci_root() / project_code / "workspace"


def get_artifacts_path(project_code: str, run_id: str) -> Path:
    """获取 artifacts 路径（按 run 隔离，挂载到容器 /artifacts）"""
    return _ci_root() / project_code / "runs" / run_id / "artifacts"


def get_secrets_path(project_code: str, run_id: str) -> Path:
    """获取 secrets 路径（按 run 隔离，挂载到容器 /run/secrets）"""
    return _ci_root() / project_code / "runs" / run_id / "secrets"


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
    artifacts_path = get_artifacts_path(project_code, run_id)

    workspace_path.mkdir(parents=True, exist_ok=True)
    artifacts_path.mkdir(parents=True, exist_ok=True)

    return workspace_path, artifacts_path


def cleanup_run(project_code: str, run_id: str) -> None:
    """清理单次 run 目录（artifacts、secrets）。
    workspace 目录（data/ci/{code}/workspace）由项目共享，不在此清理。
    """
    run_dir = _ci_root() / project_code / "runs" / run_id
    if run_dir.exists():
        shutil.rmtree(run_dir, ignore_errors=True)


def cleanup_project(project_code: str) -> None:
    """清理项目全部数据目录（删除项目时调用）"""
    project_dir = _ci_root() / project_code
    if project_dir.exists():
        shutil.rmtree(project_dir, ignore_errors=True)
