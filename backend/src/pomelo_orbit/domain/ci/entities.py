"""CI 实体"""

from copy import deepcopy
from dataclasses import dataclass, field
from datetime import datetime
from typing import Any

from ulid import ULID

from pomelo_orbit.domain.cd.value_objects import TaskStatus
from pomelo_orbit.domain.ci.value_objects import (
    ArtifactConfig,
    CredentialType,
    PipelineRunTrigger,
    StageDefinition,
    StageOrchestration,
    VariableDeclaration,
)
from pomelo_orbit.infrastructure.time_utils import utc_now


@dataclass
class Project:
    """项目实体"""

    id: str
    name: str
    code: str  # 短标识符，固化工作目录路径，创建后不可修改
    repository_url: str
    variable_overrides: dict[str, Any]
    git_credential_id: str | None = None
    default_branch: str = "master"
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(
        name: str,
        code: str,
        repository_url: str,
        git_credential_id: str | None = None,
        variable_overrides: dict[str, Any] | None = None,
        default_branch: str = "master",
    ) -> "Project":
        now = utc_now()
        return Project(
            id=str(ULID()),
            name=name,
            code=code,
            repository_url=repository_url,
            git_credential_id=git_credential_id,
            variable_overrides=variable_overrides or {},
            default_branch=default_branch,
            created_at=now,
            updated_at=now,
        )

    def update(
        self,
        name: str | None = None,
        repository_url: str | None = None,
        variable_overrides: dict[str, Any] | None = None,
        git_credential_id: str | None = None,
        default_branch: str | None = None,
    ) -> None:
        if name is not None:
            self.name = name
        if repository_url is not None:
            self.repository_url = repository_url
        if variable_overrides is not None:
            self.variable_overrides = variable_overrides
        if git_credential_id is not None:
            self.git_credential_id = git_credential_id
        if default_branch is not None:
            self.default_branch = default_branch
        self.updated_at = utc_now()


@dataclass
class ProjectWebhook:
    """项目 Webhook 配置：每个 Webhook 绑定一个模板，有独立的签名密钥"""

    id: str
    project_id: str
    name: str
    template_id: str
    branch_filter: (
        str | None
    )  # None 或空字符串表示拒绝所有分支；"*" 表示接受所有分支；其他值用 glob 匹配（如 "main", "release/*"）
    encrypted_secret: str  # 加密存储的 HMAC 密钥
    enabled: bool
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(
        project_id: str,
        name: str,
        template_id: str,
        encrypted_secret: str,
        branch_filter: str | None = None,
    ) -> "ProjectWebhook":
        now = utc_now()
        return ProjectWebhook(
            id=str(ULID()),
            project_id=project_id,
            name=name,
            template_id=template_id,
            branch_filter=branch_filter,
            encrypted_secret=encrypted_secret,
            enabled=True,
            created_at=now,
            updated_at=now,
        )

    def update(
        self,
        name: str | None = None,
        template_id: str | None = None,
        branch_filter: str | None = None,
        encrypted_secret: str | None = None,
        enabled: bool | None = None,
    ) -> None:
        if name is not None:
            self.name = name
        if template_id is not None:
            self.template_id = template_id
        self.branch_filter = branch_filter or None  # 空字符串统一转 None，表示拒绝所有分支
        if encrypted_secret is not None:
            self.encrypted_secret = encrypted_secret
        if enabled is not None:
            self.enabled = enabled
        self.updated_at = utc_now()


@dataclass
class Credential:
    """凭据实体"""

    id: str
    name: str
    type: CredentialType
    encrypted_data: str
    created_at: datetime = field(default_factory=utc_now)

    def get_private_key(self) -> str:
        if self.type != CredentialType.GIT_SSH:
            raise ValueError(f"Credential type {self.type} has no private key")
        return self.encrypted_data

    def get_token(self) -> str:
        if self.type != CredentialType.GIT_TOKEN:
            raise ValueError(f"Credential type {self.type} has no token")
        return self.encrypted_data

    @staticmethod
    def create(name: str, type: CredentialType, encrypted_data: str) -> "Credential":
        return Credential(id=str(ULID()), name=name, type=type, encrypted_data=encrypted_data)


@dataclass
class PipelineStage:
    """流水线 Stage 实体：执行的最小单元，描述"做什么"。

    不含任何编排信息（depends_on、sort_order 属于 PipelineTemplate 的编排，不属于 Stage）。
    """

    id: str
    name: str
    image: str
    script: str
    env: dict[str, str]
    artifacts: list | None = None  # list[ArtifactConfig]
    description: str = ""
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(
        name: str,
        image: str,
        script: str,
        env: dict[str, str] | None = None,
        artifacts: list | None = None,
        description: str = "",
    ) -> "PipelineStage":
        now = utc_now()
        return PipelineStage(
            id=str(ULID()),
            name=name,
            image=image,
            script=script,
            env=env or {},
            artifacts=artifacts,
            description=description,
            created_at=now,
            updated_at=now,
        )

    def update(
        self,
        name: str | None = None,
        image: str | None = None,
        script: str | None = None,
        env: dict[str, str] | None = None,
        artifacts: list | None = None,
        description: str | None = None,
    ) -> None:
        if name is not None:
            self.name = name
        if image is not None:
            self.image = image
        if script is not None:
            self.script = script
        if env is not None:
            self.env = env
        if artifacts is not None:
            self.artifacts = artifacts
        if description is not None:
            self.description = description
        self.updated_at = utc_now()

    def to_stage_definition(self, name: str | None = None, depends_on: list[str] | None = None) -> "StageDefinition":
        """转换为值对象，用于快照和执行。name 和 depends_on 由编排层传入。"""
        return StageDefinition(
            name=name or self.name,
            id=self.id,
            image=self.image,
            depends_on=depends_on or [],
            script=self.script,
            env=self.env,
            artifacts=[ArtifactConfig(**a) if isinstance(a, dict) else a for a in self.artifacts]
            if self.artifacts
            else None,
        )


@dataclass
class PipelineTemplate:
    """流水线模板实体：将一组 Stage 编排起来，定义依赖关系和执行顺序，配合变量声明可以运行。"""

    id: str
    name: str
    orchestration: list["StageOrchestration"]  # 编排：stage_id + depends_on + sort_order
    stages: list["PipelineStage"]  # 编排引用的 Stage 实体（加载时填充）
    variable_declarations: list[VariableDeclaration]
    description: str = ""
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(
        name: str,
        variable_declarations: list[VariableDeclaration],
        description: str = "",
    ) -> "PipelineTemplate":
        if not name:
            raise ValueError("Template name cannot be empty")
        return PipelineTemplate(
            id=str(ULID()),
            name=name,
            orchestration=[],
            stages=[],
            variable_declarations=variable_declarations,
            description=description,
        )

    def update(
        self,
        name: str | None = None,
        description: str | None = None,
        variable_declarations: list[VariableDeclaration] | None = None,
    ) -> None:
        if name is not None:
            if not name:
                raise ValueError("Template name cannot be empty")
            self.name = name
        if description is not None:
            self.description = description
        if variable_declarations is not None:
            self.variable_declarations = variable_declarations
        self.updated_at = utc_now()

    def get_stage_definitions(self) -> list[StageDefinition]:
        """按 sort_order 排序，将编排 + Stage 内容合并为 StageDefinition 列表，用于快照和执行。"""
        stage_map = {s.id: s for s in self.stages}
        sorted_orch = sorted(self.orchestration, key=lambda o: o.sort_order)
        result = []
        for orch in sorted_orch:
            stage = stage_map.get(orch.stage_id)
            if stage:
                # StageDefinition.name 使用 stage_key，执行器用它做依赖解析
                result.append(stage.to_stage_definition(name=orch.stage_key, depends_on=orch.depends_on))
        return result


@dataclass
class PipelineSnapshot:
    """流水线快照：模板某一版本的不可变副本"""

    id: str
    template_id: str
    version: int  # 从 1 开始，单调递增
    stages_snapshot: list[StageDefinition]
    variable_declarations_snapshot: list[VariableDeclaration]
    created_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(template: "PipelineTemplate", version: int) -> "PipelineSnapshot":
        return PipelineSnapshot(
            id=str(ULID()),
            template_id=template.id,
            version=version,
            stages_snapshot=deepcopy(template.get_stage_definitions()),
            variable_declarations_snapshot=deepcopy(template.variable_declarations),
        )


@dataclass
class PipelineRun:
    """Pipeline 运行实例"""

    id: str
    project_id: str
    project_name: str
    pipeline_snapshot_id: str
    template_id: str
    template_name: str
    trigger: PipelineRunTrigger
    trigger_ref: str
    variables_snapshot: dict[str, Any]
    status: TaskStatus = TaskStatus.WAITING_TO_RUN
    retry_of: str | None = None
    started_at: datetime | None = None
    finished_at: datetime | None = None
    created_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(
        project_id: str,
        project_name: str,
        pipeline_snapshot_id: str,
        template_id: str,
        template_name: str,
        trigger: PipelineRunTrigger,
        trigger_ref: str,
        variables_snapshot: dict[str, Any],
        retry_of: str | None = None,
    ) -> "PipelineRun":
        return PipelineRun(
            id=str(ULID()),
            project_id=project_id,
            project_name=project_name,
            pipeline_snapshot_id=pipeline_snapshot_id,
            template_id=template_id,
            template_name=template_name,
            trigger=trigger,
            trigger_ref=trigger_ref,
            variables_snapshot=variables_snapshot,
            retry_of=retry_of,
        )

    def start(self) -> None:
        self.status = TaskStatus.RUNNING
        self.started_at = utc_now()

    def complete_success(self) -> None:
        self.status = TaskStatus.RAN_TO_COMPLETION
        self.finished_at = utc_now()

    def complete_failed(self) -> None:
        self.status = TaskStatus.FAULTED
        self.finished_at = utc_now()

    def cancel(self) -> None:
        if self.status not in (TaskStatus.WAITING_TO_RUN, TaskStatus.RUNNING):
            raise ValueError(f"Cannot cancel run with status {self.status}")
        self.status = TaskStatus.CANCELED
        self.finished_at = utc_now()


@dataclass
class StageRun:
    """Stage 执行记录（1 Stage = 1 StageRun per PipelineRun）"""

    id: str
    pipeline_run_id: str
    name: str  # 对应 StageDefinition.name
    status: TaskStatus = TaskStatus.WAITING_TO_RUN
    started_at: datetime | None = None
    finished_at: datetime | None = None
    exit_code: int | None = None
    error_message: str | None = None

    @staticmethod
    def create(pipeline_run_id: str, name: str) -> "StageRun":
        return StageRun(id=str(ULID()), pipeline_run_id=pipeline_run_id, name=name)

    def start(self) -> None:
        self.status = TaskStatus.RUNNING
        self.started_at = utc_now()

    def complete_success(self, exit_code: int = 0) -> None:
        self.status = TaskStatus.RAN_TO_COMPLETION
        self.exit_code = exit_code
        self.finished_at = utc_now()

    def complete_failed(self, exit_code: int, error_message: str | None = None) -> None:
        self.status = TaskStatus.FAULTED
        self.exit_code = exit_code
        self.error_message = error_message
        self.finished_at = utc_now()

    def complete_faulted(self, error_message: str) -> None:
        self.status = TaskStatus.FAULTED
        self.error_message = error_message
        self.finished_at = utc_now()


@dataclass
class Artifact:
    """制品记录"""

    id: str
    pipeline_run_id: str
    stage_name: str
    type: str  # docker_image | file
    name: str
    path: str | None = None
    created_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(
        pipeline_run_id: str,
        stage_name: str,
        artifact_type: str,
        name: str,
        path: str | None = None,
    ) -> "Artifact":
        return Artifact(
            id=str(ULID()),
            pipeline_run_id=pipeline_run_id,
            stage_name=stage_name,
            type=artifact_type,
            name=name,
            path=path,
        )
