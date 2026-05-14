from typing import Annotated

from fastapi import APIRouter, Depends

from pomelo_orbit.application.project.di import get_project_service
from pomelo_orbit.application.project.project_service import ProjectService
from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.project.dto import ProjectCreateReq, ProjectResp, ProjectUpdateReq

router = APIRouter(prefix="/project", tags=["project"])


@router.get("", response_model=list[ProjectResp])
def list_projects(
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> list[ProjectResp]:
    projects = project_service.list_projects(current_user.id)
    return [ProjectResp.model_validate(project) for project in projects]


@router.post("", response_model=ProjectResp, status_code=201)
def create_project(
    data: ProjectCreateReq,
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> ProjectResp:
    project = project_service.create_project(
        owner_user_id=current_user.id,
        name=data.name,
        code=data.code,
    )
    return ProjectResp.model_validate(project)


@router.get("/{project_id}", response_model=ProjectResp)
def get_project(
    project_id: str,
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> ProjectResp:
    project = project_service.get_project(current_user.id, project_id)
    return ProjectResp.model_validate(project)


@router.put("/{project_id}", response_model=ProjectResp)
def update_project(
    project_id: str,
    data: ProjectUpdateReq,
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> ProjectResp:
    project = project_service.update_project(
        owner_user_id=current_user.id,
        project_id=project_id,
        name=data.name,
        code=data.code,
    )
    return ProjectResp.model_validate(project)
