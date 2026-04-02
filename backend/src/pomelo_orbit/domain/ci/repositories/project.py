"""CI 项目仓储接口"""

from abc import ABC, abstractmethod

from pomelo_orbit.domain.ci.entities import Project


class ProjectRepository(ABC):
    @abstractmethod
    def find_by_id(self, project_id: str) -> Project | None: ...

    @abstractmethod
    def find_paginated(self, page: int, per_page: int) -> tuple[list[Project], int]: ...

    @abstractmethod
    def find_by_repository_url(self, repository_url: str) -> list[Project]: ...

    @abstractmethod
    def has_running_pipelines(self, project_id: str) -> bool: ...

    @abstractmethod
    def save(self, project: Project) -> None: ...

    @abstractmethod
    def delete(self, project: Project) -> None: ...
