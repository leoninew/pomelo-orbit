"""
应用仓储实现
"""

from sqlalchemy.orm import Session

from pomelo_orbit.domain.cd.entities import Application
from pomelo_orbit.domain.cd.repositories import ApplicationRepository
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository
from pomelo_orbit.infrastructure.persistence.mappers import ApplicationMapper
from pomelo_orbit.infrastructure.persistence.models import ApplicationModel


class ApplicationRepositoryImpl(BaseRepository[Application, ApplicationModel], ApplicationRepository):
    """应用仓储实现"""

    def __init__(self, db: Session):
        super().__init__(db, ApplicationModel, ApplicationMapper)

    def find_by_name(self, name: str) -> Application | None:
        """根据名称查找应用"""
        orm = self._session.query(ApplicationModel).filter(ApplicationModel.name == name).first()
        return ApplicationMapper.to_domain(orm) if orm else None

    def find_by_code(self, code: str) -> Application | None:
        """根据编码查找应用"""
        orm = self._session.query(ApplicationModel).filter(ApplicationModel.code == code).first()
        return ApplicationMapper.to_domain(orm) if orm else None

    def find_paginated(  # type: ignore[override]
        self, project_id: str, page: int = 1, per_page: int = 20, search: str | None = None
    ) -> tuple[list[Application], int]:
        """分页查询应用（重写以支持自定义排序和搜索）"""
        query = self._session.query(ApplicationModel).filter(ApplicationModel.project_id == project_id)
        if search:
            query = query.filter(ApplicationModel.name.ilike(f"%{search}%"))
        total = query.count()
        offset = (page - 1) * per_page
        orms = query.order_by(ApplicationModel.id.desc()).offset(offset).limit(per_page).all()
        return [ApplicationMapper.to_domain(orm) for orm in orms], total
