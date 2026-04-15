from sqlalchemy.orm import Session

from pomelo_orbit.domain.cd.entities import ApplicationRoute
from pomelo_orbit.domain.cd.repositories import ApplicationRouteRepository
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository
from pomelo_orbit.infrastructure.persistence.mappers import ApplicationRouteMapper
from pomelo_orbit.infrastructure.persistence.models import ApplicationRouteModel


class ApplicationRouteRepositoryImpl(
    BaseRepository[ApplicationRoute, ApplicationRouteModel], ApplicationRouteRepository
):
    def __init__(self, db: Session):
        super().__init__(db, ApplicationRouteModel, ApplicationRouteMapper)

    def find_by_application(self, application_id: str) -> list[ApplicationRoute]:
        orms = (
            self._session.query(ApplicationRouteModel)
            .filter(ApplicationRouteModel.application_id == application_id)
            .all()
        )
        return [ApplicationRouteMapper.to_domain(o) for o in orms]
