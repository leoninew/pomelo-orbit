"""CI 模块依赖注入"""

import contextlib
from functools import lru_cache
from typing import Annotated

from dynaconf import Dynaconf
from fastapi import Depends
from sqlalchemy.orm import Session

from pomelo_orbit.application.ci_webhook_service import CIWebhookService
from pomelo_orbit.application.pipeline_service import PipelineService
from pomelo_orbit.infrastructure.ci.container import ContainerExecutor
from pomelo_orbit.infrastructure.ci.repositories import (
    ArtifactRepositoryImpl,
    CredentialRepositoryImpl,
    JobLogRepositoryImpl,
    JobRepositoryImpl,
    PipelineRunRepositoryImpl,
    PipelineTemplateRepositoryImpl,
    ProjectRepositoryImpl,
)
from pomelo_orbit.infrastructure.config import get_settings
from pomelo_orbit.infrastructure.persistence.database import get_session_factory
from pomelo_orbit.infrastructure.persistence.di import get_db


@lru_cache
def get_container_executor() -> ContainerExecutor:
    """ContainerExecutor 单例（复用 Docker client 连接）"""
    return ContainerExecutor()


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
        run_repo=PipelineRunRepositoryImpl(db),
        artifact_repo=ArtifactRepositoryImpl(db),
        job_repo=JobRepositoryImpl(db),
        job_log_repo=JobLogRepositoryImpl(db),
        global_variables=global_vars,
        session_factory=get_session_factory(),
    )


def get_ci_webhook_service(
    db: Annotated[Session, Depends(get_db)],
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
) -> CIWebhookService:
    return CIWebhookService(
        project_repo=ProjectRepositoryImpl(db),
        pipeline_service=pipeline_service,
    )
