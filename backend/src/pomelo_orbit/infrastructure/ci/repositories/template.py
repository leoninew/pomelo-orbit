"""CI 流水线模板与快照仓储实现"""

from sqlalchemy.orm import Session

from pomelo_orbit.domain.ci.entities import BuildStage, PipelineSnapshot, PipelineTemplate
from pomelo_orbit.domain.ci.repositories import (
    BuildStageRepository,
    PipelineSnapshotRepository,
    PipelineTemplateRepository,
)
from pomelo_orbit.domain.ci.value_objects import StageOrchestration
from pomelo_orbit.infrastructure.ci.mappers import (
    BuildStageMapper,
    PipelineSnapshotMapper,
    PipelineTemplateMapper,
    PipelineTemplateStageMapper,
)
from pomelo_orbit.infrastructure.ci.models import (
    BuildStageModel,
    PipelineSnapshotModel,
    PipelineTemplateModel,
    PipelineTemplateStageModel,
    ProjectWebhookModel,
)


class BuildStageRepositoryImpl(BuildStageRepository):
    def __init__(self, session: Session):
        self._session = session
        self._mapper = BuildStageMapper()

    def find_by_id_in_project(self, project_id: str, stage_id: str) -> BuildStage | None:
        orm = (
            self._session.query(BuildStageModel)
            .filter(BuildStageModel.project_id == project_id, BuildStageModel.id == stage_id)
            .first()
        )
        return self._mapper.to_domain(orm) if orm else None

    def find_by_name(self, project_id: str, name: str) -> BuildStage | None:
        orm = (
            self._session.query(BuildStageModel)
            .filter(BuildStageModel.project_id == project_id, BuildStageModel.name == name)
            .first()
        )
        return self._mapper.to_domain(orm) if orm else None

    def find_paginated_by_project_id(
        self, project_id: str, page: int, per_page: int, search: str | None = None
    ) -> tuple[list[BuildStage], int]:
        query = self._session.query(BuildStageModel).filter(BuildStageModel.project_id == project_id)
        if search:
            query = query.filter(BuildStageModel.name.ilike(f"%{search}%"))
        total = query.count()
        orms = query.order_by(BuildStageModel.id.desc()).offset((page - 1) * per_page).limit(per_page).all()
        return [self._mapper.to_domain(orm) for orm in orms], total

    def find_by_ids(self, project_id: str, stage_ids: list[str]) -> list[BuildStage]:
        orms = (
            self._session.query(BuildStageModel)
            .filter(BuildStageModel.project_id == project_id, BuildStageModel.id.in_(stage_ids))
            .all()
        )
        return [self._mapper.to_domain(orm) for orm in orms]

    def save(self, stage: BuildStage) -> None:
        existing = self._session.get(BuildStageModel, stage.id)
        orm = self._mapper.to_orm(stage)
        if existing:
            for k, v in orm.__dict__.items():
                if not k.startswith("_"):
                    setattr(existing, k, v)
        else:
            self._session.add(orm)

    def delete(self, stage: BuildStage) -> None:
        orm = self._session.get(BuildStageModel, stage.id)
        if orm:
            self._session.delete(orm)

    def is_referenced_by_templates(self, project_id: str, stage_id: str) -> bool:
        return (
            self._session.query(PipelineTemplateStageModel)
            .join(PipelineTemplateModel, PipelineTemplateModel.id == PipelineTemplateStageModel.template_id)
            .filter(PipelineTemplateModel.project_id == project_id, PipelineTemplateStageModel.stage_id == stage_id)
            .first()
            is not None
        )


class PipelineTemplateRepositoryImpl(PipelineTemplateRepository):
    def __init__(self, session: Session):
        self._session = session
        self._stage_mapper = BuildStageMapper()
        self._orch_mapper = PipelineTemplateStageMapper()

    def _load(self, orm: PipelineTemplateModel) -> PipelineTemplate:
        orch_orms = (
            self._session.query(PipelineTemplateStageModel)
            .filter(PipelineTemplateStageModel.template_id == orm.id)
            .order_by(PipelineTemplateStageModel.sort_order)
            .all()
        )
        orchestration = [self._orch_mapper.to_domain(o) for o in orch_orms]
        stage_ids = [o.stage_id for o in orch_orms]
        stage_orms = (
            (
                self._session.query(BuildStageModel)
                .filter(BuildStageModel.project_id == orm.project_id, BuildStageModel.id.in_(stage_ids))
                .all()
            )
            if stage_ids
            else []
        )
        stages = [self._stage_mapper.to_domain(s) for s in stage_orms]
        return PipelineTemplateMapper.to_domain(orm, orchestration, stages)

    def find_by_id_in_project(self, project_id: str, template_id: str) -> PipelineTemplate | None:
        orm = (
            self._session.query(PipelineTemplateModel)
            .filter(PipelineTemplateModel.project_id == project_id, PipelineTemplateModel.id == template_id)
            .first()
        )
        return self._load(orm) if orm else None

    def find_by_name(self, project_id: str, name: str) -> PipelineTemplate | None:
        orm = (
            self._session.query(PipelineTemplateModel)
            .filter(PipelineTemplateModel.project_id == project_id, PipelineTemplateModel.name == name)
            .first()
        )
        return self._load(orm) if orm else None

    def find_paginated_by_project_id(
        self, project_id: str, page: int, per_page: int, search: str | None = None
    ) -> tuple[list[PipelineTemplate], int]:
        query = self._session.query(PipelineTemplateModel).filter(PipelineTemplateModel.project_id == project_id)
        if search:
            query = query.filter(PipelineTemplateModel.name.ilike(f"%{search}%"))
        total = query.count()
        orms = query.order_by(PipelineTemplateModel.id.desc()).offset((page - 1) * per_page).limit(per_page).all()
        return [self._load(orm) for orm in orms], total

    def save(self, template: PipelineTemplate) -> None:
        existing = self._session.get(PipelineTemplateModel, template.id)
        orm = PipelineTemplateMapper.to_orm(template)
        if existing:
            for k, v in orm.__dict__.items():
                if not k.startswith("_"):
                    setattr(existing, k, v)
        else:
            self._session.add(orm)

    def save_orchestration(self, template_id: str, orchestration: list[StageOrchestration]) -> None:
        """整体替换模板编排（先删后插）"""
        self._session.query(PipelineTemplateStageModel).filter(
            PipelineTemplateStageModel.template_id == template_id
        ).delete()
        for orch in orchestration:
            self._session.add(PipelineTemplateStageMapper.to_orm(template_id, orch))

    def delete(self, template: PipelineTemplate) -> None:
        orm = self._session.get(PipelineTemplateModel, template.id)
        if orm:
            self._session.delete(orm)

    def is_referenced_by_webhooks(self, template_id: str) -> bool:
        """检查模板是否被任何 Webhook 引用"""
        return (
            self._session.query(ProjectWebhookModel).filter(ProjectWebhookModel.template_id == template_id).first()
            is not None
        )


class PipelineSnapshotRepositoryImpl(PipelineSnapshotRepository):
    def __init__(self, session: Session):
        self._session = session
        self._mapper = PipelineSnapshotMapper()

    def find_by_id_in_project(self, project_id: str, snapshot_id: str) -> PipelineSnapshot | None:
        orm = (
            self._session.query(PipelineSnapshotModel)
            .filter(PipelineSnapshotModel.project_id == project_id, PipelineSnapshotModel.id == snapshot_id)
            .first()
        )
        return self._mapper.to_domain(orm) if orm else None

    def find_latest(self, project_id: str, template_id: str) -> PipelineSnapshot | None:
        orm = (
            self._session.query(PipelineSnapshotModel)
            .filter(PipelineSnapshotModel.project_id == project_id, PipelineSnapshotModel.template_id == template_id)
            .order_by(PipelineSnapshotModel.version.desc())
            .first()
        )
        return self._mapper.to_domain(orm) if orm else None

    def save(self, snapshot: PipelineSnapshot) -> None:
        existing = self._session.get(PipelineSnapshotModel, snapshot.id)
        if existing:
            return
        self._session.add(self._mapper.to_orm(snapshot))
