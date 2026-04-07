"""项目管理 API"""

import logging
import math
from typing import Annotated

from fastapi import APIRouter, BackgroundTasks, Depends, Query

from pomelo_orbit.application.ci.di import get_pipeline_service
from pomelo_orbit.application.ci.pipeline_service import PipelineService
from pomelo_orbit.domain.ci.value_objects import PipelineRunTrigger
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.ci.dto.project import ProjectCreateReq, ProjectResp, ProjectUpdateReq
from pomelo_orbit.interfaces.api.ci.dto.run import PipelineRunResp, TriggerPipelineReq
from pomelo_orbit.interfaces.api.common import PaginatedResp

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/projects", tags=["projects"])


@router.get("", response_model=PaginatedResp[ProjectResp])
def list_projects(
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 20,
) -> PaginatedResp[ProjectResp]:
    projects, credential_names, total = pipeline_service.list_projects(page=page, per_page=per_page)
    return PaginatedResp(
        items=[
            ProjectResp.from_domain(p, cred_name) for p, cred_name in zip(projects, credential_names, strict=True)
        ],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.post("", response_model=ProjectResp, status_code=201)
def create_project(
    data: ProjectCreateReq,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> ProjectResp:
    project = pipeline_service.create_project(
        name=data.name,
        code=data.code,
        repository_url=data.repository_url,
        git_credential_id=data.git_credential_id,
        variable_overrides=data.variable_overrides,
        default_branch=data.default_branch,
    )
    project, credential_name = pipeline_service.get_project_with_credential_name(project.id)
    return ProjectResp.from_domain(project, credential_name)


@router.get("/{project_id}", response_model=ProjectResp)
def get_project(
    project_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> ProjectResp:
    project, credential_name = pipeline_service.get_project_with_credential_name(project_id)
    return ProjectResp.from_domain(project, credential_name)


@router.put("/{project_id}", response_model=ProjectResp)
def update_project(
    project_id: str,
    data: ProjectUpdateReq,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> ProjectResp:
    pipeline_service.update_project(
        project_id=project_id,
        name=data.name,
        repository_url=data.repository_url,
        variable_overrides=data.variable_overrides,
        git_credential_id=data.git_credential_id,
        default_branch=data.default_branch,
    )
    project, credential_name = pipeline_service.get_project_with_credential_name(project_id)
    return ProjectResp.from_domain(project, credential_name)


@router.delete("/{project_id}", status_code=204)
def delete_project(
    project_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> None:
    pipeline_service.delete_project(project_id)


@router.get("/{project_id}/runs", response_model=PaginatedResp[PipelineRunResp])
def list_project_runs(
    project_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 20,
) -> PaginatedResp[PipelineRunResp]:
    runs, total = pipeline_service.list_runs(project_id=project_id, page=page, per_page=per_page)
    return PaginatedResp(
        items=[PipelineRunResp.model_validate(r) for r in runs],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.post("/{project_id}/trigger", response_model=PipelineRunResp, status_code=201)
async def trigger_pipeline(
    project_id: str,
    data: TriggerPipelineReq,
    background_tasks: BackgroundTasks,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> PipelineRunResp:
    run, project, merged_vars, snapshot = pipeline_service.create_run(
        project_id=project_id,
        template_id=data.template_id,
        trigger=PipelineRunTrigger.MANUAL,
        trigger_ref=data.trigger_ref,
        runtime_variables=data.variables,
    )
    background_tasks.add_task(pipeline_service.execute_run, run, project, merged_vars, snapshot)
    return PipelineRunResp.model_validate(run)
