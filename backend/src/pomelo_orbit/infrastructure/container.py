"""Dishka 依赖注入容器配置"""

import contextlib
from collections.abc import Iterator

from dishka import AsyncContainer, Provider, Scope, make_async_container
from dynaconf import Dynaconf
from sqlalchemy.orm import Session

from pomelo_orbit.application.cd.application_service import ApplicationService
from pomelo_orbit.application.ci.pipeline_run_service import PipelineRunService
from pomelo_orbit.domain.cd.application_manager import ApplicationManager
from pomelo_orbit.domain.cd.repositories import (
    ApplicationRepository,
    ApplicationRouteRepository,
    ApplicationServiceConfigRepository,
    ConfigFileRepository,
    DeploymentRepository,
)
from pomelo_orbit.domain.ci.executor import PipelineExecutor
from pomelo_orbit.domain.ci.repositories import (
    ArtifactRepository,
    BuildStageRepository,
    CredentialRepository,
    PipelineRunRepository,
    PipelineSnapshotRepository,
    PipelineTemplateRepository,
    RepositoryRepository,
    RepositoryWebhookRepository,
    StageRunRepository,
)
from pomelo_orbit.domain.ci.snapshot_manager import SnapshotManager
from pomelo_orbit.domain.ci.variable_resolver import VariableResolver
from pomelo_orbit.infrastructure.cd.docker.manager import ApplicationManagerImpl
from pomelo_orbit.infrastructure.cd.repositories.di import (
    get_application_repo,
    get_application_route_repo,
    get_application_service_config_repo,
    get_config_file_repo,
    get_deployment_repo,
)
from pomelo_orbit.infrastructure.ci.container import ContainerExecutor
from pomelo_orbit.infrastructure.ci.di import (
    get_artifact_repo,
    get_build_stage_repo,
    get_container_executor,
    get_credential_repo,
    get_pipeline_executor,
    get_pipeline_run_repo,
    get_repository_repo,
    get_snapshot_repo,
    get_stage_run_repo,
    get_template_repo,
    get_webhook_repo,
)
from pomelo_orbit.infrastructure.config import get_settings
from pomelo_orbit.infrastructure.persistence.database import get_session_factory
from pomelo_orbit.infrastructure.security import SecurityService


# Dishka 无法分析 @lru_cache 包装的函数签名，需要包一层普通函数
def _get_settings() -> Dynaconf:
    return get_settings()


def _get_app_manager(settings: Dynaconf) -> ApplicationManager:
    return ApplicationManagerImpl(settings)


# Dishka 无法分析 @lru_cache 包装的函数签名，需要包一层普通函数
def _get_container_executor() -> ContainerExecutor:
    return get_container_executor()


def _get_security_service(settings: Dynaconf) -> SecurityService:
    return SecurityService(settings)


def _get_variable_resolver(settings: Dynaconf) -> VariableResolver:
    global_vars: dict = {}
    # 读取失败时静默降级为空变量集，避免配置缺失导致服务启动失败
    with contextlib.suppress(Exception):
        ci_settings = settings.get("ci", {})
        global_vars = dict(ci_settings.get("global_variables", {}))
    return VariableResolver(global_variables=global_vars)


def _get_session() -> Iterator[Session]:
    factory = get_session_factory()
    session = factory()
    try:
        yield session
        session.commit()
    except Exception:
        session.rollback()
        raise
    finally:
        session.close()


def _get_snapshot_manager(snapshot_repo: PipelineSnapshotRepository) -> SnapshotManager:
    return SnapshotManager(snapshot_repo)


def _build_provider() -> Provider:
    provider = Provider()

    # APP scope
    provider.provide(_get_settings, scope=Scope.APP, provides=Dynaconf)
    provider.provide(_get_app_manager, scope=Scope.APP, provides=ApplicationManager)
    provider.provide(_get_security_service, scope=Scope.APP, provides=SecurityService)
    provider.provide(_get_container_executor, scope=Scope.APP, provides=ContainerExecutor)
    provider.provide(_get_variable_resolver, scope=Scope.APP, provides=VariableResolver)

    # REQUEST scope — session
    provider.provide(_get_session, scope=Scope.REQUEST, provides=Session)

    # REQUEST scope — CD
    provider.provide(get_application_repo, scope=Scope.REQUEST, provides=ApplicationRepository)
    provider.provide(get_application_route_repo, scope=Scope.REQUEST, provides=ApplicationRouteRepository)
    provider.provide(get_application_service_config_repo, scope=Scope.REQUEST, provides=ApplicationServiceConfigRepository)
    provider.provide(get_config_file_repo, scope=Scope.REQUEST, provides=ConfigFileRepository)
    provider.provide(get_deployment_repo, scope=Scope.REQUEST, provides=DeploymentRepository)
    provider.provide(ApplicationService, scope=Scope.REQUEST)

    # REQUEST scope — CI repos (reuse infrastructure/ci/di.py functions)
    provider.provide(get_pipeline_run_repo, scope=Scope.REQUEST, provides=PipelineRunRepository)
    provider.provide(get_artifact_repo, scope=Scope.REQUEST, provides=ArtifactRepository)
    provider.provide(get_stage_run_repo, scope=Scope.REQUEST, provides=StageRunRepository)
    provider.provide(get_repository_repo, scope=Scope.REQUEST, provides=RepositoryRepository)
    provider.provide(get_template_repo, scope=Scope.REQUEST, provides=PipelineTemplateRepository)
    provider.provide(get_snapshot_repo, scope=Scope.REQUEST, provides=PipelineSnapshotRepository)
    provider.provide(get_credential_repo, scope=Scope.REQUEST, provides=CredentialRepository)
    provider.provide(get_build_stage_repo, scope=Scope.REQUEST, provides=BuildStageRepository)
    provider.provide(get_webhook_repo, scope=Scope.REQUEST, provides=RepositoryWebhookRepository)

    # REQUEST scope — CI services
    provider.provide(_get_snapshot_manager, scope=Scope.REQUEST, provides=SnapshotManager)
    provider.provide(get_pipeline_executor, scope=Scope.REQUEST, provides=PipelineExecutor)
    provider.provide(PipelineRunService, scope=Scope.REQUEST)

    return provider


def create_container() -> AsyncContainer:
    return make_async_container(_build_provider())
