"""Repositories - dependency injection."""

from typing import Annotated

from fastapi import Depends
from sqlalchemy.orm import Session

from pomelo_orbit.domain.cd.repositories import RouteRepository
from pomelo_orbit.infrastructure.cd.repositories.route import RouteRepositoryImpl
from pomelo_orbit.infrastructure.persistence.di import get_db


def get_route_repository(db: Annotated[Session, Depends(get_db)]) -> RouteRepository:
    """获取路由仓储实例"""
    return RouteRepositoryImpl(db)
