"""
部署操作 API
"""

import logging
import math
from typing import Annotated

from fastapi import APIRouter, Depends, Query

from pomelo_orbit.application.deployment_service import DeploymentService
from pomelo_orbit.application.di import get_deployment_service
from pomelo_orbit.interfaces.api.auth import get_current_user
from pomelo_orbit.interfaces.api.dto import DeploymentDetailResp, DeploymentResp, PaginatedResp

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/deployment", tags=["deployment"])


@router.get("", response_model=PaginatedResp[DeploymentResp])
def list_deployments(
    deployment_service: Annotated[DeploymentService, Depends(get_deployment_service)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 10,
    application_id: Annotated[str | None, Query()] = None,
    status_filter: Annotated[str | None, Query(alias="status")] = None,
    search: Annotated[str | None, Query()] = None,
    date_from: Annotated[str | None, Query()] = None,
    date_to: Annotated[str | None, Query()] = None,
) -> PaginatedResp[DeploymentResp]:
    """列出所有部署"""
    deployments, total = deployment_service.list_deployments(
        page=page,
        per_page=per_page,
        application_id=application_id,
        status_filter=status_filter,
        search=search,
        date_from=date_from,
        date_to=date_to,
    )
    return PaginatedResp(
        items=[DeploymentResp.model_validate(d) for d in deployments],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.get("/{deployment_id}", response_model=DeploymentDetailResp)
def get_deployment(
    deployment_id: str,
    deployment_service: Annotated[DeploymentService, Depends(get_deployment_service)],
    _current_user=Depends(get_current_user),
) -> DeploymentDetailResp:
    """获取部署详情"""
    deployment = deployment_service.get_deployment(deployment_id)
    return DeploymentDetailResp.model_validate(deployment)


@router.get("/{deployment_id}/logs")
async def get_deployment_logs(
    deployment_id: str,
    deployment_service: Annotated[DeploymentService, Depends(get_deployment_service)],
    _current_user=Depends(get_current_user),
    offset: Annotated[int, Query(ge=0)] = 0,
):
    """获取部署日志（增量读取）"""
    logs, current_offset, is_complete = deployment_service.read_deployment_log(deployment_id, offset)
    deployment = deployment_service.get_deployment(deployment_id)

    return {
        "logs": logs,
        "offset": current_offset,
        "is_complete": is_complete,
        "status": deployment.status,
    }


@router.post("/{deployment_id}/cancel")
def cancel_deployment(
    deployment_id: str,
    deployment_service: Annotated[DeploymentService, Depends(get_deployment_service)],
    _current_user=Depends(get_current_user),
):
    """取消正在进行的部署"""
    deployment_service.cancel_deployment(deployment_id)
    return {"message": "Deployment cancelled"}
