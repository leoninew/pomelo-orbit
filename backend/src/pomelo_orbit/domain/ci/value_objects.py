"""CI 值对象和枚举"""

from enum import StrEnum
from typing import Any

from pydantic import BaseModel


class CredentialType(StrEnum):
    GIT_SSH = "git_ssh"
    GIT_TOKEN = "git_token"


class PipelineRunTrigger(StrEnum):
    MANUAL = "manual"
    WEBHOOK = "webhook"


class VariableSource(StrEnum):
    """变量来源"""

    GLOBAL = "global"  # 全局内置变量
    REPOSITORY = "repository"  # 项目内置变量
    REPOSITORY_CUSTOM = "repository_custom"  # 项目自定义变量
    TEMPLATE = "template"  # 模板内置变量
    TEMPLATE_STAGE = "template_stage"  # 模板 Stage 发现的变量
    TEMPLATE_CUSTOM = "template_custom"  # 模板自定义变量
    RUNTIME = "runtime"  # 运行时临时变量


class VariableDeclaration(BaseModel):
    name: str
    description: str = ""
    value: Any = None
    secret: bool = False
    source: VariableSource = VariableSource.TEMPLATE_CUSTOM  # 默认为自定义


type BuiltinVariableSpecs = dict[str, str]


class ArtifactConfig(BaseModel):
    path: str
    name: str


class StageDefinition(BaseModel):
    """Stage 定义值对象：用于快照和执行，包含编排信息（已展开）"""

    name: str
    id: str
    image: str
    depends_on: list[str] = []  # 编排属性，存储依赖的 stage_id 列表
    script: str
    env: dict[str, str] = {}
    artifacts: list[ArtifactConfig] | None = None


class StageOrchestration(BaseModel):
    """模板对 Stage 的编排：引用 + 依赖 + 顺序。依赖和顺序属于编排，不属于 Stage 本身。"""

    stage_id: str
    stage_name: str  # 模板内唯一标识，默认为 stage 名，用于展示
    depends_on: list[str] = []  # 依赖的 stage_id 列表
    sort_order: int = 0  # 列表视图显示顺序
