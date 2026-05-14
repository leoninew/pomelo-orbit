from abc import ABC, abstractmethod

from pomelo_orbit.domain.project.entities import Project


class ProjectRepository(ABC):
    @abstractmethod
    def find_by_id(self, project_id: str) -> Project | None: ...

    @abstractmethod
    def find_by_owner(self, owner_user_id: str) -> list[Project]: ...

    @abstractmethod
    def find_by_owner_and_id(self, owner_user_id: str, project_id: str) -> Project | None: ...

    @abstractmethod
    def find_by_owner_and_code(self, owner_user_id: str, code: str) -> Project | None: ...

    @abstractmethod
    def save(self, project: Project) -> None: ...

    @abstractmethod
    def delete(self, project: Project) -> None: ...
