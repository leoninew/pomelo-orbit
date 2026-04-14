"""Traefik 应用服务"""

from dynaconf import Dynaconf

from pomelo_orbit.application.cd.dto.traefik import TraefikConfigResp, TraefikRouteListResp
from pomelo_orbit.domain.cd.repositories import RouteRepository
from pomelo_orbit.infrastructure.traefik import TraefikAPIClient


class TraefikService:
    """Traefik 相关业务逻辑"""

    def __init__(
        self,
        route_repo: RouteRepository,
        traefik_client: TraefikAPIClient,
        settings: Dynaconf,
    ):
        self.route_repo = route_repo
        self.traefik_client = traefik_client
        self.settings = settings

    def get_config(self) -> TraefikConfigResp:
        """获取 Traefik 配置"""
        dashboard_domain = f"traefik.{self.settings.traefik.domain_suffix}"
        route = self.route_repo.find_by_domain(dashboard_domain)

        return TraefikConfigResp(
            dashboard_domain=dashboard_domain,
            https_enabled=route.enabled and route.https_enabled if route else False,
        )

    def list_routes(self) -> TraefikRouteListResp:
        """列出所有 Traefik 路由"""
        routers = self.traefik_client.get_routers(self.settings.traefik.api_url)
        return TraefikRouteListResp(items=routers, total=len(routers))
