"""CI Mapper - ORM 和领域实体转换"""

import json

from pomelo_orbit.domain.cd.value_objects import TaskStatus
from pomelo_orbit.domain.ci.entities import (
    Artifact,
    Credential,
    PipelineRun,
    PipelineSnapshot,
    PipelineStage,
    PipelineTemplate,
    Project,
    ProjectWebhook,
    StageLog,
    StageRun,
)
from pomelo_orbit.domain.ci.value_objects import (
    ArtifactConfig,
    CredentialType,
    PipelineRunTrigger,
    StageDefinition,
    StageOrchestration,
    VariableDeclaration,
)
from pomelo_orbit.infrastructure.ci.models import (
    ArtifactModel,
    CredentialModel,
    PipelineRunModel,
    PipelineSnapshotModel,
    PipelineStageModel,
    PipelineTemplateModel,
    PipelineTemplateStageModel,
    ProjectModel,
    ProjectWebhookModel,
    StageLogModel,
    StageRunModel,
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


class PipelineStageMapper:
    @staticmethod
    def to_domain(orm: PipelineStageModel) -> PipelineStage:
        artifacts_raw = json.loads(orm.artifacts) if orm.artifacts else None
        artifacts = [ArtifactConfig(**a) for a in artifacts_raw] if artifacts_raw else None
        return PipelineStage(
            id=orm.id,
            name=orm.name,
            image=orm.image,
            script=orm.script,
            env=json.loads(orm.env),
            artifacts=artifacts,
            description=orm.description,
            created_at=orm.created_at,
            updated_at=orm.updated_at,
        )

    @staticmethod
    def to_orm(entity: PipelineStage) -> PipelineStageModel:
        artifacts_json = (
            json.dumps([a.model_dump() if hasattr(a, "model_dump") else a for a in entity.artifacts])
            if entity.artifacts
            else None
        )
        return PipelineStageModel(
            id=entity.id,
            name=entity.name,
            image=entity.image,
            script=entity.script,
            env=json.dumps(entity.env),
            artifacts=artifacts_json,
            description=entity.description,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )


class PipelineTemplateStageMapper:
    @staticmethod
    def to_domain(orm: PipelineTemplateStageModel) -> StageOrchestration:
        return StageOrchestration(
            stage_id=orm.stage_id,
            stage_key=orm.stage_key,
            depends_on=json.loads(orm.depends_on),
            sort_order=orm.sort_order,
        )

    @staticmethod
    def to_orm(template_id: str, orch: StageOrchestration) -> PipelineTemplateStageModel:
        return PipelineTemplateStageModel(
            template_id=template_id,
            stage_id=orch.stage_id,
            stage_key=orch.stage_key,
            depends_on=json.dumps(orch.depends_on),
            sort_order=orch.sort_order,
        )


class PipelineTemplateMapper:
    @staticmethod
    def _normalize_snapshot_stage(s: dict) -> dict:
        s.pop("builtin", None)
        s.pop("readonly", None)
        return s

    @staticmethod
    def to_domain(
        orm: PipelineTemplateModel, orchestration: list[StageOrchestration], stages: list[PipelineStage]
    ) -> PipelineTemplate:
        variable_declarations = [VariableDeclaration(**vd) for vd in json.loads(orm.variable_declarations)]
        return PipelineTemplate(
            id=orm.id,
            name=orm.name,
            description=orm.description,
            orchestration=orchestration,
            stages=stages,
            variable_declarations=variable_declarations,
            created_at=orm.created_at,
            updated_at=orm.updated_at,
        )

    @staticmethod
    def to_orm(entity: PipelineTemplate) -> PipelineTemplateModel:
        return PipelineTemplateModel(
            id=entity.id,
            name=entity.name,
            description=entity.description,
            variable_declarations=json.dumps([vd.model_dump() for vd in entity.variable_declarations]),
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )


class PipelineSnapshotMapper:
    @staticmethod
    def to_domain(orm: PipelineSnapshotModel) -> PipelineSnapshot:
        raw_stages = json.loads(orm.stages_snapshot)
        stages_snapshot = [StageDefinition(**PipelineTemplateMapper._normalize_snapshot_stage(s)) for s in raw_stages]
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
            code=orm.code,
            repository_url=orm.repository_url,
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
            code=entity.code,
            repository_url=entity.repository_url,
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
            status=TaskStatus(orm.status),
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


class StageRunMapper:
    @staticmethod
    def to_domain(orm: StageRunModel) -> StageRun:
        return StageRun(
            id=orm.id,
            pipeline_run_id=orm.pipeline_run_id,
            name=orm.name,
            status=TaskStatus(orm.status),
            started_at=orm.started_at,
            finished_at=orm.finished_at,
            exit_code=orm.exit_code,
            error_message=orm.error_message,
        )

    @staticmethod
    def to_orm(entity: StageRun) -> StageRunModel:
        return StageRunModel(
            id=entity.id,
            pipeline_run_id=entity.pipeline_run_id,
            name=entity.name,
            status=entity.status.value,
            started_at=entity.started_at,
            finished_at=entity.finished_at,
            exit_code=entity.exit_code,
            error_message=entity.error_message,
        )


class StageLogMapper:
    @staticmethod
    def to_domain(orm: StageLogModel) -> StageLog:
        return StageLog(
            id=orm.id,
            stage_run_id=orm.stage_run_id,
            content=orm.content,
            created_at=orm.created_at,
        )

    @staticmethod
    def to_orm(entity: StageLog) -> StageLogModel:
        return StageLogModel(
            id=entity.id,
            stage_run_id=entity.stage_run_id,
            content=entity.content,
            created_at=entity.created_at,
        )


class ArtifactMapper:
    @staticmethod
    def to_domain(orm: ArtifactModel) -> Artifact:
        return Artifact(
            id=orm.id,
            pipeline_run_id=orm.pipeline_run_id,
            stage_name=orm.stage_name,
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
            stage_name=entity.stage_name,
            type=entity.type,
            name=entity.name,
            path=entity.path,
            created_at=entity.created_at,
        )


class ProjectWebhookMapper:
    @staticmethod
    def to_domain(orm: ProjectWebhookModel) -> ProjectWebhook:
        return ProjectWebhook(
            id=orm.id,
            project_id=orm.project_id,
            name=orm.name,
            template_id=orm.template_id,
            branch_filter=orm.branch_filter,
            encrypted_secret=orm.encrypted_secret,
            enabled=bool(orm.enabled),
            created_at=orm.created_at,
            updated_at=orm.updated_at,
        )

    @staticmethod
    def to_orm(entity: ProjectWebhook) -> ProjectWebhookModel:
        return ProjectWebhookModel(
            id=entity.id,
            project_id=entity.project_id,
            name=entity.name,
            template_id=entity.template_id,
            branch_filter=entity.branch_filter,
            encrypted_secret=entity.encrypted_secret,
            enabled=1 if entity.enabled else 0,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )
