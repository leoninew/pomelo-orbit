"""构建 Stage 管理 API"""

import math
from typing import Annotated

from fastapi import APIRouter, Depends, Query

from pomelo_orbit.application.ci.build_stage_service import BuildStageService
from pomelo_orbit.application.ci.di import get_stage_service
from pomelo_orbit.domain.project.entities import Project
from pomelo_orbit.interfaces.api.ci.dto.build_stage import (
    BuildStageCreateReq,
    BuildStageResp,
    BuildStageUpdateReq,
)
from pomelo_orbit.interfaces.api.common import PaginatedResp
from pomelo_orbit.interfaces.api.project.dependencies import get_current_project

router = APIRouter(prefix="/build-stage", tags=["build-stage"])


@router.get("", response_model=PaginatedResp[BuildStageResp])
def list_stages(
    stage_service: Annotated[BuildStageService, Depends(get_stage_service)],
    current_project: Annotated[Project, Depends(get_current_project)],
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 20,
    search: Annotated[str | None, Query()] = None,
) -> PaginatedResp[BuildStageResp]:
    stages, total = stage_service.list_stages(current_project.id, page=page, per_page=per_page, search=search)
    return PaginatedResp(
        items=[BuildStageResp.model_validate(s) for s in stages],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.post("", response_model=BuildStageResp, status_code=201)
def create_stage(
    data: BuildStageCreateReq,
    stage_service: Annotated[BuildStageService, Depends(get_stage_service)],
    current_project: Annotated[Project, Depends(get_current_project)],
) -> BuildStageResp:
    stage = stage_service.create_stage(
        project_id=current_project.id,
        name=data.name,
        image=data.image,
        script=data.script,
        artifacts=[a.to_artifact_config() for a in data.artifacts] if data.artifacts else None,
        description=data.description,
    )
    return BuildStageResp.model_validate(stage)


@router.get("/{stage_id}", response_model=BuildStageResp)
def get_stage(
    stage_id: str,
    stage_service: Annotated[BuildStageService, Depends(get_stage_service)],
) -> BuildStageResp:
    return BuildStageResp.model_validate(stage_service.get_stage(stage_id))


@router.put("/{stage_id}", response_model=BuildStageResp)
def update_stage(
    stage_id: str,
    data: BuildStageUpdateReq,
    stage_service: Annotated[BuildStageService, Depends(get_stage_service)],
) -> BuildStageResp:
    stage = stage_service.update_stage(
        stage_id=stage_id,
        name=data.name,
        image=data.image,
        script=data.script,
        artifacts=[a.to_artifact_config() for a in data.artifacts] if data.artifacts is not None else None,
        description=data.description,
    )
    return BuildStageResp.model_validate(stage)


@router.delete("/{stage_id}", status_code=204)
def delete_stage(
    stage_id: str,
    stage_service: Annotated[BuildStageService, Depends(get_stage_service)],
) -> None:
    stage_service.delete_stage(stage_id)


@router.post("/{stage_id}/duplicate", response_model=BuildStageResp, status_code=201)
def duplicate_stage(
    stage_id: str,
    stage_service: Annotated[BuildStageService, Depends(get_stage_service)],
) -> BuildStageResp:
    stage = stage_service.duplicate_stage(stage_id)
    return BuildStageResp.model_validate(stage)
