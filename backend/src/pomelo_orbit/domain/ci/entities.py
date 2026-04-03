"""CI 实体"""

from copy import deepcopy
from dataclasses import dataclass, field
from datetime import datetime
from typing import Any

from ulid import ULID

from pomelo_orbit.domain.ci.value_objects import (
    CredentialType,
    JobStatus,
    PipelineRunStatus,
    PipelineRunTrigger,
    StageDefinition,
    VariableDeclaration,
)
from pomelo_orbit.infrastructure.time_utils import utc_now


@dataclass
class Project:
    """项目实体"""

    id: str
    name: str
    repository_url: str
    pipeline_snapshot_id: str
    variable_overrides: dict[str, Any]
    git_credential_id: str | None = None
    default_branch: str = "master"
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(
        name: str,
        repository_url: str,
        pipeline_snapshot_id: str,
        git_credential_id: str | None = None,
        variable_overrides: dict[str, Any] | None = None,
        default_branch: str = "master",
    ) -> "Project":
        return Project(
            id=str(ULID()),
            name=name,
            repository_url=repository_url,
            pipeline_snapshot_id=pipeline_snapshot_id,
            git_credential_id=git_credential_id,
            variable_overrides=variable_overrides or {},
            default_branch=default_branch,
        )

    def update(
        self,
        name: str | None = None,
        repository_url: str | None = None,
        variable_overrides: dict[str, Any] | None = None,
        pipeline_snapshot_id: str | None = None,
        git_credential_id: str | None = None,
        default_branch: str | None = None,
    ) -> None:
        if name is not None:
            self.name = name
        if repository_url is not None:
            self.repository_url = repository_url
        if variable_overrides is not None:
            self.variable_overrides = variable_overrides
        if pipeline_snapshot_id is not None:
            self.pipeline_snapshot_id = pipeline_snapshot_id
        if git_credential_id is not None:
            self.git_credential_id = git_credential_id
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
class PipelineTemplate:
    """流水线模板实体"""

    id: str
    name: str
    stages: list[StageDefinition]
    variable_declarations: list[VariableDeclaration]
    description: str = ""
    is_builtin: bool = False
    created_at: datetime = field(default_factory=utc_now)
    updated_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(
        name: str,
        stages: list[StageDefinition],
        variable_declarations: list[VariableDeclaration],
        description: str = "",
        is_builtin: bool = False,
    ) -> "PipelineTemplate":
        if not name:
            raise ValueError("Template name cannot be empty")
        return PipelineTemplate(
            id=str(ULID()),
            name=name,
            stages=stages,
            variable_declarations=variable_declarations,
            description=description,
            is_builtin=is_builtin,
        )

    def update(
        self,
        name: str | None = None,
        description: str | None = None,
        stages: list[StageDefinition] | None = None,
        variable_declarations: list[VariableDeclaration] | None = None,
    ) -> None:
        if name is not None:
            if not name:
                raise ValueError("Template name cannot be empty")
            self.name = name
        if description is not None:
            self.description = description
        if stages is not None:
            self.stages = stages
        if variable_declarations is not None:
            self.variable_declarations = variable_declarations
        self.updated_at = utc_now()


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
    def create(template: PipelineTemplate, version: int) -> "PipelineSnapshot":
        return PipelineSnapshot(
            id=str(ULID()),
            template_id=template.id,
            version=version,
            stages_snapshot=deepcopy(template.stages),
            variable_declarations_snapshot=deepcopy(template.variable_declarations),
        )


@dataclass
class PipelineRun:
    """Pipeline 运行实例"""

    id: str
    project_id: str
    pipeline_snapshot_id: str
    trigger: PipelineRunTrigger
    trigger_ref: str
    variables_snapshot: dict[str, Any]
    status: PipelineRunStatus = PipelineRunStatus.WAITING
    retry_of: str | None = None
    started_at: datetime | None = None
    finished_at: datetime | None = None
    created_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(
        project_id: str,
        pipeline_snapshot_id: str,
        trigger: PipelineRunTrigger,
        trigger_ref: str,
        variables_snapshot: dict[str, Any],
        retry_of: str | None = None,
    ) -> "PipelineRun":
        return PipelineRun(
            id=str(ULID()),
            project_id=project_id,
            pipeline_snapshot_id=pipeline_snapshot_id,
            trigger=trigger,
            trigger_ref=trigger_ref,
            variables_snapshot=variables_snapshot,
            retry_of=retry_of,
        )

    def start(self) -> None:
        self.status = PipelineRunStatus.RUNNING
        self.started_at = utc_now()

    def complete_success(self) -> None:
        self.status = PipelineRunStatus.SUCCESS
        self.finished_at = utc_now()

    def complete_failed(self) -> None:
        self.status = PipelineRunStatus.FAILED
        self.finished_at = utc_now()

    def cancel(self) -> None:
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
    def create(pipeline_run_id: str, name: str, parent_job_id: str | None = None) -> "Job":
        return Job(id=str(ULID()), pipeline_run_id=pipeline_run_id, name=name, parent_job_id=parent_job_id)

    def start(self) -> None:
        self.status = JobStatus.RUNNING
        self.started_at = utc_now()

    def complete_success(self, exit_code: int = 0) -> None:
        self.status = JobStatus.SUCCESS
        self.exit_code = exit_code
        self.finished_at = utc_now()

    def complete_failed(self, exit_code: int, error_message: str | None = None) -> None:
        self.status = JobStatus.FAILED
        self.exit_code = exit_code
        self.error_message = error_message
        self.finished_at = utc_now()

    def complete_faulted(self, error_message: str) -> None:
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
        return JobLog(id=str(ULID()), job_id=job_id, content=content)


@dataclass
class Artifact:
    """制品记录"""

    id: str
    pipeline_run_id: str
    job_name: str
    type: str  # docker_image | file
    name: str
    path: str | None = None
    created_at: datetime = field(default_factory=utc_now)

    @staticmethod
    def create(
        pipeline_run_id: str,
        job_name: str,
        artifact_type: str,
        name: str,
        path: str | None = None,
    ) -> "Artifact":
        return Artifact(
            id=str(ULID()),
            pipeline_run_id=pipeline_run_id,
            job_name=job_name,
            type=artifact_type,
            name=name,
            path=path,
        )
