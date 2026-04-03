"""Pipeline Snapshot API"""

import logging
from typing import Annotated

from fastapi import APIRouter, Depends

from pomelo_orbit.application.ci.di import get_pipeline_service
from pomelo_orbit.application.ci.pipeline_service import PipelineService
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.ci.dto.template import PipelineSnapshotResp

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/snapshots", tags=["snapshots"])


@router.get("/{snapshot_id}", response_model=PipelineSnapshotResp)
def get_snapshot(
    snapshot_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineSnapshotResp:
    """获取 Pipeline Snapshot 详情"""
    snapshot = pipeline_service.get_snapshot(snapshot_id)
    return PipelineSnapshotResp.model_validate(snapshot)
