"""流水线运行管理 API"""

import logging
import math
from typing import Annotated

from fastapi import APIRouter, BackgroundTasks, Depends, Query

from pomelo_orbit.application.ci.di import get_pipeline_service
from pomelo_orbit.application.ci.pipeline_service import PipelineService
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.ci.dto.run import ArtifactResp, PipelineRunResp
from pomelo_orbit.interfaces.api.common import PaginatedResp

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/runs", tags=["runs"])


@router.get("", response_model=PaginatedResp[PipelineRunResp])
def list_all_runs(
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 20,
    project_id: Annotated[str | None, Query()] = None,
) -> PaginatedResp[PipelineRunResp]:
    # 列表场景不需要 stage_runs，直接 model_validate 保持 stage_runs=[]
    runs, total = pipeline_service.list_runs(project_id=project_id, page=page, per_page=per_page)
    return PaginatedResp(
        items=[PipelineRunResp.model_validate(r) for r in runs],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.get("/{run_id}", response_model=PipelineRunResp)
def get_run(
    run_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineRunResp:
    return PipelineRunResp.with_stage_runs(run_id, pipeline_service)


@router.get("/{run_id}/artifacts", response_model=list[ArtifactResp])
def list_artifacts(
    run_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> list[ArtifactResp]:
    return [ArtifactResp.model_validate(a) for a in pipeline_service.list_artifacts(run_id)]


@router.get("/{run_id}/stages/{stage_run_id}/log")
def get_stage_log(
    run_id: str,
    stage_run_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
    offset: Annotated[int, Query(ge=0)] = 0,
):
    """读取 stage 日志（增量），对齐 CD 的 /deployments/{id}/logs 接口。"""
    logs, new_offset, is_complete = pipeline_service.read_stage_log(run_id, stage_run_id, offset)
    return {
        "logs": logs,
        "offset": new_offset,
        "is_complete": is_complete,
    }


@router.post("/{run_id}/cancel", response_model=PipelineRunResp)
def cancel_pipeline(
    run_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineRunResp:
    pipeline_service.cancel_run(run_id)
    return PipelineRunResp.with_stage_runs(run_id, pipeline_service)


@router.post("/{run_id}/retry", response_model=PipelineRunResp, status_code=201)
async def retry_pipeline(
    run_id: str,
    background_tasks: BackgroundTasks,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineRunResp:
    new_run, project, variables, snapshot = pipeline_service.create_retry_run(run_id)
    background_tasks.add_task(pipeline_service.execute_run, new_run, project, variables, snapshot)
    return PipelineRunResp.with_stage_runs(new_run.id, pipeline_service)
