from abc import ABC, abstractmethod

from pomelo_orbit.domain.auth.entities import LoginHistory, Permission, Role, User


class UserRepository(ABC):
    @abstractmethod
    def find_by_id(self, user_id: str) -> User | None: ...

    @abstractmethod
    def find_by_username(self, username: str) -> User | None: ...

    @abstractmethod
    def find_by_oauth_account(self, provider: str, provider_id: str) -> User | None: ...

    @abstractmethod
    def find_by_email(self, email: str) -> User | None: ...

    @abstractmethod
    def find_paginated(self, page: int, per_page: int, search: str | None) -> tuple[list[User], int]: ...

    @abstractmethod
    def save(self, user: User) -> None: ...

    @abstractmethod
    def delete(self, user: User) -> None: ...

    @abstractmethod
    def find_roles(self, user_id: str) -> list[Role]: ...

    @abstractmethod
    def find_roles_by_user_ids(self, user_ids: list[str]) -> dict[str, list[Role]]: ...

    @abstractmethod
    def find_permissions(self, user_id: str) -> list[Permission]: ...

    @abstractmethod
    def set_roles(self, user_id: str, role_ids: list[str]) -> None: ...

    @abstractmethod
    def save_login_history(self, history: LoginHistory) -> None: ...

    @abstractmethod
    def find_login_history(self, page: int, per_page: int, search: str | None) -> tuple[list[LoginHistory], int]: ...
