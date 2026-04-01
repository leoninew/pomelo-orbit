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
    ArtifactRepository,
    CredentialRepository,
    PipelineRunRepository,
    PipelineTemplateRepository,
    ProjectRepository,
)
from pomelo_orbit.infrastructure.config import get_settings
from pomelo_orbit.infrastructure.persistence.database import get_session_factory
from pomelo_orbit.infrastructure.persistence.di import get_db


@lru_cache
def get_container_executor() -> ContainerExecutor:
    """ContainerExecutor 单例（复用 Docker client 连接）"""
    return ContainerExecutor()


def get_project_repo(db: Annotated[Session, Depends(get_db)]) -> ProjectRepository:
    return ProjectRepository(db)


def get_credential_repo(db: Annotated[Session, Depends(get_db)]) -> CredentialRepository:
    return CredentialRepository(db)


def get_template_repo(db: Annotated[Session, Depends(get_db)]) -> PipelineTemplateRepository:
    return PipelineTemplateRepository(db)


def get_run_repo(db: Annotated[Session, Depends(get_db)]) -> PipelineRunRepository:
    return PipelineRunRepository(db)


def get_artifact_repo(db: Annotated[Session, Depends(get_db)]) -> ArtifactRepository:
    return ArtifactRepository(db)


def get_pipeline_service(
    project_repo: Annotated[ProjectRepository, Depends(get_project_repo)],
    credential_repo: Annotated[CredentialRepository, Depends(get_credential_repo)],
    template_repo: Annotated[PipelineTemplateRepository, Depends(get_template_repo)],
    run_repo: Annotated[PipelineRunRepository, Depends(get_run_repo)],
    artifact_repo: Annotated[ArtifactRepository, Depends(get_artifact_repo)],
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> PipelineService:
    global_vars: dict = {}
    with contextlib.suppress(Exception):
        global_vars = dict(settings.get("ci", {}).get("global_variables", {}))

    return PipelineService(
        project_repo=project_repo,
        credential_repo=credential_repo,
        template_repo=template_repo,
        run_repo=run_repo,
        artifact_repo=artifact_repo,
        global_variables=global_vars,
        session_factory=get_session_factory(),
    )


def get_ci_webhook_service(
    project_repo: Annotated[ProjectRepository, Depends(get_project_repo)],
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
) -> CIWebhookService:
    return CIWebhookService(
        project_repo=project_repo,
        pipeline_service=pipeline_service,
    )
