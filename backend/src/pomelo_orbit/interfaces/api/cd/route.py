"""路由管理 API"""

import logging
import math
from typing import Annotated

from fastapi import APIRouter, Depends, Query, UploadFile

from pomelo_orbit.application.cd.di import get_route_service
from pomelo_orbit.application.cd.route_service import RouteService
from pomelo_orbit.application.project.di import get_project_service
from pomelo_orbit.application.project.project_service import ProjectService
from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.domain.cd.entities import Route
from pomelo_orbit.domain.exceptions import BusinessError
from pomelo_orbit.interfaces.api.auth.dependencies import get_current_user
from pomelo_orbit.interfaces.api.cd.dto.route import RouteCreateReq, RouteResp, RouteUpdateReq
from pomelo_orbit.interfaces.api.common import PaginatedResp

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/route", tags=["route"])


def _authorize_route_project(
    route_service: RouteService,
    project_service: ProjectService,
    current_user: User,
    route_id: str,
) -> Route:
    route = route_service.get_route(route_id)
    if route.project_id is None:
        raise BusinessError("Permission denied", status_code=403)
    project_service.get_project(current_user.id, route.project_id)
    return route


@router.get("", response_model=PaginatedResp[RouteResp])
def list_routes(
    route_service: Annotated[RouteService, Depends(get_route_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
    project_id: Annotated[str, Query()],
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 10,
    search: Annotated[str | None, Query()] = None,
) -> PaginatedResp[RouteResp]:
    """列出所有路由"""
    project_service.get_project(current_user.id, project_id)
    routes, total = route_service.list_routes(project_id, page, per_page, search)
    return PaginatedResp(
        items=[RouteResp.model_validate(r) for r in routes],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.post("", response_model=RouteResp, status_code=201)
def create_route(
    data: RouteCreateReq,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
    project_id: Annotated[str, Query()],
) -> RouteResp:
    """创建路由"""
    project_service.get_project(current_user.id, project_id)
    route = route_service.create_route(
        project_id=project_id,
        name=data.name,
        domain=data.domain,
        path_prefix=data.path_prefix,
        target_url=data.target_url,
        enabled=data.enabled,
    )
    return RouteResp.model_validate(route)


@router.get("/{route_id}", response_model=RouteResp)
def get_route(
    route_id: str,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> RouteResp:
    """获取路由详情"""
    route = _authorize_route_project(route_service, project_service, current_user, route_id)
    return RouteResp.model_validate(route)


@router.put("/{route_id}", response_model=RouteResp)
def update_route(
    route_id: str,
    data: RouteUpdateReq,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> RouteResp:
    """更新路由"""
    _authorize_route_project(route_service, project_service, current_user, route_id)
    update_data = data.model_dump(exclude_unset=True)
    updated_route = route_service.update_route(route_id, **update_data)
    return RouteResp.model_validate(updated_route)


@router.delete("/{route_id}", status_code=204)
def delete_route(
    route_id: str,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
):
    """删除路由"""
    _authorize_route_project(route_service, project_service, current_user, route_id)
    route_service.delete_route(route_id)


@router.post("/{route_id}/enable")
def enable_route(
    route_id: str,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
):
    """启用路由"""
    _authorize_route_project(route_service, project_service, current_user, route_id)
    route_service.enable_route(route_id)
    return {"message": "Route enabled successfully"}


@router.post("/{route_id}/disable")
def disable_route(
    route_id: str,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
):
    """停用路由"""
    _authorize_route_project(route_service, project_service, current_user, route_id)
    route_service.disable_route(route_id)
    return {"message": "Route disabled successfully"}


@router.post("/sync")
def sync_routes(
    route_service: Annotated[RouteService, Depends(get_route_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
    project_id: Annotated[str, Query()],
):
    """同步路由配置"""
    project_service.get_project(current_user.id, project_id)
    route_service.sync_routes(project_id)
    return {"message": "Routes synced successfully"}


@router.post("/{route_id}/cert")
async def upload_cert(
    route_id: str,
    pem: UploadFile,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> RouteResp:
    """上传 SSL 证书"""
    _authorize_route_project(route_service, project_service, current_user, route_id)
    cert_content = await pem.read()
    route = route_service.upload_cert(route_id, cert_content)
    return RouteResp.model_validate(route)


@router.delete("/{route_id}/https")
def disable_https(
    route_id: str,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> RouteResp:
    """禁用 HTTPS"""
    _authorize_route_project(route_service, project_service, current_user, route_id)
    route = route_service.disable_https(route_id)
    return RouteResp.model_validate(route)


@router.post("/{route_id}/letsencrypt")
def enable_letsencrypt(
    route_id: str,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> RouteResp:
    """启用 Let's Encrypt 自动证书"""
    _authorize_route_project(route_service, project_service, current_user, route_id)
    route = route_service.enable_letsencrypt(route_id)
    return RouteResp.model_validate(route)


@router.post("/{route_id}/mkcert")
def enable_mkcert(
    route_id: str,
    route_service: Annotated[RouteService, Depends(get_route_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> RouteResp:
    """启用 mkcert 本地证书"""
    _authorize_route_project(route_service, project_service, current_user, route_id)
    route = route_service.enable_mkcert(route_id)
    return RouteResp.model_validate(route)
