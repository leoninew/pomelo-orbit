"""CI ProjectWebhook 仓储实现"""

from sqlalchemy.orm import Session

from pomelo_orbit.domain.ci.entities import ProjectWebhook
from pomelo_orbit.domain.ci.repositories import ProjectWebhookRepository
from pomelo_orbit.infrastructure.ci.mappers import ProjectWebhookMapper
from pomelo_orbit.infrastructure.ci.models import ProjectWebhookModel
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository


class ProjectWebhookRepositoryImpl(BaseRepository[ProjectWebhook, ProjectWebhookModel], ProjectWebhookRepository):
    """ProjectWebhook 仓储实现"""

    def __init__(self, session: Session):
        super().__init__(session, ProjectWebhookModel, ProjectWebhookMapper())

    def find_by_project(self, project_id: str) -> list[ProjectWebhook]:
        """查找项目的所有 Webhook

        Args:
            project_id: 项目 ID

        Returns:
            ProjectWebhook 列表
        """
        orms = self._session.query(ProjectWebhookModel).filter(ProjectWebhookModel.project_id == project_id).all()
        return [self._mapper.to_domain(orm) for orm in orms]

    def find_by_template(self, template_id: str) -> list[ProjectWebhook]:
        """查找使用指定模板的所有 Webhook

        Args:
            template_id: 模板 ID

        Returns:
            ProjectWebhook 列表
        """
        orms = self._session.query(ProjectWebhookModel).filter(ProjectWebhookModel.template_id == template_id).all()
        return [self._mapper.to_domain(orm) for orm in orms]
