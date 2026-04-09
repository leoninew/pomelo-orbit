"""Webhook 相关 DTO"""

from datetime import datetime

from pydantic import BaseModel


class ProjectWebhookResp(BaseModel):
    id: str
    repository_id: str
    name: str
    template_id: str
    branch_filter: str | None
    enabled: bool
    created_at: datetime
    updated_at: datetime
    # encrypted_secret 不返回给前端

    model_config = {"from_attributes": True}


class ProjectWebhookCreateReq(BaseModel):
    name: str
    template_id: str
    secret: str
    branch_filter: str | None = None  # None 或空字符串表示拒绝所有分支，有值则用 glob 匹配


class ProjectWebhookUpdateReq(BaseModel):
    name: str | None = None
    template_id: str | None = None
    branch_filter: str | None = None
    secret: str | None = None  # 留空则不修改
    enabled: bool | None = None
