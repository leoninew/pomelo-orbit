"""Traefik 路由 DTO"""

from pydantic import BaseModel


class TraefikConfigResp(BaseModel):
    """Traefik 配置响应"""

    dashboard_domain: str
    https_enabled: bool
