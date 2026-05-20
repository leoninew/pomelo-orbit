"""
部署操作 API
"""

import asyncio
import json
import logging
import math
from collections.abc import AsyncGenerator
from typing import Annotated

from fastapi import APIRouter, Depends, Query
from fastapi.responses import StreamingResponse

from pomelo_orbit.application.cd.deployment_service import DeploymentService
from pomelo_orbit.application.cd.di import get_deployment_service
from pomelo_orbit.application.project.di import get_project_service
from pomelo_orbit.application.project.project_service import ProjectService
from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.domain.cd.entities import Deployment
from pomelo_orbit.interfaces.api.auth.dependencies import get_current_user
from pomelo_orbit.interfaces.api.cd.dto.deployment import DeploymentDetailResp, DeploymentResp
from pomelo_orbit.interfaces.api.common import PaginatedResp

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/deployment", tags=["deployment"])


def _authorize_deployment_project(
    deployment_service: DeploymentService,
    project_service: ProjectService,
    current_user: User,
    deployment_id: str,
) -> Deployment:
    deployment = deployment_service.get_deployment(deployment_id)
    project_service.get_project(current_user.id, deployment.project_id)
    return deployment


@router.get("", response_model=PaginatedResp[DeploymentResp])
def list_deployments(
    deployment_service: Annotated[DeploymentService, Depends(get_deployment_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
    project_id: Annotated[str, Query()],
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 10,
    application_id: Annotated[str | None, Query()] = None,
    status_filter: Annotated[str | None, Query(alias="status")] = None,
    search: Annotated[str | None, Query()] = None,
    date_from: Annotated[str | None, Query()] = None,
    date_to: Annotated[str | None, Query()] = None,
) -> PaginatedResp[DeploymentResp]:
    """列出所有部署"""
    project_service.get_project(current_user.id, project_id)
    deployments, total = deployment_service.list_deployments(
        project_id=project_id,
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
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> DeploymentDetailResp:
    """获取部署详情"""
    deployment = _authorize_deployment_project(deployment_service, project_service, current_user, deployment_id)
    return DeploymentDetailResp.model_validate(deployment)


@router.get("/{deployment_id}/logs")
async def get_deployment_logs(
    deployment_id: str,
    deployment_service: Annotated[DeploymentService, Depends(get_deployment_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
    offset: Annotated[int, Query(ge=0)] = 0,
):
    """获取部署日志（增量读取）"""
    deployment = _authorize_deployment_project(deployment_service, project_service, current_user, deployment_id)
    logs, current_offset, is_complete = deployment_service.read_deployment_log(deployment_id, offset)

    return {
        "logs": logs,
        "offset": current_offset,
        "is_complete": is_complete,
        "status": deployment.status,
    }


@router.get("/{deployment_id}/stream-log")
async def stream_deployment_log(
    deployment_id: str,
    deployment_service: Annotated[DeploymentService, Depends(get_deployment_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> StreamingResponse:
    """SSE 流式推送部署日志"""
    _authorize_deployment_project(deployment_service, project_service, current_user, deployment_id)

    async def event_generator() -> AsyncGenerator[str, None]:
        offset = 0
        while True:
            logs, new_offset, is_complete = await asyncio.to_thread(
                deployment_service.read_deployment_log, deployment_id, offset
            )

            if logs:
                data = json.dumps({"logs": logs, "offset": new_offset}, ensure_ascii=False)
                yield f"data: {data}\n\n"
                offset = new_offset

            if is_complete:
                yield "event: complete\ndata: {}\n\n"
                break

            await asyncio.sleep(0.5)

    return StreamingResponse(
        event_generator(),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "X-Accel-Buffering": "no",
        },
    )


@router.post("/{deployment_id}/cancel")
def cancel_deployment(
    deployment_id: str,
    deployment_service: Annotated[DeploymentService, Depends(get_deployment_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
):
    """取消正在进行的部署"""
    _authorize_deployment_project(deployment_service, project_service, current_user, deployment_id)
    deployment_service.cancel_deployment(deployment_id)
    return {"message": "Deployment cancelled"}
