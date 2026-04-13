"""流水线运行管理 API"""

import logging
import math
from typing import Annotated

from fastapi import APIRouter, BackgroundTasks, Depends, Query

from pomelo_orbit.application.ci.di import get_pipeline_run_service
from pomelo_orbit.application.ci.pipeline_run_service import PipelineRunService
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.ci.dto.artifact import ArtifactResp
from pomelo_orbit.interfaces.api.ci.dto.pipeline_run import PipelineRunResp
from pomelo_orbit.interfaces.api.common import PaginatedResp

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/run", tags=["run"])


@router.get("", response_model=PaginatedResp[PipelineRunResp])
def list_all_runs(
    pipeline_run_service: Annotated[PipelineRunService, Depends(get_pipeline_run_service)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 20,
    repository_id: Annotated[str | None, Query()] = None,
    template_id: Annotated[str | None, Query()] = None,
) -> PaginatedResp[PipelineRunResp]:
    result = pipeline_run_service.list_runs(
        repository_id=repository_id, template_id=template_id, page=page, per_page=per_page
    )
    return PaginatedResp(
        items=[PipelineRunResp.model_validate(r) for r in result.runs],
        total=result.total,
        page=page,
        per_page=per_page,
        pages=math.ceil(result.total / per_page) if result.total > 0 else 1,
    )


@router.get("/{run_id}", response_model=PipelineRunResp)
def get_run(
    run_id: str,
    pipeline_run_service: Annotated[PipelineRunService, Depends(get_pipeline_run_service)],
    _current_user=Depends(get_current_user),
) -> PipelineRunResp:
    return PipelineRunResp.with_stage_runs(run_id, pipeline_run_service)


@router.get("/{run_id}/artifacts", response_model=list[ArtifactResp])
def list_artifacts(
    run_id: str,
    pipeline_run_service: Annotated[PipelineRunService, Depends(get_pipeline_run_service)],
    _current_user=Depends(get_current_user),
) -> list[ArtifactResp]:
    return [ArtifactResp.model_validate(a) for a in pipeline_run_service.list_artifacts(run_id)]


@router.get("/{run_id}/stages/{stage_run_id}/log")
def get_stage_log(
    run_id: str,
    stage_run_id: str,
    pipeline_run_service: Annotated[PipelineRunService, Depends(get_pipeline_run_service)],
    _current_user=Depends(get_current_user),
    offset: Annotated[int, Query(ge=0)] = 0,
) -> dict:
    result = pipeline_run_service.read_stage_log(run_id, stage_run_id, offset)
    return {
        "logs": result.logs,
        "offset": result.offset,
        "is_complete": result.is_complete,
    }


@router.post("/{run_id}/cancel", response_model=PipelineRunResp)
def cancel_pipeline(
    run_id: str,
    pipeline_run_service: Annotated[PipelineRunService, Depends(get_pipeline_run_service)],
    _current_user=Depends(get_current_user),
) -> PipelineRunResp:
    pipeline_run_service.cancel_run(run_id)
    return PipelineRunResp.with_stage_runs(run_id, pipeline_run_service)


@router.post("/{run_id}/retry", response_model=PipelineRunResp, status_code=201)
async def retry_pipeline(
    run_id: str,
    background_tasks: BackgroundTasks,
    pipeline_run_service: Annotated[PipelineRunService, Depends(get_pipeline_run_service)],
    _current_user=Depends(get_current_user),
) -> PipelineRunResp:
    result = pipeline_run_service.create_retry_run(run_id)
    background_tasks.add_task(
        pipeline_run_service.execute_run, result.run, result.repository, result.merged_variables, result.snapshot
    )
    return PipelineRunResp.with_stage_runs(result.run.id, pipeline_run_service)
