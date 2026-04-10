"""Traefik 路由 API - 只读展示从 Traefik API 读取的路由"""

import logging
from typing import Annotated

import httpx
from fastapi import APIRouter, Depends, HTTPException, status

from pomelo_orbit.application.cd.di import get_traefik_service
from pomelo_orbit.application.cd.traefik_service import TraefikService
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.cd.dto.traefik_route import TraefikConfigResp, TraefikRouteListResp

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/traefik-routes", tags=["traefik-route"])


@router.get("/config", response_model=TraefikConfigResp)
def get_traefik_config(
    traefik_service: Annotated[TraefikService, Depends(get_traefik_service)],
    _current_user=Depends(get_current_user),
) -> TraefikConfigResp:
    """获取 Traefik 配置"""
    return traefik_service.get_config()


@router.get("", response_model=TraefikRouteListResp)
def list_traefik_routes(
    traefik_service: Annotated[TraefikService, Depends(get_traefik_service)],
    _current_user=Depends(get_current_user),
) -> TraefikRouteListResp:
    """列出所有 Traefik 路由"""
    try:
        return traefik_service.list_routes()
    except (httpx.ConnectError, httpx.TimeoutException) as e:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail=f"无法连接到 Traefik: {e}",
        )
