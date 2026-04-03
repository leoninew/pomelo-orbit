"""流水线模板相关 DTO"""

from datetime import datetime
from typing import Any, Literal

from pydantic import BaseModel


class CheckoutConfigDto(BaseModel):
    ref: str = "{{ DEFAULT_BRANCH }}"

    model_config = {"from_attributes": True}


class DockerBuildConfigDto(BaseModel):
    context: str = "."
    dockerfile: str = "Dockerfile"
    image_name: str

    model_config = {"from_attributes": True}


class UnitTestConfigDto(BaseModel):
    image: str
    commands: list[str]
    artifact_paths: list[str] = []

    model_config = {"from_attributes": True}


class StageDefinitionDto(BaseModel):
    name: str
    type: Literal["checkout", "docker_build", "unit_test", "custom"]
    depends_on: list[str] = []
    config: CheckoutConfigDto | DockerBuildConfigDto | UnitTestConfigDto | None = None
    steps: list[dict[str, Any]] | None = None

    model_config = {"from_attributes": True}


class VariableDeclarationDto(BaseModel):
    name: str
    description: str = ""
    required: bool = False
    default: Any = None
    secret: bool = False
    locked: bool = False

    model_config = {"from_attributes": True}


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
