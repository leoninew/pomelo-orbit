"""
部署相关 Schema 定义
"""

from datetime import datetime

from pydantic import BaseModel


class DeploymentResp(BaseModel):
    """部署响应"""

    id: str
    application_id: str
    application_name: str | None = None
    operation_type: str
    trigger_type: str
    trigger_ref: str | None
    image_name: str | None
    env_file: str | None
    status: str
    started_at: datetime
    finished_at: datetime | None
    duration_ms: int | None
    error_message: str | None

    model_config = {"from_attributes": True}


class DeploymentDetailResp(DeploymentResp):
    """部署详情响应（包含日志）"""

    log_text: str | None
