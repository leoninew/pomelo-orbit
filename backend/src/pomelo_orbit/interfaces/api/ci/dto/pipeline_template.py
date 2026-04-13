"""流水线模板相关 DTO"""

from datetime import datetime
from typing import Any

from pydantic import BaseModel, Field

from pomelo_orbit.interfaces.api.ci.dto.build_stage import BuildStageResp
from pomelo_orbit.interfaces.api.ci.dto.common import ArtifactConfigDto


class StageDefinitionDto(BaseModel):
    """Stage 定义 DTO：用于快照，包含编排信息"""

    id: str = Field(min_length=1)
    name: str = Field(min_length=1)
    image: str = Field(min_length=1)
    version: int
    depends_on: list[str] = Field(default_factory=list)
    script: str = Field(min_length=1)
    artifacts: list[ArtifactConfigDto] | None = None

    model_config = {"from_attributes": True}


# ── 编排 DTO ──────────────────────────────────────────────────────────────────


class StageOrchestrationDto(BaseModel):
    """模板对 Stage 的编排：引用 + 依赖 + 顺序"""

    stage_id: str = Field(min_length=1)
    stage_name: str = Field(min_length=1)
    stage_version: int
    depends_on: list[str] = Field(default_factory=list)
    sort_order: int = 0

    model_config = {"from_attributes": True}


class OrchestrationUpdateReq(BaseModel):
    orchestration: list[StageOrchestrationDto]
    variable_declarations: list["VariableDeclarationDto"] = Field(default_factory=list)


# ── 变量声明 DTO ──────────────────────────────────────────────────────────────


class VariableDeclarationDto(BaseModel):
    name: str = Field(min_length=1)
    description: str = ""
    default: Any = None
    value: Any = None
    secret: bool = False
    source: str = "template_custom"
    editable: bool = True

    model_config = {"from_attributes": True}


class TemplateVariableResolveReq(BaseModel):
    orchestration: list[StageOrchestrationDto]
    variable_declarations: list["VariableDeclarationDto"] = Field(default_factory=list)


# ── 快照 DTO ──────────────────────────────────────────────────────────────────


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
    variables_snapshot: list[VariableDeclarationDto]
    created_at: datetime

    model_config = {"from_attributes": True}


# ── 模板 DTO ──────────────────────────────────────────────────────────────────


class PipelineTemplateResp(BaseModel):
    id: str
    name: str
    description: str
    orchestration: list[StageOrchestrationDto]
    stages: list[BuildStageResp]
    variable_declarations: list[VariableDeclarationDto]
    version: int
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}

    @classmethod
    def from_domain(cls, tmpl, variable_declarations: list) -> "PipelineTemplateResp":
        return cls(
            id=tmpl.id,
            name=tmpl.name,
            description=tmpl.description,
            orchestration=tmpl.orchestration,
            stages=tmpl.stages,
            variable_declarations=[VariableDeclarationDto.model_validate(v) for v in variable_declarations],
            version=tmpl.version,
            created_at=tmpl.created_at,
            updated_at=tmpl.updated_at,
        )


class PipelineTemplateCreateReq(BaseModel):
    name: str = Field(min_length=1)
    description: str = ""
    variable_declarations: list[VariableDeclarationDto] = Field(default_factory=list)


class PipelineTemplateUpdateReq(BaseModel):
    name: str | None = Field(default=None, min_length=1)
    description: str | None = None
    orchestration: list[StageOrchestrationDto] | None = None
    variable_declarations: list[VariableDeclarationDto] | None = None
