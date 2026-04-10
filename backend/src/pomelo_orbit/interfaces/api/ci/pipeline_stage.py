"""流水线 Stage 管理 API"""

import math
from typing import Annotated

from fastapi import APIRouter, Depends, Query

from pomelo_orbit.application.ci.di import get_stage_service
from pomelo_orbit.application.ci.stage_service import StageService
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.ci.dto.pipeline_template import (
    PipelineStageCreateReq,
    PipelineStageResp,
    PipelineStageUpdateReq,
)
from pomelo_orbit.interfaces.api.common import PaginatedResp

router = APIRouter(prefix="/pipeline-stage", tags=["pipeline-stage"])


@router.get("", response_model=PaginatedResp[PipelineStageResp])
def list_stages(
    stage_service: Annotated[StageService, Depends(get_stage_service)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 20,
) -> PaginatedResp[PipelineStageResp]:
    stages, total = stage_service.list_stages(page=page, per_page=per_page)
    return PaginatedResp(
        items=[PipelineStageResp.model_validate(s) for s in stages],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.post("", response_model=PipelineStageResp, status_code=201)
def create_stage(
    data: PipelineStageCreateReq,
    stage_service: Annotated[StageService, Depends(get_stage_service)],
    _current_user=Depends(get_current_user),
) -> PipelineStageResp:
    stage = stage_service.create_stage(
        name=data.name,
        image=data.image,
        script=data.script,
        env=data.env,
        artifacts=[a.model_dump() for a in data.artifacts] if data.artifacts else None,
        description=data.description,
    )
    return PipelineStageResp.model_validate(stage)


@router.get("/{stage_id}", response_model=PipelineStageResp)
def get_stage(
    stage_id: str,
    stage_service: Annotated[StageService, Depends(get_stage_service)],
    _current_user=Depends(get_current_user),
) -> PipelineStageResp:
    return PipelineStageResp.model_validate(stage_service.get_stage(stage_id))


@router.put("/{stage_id}", response_model=PipelineStageResp)
def update_stage(
    stage_id: str,
    data: PipelineStageUpdateReq,
    stage_service: Annotated[StageService, Depends(get_stage_service)],
    _current_user=Depends(get_current_user),
) -> PipelineStageResp:
    stage = stage_service.update_stage(
        stage_id=stage_id,
        name=data.name,
        image=data.image,
        script=data.script,
        env=data.env,
        artifacts=[a.model_dump() for a in data.artifacts] if data.artifacts is not None else None,
        description=data.description,
    )
    return PipelineStageResp.model_validate(stage)


@router.delete("/{stage_id}", status_code=204)
def delete_stage(
    stage_id: str,
    stage_service: Annotated[StageService, Depends(get_stage_service)],
    _current_user=Depends(get_current_user),
) -> None:
    stage_service.delete_stage(stage_id)


@router.post("/{stage_id}/duplicate", response_model=PipelineStageResp, status_code=201)
def duplicate_stage(
    stage_id: str,
    stage_service: Annotated[StageService, Depends(get_stage_service)],
    _current_user=Depends(get_current_user),
) -> PipelineStageResp:
    stage = stage_service.duplicate_stage(stage_id)
    return PipelineStageResp.model_validate(stage)
