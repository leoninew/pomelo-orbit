from abc import ABC, abstractmethod

from pomelo_orbit.domain.cd.entities import Route


class RouteRepository(ABC):
    @abstractmethod
    def find_by_id(self, route_id: str) -> Route | None: ...

    @abstractmethod
    def find_by_domain(self, domain: str) -> Route | None: ...

    @abstractmethod
    def find_all(self) -> list[Route]: ...

    @abstractmethod
    def find_paginated(
        self, project_id: str, page: int, per_page: int, search: str | None = None
    ) -> tuple[list[Route], int]: ...

    @abstractmethod
    def find_by_application(self, app_id: str) -> list[Route]: ...

    @abstractmethod
    def save(self, route: Route) -> None: ...

    @abstractmethod
    def delete(self, route: Route) -> None: ...

    @abstractmethod
    def commit(self) -> None: ...
