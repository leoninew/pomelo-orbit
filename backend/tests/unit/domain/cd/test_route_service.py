"""
路由领域服务测试
"""

import pytest

from pomelo_orbit.domain.cd.entities import Route
from pomelo_orbit.domain.cd.route_service import RouteDomainService


@pytest.fixture
def temp_config_dir(tmp_path):
    """创建临时配置目录"""
    return tmp_path / "traefik_config"


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


class TestGenerateRouteConfig:
    """_generate_route_config 方法测试"""

    def test_generates_basic_route_config(self, sample_route):
        """测试生成基本路由配置"""
        config = RouteDomainService.generate_route_config(sample_route)

        assert "http" in config
        assert "routers" in config["http"]
        assert "services" in config["http"]

    def test_generates_router_with_host_rule(self, sample_route):
        """测试生成包含 Host 规则的路由器"""
        config = RouteDomainService.generate_route_config(sample_route)

        router = config["http"]["routers"]["test-route-route"]
        assert "Host(`test.example.com`)" in router["rule"]

    def test_generates_router_with_path_prefix_when_not_root(self, sample_route):
        """测试当路径前缀不是根路径时生成 PathPrefix 规则"""
        sample_route.path_prefix = "/api"
        config = RouteDomainService.generate_route_config(sample_route)

        router = config["http"]["routers"]["test-route-route"]
        assert "Host(`test.example.com`)" in router["rule"]
        assert "PathPrefix(`/api`)" in router["rule"]

    def test_generates_router_without_path_prefix_for_root(self, sample_route):
        """测试根路径时不生成 PathPrefix 规则"""
        sample_route.path_prefix = "/"
        config = RouteDomainService.generate_route_config(sample_route)

        router = config["http"]["routers"]["test-route-route"]
        assert "PathPrefix" not in router["rule"]

    def test_generates_service_with_target_url(self, sample_route):
        """测试生成包含目标 URL 的服务"""
        config = RouteDomainService.generate_route_config(sample_route)

        service = config["http"]["services"]["test-route-service"]
        servers = service["loadBalancer"]["servers"]
        assert len(servers) == 1
        assert servers[0]["url"] == "http://test-app:8080"

    def test_router_references_correct_service(self, sample_route):
        """测试路由器引用正确的服务"""
        config = RouteDomainService.generate_route_config(sample_route)

        router = config["http"]["routers"]["test-route-route"]
        assert router["service"] == "test-route-service"

    def test_router_uses_web_entrypoint(self, sample_route):
        """测试路由器使用 web 入口点"""
        config = RouteDomainService.generate_route_config(sample_route)

        router = config["http"]["routers"]["test-route-route"]
        assert router["entryPoints"] == ["web"]


# 注意：RouteDomainService 已重构为纯领域服务（只有静态方法 generate_route_config）
# 文件操作和 Docker 交互已移至 TraefikManager（基础设施层）
# 以下测试已过时，需要在 TraefikManager 的测试中重新实现
