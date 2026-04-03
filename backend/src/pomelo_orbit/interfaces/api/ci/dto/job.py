"""作业相关 DTO"""

from datetime import datetime

from pydantic import BaseModel


class JobResp(BaseModel):
    id: str
    pipeline_run_id: str
    name: str
    status: str
    parent_job_id: str | None = None
    started_at: datetime | None = None
    finished_at: datetime | None = None
    error_message: str | None = None

    model_config = {"from_attributes": True}


class JobLogResp(BaseModel):
    id: str
    job_id: str
    content: str
    created_at: datetime

    model_config = {"from_attributes": True}
