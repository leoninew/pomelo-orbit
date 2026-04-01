"""CI 实体"""

from dataclasses import dataclass, field
from datetime import datetime
from typing import Any

from ulid import ULID

from pomelo_orbit.domain.ci.value_objects import (
    CredentialType,
    JobStatus,
    PipelineRunStatus,
    PipelineRunTrigger,
    VariableDeclaration,
)
from pomelo_orbit.infrastructure.time_utils import utc_now


@dataclass
class Project:
    """项目实体"""

    id: str
    name: str
    repository_url: str
    pipeline_template_id: str
    git_credential_id: str
    variable_overrides: dict[str, Any]
    webhook_secret: str | None = None
    branch_filter: str | None = None
    default_branch: str = "master"
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(
        name: str,
        repository_url: str,
        pipeline_template_id: str,
        git_credential_id: str,
        variable_overrides: dict[str, Any] | None = None,
        webhook_secret: str | None = None,
        branch_filter: str | None = None,
        default_branch: str = "master",
    ) -> "Project":
        """创建项目"""
        return Project(
            id=str(ULID()),
            name=name,
            repository_url=repository_url,
            pipeline_template_id=pipeline_template_id,
            git_credential_id=git_credential_id,
            variable_overrides=variable_overrides or {},
            webhook_secret=webhook_secret,
            branch_filter=branch_filter,
            default_branch=default_branch,
        )

    def update(
        self,
        name: str | None = None,
        repository_url: str | None = None,
        variable_overrides: dict[str, Any] | None = None,
        pipeline_template_id: str | None = None,
        git_credential_id: str | None = None,
        webhook_secret: str | None = None,
        branch_filter: str | None = None,
        default_branch: str | None = None,
    ) -> None:
        """更新项目"""
        if name is not None:
            self.name = name
        if repository_url is not None:
            self.repository_url = repository_url
        if variable_overrides is not None:
            self.variable_overrides = variable_overrides
        if pipeline_template_id is not None:
            self.pipeline_template_id = pipeline_template_id
        if git_credential_id is not None:
            self.git_credential_id = git_credential_id
        if webhook_secret is not None:
            self.webhook_secret = webhook_secret
        if branch_filter is not None:
            self.branch_filter = branch_filter
        if default_branch is not None:
            self.default_branch = default_branch
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
        """获取 SSH 私钥（仅 git_ssh 类型有效，encrypted_data 已解密）"""
        if self.type != CredentialType.GIT_SSH:
            raise ValueError(f"Credential type {self.type} has no private key")
        return self.encrypted_data

    def get_token(self) -> str:
        """获取 token（仅 git_token 类型有效，encrypted_data 已解密）"""
        if self.type != CredentialType.GIT_TOKEN:
            raise ValueError(f"Credential type {self.type} has no token")
        return self.encrypted_data

    @staticmethod
    def create(
        name: str,
        type: CredentialType,
        encrypted_data: str,
    ) -> "Credential":
        """创建凭据"""
        return Credential(
            id=str(ULID()),
            name=name,
            type=type,
            encrypted_data=encrypted_data,
        )


@dataclass
class PipelineTemplate:
    """Pipeline 模板实体"""

    id: str
    name: str
    content: str
    variable_declarations: list[VariableDeclaration]
    description: str = ""
    is_builtin: bool = False
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(
        name: str,
        content: str,
        variable_declarations: list[VariableDeclaration],
        description: str = "",
        is_builtin: bool = False,
    ) -> "PipelineTemplate":
        """创建模板"""
        return PipelineTemplate(
            id=str(ULID()),
            name=name,
            content=content,
            variable_declarations=variable_declarations,
            description=description,
            is_builtin=is_builtin,
        )

    def update(
        self,
        name: str | None = None,
        description: str | None = None,
        content: str | None = None,
        variable_declarations: list[VariableDeclaration] | None = None,
    ) -> None:
        """更新模板"""
        if name is not None:
            self.name = name
        if description is not None:
            self.description = description
        if content is not None:
            self.content = content
        if variable_declarations is not None:
            self.variable_declarations = variable_declarations
        self.updated_at = utc_now()


@dataclass
class PipelineRun:
    """Pipeline 运行实例"""

    id: str
    project_id: str
    trigger: PipelineRunTrigger
    trigger_ref: str
    resolved_pipeline: str
    variables_snapshot: dict[str, Any]
    status: PipelineRunStatus = PipelineRunStatus.WAITING
    retry_of: str | None = None
    started_at: datetime | None = None
    finished_at: datetime | None = None
    created_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(
        project_id: str,
        trigger: PipelineRunTrigger,
        trigger_ref: str,
        resolved_pipeline: str,
        variables_snapshot: dict[str, Any],
        retry_of: str | None = None,
    ) -> "PipelineRun":
        """创建 pipeline run"""
        return PipelineRun(
            id=str(ULID()),
            project_id=project_id,
            trigger=trigger,
            trigger_ref=trigger_ref,
            resolved_pipeline=resolved_pipeline,
            variables_snapshot=variables_snapshot,
            retry_of=retry_of,
        )

    def start(self) -> None:
        """开始执行"""
        self.status = PipelineRunStatus.RUNNING
        self.started_at = utc_now()

    def complete_success(self) -> None:
        """执行成功"""
        self.status = PipelineRunStatus.SUCCESS
        self.finished_at = utc_now()

    def complete_failed(self) -> None:
        """执行失败"""
        self.status = PipelineRunStatus.FAILED
        self.finished_at = utc_now()

    def cancel(self) -> None:
        """取消执行"""
        if self.status not in (PipelineRunStatus.WAITING, PipelineRunStatus.RUNNING):
            raise ValueError(f"Cannot cancel run with status {self.status}")
        self.status = PipelineRunStatus.CANCELED
        self.finished_at = utc_now()


@dataclass
class Job:
    """Job 执行单元"""

    id: str
    pipeline_run_id: str
    name: str
    status: JobStatus = JobStatus.WAITING
    parent_job_id: str | None = None
    started_at: datetime | None = None
    finished_at: datetime | None = None
    exit_code: int | None = None
    error_message: str | None = None

    @staticmethod
    def create(
        pipeline_run_id: str,
        name: str,
        parent_job_id: str | None = None,
    ) -> "Job":
        """创建 job"""
        return Job(
            id=str(ULID()),
            pipeline_run_id=pipeline_run_id,
            name=name,
            parent_job_id=parent_job_id,
        )

    def start(self) -> None:
        """开始执行"""
        self.status = JobStatus.RUNNING
        self.started_at = utc_now()

    def complete_success(self, exit_code: int = 0) -> None:
        """执行成功"""
        self.status = JobStatus.SUCCESS
        self.exit_code = exit_code
        self.finished_at = utc_now()

    def complete_failed(self, exit_code: int, error_message: str | None = None) -> None:
        """执行失败"""
        self.status = JobStatus.FAILED
        self.exit_code = exit_code
        self.error_message = error_message
        self.finished_at = utc_now()

    def complete_faulted(self, error_message: str) -> None:
        """执行故障"""
        self.status = JobStatus.FAULTED
        self.error_message = error_message
        self.finished_at = utc_now()


@dataclass
class JobLog:
    """Job 日志"""

    id: str
    job_id: str
    content: str
    created_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(job_id: str, content: str) -> "JobLog":
        """创建 job 日志"""
        return JobLog(
            id=str(ULID()),
            job_id=job_id,
            content=content,
        )


@dataclass
class Artifact:
    """制品记录"""

    id: str
    pipeline_run_id: str
    job_name: str
    type: str  # docker_image | file
    name: str
    path: str | None = None  # file 类型时的宿主机路径
    created_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(
        pipeline_run_id: str,
        job_name: str,
        artifact_type: str,
        name: str,
        path: str | None = None,
    ) -> "Artifact":
        """创建制品记录"""
        return Artifact(
            id=str(ULID()),
            pipeline_run_id=pipeline_run_id,
            job_name=job_name,
            type=artifact_type,
            name=name,
            path=path,
        )
