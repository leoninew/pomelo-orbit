"""
配置文件仓储实现
"""

from sqlalchemy.orm import Session

from pomelo_orbit.domain.cd.entities import ApplicationConfigFile
from pomelo_orbit.domain.cd.repositories import ConfigFileRepository
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository
from pomelo_orbit.infrastructure.persistence.mappers import ApplicationConfigFileMapper
from pomelo_orbit.infrastructure.persistence.models import ApplicationConfigFileModel


class ConfigFileRepositoryImpl(BaseRepository[ApplicationConfigFile, ApplicationConfigFileModel], ConfigFileRepository):
    """配置文件仓储实现"""

    def __init__(self, db: Session):
        super().__init__(db, ApplicationConfigFileModel, ApplicationConfigFileMapper)

    def find_by_application(self, app_id: str) -> list[ApplicationConfigFile]:
        """查找应用的所有配置文件"""
        orms = (
            self._session.query(ApplicationConfigFileModel)
            .filter(ApplicationConfigFileModel.application_id == app_id)
            .order_by(ApplicationConfigFileModel.path)
            .all()
        )
        return [ApplicationConfigFileMapper.to_domain(orm) for orm in orms]
