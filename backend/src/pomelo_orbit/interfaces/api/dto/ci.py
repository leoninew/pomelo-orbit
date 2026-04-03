"""CI 模块 DTO"""

from datetime import datetime
from typing import Any, Literal

from pydantic import BaseModel

# ── Credential ────────────────────────────────────────────────────────────────


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


class CredentialUpdateReq(BaseModel):
    name: str | None = None
    data: str | None = None


# ── Stage ─────────────────────────────────────────────────────────────────────


class CheckoutConfigDto(BaseModel):
    ref: str = "{{ DEFAULT_BRANCH }}"


class DockerBuildConfigDto(BaseModel):
    context: str = "."
    dockerfile: str = "Dockerfile"
    image_name: str


class UnitTestConfigDto(BaseModel):
    image: str
    commands: list[str]
    artifact_paths: list[str] = []


class StageDefinitionDto(BaseModel):
    name: str
    type: Literal["checkout", "docker_build", "unit_test", "custom"]
    depends_on: list[str] = []
    config: CheckoutConfigDto | DockerBuildConfigDto | UnitTestConfigDto | None = None
    steps: list[dict[str, Any]] | None = None  # custom 类型的 StepDefinition


# ── VariableDeclaration ───────────────────────────────────────────────────────


class VariableDeclarationDto(BaseModel):
    name: str
    description: str = ""
    required: bool = False
    default: Any = None
    secret: bool = False
    locked: bool = False


# ── PipelineTemplate ──────────────────────────────────────────────────────────


class PipelineTemplateResp(BaseModel):
    id: str
    name: str
    description: str
    stages: list[StageDefinitionDto]
    variable_declarations: list[VariableDeclarationDto]
    is_builtin: bool
    latest_snapshot_version: int | None = None
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class PipelineTemplateCreateReq(BaseModel):
    name: str
    description: str = ""
    stages: list[StageDefinitionDto] = []
    variable_declarations: list[VariableDeclarationDto] = []


class PipelineTemplateUpdateReq(BaseModel):
    name: str | None = None
    description: str | None = None
    stages: list[StageDefinitionDto] | None = None
    variable_declarations: list[VariableDeclarationDto] | None = None


# ── PipelineSnapshot ──────────────────────────────────────────────────────────


class PipelineSnapshotListItemResp(BaseModel):
    id: str
    template_id: str
    version: int
    created_at: datetime

    model_config = {"from_attributes": True}


class PipelineSnapshotResp(BaseModel):
    id: str
    template_id: str
    version: int
    stages_snapshot: list[StageDefinitionDto]
    variable_declarations_snapshot: list[VariableDeclarationDto]
    created_at: datetime

    model_config = {"from_attributes": True}


# ── Project ───────────────────────────────────────────────────────────────────


class ProjectResp(BaseModel):
    id: str
    name: str
    repository_url: str
    pipeline_snapshot_id: str
    git_credential_id: str | None
    variable_overrides: dict[str, Any]
    default_branch: str
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class ProjectCreateReq(BaseModel):
    name: str
    repository_url: str
    pipeline_snapshot_id: str
    git_credential_id: str | None = None
    variable_overrides: dict[str, Any] = {}
    default_branch: str = "master"


class ProjectUpdateReq(BaseModel):
    name: str | None = None
    repository_url: str | None = None
    pipeline_snapshot_id: str | None = None
    git_credential_id: str | None = None
    variable_overrides: dict[str, Any] | None = None
    default_branch: str | None = None


# ── PipelineRun ───────────────────────────────────────────────────────────────


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


# ── Job ───────────────────────────────────────────────────────────────────────


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


# ── Artifact ──────────────────────────────────────────────────────────────────


class ArtifactResp(BaseModel):
    id: str
    pipeline_run_id: str
    job_name: str
    type: str
    name: str
    path: str | None
    created_at: datetime

    model_config = {"from_attributes": True}
