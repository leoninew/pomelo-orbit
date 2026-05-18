from abc import ABC, abstractmethod

from pomelo_orbit.domain.auth.entities import Permission


class PermissionRepository(ABC):
    @abstractmethod
    def list_all(self) -> list[Permission]: ...

    @abstractmethod
    def find_by_codes(self, codes: list[str]) -> list[Permission]: ...

    @abstractmethod
    def find_by_user_id(self, user_id: str) -> list[Permission]: ...

    @abstractmethod
    def find_by_role_id(self, role_id: str) -> list[Permission]: ...

    @abstractmethod
    def set_role_permissions(self, role_id: str, permission_codes: list[str]) -> None: ...
