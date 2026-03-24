"""Certificate services - dependency injection."""

from pomelo_orbit.infrastructure.cert.mkcert import MkcertService


def get_mkcert_service() -> MkcertService:
    """获取 MkcertService 实例"""
    return MkcertService()
