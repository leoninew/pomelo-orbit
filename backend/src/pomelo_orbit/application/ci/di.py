"""CI 模块依赖注入"""

import contextlib
from functools import lru_cache
from typing import Annotated

from dynaconf import Dynaconf
from fastapi import Depends
from sqlalchemy.orm import Session

from pomelo_orbit.application.ci.pipeline_service import PipelineService
from pomelo_orbit.application.ci.webhook_service import CIWebhookService
from pomelo_orbit.domain.ci.executor import PipelineExecutor
from pomelo_orbit.infrastructure.ci.container import ContainerExecutor
from pomelo_orbit.infrastructure.ci.executor_impl import PipelineExecutorImpl
from pomelo_orbit.infrastructure.ci.repositories import (
    ArtifactRepositoryImpl,
    CredentialRepositoryImpl,
    JobLogRepositoryImpl,
    JobRepositoryImpl,
    PipelineRunRepositoryImpl,
    PipelineSnapshotRepositoryImpl,
    PipelineTemplateRepositoryImpl,
    ProjectRepositoryImpl,
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
            job_repo=JobRepositoryImpl(session),
            job_log_repo=JobLogRepositoryImpl(session),
            container_executor=container_executor,
            artifact_repo=ArtifactRepositoryImpl(session),
            credential_repo=CredentialRepositoryImpl(session),
            security_service=security_service,
        )

    return factory


def get_pipeline_service(
    db: Annotated[Session, Depends(get_db)],
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> PipelineService:
    global_vars: dict = {}
    with contextlib.suppress(Exception):
        global_vars = dict(settings.get("ci", {}).get("global_variables", {}))

    return PipelineService(
        project_repo=ProjectRepositoryImpl(db),
        credential_repo=CredentialRepositoryImpl(db),
        template_repo=PipelineTemplateRepositoryImpl(db),
        snapshot_repo=PipelineSnapshotRepositoryImpl(db),
        run_repo=PipelineRunRepositoryImpl(db),
        artifact_repo=ArtifactRepositoryImpl(db),
        job_repo=JobRepositoryImpl(db),
        job_log_repo=JobLogRepositoryImpl(db),
        session_factory=get_session_factory(),
        executor_factory=_get_cached_executor_factory(settings),
        global_variables=global_vars,
    )


def get_ci_webhook_service(
    db: Annotated[Session, Depends(get_db)],
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
) -> CIWebhookService:
    return CIWebhookService(
        project_repo=ProjectRepositoryImpl(db),
        pipeline_service=pipeline_service,
    )
