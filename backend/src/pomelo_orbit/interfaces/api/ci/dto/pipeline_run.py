"""流水线运行相关 DTO"""

from datetime import datetime
from typing import TYPE_CHECKING, Any

from pydantic import BaseModel, Field

from pomelo_orbit.interfaces.api.ci.dto.pipeline_stage_run import StageRunResp
from pomelo_orbit.interfaces.api.ci.dto.pipeline_template import VariableDeclarationDto

if TYPE_CHECKING:
    from pomelo_orbit.application.ci.pipeline_run_service import PipelineRunService


class PipelineRunResp(BaseModel):
    id: str
    repository_id: str
    repository_name: str
    snapshot_id: str
    template_id: str
    template_name: str
    template_version: int
    trigger: str
    trigger_ref: str
    variables_snapshot: list[VariableDeclarationDto]
    status: str
    retry_of: str | None = None
    started_at: datetime | None
    finished_at: datetime | None
    error_message: str | None = None
    created_at: datetime
    stage_runs: list[StageRunResp] = []

    model_config = {"from_attributes": True}

    @classmethod
    def with_stage_runs(cls, run_id: str, pipeline_run_service: "PipelineRunService") -> "PipelineRunResp":
        """构建包含 stage_runs 的详情响应（详情/cancel/retry 场景使用）。"""
        run = pipeline_run_service.get_run(run_id)
        stage_runs = pipeline_run_service.list_stage_runs(run_id)
        resp = cls.model_validate(run)
        resp.stage_runs = [StageRunResp.model_validate(s) for s in stage_runs]
        return resp


class TriggerPipelineReq(BaseModel):
    template_id: str = Field(min_length=1)
    trigger_ref: str = Field(min_length=1)
    variables: dict[str, Any] = Field(default_factory=dict)


class ArtifactResp(BaseModel):
    id: str
    pipeline_run_id: str
    stage_name: str
    type: str
    name: str
    path: str | None
    created_at: datetime

    model_config = {"from_attributes": True}
