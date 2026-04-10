"""Pipeline Snapshot API"""

import logging
from typing import Annotated

from fastapi import APIRouter, Depends

from pomelo_orbit.application.ci.di import get_template_service
from pomelo_orbit.application.ci.template_service import TemplateService
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.ci.dto.pipeline_template import PipelineSnapshotResp

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/snapshot", tags=["snapshot"])


@router.get("/{snapshot_id}", response_model=PipelineSnapshotResp)
def get_snapshot(
    snapshot_id: str,
    template_service: Annotated[TemplateService, Depends(get_template_service)],
    _current_user=Depends(get_current_user),
) -> PipelineSnapshotResp:
    """获取 Pipeline Snapshot 详情"""
    snapshot = template_service.get_snapshot(snapshot_id)
    return PipelineSnapshotResp.model_validate(snapshot)
