"""Application layer - dependency injection."""

from typing import Annotated

from dynaconf import Dynaconf
from fastapi import Depends
from sqlalchemy.orm import Session

from pomelo_orbit.application.cd.application_service import ApplicationService
from pomelo_orbit.application.cd.deployment_service import DeploymentService
from pomelo_orbit.application.cd.route_service import RouteService
from pomelo_orbit.application.setting_service import SettingService
from pomelo_orbit.domain.cd.application_manager import ApplicationManager
from pomelo_orbit.domain.cd.repositories import (
    RouteRepository,
)
from pomelo_orbit.infrastructure.cd.cert.di import get_mkcert_service
from pomelo_orbit.infrastructure.cd.cert.mkcert import MkcertService
from pomelo_orbit.infrastructure.cd.docker.manager import ApplicationManagerImpl
from pomelo_orbit.infrastructure.cd.repositories.di import get_route_repository
from pomelo_orbit.infrastructure.cd.traefik.di import get_traefik_manager
from pomelo_orbit.infrastructure.cd.traefik.manager import TraefikManager
from pomelo_orbit.infrastructure.config import get_settings
from pomelo_orbit.infrastructure.persistence.di import get_db
from pomelo_orbit.infrastructure.repositories import (
    ApplicationRepositoryImpl,
    ConfigFileRepositoryImpl,
    DeploymentRepositoryImpl,
)


def get_app_manager(
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> ApplicationManager:
    """获取应用管理器实例"""
    return ApplicationManagerImpl(settings)


def get_application_service(
    db: Annotated[Session, Depends(get_db)],
    app_manager: Annotated[ApplicationManager, Depends(get_app_manager)],
) -> ApplicationService:
    """获取应用服务实例"""
    return ApplicationService(
        app_repo=ApplicationRepositoryImpl(db),
        deployment_repo=DeploymentRepositoryImpl(db),
        config_file_repo=ConfigFileRepositoryImpl(db),
        app_manager=app_manager,
    )


def get_route_service(
    route_repo: Annotated[RouteRepository, Depends(get_route_repository)],
    traefik_manager: Annotated[TraefikManager, Depends(get_traefik_manager)],
    mkcert_service: Annotated[MkcertService, Depends(get_mkcert_service)],
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> RouteService:
    """获取路由服务实例"""
    return RouteService(
        route_repo=route_repo,
        traefik_manager=traefik_manager,
        mkcert_service=mkcert_service,
        settings=settings,
    )


def get_deployment_service(
    db: Annotated[Session, Depends(get_db)],
    app_manager: Annotated[ApplicationManager, Depends(get_app_manager)],
) -> DeploymentService:
    """获取部署服务实例"""
    return DeploymentService(
        deployment_repo=DeploymentRepositoryImpl(db),
        app_repo=ApplicationRepositoryImpl(db),
        app_manager=app_manager,
    )


def get_setting_service(
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> SettingService:
    """获取配置服务实例"""
    return SettingService(settings=settings)
