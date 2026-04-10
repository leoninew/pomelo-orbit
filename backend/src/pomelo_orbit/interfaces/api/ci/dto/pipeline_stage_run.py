"""Stage 执行记录相关 DTO"""

from datetime import datetime

from pydantic import BaseModel


class StageRunResp(BaseModel):
    id: str
    pipeline_run_id: str
    stage_id: str
    stage_name: str
    status: str
    started_at: datetime | None = None
    finished_at: datetime | None = None
    exit_code: int | None = None
    error_message: str | None = None

    model_config = {"from_attributes": True}
