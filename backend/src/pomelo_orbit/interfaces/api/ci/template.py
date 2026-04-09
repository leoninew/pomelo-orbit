"""流水线模板管理 API"""

import logging
import math
from typing import Annotated

from fastapi import APIRouter, Depends, Query

from pomelo_orbit.application.ci.di import get_pipeline_service
from pomelo_orbit.application.ci.pipeline_service import PipelineService
from pomelo_orbit.domain.ci.value_objects import VariableDeclaration
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.ci.dto.pipeline_template import (
    PipelineTemplateCreateReq,
    PipelineTemplateResp,
    PipelineTemplateUpdateReq,
)
from pomelo_orbit.interfaces.api.common import PaginatedResp

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/template", tags=["template"])


@router.get("", response_model=PaginatedResp[PipelineTemplateResp])
def list_templates(
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 20,
) -> PaginatedResp[PipelineTemplateResp]:
    items, total = pipeline_service.list_templates(page=page, per_page=per_page)
    return PaginatedResp(
        items=[PipelineTemplateResp.model_validate(t) for t in items],
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
    decls = [VariableDeclaration(**d.model_dump()) for d in data.variable_declarations]
    tmpl = pipeline_service.create_template(name=data.name, description=data.description, variable_declarations=decls)
    return PipelineTemplateResp.model_validate(tmpl)


@router.get("/{template_id}", response_model=PipelineTemplateResp)
def get_template(
    template_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineTemplateResp:
    tmpl = pipeline_service.get_template(template_id)
    return PipelineTemplateResp.model_validate(tmpl)


@router.put("/{template_id}", response_model=PipelineTemplateResp)
def update_template(
    template_id: str,
    data: PipelineTemplateUpdateReq,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineTemplateResp:
    decls = (
        [VariableDeclaration(**d.model_dump()) for d in data.variable_declarations]
        if data.variable_declarations is not None
        else None
    )
    orch = [o.model_dump() for o in data.orchestration] if data.orchestration is not None else None
    tmpl = pipeline_service.update_template(
        template_id=template_id,
        name=data.name,
        description=data.description,
        orchestration=orch,
        variable_declarations=decls,
    )
    return PipelineTemplateResp.model_validate(tmpl)


@router.delete("/{template_id}", status_code=204)
def delete_template(
    template_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> None:
    pipeline_service.delete_template(template_id)
