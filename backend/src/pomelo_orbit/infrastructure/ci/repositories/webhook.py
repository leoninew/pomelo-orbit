"""CI RepositoryWebhook 仓储实现"""

from sqlalchemy.orm import Session

from pomelo_orbit.domain.ci.entities import RepositoryWebhook
from pomelo_orbit.domain.ci.repositories import RepositoryWebhookRepository
from pomelo_orbit.infrastructure.ci.mappers import ProjectWebhookMapper
from pomelo_orbit.infrastructure.ci.models import ProjectWebhookModel
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository


class RepositoryWebhookRepositoryImpl(
    BaseRepository[RepositoryWebhook, ProjectWebhookModel], RepositoryWebhookRepository
):
    """RepositoryWebhook 仓储实现"""

    def __init__(self, session: Session):
        super().__init__(session, ProjectWebhookModel, ProjectWebhookMapper())

    def find_by_repository(self, repository_id: str) -> list[RepositoryWebhook]:
        """查找项目的所有 Webhook

        Args:
            repository_id: 项目 ID

        Returns:
            RepositoryWebhook 列表
        """
        orms = self._session.query(ProjectWebhookModel).filter(ProjectWebhookModel.repository_id == repository_id).all()
        return [self._mapper.to_domain(orm) for orm in orms]

    def find_by_template(self, template_id: str) -> list[RepositoryWebhook]:
        """查找使用指定模板的所有 Webhook

        Args:
            template_id: 模板 ID

        Returns:
            RepositoryWebhook 列表
        """
        orms = self._session.query(ProjectWebhookModel).filter(ProjectWebhookModel.template_id == template_id).all()
        return [self._mapper.to_domain(orm) for orm in orms]
