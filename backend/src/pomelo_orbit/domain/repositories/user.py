from abc import ABC, abstractmethod

from pomelo_orbit.domain.entities import LoginHistory, User


class UserRepository(ABC):
    @abstractmethod
    def find_by_id(self, user_id: str) -> User | None: ...

    @abstractmethod
    def find_by_username(self, username: str) -> User | None: ...

    @abstractmethod
    def save(self, user: User) -> None: ...

    @abstractmethod
    def save_login_history(self, history: LoginHistory) -> None: ...

    @abstractmethod
    def find_login_history(self, page: int, per_page: int, search: str | None) -> tuple[list[LoginHistory], int]: ...
