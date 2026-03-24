"""
凭据仓储实现
"""

from typing import Annotated

from fastapi import Depends
from sqlalchemy.orm import Session

from pomelo_orbit.domain.entities import Credential
from pomelo_orbit.domain.repositories import CredentialRepository
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository
from pomelo_orbit.infrastructure.persistence.di import get_db
from pomelo_orbit.infrastructure.persistence.mappers import CredentialMapper
from pomelo_orbit.infrastructure.persistence.models import CredentialModel


class CredentialRepositoryImpl(BaseRepository[Credential, CredentialModel], CredentialRepository):
    """凭据仓储实现"""

    def __init__(self, db: Session):
        super().__init__(db, CredentialModel, CredentialMapper)

    def find_by_application(self, app_id: str) -> Credential | None:
        """查找应用的凭据"""
        orm = self._session.query(CredentialModel).filter(CredentialModel.application_id == app_id).first()
        return CredentialMapper.to_domain(orm) if orm else None

    def find_paginated(
        self, page: int = 1, per_page: int = 10, search: str | None = None
    ) -> tuple[list[Credential], int]:
        """分页查询凭据"""
        query = self._session.query(CredentialModel)
        if search:
            query = query.filter(CredentialModel.name.contains(search))
        total = query.count()
        offset = (page - 1) * per_page
        orms = query.order_by(CredentialModel.created_at.desc()).offset(offset).limit(per_page).all()
        return [CredentialMapper.to_domain(orm) for orm in orms], total

    def find_by_name_in_app(self, app_id: str, name: str) -> Credential | None:
        """在应用内根据名称查找凭据"""
        orm = (
            self._session.query(CredentialModel)
            .filter(CredentialModel.application_id == app_id, CredentialModel.name == name)
            .first()
        )
        return CredentialMapper.to_domain(orm) if orm else None


def get_credential_repository(db: Annotated[Session, Depends(get_db)]) -> CredentialRepository:
    """获取凭据仓储实例（依赖注入）- 返回接口类型"""
    return CredentialRepositoryImpl(db)
