"""
配置管理模块
使用 dynaconf 管理应用配置
"""

import re
from functools import lru_cache
from pathlib import Path
from typing import Annotated

from dynaconf import Dynaconf
from fastapi import Depends


def get_project_root() -> Path:
    """获取项目根目录（backend/）"""
    for parent in Path(__file__).resolve().parents:
        if (parent / "pyproject.toml").exists():
            return parent
    return Path(__file__).parent.parent.parent.parent


@lru_cache
def get_settings() -> Dynaconf:
    """
    获取配置实例（单例模式）

    配置加载优先级（从低到高）：
      config.defaults.yaml  →  config.yaml  →  .env  →  环境变量

    使用 @lru_cache 实现单例，避免重复初始化。
    """
    base_dir = get_project_root()

    settings_files: list[str] = []

    defaults_file = base_dir / "config.defaults.yaml"
    if defaults_file.exists():
        settings_files.append(str(defaults_file))

    user_config = base_dir / "config.yaml"
    if user_config.exists():
        settings_files.append(str(user_config))

    dotenv = base_dir / ".env"

    return Dynaconf(
        settings_files=settings_files,
        environments=False,
        load_dotenv=True,
        dotenv_path=str(dotenv) if dotenv.exists() else None,
        dotenv_override=False,  # .env 不覆盖已有的系统环境变量
        envvar_prefix="POMELO_ORBIT",
        merge_enabled=True,
        dotenv_encoding="utf-8",
        encoding="utf-8",
    )


# FastAPI 依赖注入类型
Settings = Annotated[Dynaconf, Depends(get_settings)]


def get_cors_config() -> dict:
    """获取CORS配置"""
    settings = get_settings()
    return {
        "allow_origins": settings.cors.origins,
        "allow_credentials": settings.cors.allow_credentials,
        "allow_methods": settings.cors.allow_methods,
        "allow_headers": settings.cors.allow_headers,
    }


def write_env(updates: dict[str, str]) -> None:
    """将 key=value 写入 .env，已有的行原地替换，不存在则追加"""
    env_path = get_project_root() / ".env"
    lines: list[str] = env_path.read_text(encoding="utf-8").splitlines(keepends=True) if env_path.exists() else []

    remaining = dict(updates)
    new_lines: list[str] = []
    for line in lines:
        match = re.match(r"^([A-Z0-9_]+)\s*=", line)
        if match and match.group(1) in remaining:
            key = match.group(1)
            new_lines.append(f"{key}={remaining.pop(key)}\n")
        else:
            new_lines.append(line)

    if remaining:
        # 确保文件末尾有换行再追加
        if new_lines and not new_lines[-1].endswith("\n"):
            new_lines[-1] += "\n"
        for key, value in remaining.items():
            new_lines.append(f"{key}={value}\n")

    env_path.write_text("".join(new_lines), encoding="utf-8")


def delete_env_keys(keys: list[str]) -> None:
    """从 .env 中删除指定的 key 行（重置为默认值）"""
    env_path = get_project_root() / ".env"
    if not env_path.exists():
        return
    key_set = set(keys)
    lines = env_path.read_text(encoding="utf-8").splitlines(keepends=True)
    new_lines: list[str] = []
    for line in lines:
        m = re.match(r"^([A-Z0-9_]+)\s*=", line)
        if m and m.group(1) in key_set:
            continue
        new_lines.append(line)
    env_path.write_text("".join(new_lines), encoding="utf-8")


def get_default_settings() -> Dynaconf:
    """只加载 config.defaults.yaml，用于读取原始默认值（不受 .env 和环境变量影响）"""
    base_dir = get_project_root()
    defaults_file = base_dir / "config.defaults.yaml"
    return Dynaconf(
        settings_files=[str(defaults_file)],
        environments=False,
        load_dotenv=False,
        envvar_prefix="__DISABLED__",  # 禁用环境变量读取，避免 POMELO_ORBIT_* 污染默认值
        encoding="utf-8",
    )


__all__ = [
    "Settings",
    "delete_env_keys",
    "get_cors_config",
    "get_default_settings",
    "get_project_root",
    "get_settings",
    "write_env",
]
