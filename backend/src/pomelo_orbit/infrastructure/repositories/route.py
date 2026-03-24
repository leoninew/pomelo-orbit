from sqlalchemy.orm import Session

from pomelo_orbit.domain.entities import Route
from pomelo_orbit.domain.repositories import RouteRepository
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository
from pomelo_orbit.infrastructure.persistence.mappers import RouteMapper
from pomelo_orbit.infrastructure.persistence.models import RouteModel


class RouteRepositoryImpl(BaseRepository[Route, RouteModel], RouteRepository):
    """路由仓储实现"""

    def __init__(self, db: Session):
        super().__init__(db, RouteModel, RouteMapper)

    def find_by_domain(self, domain: str) -> Route | None:
        model = self._session.query(RouteModel).filter(RouteModel.domain == domain).first()
        return self._mapper.to_domain(model) if model else None

    def find_by_application(self, app_id: str) -> list[Route]:
        models = self._session.query(RouteModel).filter(RouteModel.domain.contains(app_id)).all()
        return [self._mapper.to_domain(m) for m in models]
