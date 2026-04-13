"""
通用仓储基类
"""

from typing import Any

from sqlalchemy.orm import Session

from pomelo_orbit.infrastructure.persistence.models import Base


class BaseRepository[TDomain, TORM: Base]:
    """通用仓储基类，提供常用 CRUD 操作"""

    def __init__(self, session: Session, orm_class: type[TORM], mapper: Any):
        self._session = session
        self._orm_class = orm_class
        self._mapper = mapper

    def find_by_id(self, entity_id: str) -> TDomain | None:
        """根据 ID 查找实体"""
        orm = (
            self._session.query(self._orm_class)
            .filter(self._orm_class.id == entity_id)  # type: ignore[attr-defined]
            .first()
        )
        return self._mapper.to_domain(orm) if orm else None

    def find_all(self) -> list[TDomain]:
        """查找所有实体"""
        orms = self._session.query(self._orm_class).all()
        return [self._mapper.to_domain(orm) for orm in orms]

    def find_paginated(self, page: int = 1, per_page: int = 20, search: str | None = None) -> tuple[list[TDomain], int]:
        """分页查询"""
        query = self._session.query(self._orm_class)

        if search:
            query = query.filter(
                self._orm_class.name.ilike(f"%{search}%")  # type: ignore[attr-defined]
            )

        total = query.count()
        orms = (
            query.order_by(self._orm_class.id.desc())  # type: ignore[attr-defined]
            .offset((page - 1) * per_page)
            .limit(per_page)
            .all()
        )
        return [self._mapper.to_domain(orm) for orm in orms], total

    def save(self, entity: TDomain) -> None:
        """保存实体"""
        orm = self._mapper.to_orm(entity)

        if hasattr(entity, "id") and entity.id:
            orm = self._session.merge(orm)
        else:
            self._session.add(orm)

    def delete(self, entity: TDomain) -> None:
        """删除实体"""
        # 需要先从数据库查询出来才能删除
        if hasattr(entity, "id"):
            db_orm = (
                self._session.query(self._orm_class)
                .filter(self._orm_class.id == entity.id)  # type: ignore[attr-defined]
                .first()
            )
            if db_orm:
                self._session.delete(db_orm)

    def exists_by_id(self, entity_id: str) -> bool:
        """检查实体是否存在"""
        return (
            self._session.query(self._orm_class)
            .filter(self._orm_class.id == entity_id)  # type: ignore[attr-defined]
            .first()
            is not None
        )

    def commit(self) -> None:
        """提交事务"""
        self._session.commit()

    def rollback(self) -> None:
        """回滚事务"""
        self._session.rollback()
