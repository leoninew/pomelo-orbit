"""Stage 执行记录 API"""

import logging
from typing import Annotated

from fastapi import APIRouter, Depends

from pomelo_orbit.application.ci.di import get_pipeline_service
from pomelo_orbit.application.ci.pipeline_service import PipelineService
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.ci.dto.stage_run import StageLogResp

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/stages", tags=["stages"])


@router.get("/{stage_run_id}/log", response_model=StageLogResp | None)
def get_stage_log(
    stage_run_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> StageLogResp | None:
    log = pipeline_service.get_stage_log(stage_run_id)
    return StageLogResp.model_validate(log) if log else None
