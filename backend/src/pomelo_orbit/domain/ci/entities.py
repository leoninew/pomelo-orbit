"""CI 实体"""

from copy import deepcopy
from dataclasses import MISSING, dataclass, field
from datetime import datetime

from ulid import ULID

from pomelo_orbit.domain.cd.value_objects import TaskStatus
from pomelo_orbit.domain.ci.value_objects import (
    ArtifactConfig,
    ArtifactType,
    CredentialType,
    PipelineRunTrigger,
    StageDefinition,
    StageOrchestration,
    VariableDeclaration,
)
from pomelo_orbit.infrastructure.time_utils import utc_now


@dataclass
class Repository:
    """项目实体"""

    id: str
    name: str
    code: str  # 短标识符，固化工作目录路径，创建后不可修改
    repository_url: str
    variable_overrides: list[VariableDeclaration]  # 项目自定义变量列表
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
        variable_overrides: list[VariableDeclaration] | None = None,
        default_branch: str = "master",
    ) -> "Repository":
        now = utc_now()
        return Repository(
            id=str(ULID()),
            name=name,
            code=code,
            repository_url=repository_url,
            git_credential_id=git_credential_id,
            variable_overrides=variable_overrides or [],
            default_branch=default_branch,
            created_at=now,
            updated_at=now,
        )

    def update(
        self,
        name: str | None = None,
        repository_url: str | None = None,
        variable_overrides: list[VariableDeclaration] | None = None,
        git_credential_id: str | None = MISSING,  # type: ignore[assignment]  # MISSING 作哨兵，区分"未传"和"传了 None（置空）"
        default_branch: str | None = None,
    ) -> None:
        if name is not None:
            self.name = name
        if repository_url is not None:
            self.repository_url = repository_url
        if variable_overrides is not None:
            self.variable_overrides = variable_overrides
        if git_credential_id is not MISSING:  # type: ignore[comparison-overlap]
            self.git_credential_id = git_credential_id
        if default_branch is not None:
            self.default_branch = default_branch
        self.updated_at = utc_now()


@dataclass
class RepositoryWebhook:
    """项目 Webhook 配置：每个 Webhook 绑定一个模板，有独立的签名密钥"""

    id: str
    repository_id: str
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
        repository_id: str,
        name: str,
        template_id: str,
        encrypted_secret: str,
        branch_filter: str | None = None,
    ) -> "RepositoryWebhook":
        now = utc_now()
        return RepositoryWebhook(
            id=str(ULID()),
            repository_id=repository_id,
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
class BuildStage:
    """构建 Stage 实体：执行的最小单元，描述"做什么"。

    不含任何编排信息（depends_on、sort_order 属于 PipelineTemplate 的编排，不属于 Stage）。
    """

    id: str
    name: str
    image: str
    script: str
    version: int
    artifacts: list | None = None  # list[ArtifactConfig]
    description: str = ""
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(
        name: str,
        image: str,
        script: str,
        artifacts: list | None = None,
        description: str = "",
    ) -> "BuildStage":
        now = utc_now()
        return BuildStage(
            id=str(ULID()),
            name=name,
            image=image,
            script=script,
            version=1,
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
        artifacts: list | None = None,
        description: str | None = None,
    ) -> None:
        execution_changed = False
        if name is not None:
            self.name = name
        if image is not None and image != self.image:
            self.image = image
            execution_changed = True
        if script is not None and script != self.script:
            self.script = script
            execution_changed = True
        if artifacts is not None and artifacts != self.artifacts:
            self.artifacts = artifacts
            execution_changed = True
        if description is not None:
            self.description = description
        if execution_changed:
            self.version += 1
        self.updated_at = utc_now()

    def to_stage_definition(self, name: str | None = None, depends_on: list | None = None) -> "StageDefinition":
        """转换为值对象，用于快照和执行。name 和 depends_on 由编排层传入。"""
        return StageDefinition(
            name=name or self.name,
            id=self.id,
            image=self.image,
            version=self.version,
            depends_on=depends_on or [],
            script=self.script,
            artifacts=[ArtifactConfig(**a) if isinstance(a, dict) else a for a in self.artifacts]
            if self.artifacts
            else None,
        )


@dataclass
class PipelineTemplate:
    """流水线模板实体：将一组 Stage 编排起来，定义依赖关系和执行顺序，配合变量声明可以运行。"""

    id: str
    name: str
    version: int
    orchestration: list["StageOrchestration"]  # 编排：stage_id + depends_on + sort_order
    stages: list["BuildStage"]  # 编排引用的 Stage 实体（加载时填充）
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
            version=1,
            description=description,
        )

    def update(
        self,
        name: str | None = None,
        description: str | None = None,
        variable_declarations: list[VariableDeclaration] | None = None,
    ) -> bool:
        """更新模板字段，返回是否有实质变更。version 递增由调用方负责。"""
        changed = False
        if name is not None and name != self.name:
            self.name = name
            changed = True
        if description is not None and description != self.description:
            self.description = description
            changed = True
        if variable_declarations is not None:
            # 深度比较：仅在实际变更时才更新
            existing = [d.model_dump() for d in self.variable_declarations]
            incoming = [d.model_dump() for d in variable_declarations]
            if existing != incoming:
                self.variable_declarations = variable_declarations
                changed = True
        return changed

    def bump_version(self) -> None:
        """版本号递增，同步更新 updated_at。"""
        self.version += 1
        self.updated_at = utc_now()

    def has_orchestration_changed(self, new_orchestration: list["StageOrchestration"]) -> bool:
        """深度比较编排是否实际变更。"""
        existing = sorted(self.orchestration, key=lambda o: o.sort_order)
        incoming = sorted(new_orchestration, key=lambda o: o.sort_order)
        return [o.model_dump() for o in existing] != [o.model_dump() for o in incoming]

    def get_stage_definitions(self) -> list[StageDefinition]:
        """按 sort_order 排序，将编排 + Stage 内容合并为 StageDefinition 列表，用于快照和执行。"""
        stage_map = {s.id: s for s in self.stages}
        sorted_orch = sorted(self.orchestration, key=lambda o: o.sort_order)
        result = []
        for orch in sorted_orch:
            stage = stage_map.get(orch.stage_id)
            if stage:
                # StageDefinition.name 使用 stage_name，执行器用它做依赖解析
                result.append(stage.to_stage_definition(name=orch.stage_name, depends_on=orch.depends_on))
        return result


@dataclass
class PipelineSnapshot:
    """流水线快照：模板某一版本的不可变副本"""

    id: str
    template_id: str
    version: int  # 从 1 开始，单调递增
    stages_snapshot: list[StageDefinition]
    variables_snapshot: list[VariableDeclaration]
    created_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(
        template: "PipelineTemplate",
        version: int,
        variable_declarations: list[VariableDeclaration],
    ) -> "PipelineSnapshot":
        """创建快照

        Args:
            template: 模板实体
            version: 快照版本号
            variable_declarations: 完整的变量声明列表（内置 + stage + 自定义）
        """
        return PipelineSnapshot(
            id=str(ULID()),
            template_id=template.id,
            version=version,
            stages_snapshot=deepcopy(template.get_stage_definitions()),
            variables_snapshot=deepcopy(variable_declarations),
        )


@dataclass
class PipelineRun:
    """Pipeline 运行实例"""

    id: str
    repository_id: str
    repository_name: str
    snapshot_id: str
    template_id: str
    template_name: str
    template_version: int
    trigger: PipelineRunTrigger
    trigger_ref: str
    variables_snapshot: list[VariableDeclaration]
    status: TaskStatus = TaskStatus.WAITING_TO_RUN
    retry_of: str | None = None
    started_at: datetime | None = None
    finished_at: datetime | None = None
    error_message: str | None = None
    created_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(
        repository_id: str,
        repository_name: str,
        snapshot_id: str,
        template_id: str,
        template_name: str,
        template_version: int,
        trigger: PipelineRunTrigger,
        trigger_ref: str,
        variables_snapshot: list[VariableDeclaration],
        retry_of: str | None = None,
    ) -> "PipelineRun":
        return PipelineRun(
            id=str(ULID()),
            repository_id=repository_id,
            repository_name=repository_name,
            snapshot_id=snapshot_id,
            template_id=template_id,
            template_name=template_name,
            template_version=template_version,
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

    def complete_failed(self, error_message: str | None = None) -> None:
        self.status = TaskStatus.FAULTED
        self.finished_at = utc_now()
        self.error_message = error_message

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
    stage_id: str  # 对应 StageDefinition.id
    stage_name: str  # 对应 StageDefinition.name（用于日志展示）
    status: TaskStatus = TaskStatus.WAITING_TO_RUN
    started_at: datetime | None = None
    finished_at: datetime | None = None
    exit_code: int | None = None
    error_message: str | None = None

    @staticmethod
    def create(pipeline_run_id: str, stage_id: str, stage_name: str) -> "StageRun":
        return StageRun(id=str(ULID()), pipeline_run_id=pipeline_run_id, stage_id=stage_id, stage_name=stage_name)

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
    repository_id: str
    repository_name: str
    template_id: str
    template_name: str
    stage_name: str
    type: ArtifactType
    name: str
    path: str | None = None
    created_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(
        pipeline_run_id: str,
        repository_id: str,
        repository_name: str,
        template_id: str,
        template_name: str,
        stage_name: str,
        artifact_type: ArtifactType,
        name: str,
        path: str | None = None,
    ) -> "Artifact":
        return Artifact(
            id=str(ULID()),
            pipeline_run_id=pipeline_run_id,
            repository_id=repository_id,
            repository_name=repository_name,
            template_id=template_id,
            template_name=template_name,
            stage_name=stage_name,
            type=artifact_type,
            name=name,
            path=path,
        )
