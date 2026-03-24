"""
应用仓储实现
"""

from typing import Annotated

from fastapi import Depends
from sqlalchemy.orm import Session

from pomelo_orbit.domain.entities import Application
from pomelo_orbit.domain.repositories import ApplicationRepository
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository
from pomelo_orbit.infrastructure.persistence.di import get_db
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

    def find_paginated(
        self, page: int = 1, per_page: int = 10, search: str | None = None
    ) -> tuple[list[Application], int]:
        """分页查询应用（重写以支持自定义排序和搜索）"""
        query = self._session.query(ApplicationModel)
        if search:
            query = query.filter(ApplicationModel.name.contains(search))
        total = query.count()
        offset = (page - 1) * per_page
        orms = query.order_by(ApplicationModel.created_at.desc()).offset(offset).limit(per_page).all()
        return [ApplicationMapper.to_domain(orm) for orm in orms], total


def get_application_repository(db: Annotated[Session, Depends(get_db)]) -> ApplicationRepository:
    """获取应用仓储实例（依赖注入）- 返回接口类型"""
    return ApplicationRepositoryImpl(db)
