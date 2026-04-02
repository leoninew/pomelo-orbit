"""Route management API endpoints."""

import logging
import math
from typing import Annotated

from fastapi import APIRouter, Depends, Query, UploadFile, status

from pomelo_orbit.application.cd.di import get_route_service
from pomelo_orbit.application.cd.route_service import RouteService
from pomelo_orbit.infrastructure.persistence.mappers import RouteMapper
from pomelo_orbit.interfaces.api.auth import get_current_user
from pomelo_orbit.interfaces.api.dto import (
    PaginatedResp,
    RouteCreateReq,
    RouteResp,
    RouteUpdateReq,
)

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/route", tags=["route"])


@router.get("", response_model=PaginatedResp[RouteResp])
def list_routes(
    route_service: Annotated[RouteService, Depends(get_route_service)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 10,
) -> PaginatedResp[RouteResp]:
    """列出所有路由"""
    routes, total = route_service.list_routes(page, per_page)
    return PaginatedResp(
        items=[RouteResp.model_validate(RouteMapper.to_orm(r)) for r in routes],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.post("", response_model=RouteResp, status_code=status.HTTP_201_CREATED)
def create_route(
    data: RouteCreateReq,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    _current_user=Depends(get_current_user),
) -> RouteResp:
    """创建路由"""
    route = route_service.create_route(
        name=data.name,
        domain=data.domain,
        path_prefix=data.path_prefix,
        target_url=data.target_url,
        enabled=data.enabled,
    )
    return RouteResp.model_validate(RouteMapper.to_orm(route))


@router.get("/{route_id}", response_model=RouteResp)
def get_route(
    route_id: str,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    _current_user=Depends(get_current_user),
) -> RouteResp:
    """获取路由详情"""
    route = route_service.get_route(route_id)
    return RouteResp.model_validate(RouteMapper.to_orm(route))


@router.put("/{route_id}", response_model=RouteResp)
def update_route(
    route_id: str,
    data: RouteUpdateReq,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    _current_user=Depends(get_current_user),
) -> RouteResp:
    """更新路由"""
    update_data = data.model_dump(exclude_unset=True)
    updated_route = route_service.update_route(route_id, **update_data)
    return RouteResp.model_validate(RouteMapper.to_orm(updated_route))


@router.delete("/{route_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_route(
    route_id: str,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    _current_user=Depends(get_current_user),
):
    """删除路由（要求路由已停用）"""
    route_service.delete_route(route_id)


@router.put("/{route_id}/enable")
def enable_route(
    route_id: str,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    _current_user=Depends(get_current_user),
):
    """启用路由"""
    route_service.enable_route(route_id)
    return {"message": "Route enabled successfully"}


@router.put("/{route_id}/disable")
def disable_route(
    route_id: str,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    _current_user=Depends(get_current_user),
):
    """停用路由"""
    route_service.disable_route(route_id)
    return {"message": "Route disabled successfully"}


@router.post("/sync")
def sync_routes(
    route_service: Annotated[RouteService, Depends(get_route_service)],
    _current_user=Depends(get_current_user),
):
    """同步路由配置"""
    route_service.sync_routes()
    return {"message": "Routes synced successfully"}


@router.post("/{route_id}/cert")
async def upload_cert(
    route_id: str,
    pem: UploadFile,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    _current_user=Depends(get_current_user),
) -> RouteResp:
    """上传 SSL 证书 (PEM 格式)"""
    cert_content = await pem.read()
    route = route_service.upload_cert(route_id, cert_content)
    return RouteResp.model_validate(RouteMapper.to_orm(route))


@router.delete("/{route_id}/https")
def disable_https(
    route_id: str,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    _current_user=Depends(get_current_user),
) -> RouteResp:
    """禁用 HTTPS"""
    route = route_service.disable_https(route_id)
    return RouteResp.model_validate(RouteMapper.to_orm(route))


@router.post("/{route_id}/letsencrypt")
def enable_letsencrypt(
    route_id: str,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    _current_user=Depends(get_current_user),
) -> RouteResp:
    """启用 Let's Encrypt 自动证书"""
    route = route_service.enable_letsencrypt(route_id)
    return RouteResp.model_validate(RouteMapper.to_orm(route))


@router.post("/{route_id}/mkcert")
def enable_mkcert(
    route_id: str,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    _current_user=Depends(get_current_user),
) -> RouteResp:
    """启用 mkcert 本地证书"""
    route = route_service.enable_mkcert(route_id)
    return RouteResp.model_validate(RouteMapper.to_orm(route))
