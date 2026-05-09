from sqlalchemy import or_
from sqlalchemy.orm import Session

from pomelo_orbit.domain.cd.entities import Route
from pomelo_orbit.domain.cd.repositories import RouteRepository
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository
from pomelo_orbit.infrastructure.persistence.mappers import RouteMapper
from pomelo_orbit.infrastructure.persistence.models import RouteModel


class RouteRepositoryImpl(BaseRepository[Route, RouteModel], RouteRepository):
    """路由仓储实现"""

    def __init__(self, db: Session):
        super().__init__(db, RouteModel, RouteMapper)

    def find_paginated(self, page: int = 1, per_page: int = 20, search: str | None = None) -> tuple[list[Route], int]:
        """分页查询路由（支持按名称/域名/目标地址搜索）"""
        query = self._session.query(RouteModel)
        if search:
            pattern = f"%{search}%"
            query = query.filter(
                or_(
                    RouteModel.name.ilike(pattern),
                    RouteModel.domain.ilike(pattern),
                    RouteModel.target_url.ilike(pattern),
                )
            )
        total = query.count()
        orms = query.order_by(RouteModel.id.desc()).offset((page - 1) * per_page).limit(per_page).all()
        return [self._mapper.to_domain(orm) for orm in orms], total

    def find_by_domain(self, domain: str) -> Route | None:
        model = self._session.query(RouteModel).filter(RouteModel.domain == domain).first()
        return self._mapper.to_domain(model) if model else None

    def find_by_application(self, app_id: str) -> list[Route]:
        models = self._session.query(RouteModel).filter(RouteModel.domain.contains(app_id)).all()
        return [self._mapper.to_domain(m) for m in models]
