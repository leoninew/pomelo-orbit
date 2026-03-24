"""Traefik 路由 API - 只读展示从 Traefik API 读取的路由"""

from typing import Annotated

import httpx
from dynaconf import Dynaconf
from fastapi import APIRouter, Depends, HTTPException, status

from pomelo_orbit.infrastructure.config import get_settings
from pomelo_orbit.infrastructure.repositories.di import get_route_repository
from pomelo_orbit.infrastructure.traefik import TraefikAPIClient
from pomelo_orbit.infrastructure.traefik.di import get_traefik_api_client
from pomelo_orbit.interfaces.api.auth import get_current_user
from pomelo_orbit.interfaces.api.dto.traefik_route import TraefikConfigResp

router = APIRouter(prefix="/traefik-route", tags=["traefik-route"])


@router.get("/config", response_model=TraefikConfigResp)
def get_traefik_config(
    settings: Annotated[Dynaconf, Depends(get_settings)],
    _current_user=Depends(get_current_user),
    route_repo=Depends(get_route_repository),
) -> TraefikConfigResp:
    """获取 Traefik 配置"""
    dashboard_domain = f"traefik.{settings.traefik.domain_suffix}"
    route = route_repo.find_by_domain(dashboard_domain)

    return TraefikConfigResp(
        dashboard_domain=dashboard_domain,
        https_enabled=route.enabled and route.https_enabled if route else False,
    )


@router.get("")
def list_traefik_routes(
    settings: Annotated[Dynaconf, Depends(get_settings)],
    client: Annotated[TraefikAPIClient, Depends(get_traefik_api_client)],
    _current_user=Depends(get_current_user),
):
    """列出所有 Traefik 路由（从 Traefik API 读取）"""
    try:
        routers = client.get_routers(settings.traefik.api_url)
        return {"items": routers, "total": len(routers)}
    except (httpx.ConnectError, httpx.TimeoutException) as e:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail=f"无法连接到 Traefik: {e}",
        )
