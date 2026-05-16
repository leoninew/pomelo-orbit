"""CI 值对象和枚举"""

from dataclasses import dataclass
from enum import StrEnum
from typing import Any

from pydantic import BaseModel


class CredentialType(StrEnum):
    """凭据类型枚举

    - GIT_SSH: Git SSH 私钥
    - GITHUB_TOKEN: GitHub Token，仅需 token
    - GITEE_TOKEN: Gitee 私人令牌（需要 username:token 格式）
    - REGISTRY_TOKEN: 镜像仓库 Token
    """

    GIT_SSH = "git_ssh"
    GITHUB_TOKEN = "github_token"
    GITEE_TOKEN = "gitee_token"
    REGISTRY_TOKEN = "registry_token"


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
    default: Any = None  # 系统/脚本提供的原始默认值，只读，用于展示和还原
    value: Any = None  # 用户的显式覆盖值，None 表示"未覆盖，运行时用 default"
    secret: bool = False
    source: VariableSource = VariableSource.TEMPLATE_CUSTOM
    editable: bool = True  # 展示层属性，由 VariableResolver 在返回时根据 source 设置，不参与持久化语义判断


type BuiltinVariableSpecs = dict[str, str]


class ArtifactType(StrEnum):
    DOCKER_IMAGE = "docker_image"
    BINARY = "binary"


@dataclass
class ArtifactConfig:
    type: ArtifactType
    path: str
    name: str


class StageDefinition(BaseModel):
    """Stage 定义值对象：用于快照和执行，包含编排信息（已展开）"""

    name: str
    id: str
    image: str
    version: int  # 快照时的 stage 版本
    depends_on: list[str] = []  # 编排属性，存储依赖的 stage_id 列表
    script: str
    artifacts: list[ArtifactConfig] | None = None


class StageOrchestration(BaseModel):
    """模板对 Stage 的编排：引用 + 依赖 + 顺序。依赖和顺序属于编排，不属于 Stage 本身。"""

    stage_id: str
    stage_name: str  # 模板内唯一标识，默认为 stage 名，用于展示
    stage_version: int  # 编排时记录的 stage 版本，用于检测 stage 是否有更新
    depends_on: list[str] = []  # 依赖的 stage_id 列表
    sort_order: int = 0  # 列表视图显示顺序
