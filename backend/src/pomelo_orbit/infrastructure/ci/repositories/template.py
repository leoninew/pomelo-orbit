"""CI 流水线模板与快照仓储实现"""

from sqlalchemy import func
from sqlalchemy.orm import Session

from pomelo_orbit.domain.ci.entities import PipelineSnapshot, PipelineTemplate
from pomelo_orbit.domain.ci.repositories import PipelineSnapshotRepository, PipelineTemplateRepository
from pomelo_orbit.infrastructure.ci.mappers import PipelineSnapshotMapper, PipelineTemplateMapper
from pomelo_orbit.infrastructure.ci.models import PipelineSnapshotModel, PipelineTemplateModel, ProjectModel
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository


class PipelineTemplateRepositoryImpl(
    BaseRepository[PipelineTemplate, PipelineTemplateModel], PipelineTemplateRepository
):
    def __init__(self, session: Session):
        super().__init__(session, PipelineTemplateModel, PipelineTemplateMapper())

    def is_referenced_by_projects(self, template_id: str) -> bool:
        """检查模板是否有任何快照被项目引用（通过 pipeline_snapshots 间接查询）"""
        return (
            self._session.query(ProjectModel)
            .join(PipelineSnapshotModel, ProjectModel.pipeline_snapshot_id == PipelineSnapshotModel.id)
            .filter(PipelineSnapshotModel.template_id == template_id)
            .first()
            is not None
        )

    def find_builtin(self) -> list[PipelineTemplate]:
        orms = self._session.query(PipelineTemplateModel).filter(PipelineTemplateModel.is_builtin == 1).all()
        return [self._mapper.to_domain(orm) for orm in orms]


class PipelineSnapshotRepositoryImpl(PipelineSnapshotRepository):
    def __init__(self, session: Session):
        self._session = session
        self._mapper = PipelineSnapshotMapper()

    def find_by_id(self, snapshot_id: str) -> PipelineSnapshot | None:
        orm = self._session.get(PipelineSnapshotModel, snapshot_id)
        return self._mapper.to_domain(orm) if orm else None

    def find_by_template(self, template_id: str) -> list[PipelineSnapshot]:
        """按模板 ID 查询所有快照，version 降序"""
        orms = (
            self._session.query(PipelineSnapshotModel)
            .filter(PipelineSnapshotModel.template_id == template_id)
            .order_by(PipelineSnapshotModel.version.desc())
            .all()
        )
        return [self._mapper.to_domain(orm) for orm in orms]

    def get_next_version(self, template_id: str) -> int:
        """返回下一个可用版本号"""
        result = (
            self._session.query(func.max(PipelineSnapshotModel.version))
            .filter(PipelineSnapshotModel.template_id == template_id)
            .scalar()
        )
        return (result or 0) + 1

    def save(self, snapshot: PipelineSnapshot) -> None:
        existing = self._session.get(PipelineSnapshotModel, snapshot.id)
        if existing:
            # 快照不可变，不允许更新
            return
        orm = self._mapper.to_orm(snapshot)
        self._session.add(orm)

    def find_latest_versions(self, template_ids: list[str]) -> dict[str, int]:
        """批量查询多个模板的最新快照版本号"""
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
