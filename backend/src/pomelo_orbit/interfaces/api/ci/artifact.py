"""制品 API"""

import math
from typing import Annotated

from fastapi import APIRouter, Depends, Query

from pomelo_orbit.application.ci.di import get_pipeline_run_service
from pomelo_orbit.application.ci.pipeline_run_service import PipelineRunService
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.ci.dto.artifact import ArtifactResp
from pomelo_orbit.interfaces.api.common import PaginatedResp

router = APIRouter(prefix="/artifact", tags=["artifact"])


@router.get("", response_model=PaginatedResp[ArtifactResp])
def list_artifacts(
    pipeline_run_service: Annotated[PipelineRunService, Depends(get_pipeline_run_service)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 20,
    repository_id: Annotated[str | None, Query()] = None,
    template_id: Annotated[str | None, Query()] = None,
    search: Annotated[str | None, Query()] = None,
) -> PaginatedResp[ArtifactResp]:
    artifacts, total = pipeline_run_service.list_all_artifacts(
        repository_id=repository_id, template_id=template_id, search=search, page=page, per_page=per_page
    )
    return PaginatedResp(
        items=[ArtifactResp.model_validate(a) for a in artifacts],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )
