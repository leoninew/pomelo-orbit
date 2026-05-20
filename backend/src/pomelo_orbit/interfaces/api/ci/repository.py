"""项目管理 API"""

import logging
import math
from dataclasses import MISSING
from typing import Annotated

from dishka import AsyncContainer
from dishka.integrations.fastapi import FromDishka, inject
from fastapi import APIRouter, BackgroundTasks, Depends, Query

from pomelo_orbit.application.ci.credential_service import CredentialService
from pomelo_orbit.application.ci.di import (
    get_credential_service,
    get_pipeline_run_service,
    get_repository_service,
    get_variable_resolver,
)
from pomelo_orbit.application.ci.pipeline_run_service import PipelineRunService
from pomelo_orbit.application.ci.repository_service import RepositoryService
from pomelo_orbit.application.project.di import get_project_service
from pomelo_orbit.application.project.project_service import ProjectService
from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.domain.ci.value_objects import PipelineRunTrigger
from pomelo_orbit.domain.ci.variable_resolver import VariableResolver
from pomelo_orbit.domain.exceptions import BusinessError
from pomelo_orbit.interfaces.api.auth.dependencies import get_current_user
from pomelo_orbit.interfaces.api.ci.dto.pipeline_run import PipelineRunResp, TriggerPipelineReq
from pomelo_orbit.interfaces.api.ci.dto.repository import (
    RepositoryCreateReq,
    RepositoryListResp,
    RepositoryResp,
    RepositoryUpdateReq,
)
from pomelo_orbit.interfaces.api.common import PaginatedResp
from pomelo_orbit.interfaces.api.utils import run_in_new_scope

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/repository", tags=["repository"])


@router.get("", response_model=PaginatedResp[RepositoryListResp])
def list_repository(
    repository_service: Annotated[RepositoryService, Depends(get_repository_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
    project_id: Annotated[str, Query()],
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 20,
    search: Annotated[str | None, Query()] = None,
) -> PaginatedResp[RepositoryListResp]:
    project_service.get_project(current_user.id, project_id)
    repositories, total = repository_service.list_repositories(project_id, page=page, per_page=per_page, search=search)
    return PaginatedResp(
        items=[RepositoryListResp.from_domain(repo) for repo in repositories],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


def _resolve_cred_name(credential_id: str | None, credential_service: CredentialService) -> str | None:
    if not credential_id:
        return None
    cred = credential_service.get_credential(credential_id)
    if not cred:
        raise BusinessError(f"凭据不存在: {credential_id}", status_code=404)
    return cred.name


@router.post("", response_model=RepositoryResp, status_code=201)
def create_repository(
    data: RepositoryCreateReq,
    repository_service: Annotated[RepositoryService, Depends(get_repository_service)],
    credential_service: Annotated[CredentialService, Depends(get_credential_service)],
    variable_resolver: Annotated[VariableResolver, Depends(get_variable_resolver)],
    project_id: Annotated[str, Query()],
) -> RepositoryResp:
    repository = repository_service.create_repository(
        project_id=project_id,
        name=data.name,
        code=data.code,
        repository_url=data.repository_url,
        git_credential_id=data.git_credential_id,
        variable_overrides=data.variable_overrides,
        default_branch=data.default_branch,
    )
    return RepositoryResp.from_domain(
        repository,
        _resolve_cred_name(repository.git_credential_id, credential_service),
        variable_declarations=variable_resolver.get_repository_variables(repository),
    )


@router.get("/{repository_id}", response_model=RepositoryResp)
def get_repository(
    repository_id: str,
    repository_service: Annotated[RepositoryService, Depends(get_repository_service)],
    credential_service: Annotated[CredentialService, Depends(get_credential_service)],
    variable_resolver: Annotated[VariableResolver, Depends(get_variable_resolver)],
) -> RepositoryResp:
    repository = repository_service.get_repository(repository_id)
    return RepositoryResp.from_domain(
        repository,
        _resolve_cred_name(repository.git_credential_id, credential_service),
        variable_declarations=variable_resolver.get_repository_variables(repository),
    )


@router.put("/{repository_id}", response_model=RepositoryResp)
def update_repository(
    repository_id: str,
    data: RepositoryUpdateReq,
    repository_service: Annotated[RepositoryService, Depends(get_repository_service)],
    credential_service: Annotated[CredentialService, Depends(get_credential_service)],
    variable_resolver: Annotated[VariableResolver, Depends(get_variable_resolver)],
) -> RepositoryResp:
    repository_service.update_repository(
        repository_id=repository_id,
        name=data.name,
        repository_url=data.repository_url,
        variable_overrides=(
            data.variable_overrides if "variable_overrides" in data.model_fields_set else MISSING  # type: ignore[arg-type]
        ),
        git_credential_id=(
            data.git_credential_id if "git_credential_id" in data.model_fields_set else MISSING  # type: ignore[arg-type]
        ),
        default_branch=data.default_branch,
    )

    repository = repository_service.get_repository(repository_id)
    return RepositoryResp.from_domain(
        repository,
        _resolve_cred_name(repository.git_credential_id, credential_service),
        variable_declarations=variable_resolver.get_repository_variables(repository),
    )


@router.delete("/{repository_id}", status_code=204)
def delete_repository(
    repository_id: str,
    repository_service: Annotated[RepositoryService, Depends(get_repository_service)],
    delete_workspace: Annotated[bool, Query()] = False,
) -> None:
    repository_service.delete_repository(repository_id, delete_workspace=delete_workspace)


@router.get("/{repository_id}/run", response_model=PaginatedResp[PipelineRunResp])
def list_repository_runs(
    repository_id: str,
    pipeline_run_service: Annotated[PipelineRunService, Depends(get_pipeline_run_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
    project_id: Annotated[str, Query()],
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 20,
) -> PaginatedResp[PipelineRunResp]:
    project_service.get_project(current_user.id, project_id)
    result = pipeline_run_service.list_runs(
        project_id=project_id, repository_id=repository_id, page=page, per_page=per_page
    )
    return PaginatedResp(
        items=[PipelineRunResp.model_validate(r) for r in result.runs],
        total=result.total,
        page=page,
        per_page=per_page,
        pages=math.ceil(result.total / per_page) if result.total > 0 else 1,
    )


@router.post("/{repository_id}/trigger", response_model=PipelineRunResp, status_code=201)
@inject
async def trigger_pipeline(
    repository_id: str,
    data: TriggerPipelineReq,
    background_tasks: BackgroundTasks,
    pipeline_run_service: Annotated[PipelineRunService, Depends(get_pipeline_run_service)],
    repository_service: Annotated[RepositoryService, Depends(get_repository_service)],
    container: FromDishka[AsyncContainer],
) -> PipelineRunResp:
    # 从 repository 获取 project_id
    repository = repository_service.get_repository(repository_id)

    result = pipeline_run_service.create_run(
        project_id=repository.project_id,
        repository_id=repository_id,
        template_id=data.template_id,
        trigger=PipelineRunTrigger.MANUAL,
        trigger_ref=data.trigger_ref,
        runtime_variables=data.variables,
    )

    async def _run() -> None:
        await run_in_new_scope(
            container,
            PipelineRunService,
            lambda svc: svc.execute_run(result.run, result.repository, result.merged_variables, result.snapshot),
        )

    background_tasks.add_task(_run)
    return PipelineRunResp.model_validate(result.run)
