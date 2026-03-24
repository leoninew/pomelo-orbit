from abc import ABC, abstractmethod

from pomelo_orbit.domain.entities import Application


class ApplicationRepository(ABC):
    @abstractmethod
    def find_by_id(self, application_id: str) -> Application | None: ...

    @abstractmethod
    def find_all(self) -> list[Application]: ...

    @abstractmethod
    def find_paginated(self, page: int, per_page: int, search: str | None) -> tuple[list[Application], int]: ...

    @abstractmethod
    def find_by_name(self, name: str) -> Application | None: ...

    @abstractmethod
    def save(self, application: Application) -> None: ...

    @abstractmethod
    def delete(self, application: Application) -> None: ...

    @abstractmethod
    def commit(self) -> None: ...
