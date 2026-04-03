"""CI Mapper - ORM 和领域实体转换"""

import json

from pomelo_orbit.domain.ci.entities import (
    Artifact,
    Credential,
    Job,
    JobLog,
    PipelineRun,
    PipelineSnapshot,
    PipelineTemplate,
    Project,
)
from pomelo_orbit.domain.ci.value_objects import (
    CredentialType,
    JobStatus,
    PipelineRunStatus,
    PipelineRunTrigger,
    StageDefinition,
    VariableDeclaration,
)
from pomelo_orbit.infrastructure.ci.models import (
    ArtifactModel,
    CredentialModel,
    JobLogModel,
    JobModel,
    PipelineRunModel,
    PipelineSnapshotModel,
    PipelineTemplateModel,
    ProjectModel,
)


class CredentialMapper:
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
    @staticmethod
    def to_domain(orm: PipelineTemplateModel) -> PipelineTemplate:
        stages = [StageDefinition(**s) for s in json.loads(orm.stages)]
        variable_declarations = [VariableDeclaration(**vd) for vd in json.loads(orm.variable_declarations)]
        return PipelineTemplate(
            id=orm.id,
            name=orm.name,
            description=orm.description,
            stages=stages,
            variable_declarations=variable_declarations,
            is_builtin=bool(orm.is_builtin),
            created_at=orm.created_at,
            updated_at=orm.updated_at,
        )

    @staticmethod
    def to_orm(entity: PipelineTemplate) -> PipelineTemplateModel:
        return PipelineTemplateModel(
            id=entity.id,
            name=entity.name,
            description=entity.description,
            stages=json.dumps([s.model_dump() for s in entity.stages]),
            variable_declarations=json.dumps([vd.model_dump() for vd in entity.variable_declarations]),
            is_builtin=1 if entity.is_builtin else 0,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )


class PipelineSnapshotMapper:
    @staticmethod
    def to_domain(orm: PipelineSnapshotModel) -> PipelineSnapshot:
        stages_snapshot = [StageDefinition(**s) for s in json.loads(orm.stages_snapshot)]
        variable_declarations_snapshot = [
            VariableDeclaration(**vd) for vd in json.loads(orm.variable_declarations_snapshot)
        ]
        return PipelineSnapshot(
            id=orm.id,
            template_id=orm.template_id,
            version=orm.version,
            stages_snapshot=stages_snapshot,
            variable_declarations_snapshot=variable_declarations_snapshot,
            created_at=orm.created_at,
        )

    @staticmethod
    def to_orm(entity: PipelineSnapshot) -> PipelineSnapshotModel:
        return PipelineSnapshotModel(
            id=entity.id,
            template_id=entity.template_id,
            version=entity.version,
            stages_snapshot=json.dumps([s.model_dump() for s in entity.stages_snapshot]),
            variable_declarations_snapshot=json.dumps(
                [vd.model_dump() for vd in entity.variable_declarations_snapshot]
            ),
            created_at=entity.created_at,
        )


class ProjectMapper:
    @staticmethod
    def to_domain(orm: ProjectModel) -> Project:
        return Project(
            id=orm.id,
            name=orm.name,
            repository_url=orm.repository_url,
            pipeline_snapshot_id=orm.pipeline_snapshot_id,
            git_credential_id=orm.git_credential_id,
            variable_overrides=json.loads(orm.variable_overrides),
            default_branch=orm.default_branch,
            created_at=orm.created_at,
            updated_at=orm.updated_at,
        )

    @staticmethod
    def to_orm(entity: Project) -> ProjectModel:
        return ProjectModel(
            id=entity.id,
            name=entity.name,
            repository_url=entity.repository_url,
            pipeline_snapshot_id=entity.pipeline_snapshot_id,
            git_credential_id=entity.git_credential_id,
            variable_overrides=json.dumps(entity.variable_overrides),
            default_branch=entity.default_branch,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )


class PipelineRunMapper:
    @staticmethod
    def to_domain(orm: PipelineRunModel) -> PipelineRun:
        return PipelineRun(
            id=orm.id,
            project_id=orm.project_id,
            pipeline_snapshot_id=orm.pipeline_snapshot_id,
            trigger=PipelineRunTrigger(orm.trigger),
            trigger_ref=orm.trigger_ref,
            variables_snapshot=json.loads(orm.variables_snapshot),
            status=PipelineRunStatus(orm.status),
            retry_of=orm.retry_of,
            started_at=orm.started_at,
            finished_at=orm.finished_at,
            created_at=orm.created_at,
        )

    @staticmethod
    def to_orm(entity: PipelineRun) -> PipelineRunModel:
        return PipelineRunModel(
            id=entity.id,
            project_id=entity.project_id,
            pipeline_snapshot_id=entity.pipeline_snapshot_id,
            trigger=entity.trigger.value,
            trigger_ref=entity.trigger_ref,
            variables_snapshot=json.dumps(entity.variables_snapshot),
            status=entity.status.value,
            retry_of=entity.retry_of,
            started_at=entity.started_at,
            finished_at=entity.finished_at,
            created_at=entity.created_at,
        )


class JobMapper:
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
    @staticmethod
    def to_domain(orm: JobLogModel) -> JobLog:
        return JobLog(id=orm.id, job_id=orm.job_id, content=orm.content, created_at=orm.created_at)

    @staticmethod
    def to_orm(entity: JobLog) -> JobLogModel:
        return JobLogModel(id=entity.id, job_id=entity.job_id, content=entity.content, created_at=entity.created_at)


class ArtifactMapper:
    @staticmethod
    def to_domain(orm: ArtifactModel) -> Artifact:
        return Artifact(
            id=orm.id,
            pipeline_run_id=orm.pipeline_run_id,
            job_name=orm.job_name,
            type=orm.type,
            name=orm.name,
            path=orm.path,
            created_at=orm.created_at,
        )

    @staticmethod
    def to_orm(entity: Artifact) -> ArtifactModel:
        return ArtifactModel(
            id=entity.id,
            pipeline_run_id=entity.pipeline_run_id,
            job_name=entity.job_name,
            type=entity.type,
            name=entity.name,
            path=entity.path,
            created_at=entity.created_at,
        )
