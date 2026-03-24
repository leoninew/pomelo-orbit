"""Traefik API 数据模型"""

from dataclasses import dataclass


@dataclass
class TraefikRouter:
    """Traefik 路由信息"""

    name: str
    provider: str
    status: str
    rule: str
    service: str
    entrypoints: list[str]
    tls: bool
