from typing import Annotated

from fastapi import Depends
from sqlalchemy.orm import Session

from pomelo_orbit.domain.project.repositories import ProjectRepository
from pomelo_orbit.infrastructure.persistence.di import get_db
from pomelo_orbit.infrastructure.project.repositories import ProjectRepositoryImpl


def get_project_repo(db: Annotated[Session, Depends(get_db)]) -> ProjectRepository:
    return ProjectRepositoryImpl(db)
