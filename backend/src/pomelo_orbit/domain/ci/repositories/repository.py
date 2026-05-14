"""CI 项目仓储接口"""

from abc import ABC, abstractmethod

from pomelo_orbit.domain.ci.entities import Repository


class RepositoryRepository(ABC):
    @abstractmethod
    def find_by_id(self, repository_id: str) -> Repository | None: ...

    @abstractmethod
    def find_by_id_in_project(self, project_id: str, repository_id: str) -> Repository | None: ...

    @abstractmethod
    def find_by_code(self, project_id: str, code: str) -> Repository | None: ...

    @abstractmethod
    def find_paginated_by_project_id(
        self, project_id: str, page: int, per_page: int, search: str | None = None
    ) -> tuple[list[Repository], int]: ...

    @abstractmethod
    def find_by_repository_url(self, project_id: str, repository_url: str) -> list[Repository]: ...

    @abstractmethod
    def has_running_pipelines(self, project_id: str, repository_id: str) -> bool: ...

    @abstractmethod
    def save(self, project: Repository) -> None: ...

    @abstractmethod
    def delete(self, project: Repository) -> None: ...
