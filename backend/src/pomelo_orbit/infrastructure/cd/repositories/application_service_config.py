from sqlalchemy.orm import Session

from pomelo_orbit.domain.cd.entities import ApplicationServiceConfig
from pomelo_orbit.domain.cd.repositories import ApplicationServiceConfigRepository
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository
from pomelo_orbit.infrastructure.persistence.mappers import ApplicationServiceConfigMapper
from pomelo_orbit.infrastructure.persistence.models import ApplicationServiceConfigModel


class ApplicationServiceConfigRepositoryImpl(
    BaseRepository[ApplicationServiceConfig, ApplicationServiceConfigModel], ApplicationServiceConfigRepository
):
    def __init__(self, db: Session):
        super().__init__(db, ApplicationServiceConfigModel, ApplicationServiceConfigMapper)

    def find_by_application(self, application_id: str) -> list[ApplicationServiceConfig]:
        orms = (
            self._session.query(ApplicationServiceConfigModel)
            .filter(ApplicationServiceConfigModel.application_id == application_id)
            .order_by(ApplicationServiceConfigModel.service_name)
            .all()
        )
        return [ApplicationServiceConfigMapper.to_domain(orm) for orm in orms]

    def find_by_application_and_service(
        self, application_id: str, service_name: str
    ) -> ApplicationServiceConfig | None:
        orm = (
            self._session.query(ApplicationServiceConfigModel)
            .filter(ApplicationServiceConfigModel.application_id == application_id)
            .filter(ApplicationServiceConfigModel.service_name == service_name)
            .first()
        )
        return ApplicationServiceConfigMapper.to_domain(orm) if orm else None
