"""CI 值对象和枚举"""

from enum import StrEnum
from typing import Any

from pydantic import BaseModel


class CredentialType(StrEnum):
    GIT_SSH = "git_ssh"
    GIT_TOKEN = "git_token"


class PipelineRunStatus(StrEnum):
    WAITING = "waiting"
    RUNNING = "running"
    SUCCESS = "success"
    FAILED = "failed"
    CANCELED = "canceled"


class PipelineRunTrigger(StrEnum):
    MANUAL = "manual"
    WEBHOOK = "webhook"


class StageStatus(StrEnum):
    WAITING = "waiting"
    RUNNING = "running"
    SUCCESS = "success"
    FAILED = "failed"
    FAULTED = "faulted"
    SKIPPED = "skipped"
    CANCELED = "canceled"


class VariableDeclaration(BaseModel):
    name: str
    description: str = ""
    required: bool = False
    default: Any = None
    secret: bool = False
    locked: bool = False


class ArtifactConfig(BaseModel):
    path: str
    name: str


class StageDefinition(BaseModel):
    """Stage 定义值对象：用于快照和执行，包含编排信息（已展开）"""

    name: str
    image: str
    depends_on: list[str] = []  # 编排属性，来自 StageOrchestration
    script: str
    env: dict[str, str] = {}
    artifacts: list[ArtifactConfig] | None = None


class StageOrchestration(BaseModel):
    """模板对 Stage 的编排：引用 + 依赖 + 顺序。依赖和顺序属于编排，不属于 Stage 本身。"""

    stage_id: str
    depends_on: list[str] = []  # 依赖的 stage_id 列表，决定串并行
    sort_order: int = 0  # 列表视图显示顺序
