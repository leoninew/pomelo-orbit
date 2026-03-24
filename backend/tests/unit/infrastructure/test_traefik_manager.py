"""
Traefik Manager 基础设施层测试
"""

from unittest.mock import Mock, patch

import pytest
import yaml

from pomelo_orbit.domain.entities import Route
from pomelo_orbit.infrastructure.traefik.manager import TraefikManager


@pytest.fixture
def temp_config_dir(tmp_path):
    """创建临时配置目录"""
    return tmp_path / "traefik_config"


@pytest.fixture
def temp_cert_dir(tmp_path):
    """创建临时证书目录"""
    return tmp_path / "certs"


@pytest.fixture
def traefik_manager():
    """创建 TraefikManager 实例（无状态）"""
    return TraefikManager()


@pytest.fixture
def sample_route():
    """创建示例路由实体"""
    return Route(
        id="route-1",
        name="test-route",
        domain="test.example.com",
        path_prefix="/",
        target_url="http://test-app:8080",
        enabled=True,
        https_enabled=False,
    )


class TestTraefikManagerInit:
    """TraefikManager 初始化测试"""

    def test_creates_stateless_instance(self):
        """测试创建无状态实例"""
        manager = TraefikManager()
        assert manager is not None


class TestDeployRoute:
    """deploy_route 方法测试"""

    @patch.object(TraefikManager, "_reload_traefik")
    def test_creates_yaml_config_file(self, mock_reload, traefik_manager, sample_route, temp_config_dir):
        """测试创建 YAML 配置文件"""
        traefik_manager.deploy_route(sample_route, temp_config_dir, "test-traefik")

        config_file = temp_config_dir / "test-route.yml"
        assert config_file.exists()

    @patch.object(TraefikManager, "_reload_traefik")
    def test_writes_valid_yaml_content(self, mock_reload, traefik_manager, sample_route, temp_config_dir):
        """测试写入有效的 YAML 内容"""
        traefik_manager.deploy_route(sample_route, temp_config_dir, "test-traefik")

        config_file = temp_config_dir / "test-route.yml"
        with config_file.open("r", encoding="utf-8") as f:
            content = yaml.safe_load(f)

        assert "http" in content
        assert "routers" in content["http"]
        assert "services" in content["http"]

    @patch.object(TraefikManager, "_reload_traefik")
    def test_calls_reload_traefik(self, mock_reload, traefik_manager, sample_route, temp_config_dir):
        """测试调用 Traefik 重载"""
        traefik_manager.deploy_route(sample_route, temp_config_dir, "test-traefik")
        mock_reload.assert_called_once_with("test-traefik")


class TestRevokeRoute:
    """revoke_route 方法测试"""

    @patch.object(TraefikManager, "_reload_traefik")
    def test_deletes_existing_config_file(self, mock_reload, traefik_manager, sample_route, temp_config_dir):
        """测试删除已存在的配置文件"""
        config_file = temp_config_dir / "test-route.yml"
        temp_config_dir.mkdir(parents=True, exist_ok=True)
        config_file.write_text("test content")

        traefik_manager.revoke_route(sample_route, temp_config_dir, "test-traefik")

        assert not config_file.exists()

    @patch.object(TraefikManager, "_reload_traefik")
    def test_handles_non_existent_file_gracefully(self, mock_reload, traefik_manager, sample_route, temp_config_dir):
        """测试优雅处理不存在的文件"""
        traefik_manager.revoke_route(sample_route, temp_config_dir, "test-traefik")
        mock_reload.assert_not_called()

    @patch.object(TraefikManager, "_reload_traefik")
    def test_calls_reload_traefik(self, mock_reload, traefik_manager, sample_route, temp_config_dir):
        """测试调用 Traefik 重载"""
        config_file = temp_config_dir / "test-route.yml"
        temp_config_dir.mkdir(parents=True, exist_ok=True)
        config_file.write_text("test")

        traefik_manager.revoke_route(sample_route, temp_config_dir, "test-traefik")
        mock_reload.assert_called_once_with("test-traefik")


class TestReloadTraefik:
    """_reload_traefik 方法测试"""

    @patch("platform.system", return_value="Linux")
    def test_skips_reload_on_non_windows(self, mock_system, traefik_manager):
        """测试在非 Windows 系统上跳过重载"""
        with patch("subprocess.run") as mock_run:
            traefik_manager._reload_traefik("test-traefik")
            mock_run.assert_not_called()

    @patch("platform.system", return_value="Windows")
    @patch("subprocess.run")
    def test_checks_container_running_on_windows(self, mock_run, mock_system, traefik_manager):
        """测试在 Windows 上检查容器是否运行"""
        mock_run.return_value = Mock(stdout="container-id\n")
        traefik_manager._reload_traefik("test-traefik")
        assert mock_run.call_count == 2

    @patch("platform.system", return_value="Windows")
    @patch("subprocess.run")
    def test_sends_sighup_when_container_running(self, mock_run, mock_system, traefik_manager):
        """测试当容器运行时发送 SIGHUP 信号"""
        mock_run.side_effect = [
            Mock(stdout="container-id\n", returncode=0),
            Mock(stdout="", stderr="", returncode=0),
        ]
        traefik_manager._reload_traefik("test-traefik")

        calls = mock_run.call_args_list
        assert len(calls) == 2
        assert "docker" in calls[1][0][0]
        assert "kill" in calls[1][0][0]
        assert "--signal=HUP" in calls[1][0][0]

    @patch("platform.system", return_value="Windows")
    @patch("subprocess.run")
    def test_skips_sighup_when_container_not_running(self, mock_run, mock_system, traefik_manager):
        """测试当容器未运行时跳过 SIGHUP"""
        mock_run.return_value = Mock(stdout="")
        traefik_manager._reload_traefik("test-traefik")
        assert mock_run.call_count == 1
