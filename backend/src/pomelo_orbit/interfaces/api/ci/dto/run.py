"""流水线运行相关 DTO"""

from datetime import datetime
from typing import TYPE_CHECKING, Any

from pydantic import BaseModel

from pomelo_orbit.interfaces.api.ci.dto.stage_run import StageRunResp

if TYPE_CHECKING:
    from pomelo_orbit.application.ci.pipeline_service import PipelineService


class PipelineRunResp(BaseModel):
    id: str
    project_id: str
    project_name: str
    pipeline_snapshot_id: str
    template_id: str
    template_name: str
    trigger: str
    trigger_ref: str
    variables_snapshot: dict[str, Any]
    status: str
    retry_of: str | None = None
    started_at: datetime | None
    finished_at: datetime | None
    created_at: datetime
    stage_runs: list[StageRunResp] = []

    model_config = {"from_attributes": True}

    @classmethod
    def with_stage_runs(cls, run_id: str, pipeline_service: "PipelineService") -> "PipelineRunResp":
        """构建包含 stage_runs 的详情响应（详情/cancel/retry 场景使用）。"""
        run = pipeline_service.get_run(run_id)
        stage_runs = pipeline_service.list_stage_runs(run_id)
        resp = cls.model_validate(run)
        resp.stage_runs = [StageRunResp.model_validate(s) for s in stage_runs]
        return resp


class TriggerPipelineReq(BaseModel):
    template_id: str
    trigger_ref: str = ""
    variables: dict[str, Any] = {}


class ArtifactResp(BaseModel):
    id: str
    pipeline_run_id: str
    stage_name: str
    type: str
    name: str
    path: str | None
    created_at: datetime

    model_config = {"from_attributes": True}
