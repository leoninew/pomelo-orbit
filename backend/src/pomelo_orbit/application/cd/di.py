"""Application layer - dependency injection."""

from typing import Annotated

from dynaconf import Dynaconf
from fastapi import Depends

from pomelo_orbit.application.cd.application_service import ApplicationService
from pomelo_orbit.application.cd.deployment_service import DeploymentService
from pomelo_orbit.application.cd.route_service import RouteService
from pomelo_orbit.application.cd.traefik_service import TraefikService
from pomelo_orbit.application.settings.setting_service import SettingService
from pomelo_orbit.domain.cd.application_manager import ApplicationManager
from pomelo_orbit.domain.cd.repositories import (
    ApplicationRepository,
    ApplicationRouteRepository,
    ConfigFileRepository,
    DeploymentRepository,
    RouteRepository,
)
from pomelo_orbit.infrastructure.cd.cert.di import get_mkcert_service
from pomelo_orbit.infrastructure.cd.cert.mkcert import MkcertService
from pomelo_orbit.infrastructure.cd.docker.manager import ApplicationManagerImpl
from pomelo_orbit.infrastructure.cd.repositories.di import (
    get_application_repo,
    get_application_route_repo,
    get_config_file_repo,
    get_deployment_repo,
    get_route_repository,
)
from pomelo_orbit.infrastructure.cd.traefik.di import get_traefik_api_client, get_traefik_manager
from pomelo_orbit.infrastructure.cd.traefik.manager import TraefikManager
from pomelo_orbit.infrastructure.config import get_settings
from pomelo_orbit.infrastructure.traefik import TraefikAPIClient


def get_app_manager(
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> ApplicationManager:
    """获取应用管理器实例"""
    return ApplicationManagerImpl(settings)


def get_application_service(
    app_repo: Annotated[ApplicationRepository, Depends(get_application_repo)],
    deployment_repo: Annotated[DeploymentRepository, Depends(get_deployment_repo)],
    config_file_repo: Annotated[ConfigFileRepository, Depends(get_config_file_repo)],
    app_route_repo: Annotated[ApplicationRouteRepository, Depends(get_application_route_repo)],
    app_manager: Annotated[ApplicationManager, Depends(get_app_manager)],
) -> ApplicationService:
    """获取应用服务实例"""
    return ApplicationService(
        app_repo=app_repo,
        deployment_repo=deployment_repo,
        config_file_repo=config_file_repo,
        app_route_repo=app_route_repo,
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
    deployment_repo: Annotated[DeploymentRepository, Depends(get_deployment_repo)],
    app_repo: Annotated[ApplicationRepository, Depends(get_application_repo)],
    app_manager: Annotated[ApplicationManager, Depends(get_app_manager)],
) -> DeploymentService:
    """获取部署服务实例"""
    return DeploymentService(
        deployment_repo=deployment_repo,
        app_repo=app_repo,
        app_manager=app_manager,
    )


def get_setting_service(
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> SettingService:
    """获取配置服务实例"""
    return SettingService(settings=settings)


def get_traefik_service(
    route_repo: Annotated[RouteRepository, Depends(get_route_repository)],
    traefik_client: Annotated[TraefikAPIClient, Depends(get_traefik_api_client)],
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> TraefikService:
    """获取 Traefik 服务实例"""
    return TraefikService(
        route_repo=route_repo,
        traefik_client=traefik_client,
        settings=settings,
    )
