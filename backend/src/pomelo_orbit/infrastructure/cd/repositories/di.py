"""Repositories - dependency injection."""

from typing import Annotated

from fastapi import Depends
from sqlalchemy.orm import Session

from pomelo_orbit.domain.cd.repositories import (
    ApplicationRepository,
    ApplicationRouteRepository,
    ApplicationServiceConfigRepository,
    ConfigFileRepository,
    DeploymentRepository,
    RouteRepository,
    UserRepository,
)
from pomelo_orbit.infrastructure.cd.repositories.application import ApplicationRepositoryImpl
from pomelo_orbit.infrastructure.cd.repositories.application_route import ApplicationRouteRepositoryImpl
from pomelo_orbit.infrastructure.cd.repositories.application_service_config import (
    ApplicationServiceConfigRepositoryImpl,
)
from pomelo_orbit.infrastructure.cd.repositories.config_file import ConfigFileRepositoryImpl
from pomelo_orbit.infrastructure.cd.repositories.deployment import DeploymentRepositoryImpl
from pomelo_orbit.infrastructure.cd.repositories.route import RouteRepositoryImpl
from pomelo_orbit.infrastructure.cd.repositories.user import UserRepositoryImpl
from pomelo_orbit.infrastructure.persistence.di import get_db


def get_route_repository(db: Annotated[Session, Depends(get_db)]) -> RouteRepository:
    return RouteRepositoryImpl(db)


def get_application_repo(db: Annotated[Session, Depends(get_db)]) -> ApplicationRepository:
    return ApplicationRepositoryImpl(db)


def get_application_route_repo(db: Annotated[Session, Depends(get_db)]) -> ApplicationRouteRepository:
    return ApplicationRouteRepositoryImpl(db)


def get_application_service_config_repo(
    db: Annotated[Session, Depends(get_db)],
) -> ApplicationServiceConfigRepository:
    return ApplicationServiceConfigRepositoryImpl(db)


def get_config_file_repo(db: Annotated[Session, Depends(get_db)]) -> ConfigFileRepository:
    return ConfigFileRepositoryImpl(db)


def get_deployment_repo(db: Annotated[Session, Depends(get_db)]) -> DeploymentRepository:
    return DeploymentRepositoryImpl(db)


def get_user_repo(db: Annotated[Session, Depends(get_db)]) -> UserRepository:
    return UserRepositoryImpl(db)
