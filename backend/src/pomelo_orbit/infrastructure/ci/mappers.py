"""CI Mapper - ORM 和领域实体转换"""

import json

from pomelo_orbit.domain.ci.entities import (
    Credential,
    Job,
    JobLog,
    PipelineRun,
    PipelineTemplate,
    Project,
)
from pomelo_orbit.domain.ci.value_objects import (
    CredentialType,
    JobStatus,
    PipelineRunStatus,
    PipelineRunTrigger,
    VariableDeclaration,
)
from pomelo_orbit.infrastructure.ci.models import (
    CredentialModel,
    JobLogModel,
    JobModel,
    PipelineRunModel,
    PipelineTemplateModel,
    ProjectModel,
)


class CredentialMapper:
    """凭据 Mapper"""

    @staticmethod
    def to_domain(orm: CredentialModel) -> Credential:
        return Credential(
            id=orm.id,
            name=orm.name,
            type=CredentialType(orm.type),
            encrypted_data=orm.encrypted_data,
            created_at=orm.created_at,
        )

    @staticmethod
    def to_orm(entity: Credential) -> CredentialModel:
        return CredentialModel(
            id=entity.id,
            name=entity.name,
            type=entity.type.value,
            encrypted_data=entity.encrypted_data,
            created_at=entity.created_at,
        )


class PipelineTemplateMapper:
    """Pipeline 模板 Mapper"""

    @staticmethod
    def to_domain(orm: PipelineTemplateModel) -> PipelineTemplate:
        variable_declarations_data = json.loads(orm.variable_declarations)
        variable_declarations = [VariableDeclaration(**vd) for vd in variable_declarations_data]

        return PipelineTemplate(
            id=orm.id,
            name=orm.name,
            description=orm.description,
            content=orm.content,
            variable_declarations=variable_declarations,
            is_builtin=bool(orm.is_builtin),
            created_at=orm.created_at,
            updated_at=orm.updated_at,
        )

    @staticmethod
    def to_orm(entity: PipelineTemplate) -> PipelineTemplateModel:
        variable_declarations_data = [vd.model_dump() for vd in entity.variable_declarations]

        return PipelineTemplateModel(
            id=entity.id,
            name=entity.name,
            description=entity.description,
            content=entity.content,
            variable_declarations=json.dumps(variable_declarations_data),
            is_builtin=1 if entity.is_builtin else 0,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )


class ProjectMapper:
    """项目 Mapper"""

    @staticmethod
    def to_domain(orm: ProjectModel) -> Project:
        variable_overrides = json.loads(orm.variable_overrides)

        return Project(
            id=orm.id,
            name=orm.name,
            repository_url=orm.repository_url,
            pipeline_template_id=orm.pipeline_template_id,
            git_credential_id=orm.git_credential_id,
            variable_overrides=variable_overrides,
            webhook_secret=orm.webhook_secret,
            branch_filter=orm.branch_filter,
            created_at=orm.created_at,
            updated_at=orm.updated_at,
        )

    @staticmethod
    def to_orm(entity: Project) -> ProjectModel:
        return ProjectModel(
            id=entity.id,
            name=entity.name,
            repository_url=entity.repository_url,
            pipeline_template_id=entity.pipeline_template_id,
            git_credential_id=entity.git_credential_id,
            variable_overrides=json.dumps(entity.variable_overrides),
            webhook_secret=entity.webhook_secret,
            branch_filter=entity.branch_filter,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )


class PipelineRunMapper:
    """Pipeline 运行 Mapper"""

    @staticmethod
    def to_domain(orm: PipelineRunModel) -> PipelineRun:
        variables_snapshot = json.loads(orm.variables_snapshot)

        return PipelineRun(
            id=orm.id,
            project_id=orm.project_id,
            trigger=PipelineRunTrigger(orm.trigger),
            trigger_ref=orm.trigger_ref,
            resolved_pipeline=orm.resolved_pipeline,
            variables_snapshot=variables_snapshot,
            status=PipelineRunStatus(orm.status),
            started_at=orm.started_at,
            finished_at=orm.finished_at,
            created_at=orm.created_at,
        )

    @staticmethod
    def to_orm(entity: PipelineRun) -> PipelineRunModel:
        return PipelineRunModel(
            id=entity.id,
            project_id=entity.project_id,
            trigger=entity.trigger.value,
            trigger_ref=entity.trigger_ref,
            resolved_pipeline=entity.resolved_pipeline,
            variables_snapshot=json.dumps(entity.variables_snapshot),
            status=entity.status.value,
            started_at=entity.started_at,
            finished_at=entity.finished_at,
            created_at=entity.created_at,
        )


class JobMapper:
    """Job Mapper"""

    @staticmethod
    def to_domain(orm: JobModel) -> Job:
        return Job(
            id=orm.id,
            pipeline_run_id=orm.pipeline_run_id,
            name=orm.name,
            parent_job_id=orm.parent_job_id,
            status=JobStatus(orm.status),
            started_at=orm.started_at,
            finished_at=orm.finished_at,
            exit_code=orm.exit_code,
            error_message=orm.error_message,
        )

    @staticmethod
    def to_orm(entity: Job) -> JobModel:
        return JobModel(
            id=entity.id,
            pipeline_run_id=entity.pipeline_run_id,
            name=entity.name,
            parent_job_id=entity.parent_job_id,
            status=entity.status.value,
            started_at=entity.started_at,
            finished_at=entity.finished_at,
            exit_code=entity.exit_code,
            error_message=entity.error_message,
        )


class JobLogMapper:
    """Job 日志 Mapper"""

    @staticmethod
    def to_domain(orm: JobLogModel) -> JobLog:
        return JobLog(
            id=orm.id,
            job_id=orm.job_id,
            content=orm.content,
            created_at=orm.created_at,
        )

    @staticmethod
    def to_orm(entity: JobLog) -> JobLogModel:
        return JobLogModel(
            id=entity.id,
            job_id=entity.job_id,
            content=entity.content,
            created_at=entity.created_at,
        )
