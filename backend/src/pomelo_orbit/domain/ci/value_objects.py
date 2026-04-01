"""CI 值对象和枚举"""

from enum import StrEnum
from typing import Any

from pydantic import BaseModel, Field


class CredentialType(StrEnum):
    """凭据类型"""

    GIT_SSH = "git_ssh"
    GIT_TOKEN = "git_token"


class PipelineRunStatus(StrEnum):
    """Pipeline 运行状态"""

    WAITING = "waiting"
    RUNNING = "running"
    SUCCESS = "success"
    FAILED = "failed"


class PipelineRunTrigger(StrEnum):
    """Pipeline 触发方式"""

    MANUAL = "manual"
    WEBHOOK = "webhook"  # Phase 2


class JobStatus(StrEnum):
    """Job 状态"""

    WAITING = "waiting"
    RUNNING = "running"
    SUCCESS = "success"
    FAILED = "failed"
    FAULTED = "faulted"
    SKIPPED = "skipped"  # Phase 2
    CANCELED = "canceled"  # Phase 3


class VariableDeclaration(BaseModel):
    """变量声明"""

    name: str
    description: str = ""
    required: bool = False
    default: Any = None
    secret: bool = False  # 是否敏感数据


class StepDefinition(BaseModel):
    """Step 定义（递归）"""

    model_config = {"populate_by_name": True}

    name: str
    image: str | None = None
    commands: list[str] | None = None
    uses: str | None = None  # "checkout"
    with_: dict[str, Any] | None = Field(None, alias="with")
    volumes: list[str] | None = None
    depends_on: list[str] | None = None  # Phase 2
    timeout: int | None = None  # Phase 3
    retry_policy: str | None = None  # Phase 2
    artifacts: list[dict[str, Any]] | None = None  # Phase 2
    outputs: list[str] | None = None
    steps: list["StepDefinition"] | None = None  # 嵌套


class PipelineDefinition(BaseModel):
    """Pipeline 定义"""

    version: str
    timeout: int | None = None
    steps: list[StepDefinition]
