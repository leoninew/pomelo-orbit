from sqlalchemy.orm import Session

from pomelo_orbit.domain.entities import WebhookEvent
from pomelo_orbit.domain.repositories import WebhookEventRepository
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository
from pomelo_orbit.infrastructure.persistence.mappers import WebhookEventMapper
from pomelo_orbit.infrastructure.persistence.models import WebhookEventModel


class WebhookEventRepositoryImpl(BaseRepository[WebhookEvent, WebhookEventModel], WebhookEventRepository):
    """Webhook 事件仓储实现"""

    def __init__(self, db: Session):
        super().__init__(db, WebhookEventModel, WebhookEventMapper)

    def find_by_application(self, app_id: str, page: int = 1, per_page: int = 20) -> tuple[list[WebhookEvent], int]:
        query = self._session.query(WebhookEventModel).filter(WebhookEventModel.matched_application_id == app_id)
        total = query.count()
        models = query.order_by(WebhookEventModel.id.desc()).offset((page - 1) * per_page).limit(per_page).all()
        return [self._mapper.to_domain(m) for m in models], total
