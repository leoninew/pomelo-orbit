from typing import Annotated

from fastapi import Depends, Query

from pomelo_orbit.application.project.di import get_project_service
from pomelo_orbit.application.project.project_service import ProjectService
from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.domain.project.entities import Project
from pomelo_orbit.interfaces.api.auth.router import get_current_user


def get_current_project(
    project_id: Annotated[str, Query(alias="projectId", min_length=1)],
    current_user: Annotated[User, Depends(get_current_user)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
) -> Project:
    return project_service.get_project(current_user.id, project_id)
