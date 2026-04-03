"""流水线模板管理 API"""

import logging
import math
from typing import Annotated

from fastapi import APIRouter, Depends, Query

from pomelo_orbit.application.ci.di import get_pipeline_service
from pomelo_orbit.application.ci.pipeline_service import PipelineService
from pomelo_orbit.domain.ci.value_objects import StageDefinition, VariableDeclaration
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.ci.dto.template import (
    PipelineSnapshotListItemResp,
    PipelineTemplateCreateReq,
    PipelineTemplateResp,
    PipelineTemplateUpdateReq,
)
from pomelo_orbit.interfaces.api.common import PaginatedResp

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/templates", tags=["templates"])


@router.get("", response_model=PaginatedResp[PipelineTemplateResp])
def list_templates(
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 20,
) -> PaginatedResp[PipelineTemplateResp]:
    items_with_version, total = pipeline_service.list_templates_with_latest_version(page=page, per_page=per_page)
    items = []
    for tmpl, latest_version in items_with_version:
        resp = PipelineTemplateResp.model_validate(tmpl)
        resp = resp.model_copy(update={"latest_snapshot_version": latest_version})
        items.append(resp)
    return PaginatedResp(
        items=items,
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.post("", response_model=PipelineTemplateResp, status_code=201)
def create_template(
    data: PipelineTemplateCreateReq,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineTemplateResp:
    stages = [StageDefinition(**s.model_dump()) for s in data.stages]
    decls = [VariableDeclaration(**d.model_dump()) for d in data.variable_declarations]
    tmpl = pipeline_service.create_template(
        name=data.name, stages=stages, description=data.description, variable_declarations=decls
    )
    resp = PipelineTemplateResp.model_validate(tmpl)
    return resp.model_copy(update={"latest_snapshot_version": 1})


@router.get("/{template_id}", response_model=PipelineTemplateResp)
def get_template(
    template_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineTemplateResp:
    tmpl = pipeline_service.get_template(template_id)
    latest = pipeline_service.get_template_latest_version(template_id)
    resp = PipelineTemplateResp.model_validate(tmpl)
    return resp.model_copy(update={"latest_snapshot_version": latest})


@router.put("/{template_id}", response_model=PipelineTemplateResp)
def update_template(
    template_id: str,
    data: PipelineTemplateUpdateReq,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineTemplateResp:
    stages = [StageDefinition(**s.model_dump()) for s in data.stages] if data.stages is not None else None
    decls = (
        [VariableDeclaration(**d.model_dump()) for d in data.variable_declarations]
        if data.variable_declarations is not None
        else None
    )
    tmpl = pipeline_service.update_template(
        template_id=template_id,
        name=data.name,
        description=data.description,
        stages=stages,
        variable_declarations=decls,
    )
    latest = pipeline_service.get_template_latest_version(template_id)
    resp = PipelineTemplateResp.model_validate(tmpl)
    return resp.model_copy(update={"latest_snapshot_version": latest})


@router.delete("/{template_id}", status_code=204)
def delete_template(
    template_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> None:
    pipeline_service.delete_template(template_id)


@router.get("/{template_id}/snapshots", response_model=list[PipelineSnapshotListItemResp])
def list_template_snapshots(
    template_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> list[PipelineSnapshotListItemResp]:
    """列出指定模板的所有快照"""
    snapshots = pipeline_service.list_template_snapshots(template_id)
    return [PipelineSnapshotListItemResp.model_validate(s) for s in snapshots]
