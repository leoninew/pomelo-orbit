"""CI 流水线模板与快照仓储实现"""

from sqlalchemy import func
from sqlalchemy.orm import Session

from pomelo_orbit.domain.ci.entities import PipelineSnapshot, PipelineStage, PipelineTemplate
from pomelo_orbit.domain.ci.repositories import (
    PipelineSnapshotRepository,
    PipelineStageRepository,
    PipelineTemplateRepository,
)
from pomelo_orbit.domain.ci.value_objects import StageOrchestration
from pomelo_orbit.infrastructure.ci.mappers import (
    PipelineSnapshotMapper,
    PipelineStageMapper,
    PipelineTemplateMapper,
    PipelineTemplateStageMapper,
)
from pomelo_orbit.infrastructure.ci.models import (
    PipelineSnapshotModel,
    PipelineStageModel,
    PipelineTemplateModel,
    PipelineTemplateStageModel,
    ProjectWebhookModel,
)


class PipelineStageRepositoryImpl(PipelineStageRepository):
    def __init__(self, session: Session):
        self._session = session
        self._mapper = PipelineStageMapper()

    def find_by_id(self, stage_id: str) -> PipelineStage | None:
        orm = self._session.get(PipelineStageModel, stage_id)
        return self._mapper.to_domain(orm) if orm else None

    def find_by_name(self, name: str) -> PipelineStage | None:
        orm = self._session.query(PipelineStageModel).filter(PipelineStageModel.name == name).first()
        return self._mapper.to_domain(orm) if orm else None

    def find_all(self) -> list[PipelineStage]:
        orms = self._session.query(PipelineStageModel).order_by(PipelineStageModel.name).all()
        return [self._mapper.to_domain(orm) for orm in orms]

    def find_by_ids(self, stage_ids: list[str]) -> list[PipelineStage]:
        orms = self._session.query(PipelineStageModel).filter(PipelineStageModel.id.in_(stage_ids)).all()
        return [self._mapper.to_domain(orm) for orm in orms]

    def save(self, stage: PipelineStage) -> None:
        existing = self._session.get(PipelineStageModel, stage.id)
        orm = self._mapper.to_orm(stage)
        if existing:
            for k, v in orm.__dict__.items():
                if not k.startswith("_"):
                    setattr(existing, k, v)
        else:
            self._session.add(orm)

    def delete(self, stage: PipelineStage) -> None:
        orm = self._session.get(PipelineStageModel, stage.id)
        if orm:
            self._session.delete(orm)

    def is_referenced_by_templates(self, stage_id: str) -> bool:
        return (
            self._session.query(PipelineTemplateStageModel)
            .filter(PipelineTemplateStageModel.stage_id == stage_id)
            .first()
            is not None
        )


class PipelineTemplateRepositoryImpl(PipelineTemplateRepository):
    def __init__(self, session: Session):
        self._session = session
        self._stage_mapper = PipelineStageMapper()
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
            (self._session.query(PipelineStageModel).filter(PipelineStageModel.id.in_(stage_ids)).all())
            if stage_ids
            else []
        )
        stages = [self._stage_mapper.to_domain(s) for s in stage_orms]
        return PipelineTemplateMapper.to_domain(orm, orchestration, stages)

    def find_by_id(self, template_id: str) -> PipelineTemplate | None:
        orm = self._session.get(PipelineTemplateModel, template_id)
        return self._load(orm) if orm else None

    def find_all(self) -> list[PipelineTemplate]:
        orms = self._session.query(PipelineTemplateModel).all()
        return [self._load(orm) for orm in orms]

    def find_paginated(self, page: int, per_page: int) -> tuple[list[PipelineTemplate], int]:
        total = self._session.query(func.count(PipelineTemplateModel.id)).scalar() or 0
        orms = (
            self._session.query(PipelineTemplateModel)
            .order_by(PipelineTemplateModel.created_at.desc())
            .offset((page - 1) * per_page)
            .limit(per_page)
            .all()
        )
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
        # pipeline_template_stages 有 ON DELETE CASCADE，随模板自动删除
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

    def find_by_id(self, snapshot_id: str) -> PipelineSnapshot | None:
        orm = self._session.get(PipelineSnapshotModel, snapshot_id)
        return self._mapper.to_domain(orm) if orm else None

    def find_latest(self, template_id: str) -> PipelineSnapshot | None:
        orm = (
            self._session.query(PipelineSnapshotModel)
            .filter(PipelineSnapshotModel.template_id == template_id)
            .order_by(PipelineSnapshotModel.version.desc())
            .first()
        )
        return self._mapper.to_domain(orm) if orm else None

    def find_by_template(self, template_id: str) -> list[PipelineSnapshot]:
        orms = (
            self._session.query(PipelineSnapshotModel)
            .filter(PipelineSnapshotModel.template_id == template_id)
            .order_by(PipelineSnapshotModel.version.desc())
            .all()
        )
        return [self._mapper.to_domain(orm) for orm in orms]

    def get_next_version(self, template_id: str) -> int:
        result = (
            self._session.query(func.max(PipelineSnapshotModel.version))
            .filter(PipelineSnapshotModel.template_id == template_id)
            .scalar()
        )
        return (result or 0) + 1

    def save(self, snapshot: PipelineSnapshot) -> None:
        existing = self._session.get(PipelineSnapshotModel, snapshot.id)
        if existing:
            return  # 快照不可变
        self._session.add(self._mapper.to_orm(snapshot))

    def find_latest_versions(self, template_ids: list[str]) -> dict[str, int]:
        if not template_ids:
            return {}
        rows = (
            self._session.query(
                PipelineSnapshotModel.template_id,
                func.max(PipelineSnapshotModel.version),
            )
            .filter(PipelineSnapshotModel.template_id.in_(template_ids))
            .group_by(PipelineSnapshotModel.template_id)
            .all()
        )
        return dict(tuple(row) for row in rows)
