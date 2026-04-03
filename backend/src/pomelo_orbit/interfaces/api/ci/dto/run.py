"""流水线运行相关 DTO"""

from datetime import datetime
from typing import Any

from pydantic import BaseModel


class PipelineRunResp(BaseModel):
    id: str
    project_id: str
    pipeline_snapshot_id: str
    trigger: str
    trigger_ref: str
    status: str
    retry_of: str | None = None
    started_at: datetime | None
    finished_at: datetime | None
    created_at: datetime

    model_config = {"from_attributes": True}


class TriggerPipelineReq(BaseModel):
    trigger_ref: str = ""
    variables: dict[str, Any] = {}


class ArtifactResp(BaseModel):
    id: str
    pipeline_run_id: str
    job_name: str
    type: str
    name: str
    path: str | None
    created_at: datetime

    model_config = {"from_attributes": True}
