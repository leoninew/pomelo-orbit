"""
配置加载优先级单元测试

验证 Dynaconf 加载规则：
  config.defaults.yaml → config.yaml → .env → 环境变量
后面的优先级高于前面的。
"""

import os
from pathlib import Path
from unittest.mock import patch

from dynaconf import Dynaconf

from pomelo_orbit.application.settings.setting_service import SettingService
from pomelo_orbit.infrastructure.config import get_default_settings


def make_settings(base_dir: Path, extra_env: dict[str, str] | None = None) -> Dynaconf:
    """
    按照与 get_settings() 相同的参数构造 Dynaconf 实例，
    但使用指定的临时目录，方便测试隔离。

    dotenv_path 明确指向 tmp_path/.env，防止 dynaconf 向上搜索找到
    真实的 backend/.env（Windows 下可能有编码问题）。
    """
    settings_files: list[str] = []

    defaults_file = base_dir / "config.defaults.yaml"
    if defaults_file.exists():
        settings_files.append(str(defaults_file))

    user_config = base_dir / "config.yaml"
    if user_config.exists():
        settings_files.append(str(user_config))

    dotenv = base_dir / ".env"

    env_before = os.environ.copy()
    # 先注入 extra_env，load_dotenv(override=False) 不会覆盖已有 key，
    # 从而保证 extra_env 优先级高于 .env
    if extra_env:
        os.environ.update(extra_env)

    try:
        s = Dynaconf(
            settings_files=settings_files,
            environments=False,
            load_dotenv=True,
            dotenv_path=str(dotenv) if dotenv.exists() else str(base_dir / ".env"),  # 明确路径，禁止向上搜索
            dotenv_override=False,
            envvar_prefix="POMELO_ORBIT",
            merge_enabled=True,
            dotenv_encoding="utf-8",
            encoding="utf-8",
        )
        s.as_dict()  # 强制触发加载，在环境变量还原前完成
        return s
    finally:
        os.environ.clear()
        os.environ.update(env_before)


class TestDefaultsLoading:
    """config.defaults.yaml 基础加载"""

    def test_loads_defaults(self, tmp_path: Path):
        (tmp_path / "config.defaults.yaml").write_text("app:\n  name: TestApp\n  version: '1.0'\n", encoding="utf-8")
        s = make_settings(tmp_path)
        assert s.app.name == "TestApp"
        assert s.app.version == "1.0"

    def test_missing_defaults_file_does_not_raise(self, tmp_path: Path):
        """没有 defaults 文件时不应报错"""
        s = make_settings(tmp_path)
        assert s is not None


class TestUserConfigOverridesDefaults:
    """config.yaml 覆盖 config.defaults.yaml"""

    def test_overrides_value(self, tmp_path: Path):
        (tmp_path / "config.defaults.yaml").write_text("app:\n  name: DefaultName\n  debug: false\n", encoding="utf-8")
        (tmp_path / "config.yaml").write_text("app:\n  name: OverriddenName\n", encoding="utf-8")
        s = make_settings(tmp_path)
        assert s.app.name == "OverriddenName"

    def test_keeps_unoverridden_defaults(self, tmp_path: Path):
        """config.yaml 只覆盖指定的 key，其余 defaults 保留"""
        (tmp_path / "config.defaults.yaml").write_text("app:\n  name: DefaultName\n  debug: false\n", encoding="utf-8")
        (tmp_path / "config.yaml").write_text("app:\n  name: OverriddenName\n", encoding="utf-8")
        s = make_settings(tmp_path)
        assert s.app.debug is False

    def test_no_user_config_uses_defaults(self, tmp_path: Path):
        (tmp_path / "config.defaults.yaml").write_text("app:\n  name: DefaultName\n", encoding="utf-8")
        s = make_settings(tmp_path)
        assert s.app.name == "DefaultName"


class TestDotenvOverridesConfig:
    """.env 覆盖 yaml 配置文件"""

    def test_overrides_yaml_value(self, tmp_path: Path):
        (tmp_path / "config.defaults.yaml").write_text("app:\n  debug: false\n", encoding="utf-8")
        (tmp_path / ".env").write_text("POMELO_ORBIT_APP__DEBUG=true\n", encoding="utf-8")
        s = make_settings(tmp_path)
        assert s.app.debug is True

    def test_nested_key_via_double_underscore(self, tmp_path: Path):
        """双下划线表示嵌套 key"""
        (tmp_path / "config.defaults.yaml").write_text("app:\n  name: old_name\n  debug: false\n", encoding="utf-8")
        (tmp_path / ".env").write_text("POMELO_ORBIT_APP__NAME=from_dotenv\n", encoding="utf-8")
        s = make_settings(tmp_path)
        assert s.app.name == "from_dotenv"

    def test_keeps_unoverridden_yaml_values(self, tmp_path: Path):
        (tmp_path / "config.defaults.yaml").write_text("app:\n  name: old_name\n  debug: false\n", encoding="utf-8")
        (tmp_path / ".env").write_text("POMELO_ORBIT_APP__NAME=from_dotenv\n", encoding="utf-8")
        s = make_settings(tmp_path)
        assert s.app.debug is False

    def test_missing_dotenv_falls_back_to_yaml(self, tmp_path: Path):
        (tmp_path / "config.defaults.yaml").write_text("app:\n  name: from_yaml\n", encoding="utf-8")
        s = make_settings(tmp_path)
        assert s.app.name == "from_yaml"


class TestEnvVarOverridesDotenv:
    """环境变量优先级最高，覆盖 .env"""

    def test_overrides_dotenv_value(self, tmp_path: Path):
        (tmp_path / ".env").write_text("POMELO_ORBIT_APP__NAME=from_dotenv\n", encoding="utf-8")
        s = make_settings(tmp_path, extra_env={"POMELO_ORBIT_APP__NAME": "from_env_var"})
        assert s.app.name == "from_env_var"

    def test_overrides_yaml_value(self, tmp_path: Path):
        (tmp_path / "config.defaults.yaml").write_text("app:\n  name: from_yaml\n", encoding="utf-8")
        s = make_settings(tmp_path, extra_env={"POMELO_ORBIT_APP__NAME": "from_env_var"})
        assert s.app.name == "from_env_var"

    def test_dotenv_override_false_respects_existing_env_var(self, tmp_path: Path):
        """dotenv_override=False：系统环境变量已存在时，.env 不覆盖它"""
        (tmp_path / ".env").write_text("POMELO_ORBIT_APP__NAME=from_dotenv\n", encoding="utf-8")
        # extra_env 先写入，模拟系统已有环境变量
        s = make_settings(tmp_path, extra_env={"POMELO_ORBIT_APP__NAME": "pre_existing"})
        assert s.app.name == "pre_existing"


class TestFullPriorityChain:
    """完整优先级链：defaults < config.yaml < .env < 环境变量"""

    def test_full_chain(self, tmp_path: Path):
        (tmp_path / "config.defaults.yaml").write_text("app:\n  name: from_defaults\n", encoding="utf-8")
        (tmp_path / "config.yaml").write_text("app:\n  name: from_config\n", encoding="utf-8")
        (tmp_path / ".env").write_text("POMELO_ORBIT_APP__NAME=from_dotenv\n", encoding="utf-8")
        s = make_settings(tmp_path, extra_env={"POMELO_ORBIT_APP__NAME": "from_env_var"})
        assert s.app.name == "from_env_var"

    def test_chain_stops_at_dotenv_when_no_env_var(self, tmp_path: Path):
        (tmp_path / "config.defaults.yaml").write_text("app:\n  name: from_defaults\n", encoding="utf-8")
        (tmp_path / "config.yaml").write_text("app:\n  name: from_config\n", encoding="utf-8")
        (tmp_path / ".env").write_text("POMELO_ORBIT_APP__NAME=from_dotenv\n", encoding="utf-8")
        s = make_settings(tmp_path)
        assert s.app.name == "from_dotenv"

    def test_chain_stops_at_config_yaml_when_no_dotenv(self, tmp_path: Path):
        (tmp_path / "config.defaults.yaml").write_text("app:\n  name: from_defaults\n", encoding="utf-8")
        (tmp_path / "config.yaml").write_text("app:\n  name: from_config\n", encoding="utf-8")
        s = make_settings(tmp_path)
        assert s.app.name == "from_config"

    def test_chain_stops_at_defaults_when_only_defaults(self, tmp_path: Path):
        (tmp_path / "config.defaults.yaml").write_text("app:\n  name: from_defaults\n", encoding="utf-8")
        s = make_settings(tmp_path)
        assert s.app.name == "from_defaults"


# ── helpers for SettingService tests ──────────────────────────────────────

_SETTING_DEFAULTS_YAML = """\
traefik:
  domain_suffix: "lvh.me"
  api_url: "http://traefik:8080"
cert:
  letsencrypt:
    enabled: false
    email: ""
    challenge: "http"
jwt:
  secret_key: ""
  expire_minutes: 10080
"""


def _make_setting_service(base_dir: Path):
    """构造指向临时目录的 SettingService，隔离真实 .env"""
    settings = Dynaconf(
        settings_files=[str(base_dir / "config.defaults.yaml")],
        environments=False,
        load_dotenv=False,
        envvar_prefix="__DISABLED__",
        encoding="utf-8",
    )
    svc = SettingService(settings)
    # write_env / delete_env_keys 定义在 config 模块，需同时 mock 那边的 get_project_root
    p1 = patch("pomelo_orbit.application.settings.setting_service.get_project_root", return_value=base_dir)
    p2 = patch("pomelo_orbit.infrastructure.config.get_project_root", return_value=base_dir)
    p1.start()
    p2.start()

    class _MultiPatcher:
        def stop(self):
            p1.stop()
            p2.stop()

    return svc, _MultiPatcher()


# ── get_default_settings 隔离测试 ──────────────────────────────────────────


class TestGetDefaultSettingsIsolation:
    """验证 get_default_settings() 不受环境变量和 .env 文件影响"""

    def test_env_var_does_not_pollute_defaults(self, tmp_path: Path):
        """POMELO_ORBIT_* 环境变量不应影响使用 __DISABLED__ 前缀的 Dynaconf 实例"""
        (tmp_path / "config.defaults.yaml").write_text("traefik:\n  domain_suffix: 'lvh.me'\n", encoding="utf-8")
        env_before = os.environ.copy()
        os.environ["POMELO_ORBIT_TRAEFIK__DOMAIN_SUFFIX"] = "injected.example.com"
        try:
            d = Dynaconf(
                settings_files=[str(tmp_path / "config.defaults.yaml")],
                environments=False,
                load_dotenv=False,
                envvar_prefix="__DISABLED__",
                encoding="utf-8",
            )
            assert d.get("traefik.domain_suffix") == "lvh.me"
        finally:
            os.environ.clear()
            os.environ.update(env_before)

    def test_dotenv_file_does_not_pollute_defaults(self, tmp_path: Path):
        """.env 文件存在时，load_dotenv=False 确保不被读取"""
        (tmp_path / "config.defaults.yaml").write_text("jwt:\n  expire_minutes: 10080\n", encoding="utf-8")
        (tmp_path / ".env").write_text("POMELO_ORBIT_JWT__EXPIRE_MINUTES=999\n", encoding="utf-8")
        d = Dynaconf(
            settings_files=[str(tmp_path / "config.defaults.yaml")],
            environments=False,
            load_dotenv=False,
            envvar_prefix="__DISABLED__",
            encoding="utf-8",
        )
        assert d.get("jwt.expire_minutes") == 10080

    def test_real_get_default_settings_ignores_env_var(self):
        """真实 get_default_settings() 在有 POMELO_ORBIT_* 环境变量时仍返回 yaml 默认值"""
        env_before = os.environ.copy()
        os.environ["POMELO_ORBIT_TRAEFIK__DOMAIN_SUFFIX"] = "should-not-appear.com"
        try:
            d = get_default_settings()
            assert d.get("traefik.domain_suffix") == "lvh.me"
        finally:
            os.environ.clear()
            os.environ.update(env_before)


# ── _build_items value/default 分离 ────────────────────────────────────────


class TestBuildItemsValueDefaultSeparation:
    """value 和 default 必须独立，修改 .env 只影响 value"""

    def test_default_unchanged_after_update(self, tmp_path: Path):
        """写入 .env 后，default 仍是 yaml 里的原始值"""
        (tmp_path / "config.defaults.yaml").write_text(_SETTING_DEFAULTS_YAML, encoding="utf-8")
        svc, patcher = _make_setting_service(tmp_path)
        try:
            svc.update_config("traefik__domain_suffix", "new.example.com")
            result = svc.get_config()
        finally:
            patcher.stop()

        item = next(i for i in result["items"] if i["key"] == "traefik__domain_suffix")
        assert item["value"] == "new.example.com"
        assert item["default"] == "lvh.me"

    def test_value_equals_default_when_not_overridden(self, tmp_path: Path):
        """未覆盖时 value 显示默认值，is_overridden 为 False"""
        (tmp_path / "config.defaults.yaml").write_text(_SETTING_DEFAULTS_YAML, encoding="utf-8")
        svc, patcher = _make_setting_service(tmp_path)
        try:
            result = svc.get_config()
        finally:
            patcher.stop()

        item = next(i for i in result["items"] if i["key"] == "traefik__domain_suffix")
        assert item["value"] == item["default"] == "lvh.me"
        assert item["is_overridden"] is False

    def test_default_unchanged_after_reset(self, tmp_path: Path):
        """reset 后 value 回到默认值，default 始终不变"""
        (tmp_path / "config.defaults.yaml").write_text(_SETTING_DEFAULTS_YAML, encoding="utf-8")
        (tmp_path / ".env").write_text("POMELO_ORBIT_TRAEFIK__DOMAIN_SUFFIX=custom.com\n", encoding="utf-8")
        svc, patcher = _make_setting_service(tmp_path)
        try:
            svc.reset_config(["traefik__domain_suffix"])
            result = svc.get_config()
        finally:
            patcher.stop()

        item = next(i for i in result["items"] if i["key"] == "traefik__domain_suffix")
        assert item["value"] == "lvh.me"
        assert item["default"] == "lvh.me"
        assert item["is_overridden"] is False

    def test_bool_default_not_affected_by_env_override(self, tmp_path: Path):
        """bool 类型字段：写入 .env 后 default 仍是 yaml 里的 false"""
        (tmp_path / "config.defaults.yaml").write_text(_SETTING_DEFAULTS_YAML, encoding="utf-8")
        svc, patcher = _make_setting_service(tmp_path)
        try:
            svc.update_config("cert__letsencrypt__enabled", True)
            result = svc.get_config()
        finally:
            patcher.stop()

        item = next(i for i in result["items"] if i["key"] == "cert__letsencrypt__enabled")
        assert item["value"] is True
        assert item["default"] is False

    def test_is_overridden_true_when_env_has_key(self, tmp_path: Path):
        (tmp_path / "config.defaults.yaml").write_text(_SETTING_DEFAULTS_YAML, encoding="utf-8")
        (tmp_path / ".env").write_text("POMELO_ORBIT_TRAEFIK__DOMAIN_SUFFIX=custom.com\n", encoding="utf-8")
        svc, patcher = _make_setting_service(tmp_path)
        try:
            result = svc.get_config()
        finally:
            patcher.stop()

        item = next(i for i in result["items"] if i["key"] == "traefik__domain_suffix")
        assert item["is_overridden"] is True

    def test_is_overridden_false_after_reset(self, tmp_path: Path):
        (tmp_path / "config.defaults.yaml").write_text(_SETTING_DEFAULTS_YAML, encoding="utf-8")
        (tmp_path / ".env").write_text("POMELO_ORBIT_TRAEFIK__DOMAIN_SUFFIX=custom.com\n", encoding="utf-8")
        svc, patcher = _make_setting_service(tmp_path)
        try:
            svc.reset_config(["traefik__domain_suffix"])
            result = svc.get_config()
        finally:
            patcher.stop()

        item = next(i for i in result["items"] if i["key"] == "traefik__domain_suffix")
        assert item["is_overridden"] is False
