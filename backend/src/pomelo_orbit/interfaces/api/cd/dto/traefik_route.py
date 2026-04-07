"""Traefik 路由 DTO"""

from pydantic import BaseModel


class TraefikConfigResp(BaseModel):
    """Traefik 配置响应"""

    dashboard_domain: str
    https_enabled: bool


class TraefikRouteListResp(BaseModel):
    """Traefik 路由列表响应"""

    items: list
    total: int
