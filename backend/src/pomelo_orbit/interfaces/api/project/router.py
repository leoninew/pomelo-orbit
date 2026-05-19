from typing import Annotated

from fastapi import APIRouter, Depends

from pomelo_orbit.application.project.di import get_project_service
from pomelo_orbit.application.project.project_service import ProjectService
from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.project.dto import (
    ProjectCreateReq,
    ProjectMemberReq,
    ProjectMemberResp,
    ProjectResp,
    ProjectUpdateReq,
)

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
        user_id=current_user.id,
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
        user_id=current_user.id,
        project_id=project_id,
        name=data.name,
        code=data.code,
    )
    return ProjectResp.model_validate(project)


@router.post("/{project_id}/deprecate", status_code=204)
def deprecate_project(
    project_id: str,
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> None:
    project_service.deprecate_project(current_user.id, project_id)


@router.get("/{project_id}/member", response_model=list[ProjectMemberResp])
def list_project_members(
    project_id: str,
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> list[ProjectMemberResp]:
    members = project_service.list_members(current_user.id, project_id)
    return [ProjectMemberResp.from_domain(member) for member in members]


@router.post("/{project_id}/member", response_model=list[ProjectMemberResp])
def add_project_member(
    project_id: str,
    data: ProjectMemberReq,
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> list[ProjectMemberResp]:
    members = project_service.add_member(current_user.id, project_id, data.user_id)
    return [ProjectMemberResp.from_domain(member) for member in members]


@router.delete("/{project_id}/member/{user_id}", response_model=list[ProjectMemberResp])
def remove_project_member(
    project_id: str,
    user_id: str,
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> list[ProjectMemberResp]:
    members = project_service.remove_member(current_user.id, project_id, user_id)
    return [ProjectMemberResp.from_domain(member) for member in members]
