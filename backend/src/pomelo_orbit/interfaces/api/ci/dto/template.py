"""流水线模板相关 DTO"""

from datetime import datetime
from typing import Any

from pydantic import BaseModel


class ArtifactConfigDto(BaseModel):
    path: str
    name: str
    model_config = {"from_attributes": True}


# ── PipelineStage DTO ─────────────────────────────────────────────────────────


class PipelineStageResp(BaseModel):
    id: str
    name: str
    image: str
    script: str
    env: dict[str, str]
    artifacts: list[ArtifactConfigDto] | None
    description: str
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class PipelineStageCreateReq(BaseModel):
    name: str
    image: str
    script: str
    env: dict[str, str] = {}
    artifacts: list[ArtifactConfigDto] | None = None
    description: str = ""


class PipelineStageUpdateReq(BaseModel):
    name: str | None = None
    image: str | None = None
    script: str | None = None
    env: dict[str, str] | None = None
    artifacts: list[ArtifactConfigDto] | None = None
    description: str | None = None


# ── 编排 DTO ──────────────────────────────────────────────────────────────────


class StageOrchestrationDto(BaseModel):
    """模板对 Stage 的编排：引用 + 依赖 + 顺序"""

    stage_id: str
    depends_on: list[str] = []
    sort_order: int = 0

    model_config = {"from_attributes": True}


class OrchestrationUpdateReq(BaseModel):
    orchestration: list[StageOrchestrationDto]
    variable_declarations: list["VariableDeclarationDto"] = []


# ── 变量声明 DTO ──────────────────────────────────────────────────────────────


class VariableDeclarationDto(BaseModel):
    name: str
    description: str = ""
    required: bool = False
    default: Any = None
    secret: bool = False
    locked: bool = False

    model_config = {"from_attributes": True}


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
    stages_snapshot: list[dict]
    variable_declarations_snapshot: list[VariableDeclarationDto]
    created_at: datetime

    model_config = {"from_attributes": True}


# ── 模板 DTO ──────────────────────────────────────────────────────────────────


class PipelineTemplateResp(BaseModel):
    id: str
    name: str
    description: str
    orchestration: list[StageOrchestrationDto]
    stages: list[PipelineStageResp]  # 编排引用的 Stage 详情
    variable_declarations: list[VariableDeclarationDto]
    latest_snapshot_version: int | None = None
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class PipelineTemplateCreateReq(BaseModel):
    name: str
    description: str = ""
    variable_declarations: list[VariableDeclarationDto] = []


class PipelineTemplateUpdateReq(BaseModel):
    name: str | None = None
    description: str | None = None
    orchestration: list[StageOrchestrationDto] | None = None
    variable_declarations: list[VariableDeclarationDto] | None = None
