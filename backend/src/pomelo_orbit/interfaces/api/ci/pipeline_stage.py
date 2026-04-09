"""流水线 Stage 管理 API"""

from typing import Annotated

from fastapi import APIRouter, Depends

from pomelo_orbit.application.ci.di import get_pipeline_service
from pomelo_orbit.application.ci.pipeline_service import PipelineService
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.ci.dto.pipeline_template import (
    PipelineStageCreateReq,
    PipelineStageResp,
    PipelineStageUpdateReq,
)

router = APIRouter(prefix="/pipeline-stage", tags=["pipeline-stage"])


@router.get("", response_model=list[PipelineStageResp])
def list_stages(
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> list[PipelineStageResp]:
    return [PipelineStageResp.model_validate(s) for s in pipeline_service.list_stages()]


@router.post("", response_model=PipelineStageResp, status_code=201)
def create_stage(
    data: PipelineStageCreateReq,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineStageResp:
    stage = pipeline_service.create_stage(
        name=data.name,
        image=data.image,
        script=data.script,
        env=data.env or {},
        artifacts=[a.model_dump() for a in data.artifacts] if data.artifacts else None,
        description=data.description,
    )
    return PipelineStageResp.model_validate(stage)


@router.get("/{stage_id}", response_model=PipelineStageResp)
def get_stage(
    stage_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineStageResp:
    return PipelineStageResp.model_validate(pipeline_service.get_stage(stage_id))


@router.put("/{stage_id}", response_model=PipelineStageResp)
def update_stage(
    stage_id: str,
    data: PipelineStageUpdateReq,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineStageResp:
    stage = pipeline_service.update_stage(
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
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> None:
    pipeline_service.delete_stage(stage_id)


@router.post("/{stage_id}/duplicate", response_model=PipelineStageResp, status_code=201)
def duplicate_stage(
    stage_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineStageResp:
    stage = pipeline_service.duplicate_stage(stage_id)
    return PipelineStageResp.model_validate(stage)
