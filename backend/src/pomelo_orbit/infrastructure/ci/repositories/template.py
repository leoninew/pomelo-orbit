"""CI 流水线模板仓储实现"""

from sqlalchemy.orm import Session

from pomelo_orbit.domain.ci.entities import PipelineTemplate
from pomelo_orbit.domain.ci.repositories import PipelineTemplateRepository
from pomelo_orbit.infrastructure.ci.mappers import PipelineTemplateMapper
from pomelo_orbit.infrastructure.ci.models import PipelineTemplateModel, ProjectModel
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository


class PipelineTemplateRepositoryImpl(
    BaseRepository[PipelineTemplate, PipelineTemplateModel], PipelineTemplateRepository
):
    """流水线模板仓储"""

    def __init__(self, session: Session):
        super().__init__(session, PipelineTemplateModel, PipelineTemplateMapper())

    def is_referenced_by_projects(self, template_id: str) -> bool:
        """检查模板是否被项目引用"""
        return (
            self._session.query(ProjectModel).filter(ProjectModel.pipeline_template_id == template_id).first()
            is not None
        )

    def find_builtin(self) -> list[PipelineTemplate]:
        """查询内置模板"""
        orms = self._session.query(PipelineTemplateModel).filter(PipelineTemplateModel.is_builtin == 1).all()
        return [self._mapper.to_domain(orm) for orm in orms]
