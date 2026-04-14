"""CI infrastructure 层依赖注入"""

from functools import lru_cache
from typing import Annotated

from dynaconf import Dynaconf
from fastapi import Depends
from sqlalchemy.orm import Session

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
from pomelo_orbit.infrastructure.ci.container import ContainerExecutor
from pomelo_orbit.infrastructure.ci.executor_impl import PipelineExecutorImpl
from pomelo_orbit.infrastructure.ci.repositories import (
    ArtifactRepositoryImpl,
    BuildStageRepositoryImpl,
    CredentialRepositoryImpl,
    PipelineRunRepositoryImpl,
    PipelineSnapshotRepositoryImpl,
    PipelineTemplateRepositoryImpl,
    RepositoryRepositoryImpl,
    RepositoryWebhookRepositoryImpl,
    StageRunRepositoryImpl,
)
from pomelo_orbit.infrastructure.config import get_settings
from pomelo_orbit.infrastructure.persistence.di import get_db
from pomelo_orbit.infrastructure.security import SecurityService


@lru_cache
def get_container_executor() -> ContainerExecutor:
    """ContainerExecutor 单例（复用 Docker client 连接）"""
    return ContainerExecutor()


def get_pipeline_run_repo(db: Annotated[Session, Depends(get_db)]) -> PipelineRunRepository:
    return PipelineRunRepositoryImpl(db)


def get_artifact_repo(db: Annotated[Session, Depends(get_db)]) -> ArtifactRepository:
    return ArtifactRepositoryImpl(db)


def get_stage_run_repo(db: Annotated[Session, Depends(get_db)]) -> StageRunRepository:
    return StageRunRepositoryImpl(db)


def get_repository_repo(db: Annotated[Session, Depends(get_db)]) -> RepositoryRepository:
    return RepositoryRepositoryImpl(db)


def get_template_repo(db: Annotated[Session, Depends(get_db)]) -> PipelineTemplateRepository:
    return PipelineTemplateRepositoryImpl(db)


def get_snapshot_repo(db: Annotated[Session, Depends(get_db)]) -> PipelineSnapshotRepository:
    return PipelineSnapshotRepositoryImpl(db)


def get_credential_repo(db: Annotated[Session, Depends(get_db)]) -> CredentialRepository:
    return CredentialRepositoryImpl(db)


def get_build_stage_repo(db: Annotated[Session, Depends(get_db)]) -> BuildStageRepository:
    return BuildStageRepositoryImpl(db)


def get_webhook_repo(db: Annotated[Session, Depends(get_db)]) -> RepositoryWebhookRepository:
    return RepositoryWebhookRepositoryImpl(db)


def get_pipeline_executor(
    stage_run_repo: Annotated[StageRunRepository, Depends(get_stage_run_repo)],
    artifact_repo: Annotated[ArtifactRepository, Depends(get_artifact_repo)],
    credential_repo: Annotated[CredentialRepository, Depends(get_credential_repo)],
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> PipelineExecutor:
    return PipelineExecutorImpl(
        stage_run_repo=stage_run_repo,
        container_executor=get_container_executor(),
        artifact_repo=artifact_repo,
        credential_repo=credential_repo,
        security_service=SecurityService(settings),
    )
