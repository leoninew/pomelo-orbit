"""CI 值对象和枚举"""

from enum import StrEnum
from typing import Any

from pydantic import BaseModel


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
    CANCELED = "canceled"


class PipelineRunTrigger(StrEnum):
    """Pipeline 触发方式"""

    MANUAL = "manual"
    WEBHOOK = "webhook"


class JobStatus(StrEnum):
    """Job 状态"""

    WAITING = "waiting"
    RUNNING = "running"
    SUCCESS = "success"
    FAILED = "failed"
    FAULTED = "faulted"
    SKIPPED = "skipped"
    CANCELED = "canceled"


class RetryPolicy(StrEnum):
    """重试策略"""

    ALWAYS_RERUN = "always_rerun"
    SKIP_IF_SUCCESS = "skip_if_success"


class StageType(StrEnum):
    """Stage 类型"""

    CHECKOUT = "checkout"
    DOCKER_BUILD = "docker_build"
    UNIT_TEST = "unit_test"
    CUSTOM = "custom"


class VariableDeclaration(BaseModel):
    """变量声明"""

    name: str
    description: str = ""
    required: bool = False
    default: Any = None
    secret: bool = False
    locked: bool = False  # True 时运行时临时变量不可覆盖


class StepDefinition(BaseModel):
    """Step 定义（用于 custom stage 内部）"""

    name: str
    image: str | None = None
    commands: list[str] | None = None
    uses: str | None = None
    inputs: dict[str, Any] | None = None
    volumes: list[str] | None = None
    depends_on: list[str] | None = None
    timeout: int | None = None
    retry_policy: RetryPolicy | None = None
    artifacts: list[dict[str, Any]] | None = None
    outputs: list[str] | None = None
    steps: list["StepDefinition"] | None = None  # 嵌套


# ── Stage 配置类型 ──────────────────────────────────────────────────────────


class CheckoutStageConfig(BaseModel):
    """checkout stage 配置：仓库地址和凭据由项目属性自动注入"""

    ref: str = "{{ DEFAULT_BRANCH }}"


class DockerBuildStageConfig(BaseModel):
    """docker_build stage 配置：执行成功后产出 docker_image Artifact"""

    context: str = "."
    dockerfile: str = "Dockerfile"
    image_name: str  # 必填，支持变量占位符


class UnitTestStageConfig(BaseModel):
    """unit_test stage 配置：artifact_paths 非空时产出 file Artifact"""

    image: str  # 必填，支持变量占位符
    commands: list[str]
    artifact_paths: list[str] = []


class StageDefinition(BaseModel):
    """Stage 定义"""

    name: str
    type: StageType
    depends_on: list[str] = []
    # checkout / docker_build / unit_test 使用 config；custom 使用 steps
    config: CheckoutStageConfig | DockerBuildStageConfig | UnitTestStageConfig | None = None
    steps: list[StepDefinition] | None = None  # 仅 custom 类型


class PipelineDefinition(BaseModel):
    """Pipeline 定义（保留用于兼容现有执行器接口过渡期）"""

    version: str
    timeout: int | None = None
    steps: list[StepDefinition]
