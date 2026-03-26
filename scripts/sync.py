#!/usr/bin/env python3
"""
sync.py - 与上游仓库同步文件（不保留 commit 历史）

用法:
  python scripts/sync.py fetch   # 从上游拉取变更到当前目录
  python scripts/sync.py send    # 将当前目录变更发送到上游（黑名单模式）

配置（scripts/.env）:
  SOURCE_REPO=/path/to/upstream/repo
  SOURCE_BRANCH=*  # * 表示当前分支，main 表示验证分支是否匹配
"""

import glob
import logging
import subprocess
import sys
from pathlib import Path

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
logger = logging.getLogger(__name__)

SCRIPT_DIR = Path(__file__).parent.resolve()
PROJECT_DIR = SCRIPT_DIR.parent

# send 排除规则（黑名单）：.gitignore 规则 + 固定排除项
SEND_EXCLUDES = [
    "--filter=:- .gitignore",
    "--exclude=.git",
    "--exclude=scripts",
    "--exclude=.pomelo-pw.yaml",
]

# fetch 排除规则
FETCH_EXCLUDES = [
    "--exclude=.git",
    "--filter=:- .gitignore",
]


def load_env(path: Path) -> dict[str, str]:
    env: dict[str, str] = {}
    for line in path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, _, v = line.partition("=")
        env[k.strip()] = v.strip().strip('"').strip("'")
    return env


def resolve_source_repo(source_repo: str) -> Path:
    source_path = Path(source_repo)
    if source_path.exists():
        return source_path
    matches = glob.glob(source_repo)
    if not matches:
        logger.error(f"上游仓库不存在: source_repo={source_repo}")
        sys.exit(1)
    if len(matches) > 1:
        logger.error(f"SOURCE_REPO 匹配到多个路径，请明确指定: {matches}")
        sys.exit(1)
    resolved = Path(matches[0])
    logger.info(f"上游仓库路径解析: {resolved}")
    return resolved


def check_branch(source_path: Path, source_branch: str) -> None:
    if not source_branch or source_branch == "*":
        logger.info("同步当前分支: branch=*")
        return
    if not (source_path / ".git").exists():
        logger.error(f"需要检查分支但不是 Git 仓库: source_repo={source_path}")
        sys.exit(1)
    result = subprocess.run(
        ["git", "-C", str(source_path), "rev-parse", "--abbrev-ref", "HEAD"],
        capture_output=True,
        text=True,
    )
    current_branch = result.stdout.strip()
    if current_branch != source_branch:
        logger.error(f"分支不匹配: expected={source_branch} actual={current_branch}")
        sys.exit(1)
    logger.info(f"分支验证通过: branch={source_branch}")


def to_unix_path(path: Path) -> str:
    """将路径转换为 rsync 可用的 Unix 风格路径。

    仅处理 Windows 盘符格式（如 D:\\path -> /d/path），
    不支持 UNC 路径（\\\\server\\share）。
    """
    path_str = str(path.resolve())
    if len(path_str) >= 2 and path_str[1] == ":":
        drive = path_str[0].lower()
        rest = path_str[2:].replace("\\", "/")
        return f"/{drive}{rest}"
    return path_str.replace("\\", "/")


def rsync(src: str, dst: str, *, delete: bool, excludes: list[str]) -> None:
    cmd = ["rsync", "-av"] + (["--del"] if delete else []) + excludes + [src, dst]
    logger.info(f"执行命令: {' '.join(cmd)}")
    ret = subprocess.call(cmd)
    if ret != 0:
        logger.error("rsync 失败")
        sys.exit(1)


def cmd_fetch(source_path: Path) -> None:
    """从上游拉取所有变更到当前目录。"""
    logger.info(f"fetch: {source_path} -> {PROJECT_DIR}")
    rsync(
        f"{to_unix_path(source_path)}/",
        f"{to_unix_path(PROJECT_DIR)}/",
        delete=False,
        excludes=FETCH_EXCLUDES,
    )
    logger.info("fetch 完成")


def cmd_send(source_path: Path) -> None:
    """将当前目录变更发送到上游。

    排除项：.gitignore 规则、.git、scripts、.pomelo-pw.yaml。
    目标多余文件会被删除（--del）。
    """
    logger.info(f"send: {PROJECT_DIR} -> {source_path}")
    rsync(
        f"{to_unix_path(PROJECT_DIR)}/",
        f"{to_unix_path(source_path)}/",
        delete=True,
        excludes=SEND_EXCLUDES,
    )
    logger.info("send 完成")


def main() -> None:
    if len(sys.argv) < 2 or sys.argv[1] not in ("fetch", "send"):
        print("用法: python scripts/sync.py <fetch|send>")
        sys.exit(1)

    command = sys.argv[1]

    env_file = SCRIPT_DIR / ".env"
    if not env_file.exists():
        logger.error(f"配置文件不存在: {env_file}")
        sys.exit(1)

    env = load_env(env_file)
    source_repo = env.get("SOURCE_REPO", "").strip()
    source_branch = env.get("SOURCE_BRANCH", "").strip()

    if not source_repo:
        logger.error("SOURCE_REPO 未配置")
        sys.exit(1)

    source_path = resolve_source_repo(source_repo)
    check_branch(source_path, source_branch)

    if command == "fetch":
        cmd_fetch(source_path)
    elif command == "send":
        cmd_send(source_path)


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        sys.exit(1)
