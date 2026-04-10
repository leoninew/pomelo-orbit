"""Webhook 相关 DTO"""

from datetime import datetime

from pydantic import BaseModel, Field, field_validator


class ProjectWebhookResp(BaseModel):
    id: str
    repository_id: str
    name: str
    template_id: str
    branch_filter: str | None
    enabled: bool
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class ProjectWebhookCreateReq(BaseModel):
    name: str = Field(min_length=1)
    template_id: str = Field(min_length=1)
    secret: str = Field(min_length=1)
    branch_filter: str | None = None

    @field_validator("branch_filter")
    @classmethod
    def normalize_branch_filter(cls, value: str | None) -> str | None:
        if value is not None and not value.strip():
            return None
        return value


class ProjectWebhookUpdateReq(BaseModel):
    name: str | None = Field(default=None, min_length=1)
    template_id: str | None = Field(default=None, min_length=1)
    branch_filter: str | None = None
    secret: str | None = Field(default=None, min_length=1)
    enabled: bool | None = None

    @field_validator("branch_filter")
    @classmethod
    def normalize_branch_filter(cls, value: str | None) -> str | None:
        if value is not None and not value.strip():
            return None
        return value
