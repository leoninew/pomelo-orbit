"""
回调事件相关 Schema 定义
"""

from datetime import datetime

from pydantic import BaseModel


class WebhookEventResp(BaseModel):
    """回调事件响应"""

    id: str
    source: str
    event_type: str
    repository_name: str | None
    repository_url: str | None
    branch: str | None
    sender: str | None
    signature_valid: bool | None
    status: str
    matched_application_id: str | None
    triggered_deployment_id: str | None
    error_message: str | None
    received_at: datetime
    processed_at: datetime | None

    model_config = {"from_attributes": True}


class WebhookEventDetailResp(WebhookEventResp):
    """回调事件详情响应（包含 payload）"""

    payload: str | None
