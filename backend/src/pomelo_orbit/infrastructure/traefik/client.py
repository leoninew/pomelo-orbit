"""Traefik API 客户端 - 用于读取 Traefik 路由信息"""

import httpx

from pomelo_orbit.infrastructure.traefik.models import TraefikRouter


class TraefikAPIClient:
    """Traefik API 客户端（无状态）"""

    def get_routers(self, api_url: str) -> list[TraefikRouter]:
        """获取所有 HTTP 路由"""
        url = api_url.rstrip("/")
        with httpx.Client(timeout=10.0) as client:
            response = client.get(f"{url}/api/http/routers")
            response.raise_for_status()
            routers = response.json()
            return [self._format_router(r) for r in routers]

    def _format_router(self, router: dict) -> TraefikRouter:
        """格式化路由信息"""
        return TraefikRouter(
            name=router["name"],
            provider=router["provider"],
            status=router["status"],
            rule=router["rule"],
            service=router["service"],
            entrypoints=router["entryPoints"],
            tls=router.get("tls") is not None,
        )
