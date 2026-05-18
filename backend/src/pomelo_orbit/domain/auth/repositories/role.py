from abc import ABC, abstractmethod

from pomelo_orbit.domain.auth.entities import Role


class RoleRepository(ABC):
    @abstractmethod
    def find_by_id(self, role_id: str) -> Role | None: ...

    @abstractmethod
    def find_by_code(self, code: str) -> Role | None: ...

    @abstractmethod
    def find_by_name(self, name: str) -> Role | None: ...

    @abstractmethod
    def find_paginated(self, page: int, per_page: int, search: str | None) -> tuple[list[Role], int]: ...

    @abstractmethod
    def save(self, role: Role) -> None: ...

    @abstractmethod
    def delete(self, role: Role) -> None: ...
