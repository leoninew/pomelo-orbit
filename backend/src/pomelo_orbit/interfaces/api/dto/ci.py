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
    type: Literal["git_ssh", "git_token", "registry_token"]
    data: str


# ---- PipelineTemplate ----

class VariableDeclarationResp(BaseModel):
    name: str
    description: str
    required: bool
    default: Any
    secret: bool

    model_config = {"from_attributes": True}


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
    webhook_secret: str | None
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
    repository_url: str | None = None
    pipeline_template_id: str | None = None
    git_credential_id: str | None = None
    variable_overrides: dict[str, Any] | None = None
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
    retry_of: str | None = None
    started_at: datetime | None
    finished_at: datetime | None
    created_at: datetime

    model_config = {"from_attributes": True}


class TriggerPipelineReq(BaseModel):
    trigger_ref: str = "main"
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


# ---- Job ----

class JobResp(BaseModel):
    id: str
    pipeline_run_id: str
    name: str
    status: str
    parent_job_id: str | None = None
    started_at: datetime | None = None
    finished_at: datetime | None = None
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class JobLogResp(BaseModel):
    id: str
    job_id: str
    content: str
    created_at: datetime

    model_config = {"from_attributes": True}
