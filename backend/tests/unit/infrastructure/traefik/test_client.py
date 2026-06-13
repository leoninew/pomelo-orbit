"""TraefikAPIClient 单元测试"""

from unittest.mock import Mock, patch

import httpx
import pytest

from pomelo_orbit.infrastructure.traefik import TraefikAPIClient, TraefikRouter


class TestTraefikAPIClient:
    """TraefikAPIClient 测试类"""

    def test_init(self):
        """测试初始化（无状态）"""
        client = TraefikAPIClient()
        assert client is not None

    @patch("httpx.Client.get")
    def test_get_routers_success(self, mock_get):
        """测试成功获取路由列表"""
        mock_response = Mock()
        mock_response.json.return_value = [
            {
                "name": "test-router",
                "provider": "file",
                "status": "enabled",
                "rule": "Host(`example.com`)",
                "service": "test-service",
                "entryPoints": ["web", "websecure"],
                "tls": {},
            }
        ]
        mock_get.return_value = mock_response

        client = TraefikAPIClient()
        routers = client.get_routers("http://localhost:8080")

        assert len(routers) == 1
        assert isinstance(routers[0], TraefikRouter)
        assert routers[0].name == "test-router"
        assert routers[0].provider == "file"
        assert routers[0].status == "enabled"
        assert routers[0].rule == "Host(`example.com`)"
        assert routers[0].service == "test-service"
        assert routers[0].entrypoints == ["web", "websecure"]
        assert routers[0].tls is True

        mock_get.assert_called_once_with("http://localhost:8080/api/http/routers")

    @patch("httpx.Client.get")
    def test_get_routers_strips_trailing_slash(self, mock_get):
        """测试自动移除 API URL 尾部斜杠"""
        mock_response = Mock()
        mock_response.json.return_value = []
        mock_get.return_value = mock_response

        client = TraefikAPIClient()
        client.get_routers("http://localhost:8080/")

        mock_get.assert_called_once_with("http://localhost:8080/api/http/routers")

    @patch("httpx.Client.get")
    def test_get_routers_without_tls(self, mock_get):
        """测试获取无 TLS 的路由"""
        mock_response = Mock()
        mock_response.json.return_value = [
            {
                "name": "http-router",
                "provider": "docker",
                "status": "enabled",
                "rule": "Host(`test.local`)",
                "service": "http-service",
                "entryPoints": ["web"],
            }
        ]
        mock_get.return_value = mock_response

        client = TraefikAPIClient()
        routers = client.get_routers("http://localhost:8080")

        assert len(routers) == 1
        assert routers[0].tls is False

    @patch("httpx.Client.get")
    def test_get_routers_empty(self, mock_get):
        """测试获取空路由列表"""
        mock_response = Mock()
        mock_response.json.return_value = []
        mock_get.return_value = mock_response

        client = TraefikAPIClient()
        routers = client.get_routers("http://localhost:8080")

        assert len(routers) == 0

    @patch("httpx.Client.get")
    def test_get_routers_connect_error(self, mock_get):
        """测试连接错误"""
        mock_get.side_effect = httpx.ConnectError("Connection failed")

        client = TraefikAPIClient()
        with pytest.raises(httpx.ConnectError):
            client.get_routers("http://localhost:8080")

    @patch("httpx.Client.get")
    def test_get_routers_http_error(self, mock_get):
        """测试 HTTP 错误"""
        mock_response = Mock()
        mock_response.raise_for_status.side_effect = httpx.HTTPStatusError(
            "404 Not Found", request=Mock(), response=Mock()
        )
        mock_get.return_value = mock_response

        client = TraefikAPIClient()
        with pytest.raises(httpx.HTTPStatusError):
            client.get_routers("http://localhost:8080")

    @patch("httpx.Client.get")
    def test_get_routers_missing_required_field(self, mock_get):
        """测试缺少必需字段时抛出 KeyError"""
        mock_response = Mock()
        mock_response.json.return_value = [
            {
                "name": "test-router",
                "provider": "file",
                # 缺少 status 字段
                "rule": "Host(`example.com`)",
                "service": "test-service",
                "entryPoints": ["web"],
            }
        ]
        mock_get.return_value = mock_response

        client = TraefikAPIClient()
        with pytest.raises(KeyError):
            client.get_routers("http://localhost:8080")
