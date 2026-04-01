"""CI 模块 DTO"""

from datetime import datetime
from typing import Any, Literal

from pydantic import BaseModel

# ---- Credential ----

class CredentialResp(BaseModel):
    id: str
    name: str
    type: str
    created_at: datetime

    model_config = {"from_attributes": True}


class CredentialCreateReq(BaseModel):
    name: str
    type: Literal["git_ssh", "git_token"]
    data: str  # 明文，由 API 层加密后存储


# ---- PipelineTemplate ----

class VariableDeclarationResp(BaseModel):
    name: str
    description: str
    required: bool
    default: Any
    secret: bool


class PipelineTemplateResp(BaseModel):
    id: str
    name: str
    description: str
    content: str
    variable_declarations: list[VariableDeclarationResp]
    is_builtin: bool
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class PipelineTemplateCreateReq(BaseModel):
    name: str
    content: str
    description: str = ""
    variable_declarations: list[dict] = []


class PipelineTemplateUpdateReq(BaseModel):
    name: str | None = None
    description: str | None = None
    content: str | None = None
    variable_declarations: list[dict] | None = None


# ---- Project ----

class ProjectResp(BaseModel):
    id: str
    name: str
    repository_url: str
    pipeline_template_id: str
    git_credential_id: str
    variable_overrides: dict[str, Any]
    branch_filter: str | None
    # webhook_secret 不回显
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class ProjectCreateReq(BaseModel):
    name: str
    repository_url: str
    pipeline_template_id: str
    git_credential_id: str
    variable_overrides: dict[str, Any] = {}
    branch_filter: str | None = None
    enable_webhook: bool = True


class ProjectUpdateReq(BaseModel):
    name: str | None = None
    variable_overrides: dict[str, Any] | None = None
    pipeline_template_id: str | None = None
    branch_filter: str | None = None


class WebhookConfigResp(BaseModel):
    url: str
    secret: str | None
    events: list[str]


# ---- PipelineRun ----

class PipelineRunResp(BaseModel):
    id: str
    project_id: str
    trigger: str
    trigger_ref: str
    status: str
    started_at: datetime | None
    finished_at: datetime | None
    created_at: datetime

    model_config = {"from_attributes": True}


class TriggerPipelineReq(BaseModel):
    trigger_ref: str
    variables: dict[str, Any] = {}
