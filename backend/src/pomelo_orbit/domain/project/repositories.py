from abc import ABC, abstractmethod

from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.domain.project.entities import Project


class ProjectRepository(ABC):
    @abstractmethod
    def find_by_id(self, project_id: str) -> Project | None: ...

    @abstractmethod
    def find_by_member(self, user_id: str) -> list[Project]: ...

    @abstractmethod
    def find_active_by_member(self, user_id: str) -> list[Project]: ...

    @abstractmethod
    def find_by_code(self, code: str) -> Project | None: ...

    @abstractmethod
    def is_member(self, project_id: str, user_id: str) -> bool: ...

    @abstractmethod
    def list_members(self, project_id: str) -> list[User]: ...

    @abstractmethod
    def add_member(self, project_id: str, user_id: str) -> None: ...

    @abstractmethod
    def remove_member(self, project_id: str, user_id: str) -> None: ...

    @abstractmethod
    def count_repositories(self, project_id: str) -> int: ...

    @abstractmethod
    def count_applications(self, project_id: str) -> int: ...

    @abstractmethod
    def save(self, project: Project) -> None: ...

    @abstractmethod
    def delete(self, project: Project) -> None: ...
