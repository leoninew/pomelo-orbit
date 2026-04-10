"""CI 模块依赖注入

按照 DDD 分层架构，为每个应用服务提供独立的依赖注入函数。
"""

import contextlib
from functools import lru_cache
from typing import Annotated

from dynaconf import Dynaconf
from fastapi import Depends
from sqlalchemy.orm import Session

from pomelo_orbit.application.ci import (
    CredentialService,
    PipelineRunService,
    RepositoryService,
    StageService,
    TemplateService,
    WebhookService,
)
from pomelo_orbit.domain.ci.executor import PipelineExecutor
from pomelo_orbit.domain.ci.snapshot_manager import SnapshotManager
from pomelo_orbit.domain.ci.variable_resolver import VariableResolver
from pomelo_orbit.infrastructure.ci.container import ContainerExecutor
from pomelo_orbit.infrastructure.ci.executor_impl import PipelineExecutorImpl
from pomelo_orbit.infrastructure.ci.repositories import (
    ArtifactRepositoryImpl,
    CredentialRepositoryImpl,
    PipelineRunRepositoryImpl,
    PipelineSnapshotRepositoryImpl,
    PipelineStageRepositoryImpl,
    PipelineTemplateRepositoryImpl,
    RepositoryRepositoryImpl,
    RepositoryWebhookRepositoryImpl,
    StageRunRepositoryImpl,
)
from pomelo_orbit.infrastructure.config import get_settings
from pomelo_orbit.infrastructure.persistence.database import get_session_factory
from pomelo_orbit.infrastructure.persistence.di import get_db
from pomelo_orbit.infrastructure.security import SecurityService


@lru_cache
def get_container_executor() -> ContainerExecutor:
    """ContainerExecutor 单例（复用 Docker client 连接）"""
    return ContainerExecutor()


@lru_cache
def _get_cached_executor_factory(settings: Dynaconf):
    """executor_factory 单例，避免每次请求重复创建"""
    security_service = SecurityService(settings)
    container_executor = get_container_executor()

    def factory(session: Session) -> PipelineExecutor:
        return PipelineExecutorImpl(
            stage_run_repo=StageRunRepositoryImpl(session),
            container_executor=container_executor,
            artifact_repo=ArtifactRepositoryImpl(session),
            credential_repo=CredentialRepositoryImpl(session),
            security_service=security_service,
        )

    return factory


def _get_variable_resolver(settings: Dynaconf) -> VariableResolver:
    """创建 VariableResolver 领域服务"""
    global_vars: dict = {}
    with contextlib.suppress(Exception):
        ci_settings = settings.get("ci", {})
        global_vars = dict(ci_settings.get("global_variables", {}))

    return VariableResolver(global_variables=global_vars)


# ── 应用服务依赖注入 ──────────────────────────────────────────────────────────


def get_variable_resolver(
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> VariableResolver:
    """获取 VariableResolver 领域服务"""
    return _get_variable_resolver(settings)


def get_credential_service(
    db: Annotated[Session, Depends(get_db)],
) -> CredentialService:
    """获取 Credential 应用服务"""
    return CredentialService(
        credential_repo=CredentialRepositoryImpl(db),
    )


def get_stage_service(
    db: Annotated[Session, Depends(get_db)],
) -> StageService:
    """获取 Stage 应用服务"""
    return StageService(
        stage_repo=PipelineStageRepositoryImpl(db),
    )


def get_repository_service(
    db: Annotated[Session, Depends(get_db)],
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> RepositoryService:
    """获取 Repository 应用服务"""
    return RepositoryService(
        repository_repo=RepositoryRepositoryImpl(db),
        credential_repo=CredentialRepositoryImpl(db),
        variable_resolver=_get_variable_resolver(settings),
    )


def get_template_service(
    db: Annotated[Session, Depends(get_db)],
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> TemplateService:
    """获取 Template 应用服务"""
    return TemplateService(
        template_repo=PipelineTemplateRepositoryImpl(db),
        snapshot_repo=PipelineSnapshotRepositoryImpl(db),
        webhook_repo=RepositoryWebhookRepositoryImpl(db),
        stage_repo=PipelineStageRepositoryImpl(db),
        variable_resolver=_get_variable_resolver(settings),
    )


def get_webhook_service(
    db: Annotated[Session, Depends(get_db)],
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> WebhookService:
    """获取 Webhook 应用服务"""
    return WebhookService(
        webhook_repo=RepositoryWebhookRepositoryImpl(db),
        repository_repo=RepositoryRepositoryImpl(db),
        template_repo=PipelineTemplateRepositoryImpl(db),
        security_service=SecurityService(settings),
    )


def get_pipeline_run_service(
    db: Annotated[Session, Depends(get_db)],
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> PipelineRunService:
    """获取 PipelineRun 应用服务"""
    snapshot_repo = PipelineSnapshotRepositoryImpl(db)
    return PipelineRunService(
        run_repo=PipelineRunRepositoryImpl(db),
        artifact_repo=ArtifactRepositoryImpl(db),
        stage_run_repo=StageRunRepositoryImpl(db),
        repository_repo=RepositoryRepositoryImpl(db),
        template_repo=PipelineTemplateRepositoryImpl(db),
        snapshot_repo=snapshot_repo,
        snapshot_manager=SnapshotManager(snapshot_repo),
        variable_resolver=_get_variable_resolver(settings),
        session_factory=get_session_factory(),
        executor_factory=_get_cached_executor_factory(settings),
    )
