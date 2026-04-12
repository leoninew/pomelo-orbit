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
    Repository,
    RepositoryWebhook,
    StageRun,
)
from pomelo_orbit.domain.ci.value_objects import (
    ArtifactConfig,
    ArtifactType,
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
            version=orm.version,
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
            version=entity.version,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )


class PipelineTemplateStageMapper:
    @staticmethod
    def to_domain(orm: PipelineTemplateStageModel) -> StageOrchestration:
        depends_on = json.loads(orm.depends_on or "[]")
        return StageOrchestration(
            stage_id=orm.stage_id,
            stage_name=orm.stage_name,
            stage_version=orm.stage_version,
            depends_on=depends_on,
            sort_order=orm.sort_order,
        )

    @staticmethod
    def to_orm(template_id: str, orch: StageOrchestration) -> PipelineTemplateStageModel:
        return PipelineTemplateStageModel(
            template_id=template_id,
            stage_id=orch.stage_id,
            stage_name=orch.stage_name,
            stage_version=orch.stage_version,
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
            version=orm.version,
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
            version=entity.version,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )


class PipelineSnapshotMapper:
    @staticmethod
    def to_domain(orm: PipelineSnapshotModel) -> PipelineSnapshot:
        raw_stages = json.loads(orm.stages_snapshot)
        stages_snapshot = [StageDefinition(**PipelineTemplateMapper._normalize_snapshot_stage(s)) for s in raw_stages]
        variables_snapshot = [VariableDeclaration(**vd) for vd in json.loads(orm.variables_snapshot)]
        return PipelineSnapshot(
            id=orm.id,
            template_id=orm.template_id,
            version=orm.version,
            stages_snapshot=stages_snapshot,
            variables_snapshot=variables_snapshot,
            created_at=orm.created_at,
        )

    @staticmethod
    def to_orm(entity: PipelineSnapshot) -> PipelineSnapshotModel:
        return PipelineSnapshotModel(
            id=entity.id,
            template_id=entity.template_id,
            version=entity.version,
            stages_snapshot=json.dumps([s.model_dump() for s in entity.stages_snapshot]),
            variables_snapshot=json.dumps([vd.model_dump() for vd in entity.variables_snapshot]),
            created_at=entity.created_at,
        )


class RepositoryMapper:
    @staticmethod
    def to_domain(orm: ProjectModel) -> Repository:
        variable_overrides_data = json.loads(orm.variable_overrides)
        variable_overrides = [VariableDeclaration(**item) for item in variable_overrides_data]

        return Repository(
            id=orm.id,
            name=orm.name,
            code=orm.code,
            repository_url=orm.repository_url,
            git_credential_id=orm.git_credential_id,
            variable_overrides=variable_overrides,
            default_branch=orm.default_branch,
            created_at=orm.created_at,
            updated_at=orm.updated_at,
        )

    @staticmethod
    def to_orm(entity: Repository) -> ProjectModel:
        variable_overrides_data = [var.model_dump() for var in entity.variable_overrides]
        return ProjectModel(
            id=entity.id,
            name=entity.name,
            code=entity.code,
            repository_url=entity.repository_url,
            git_credential_id=entity.git_credential_id,
            variable_overrides=json.dumps(variable_overrides_data),
            default_branch=entity.default_branch,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )


class PipelineRunMapper:
    @staticmethod
    def to_domain(orm: PipelineRunModel) -> PipelineRun:
        variables_snapshot_data = json.loads(orm.variables_snapshot)
        variables_snapshot = [VariableDeclaration(**vd) for vd in variables_snapshot_data]

        return PipelineRun(
            id=orm.id,
            repository_id=orm.repository_id,
            repository_name=orm.repository_name,
            snapshot_id=orm.snapshot_id,
            template_id=orm.template_id,
            template_name=orm.template_name,
            template_version=orm.template_version,
            trigger=PipelineRunTrigger(orm.trigger),
            trigger_ref=orm.trigger_ref,
            variables_snapshot=variables_snapshot,
            status=TaskStatus(orm.status),
            retry_of=orm.retry_of,
            started_at=orm.started_at,
            finished_at=orm.finished_at,
            error_message=orm.error_message,
            created_at=orm.created_at,
        )

    @staticmethod
    def to_orm(entity: PipelineRun) -> PipelineRunModel:
        return PipelineRunModel(
            id=entity.id,
            repository_id=entity.repository_id,
            repository_name=entity.repository_name,
            snapshot_id=entity.snapshot_id,
            template_id=entity.template_id,
            template_name=entity.template_name,
            template_version=entity.template_version,
            trigger=entity.trigger.value,
            trigger_ref=entity.trigger_ref,
            variables_snapshot=json.dumps([vd.model_dump() for vd in entity.variables_snapshot]),
            status=entity.status.value,
            retry_of=entity.retry_of,
            started_at=entity.started_at,
            finished_at=entity.finished_at,
            error_message=entity.error_message,
            created_at=entity.created_at,
        )


class StageRunMapper:
    @staticmethod
    def to_domain(orm: StageRunModel) -> StageRun:
        return StageRun(
            id=orm.id,
            pipeline_run_id=orm.pipeline_run_id,
            stage_id=orm.stage_id,
            stage_name=orm.stage_name,
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
            stage_id=entity.stage_id,
            stage_name=entity.stage_name,
            status=entity.status.value,
            started_at=entity.started_at,
            finished_at=entity.finished_at,
            exit_code=entity.exit_code,
            error_message=entity.error_message,
        )


class ArtifactMapper:
    @staticmethod
    def to_domain(orm: ArtifactModel) -> Artifact:
        return Artifact(
            id=orm.id,
            pipeline_run_id=orm.pipeline_run_id,
            stage_name=orm.stage_name,
            type=ArtifactType(orm.type),
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
    def to_domain(orm: ProjectWebhookModel) -> RepositoryWebhook:
        return RepositoryWebhook(
            id=orm.id,
            repository_id=orm.repository_id,
            name=orm.name,
            template_id=orm.template_id,
            branch_filter=orm.branch_filter,
            encrypted_secret=orm.encrypted_secret,
            enabled=bool(orm.enabled),
            created_at=orm.created_at,
            updated_at=orm.updated_at,
        )

    @staticmethod
    def to_orm(entity: RepositoryWebhook) -> ProjectWebhookModel:
        return ProjectWebhookModel(
            id=entity.id,
            repository_id=entity.repository_id,
            name=entity.name,
            template_id=entity.template_id,
            branch_filter=entity.branch_filter,
            encrypted_secret=entity.encrypted_secret,
            enabled=1 if entity.enabled else 0,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )
