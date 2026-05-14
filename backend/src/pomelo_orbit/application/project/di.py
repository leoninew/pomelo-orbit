from typing import Annotated

from fastapi import Depends

from pomelo_orbit.application.project.project_service import ProjectService
from pomelo_orbit.domain.project.repositories import ProjectRepository
from pomelo_orbit.infrastructure.project.di import get_project_repo


def get_project_service(
    project_repo: Annotated[ProjectRepository, Depends(get_project_repo)],
) -> ProjectService:
    return ProjectService(project_repo=project_repo)
