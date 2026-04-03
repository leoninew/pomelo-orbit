"""流水线运行管理 API"""

import logging
import math
from typing import Annotated

from fastapi import APIRouter, BackgroundTasks, Depends, Query

from pomelo_orbit.application.ci.di import get_pipeline_service
from pomelo_orbit.application.ci.pipeline_service import PipelineService
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.ci.dto.job import JobResp
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
    return PipelineRunResp.model_validate(pipeline_service.get_run(run_id))


@router.get("/{run_id}/artifacts", response_model=list[ArtifactResp])
def list_artifacts(
    run_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> list[ArtifactResp]:
    return [ArtifactResp.model_validate(a) for a in pipeline_service.list_artifacts(run_id)]


@router.get("/{run_id}/jobs", response_model=list[JobResp])
def list_jobs(
    run_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> list[JobResp]:
    return [JobResp.model_validate(j) for j in pipeline_service.list_jobs(run_id)]


@router.post("/{run_id}/cancel", response_model=PipelineRunResp)
def cancel_pipeline(
    run_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineRunResp:
    return PipelineRunResp.model_validate(pipeline_service.cancel_run(run_id))


@router.post("/{run_id}/retry", response_model=PipelineRunResp, status_code=201)
async def retry_pipeline(
    run_id: str,
    background_tasks: BackgroundTasks,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineRunResp:
    new_run, project, variables = pipeline_service.create_retry_run(run_id)
    background_tasks.add_task(pipeline_service.execute_run, new_run, project, variables)
    return PipelineRunResp.model_validate(new_run)
