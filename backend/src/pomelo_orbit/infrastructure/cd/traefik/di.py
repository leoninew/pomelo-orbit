"""Traefik infrastructure - dependency injection."""

from pomelo_orbit.infrastructure.cd.traefik.manager import TraefikManager
from pomelo_orbit.infrastructure.traefik.client import TraefikAPIClient


def get_traefik_manager() -> TraefikManager:
    """获取 TraefikManager 实例（无状态）"""
    return TraefikManager()


def get_traefik_api_client() -> TraefikAPIClient:
    """获取 TraefikAPIClient 实例（无状态）"""
    return TraefikAPIClient()
