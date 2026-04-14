"""CI 应用层依赖注入"""

import contextlib
from typing import Annotated

from dynaconf import Dynaconf
from fastapi import Depends

from pomelo_orbit.application.ci import (
    BuildStageService,
    CredentialService,
    PipelineRunService,
    RepositoryService,
    TemplateService,
    WebhookService,
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
from pomelo_orbit.infrastructure.ci.di import (
    get_artifact_repo,
    get_build_stage_repo,
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
from pomelo_orbit.infrastructure.security import SecurityService


def _get_variable_resolver(settings: Dynaconf) -> VariableResolver:
    global_vars: dict = {}
    with contextlib.suppress(Exception):
        ci_settings = settings.get("ci", {})
        global_vars = dict(ci_settings.get("global_variables", {}))
    return VariableResolver(global_variables=global_vars)


def get_variable_resolver(
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> VariableResolver:
    return _get_variable_resolver(settings)


def get_credential_service(
    credential_repo: Annotated[CredentialRepository, Depends(get_credential_repo)],
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> CredentialService:
    return CredentialService(
        credential_repo=credential_repo,
        security_service=SecurityService(settings),
    )


def get_stage_service(
    stage_repo: Annotated[BuildStageRepository, Depends(get_build_stage_repo)],
) -> BuildStageService:
    return BuildStageService(stage_repo=stage_repo)


def get_repository_service(
    repository_repo: Annotated[RepositoryRepository, Depends(get_repository_repo)],
    credential_repo: Annotated[CredentialRepository, Depends(get_credential_repo)],
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> RepositoryService:
    return RepositoryService(
        repository_repo=repository_repo,
        credential_repo=credential_repo,
        variable_resolver=_get_variable_resolver(settings),
    )


def get_template_service(
    template_repo: Annotated[PipelineTemplateRepository, Depends(get_template_repo)],
    snapshot_repo: Annotated[PipelineSnapshotRepository, Depends(get_snapshot_repo)],
    webhook_repo: Annotated[RepositoryWebhookRepository, Depends(get_webhook_repo)],
    stage_repo: Annotated[BuildStageRepository, Depends(get_build_stage_repo)],
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> TemplateService:
    return TemplateService(
        template_repo=template_repo,
        snapshot_repo=snapshot_repo,
        webhook_repo=webhook_repo,
        stage_repo=stage_repo,
        variable_resolver=_get_variable_resolver(settings),
    )


def get_webhook_service(
    webhook_repo: Annotated[RepositoryWebhookRepository, Depends(get_webhook_repo)],
    repository_repo: Annotated[RepositoryRepository, Depends(get_repository_repo)],
    template_repo: Annotated[PipelineTemplateRepository, Depends(get_template_repo)],
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> WebhookService:
    return WebhookService(
        webhook_repo=webhook_repo,
        repository_repo=repository_repo,
        template_repo=template_repo,
        security_service=SecurityService(settings),
    )


def get_pipeline_run_service(
    run_repo: Annotated[PipelineRunRepository, Depends(get_pipeline_run_repo)],
    artifact_repo: Annotated[ArtifactRepository, Depends(get_artifact_repo)],
    stage_run_repo: Annotated[StageRunRepository, Depends(get_stage_run_repo)],
    repository_repo: Annotated[RepositoryRepository, Depends(get_repository_repo)],
    template_repo: Annotated[PipelineTemplateRepository, Depends(get_template_repo)],
    snapshot_repo: Annotated[PipelineSnapshotRepository, Depends(get_snapshot_repo)],
    pipeline_executor: Annotated[PipelineExecutor, Depends(get_pipeline_executor)],
    settings: Annotated[Dynaconf, Depends(get_settings)],
) -> PipelineRunService:
    return PipelineRunService(
        run_repo=run_repo,
        artifact_repo=artifact_repo,
        stage_run_repo=stage_run_repo,
        repository_repo=repository_repo,
        template_repo=template_repo,
        snapshot_repo=snapshot_repo,
        snapshot_manager=SnapshotManager(snapshot_repo),
        variable_resolver=_get_variable_resolver(settings),
        pipeline_executor=pipeline_executor,
    )
