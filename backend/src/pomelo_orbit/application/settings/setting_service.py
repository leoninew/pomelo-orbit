"""Setting application service - handles runtime config read/write."""

import logging
import re
from pathlib import Path

from dynaconf import Dynaconf

from pomelo_orbit.infrastructure.config import delete_env_keys, get_default_settings, get_project_root, write_env

logger = logging.getLogger(__name__)

# 需要在 Settings 页面展示的 key（即使 .env 里没有也要展示，带默认值）
# 新增配置项时在此追加即可；.env 里有但不在此列表的 key 也会展示
_CONFIG_KEYS: list[str] = [
    "traefik__domain_suffix",
    "traefik__api_url",
    "cert__letsencrypt__enabled",
    "cert__letsencrypt__email",
    "cert__letsencrypt__challenge",
    "jwt__secret_key",
    "jwt__expire_minutes",
]

# 脱敏字段
_SECRET_KEYS: frozenset[str] = frozenset({"jwt__secret_key"})

# env var 前缀
_ENV_PREFIX = "POMELO_ORBIT_"


def _to_settings_key(field: str) -> str:
    """traefik__domain_suffix -> traefik.domain_suffix"""
    return field.replace("__", ".")


def _to_env_key(field: str) -> str:
    """traefik__domain_suffix -> POMELO_ORBIT_TRAEFIK__DOMAIN_SUFFIX"""
    return f"{_ENV_PREFIX}{field.upper()}"


def _from_env_key(env_key: str) -> str | None:
    """POMELO_ORBIT_TRAEFIK__DOMAIN_SUFFIX -> traefik__domain_suffix，非本项目前缀返回 None"""
    if not env_key.startswith(_ENV_PREFIX):
        return None
    return env_key[len(_ENV_PREFIX) :].lower()


def _get_defaults() -> dict[str, object]:
    """从 config.defaults.yaml 读取默认值（进程内缓存，只读一次）"""
    d = get_default_settings()
    return {key: d.get(_to_settings_key(key), "") for key in _CONFIG_KEYS}


def _read_env_file() -> dict[str, str]:
    """直接解析 .env 文件，返回 {ENV_KEY: value}（绕过 Dynaconf 缓存）"""
    env_path: Path = get_project_root() / ".env"
    if not env_path.exists():
        return {}
    result: dict[str, str] = {}
    for line in env_path.read_text(encoding="utf-8").splitlines():
        m = re.match(r"^([A-Z0-9_]+)\s*=\s*(.*)", line)
        if m:
            result[m.group(1)] = m.group(2).strip()
    return result


def _build_items(env_map: dict[str, str]) -> list[dict]:
    """
    合并 .env 中的 POMELO_ORBIT_* key 与 _CONFIG_KEYS，构建配置项列表。
    顺序：_CONFIG_KEYS 优先，其余按字母序追加。
    """
    defaults = _get_defaults()

    # 从 env_map 中提取属于本项目的 key
    env_field_keys: set[str] = set()
    for env_key in env_map:
        field = _from_env_key(env_key)
        if field is not None:
            env_field_keys.add(field)

    # 合并顺序：_CONFIG_KEYS 先，剩余 env key 按字母序
    config_keys_set = set(_CONFIG_KEYS)
    ordered_keys: list[str] = [*_CONFIG_KEYS, *sorted(env_field_keys - config_keys_set)]

    items = []
    for key in ordered_keys:
        env_key = _to_env_key(key)
        is_overridden = env_key in env_map
        raw_value: object = env_map[env_key] if is_overridden else defaults.get(key, "")

        # bool 类型转换（.env 里存的是字符串）
        default_val = defaults.get(key, "")
        if isinstance(default_val, bool) or (isinstance(raw_value, str) and raw_value.lower() in ("true", "false")):
            raw_value = str(raw_value).lower() == "true"

        # 脱敏
        display_value = "****" if key in _SECRET_KEYS and raw_value else raw_value

        items.append(
            {
                "key": key,
                "value": display_value,
                "default": default_val,
                "is_overridden": is_overridden,
            }
        )
    return items


class SettingService:
    def __init__(self, settings: Dynaconf):
        self.settings = settings  # 保留，供其他业务逻辑使用

    def get_config(self) -> dict:
        env_map = _read_env_file()
        return {"items": _build_items(env_map)}

    def update_config(self, key: str, value: object) -> dict:
        """写入单个 key 到 .env，返回最新配置"""
        env_key = _to_env_key(key)
        str_value = str(value).lower() if isinstance(value, bool) else str(value)
        write_env({env_key: str_value})
        logger.info(f"Config updated: {env_key}={str_value}")
        return {"items": _build_items(_read_env_file())}

    def reset_config(self, keys: list[str]) -> dict:
        """从 .env 删除指定字段，返回最新配置"""
        delete_env_keys([_to_env_key(k) for k in keys])
        logger.info(f"Config reset: keys={keys}")
        return {"items": _build_items(_read_env_file())}
