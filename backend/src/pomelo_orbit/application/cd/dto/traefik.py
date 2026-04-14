"""Traefik 应用层数据模型"""

from pydantic import BaseModel


class TraefikConfigResp(BaseModel):
    dashboard_domain: str
    https_enabled: bool


class TraefikRouteListResp(BaseModel):
    items: list
    total: int
