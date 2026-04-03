"""作业日志 API"""

import logging
from typing import Annotated

from fastapi import APIRouter, Depends

from pomelo_orbit.application.ci.di import get_pipeline_service
from pomelo_orbit.application.ci.pipeline_service import PipelineService
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.ci.dto.job import JobLogResp

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/jobs", tags=["jobs"])


@router.get("/{job_id}/logs", response_model=JobLogResp | None)
def get_job_logs(
    job_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> JobLogResp | None:
    log = pipeline_service.get_job_log(job_id)
    return JobLogResp.model_validate(log) if log else None
