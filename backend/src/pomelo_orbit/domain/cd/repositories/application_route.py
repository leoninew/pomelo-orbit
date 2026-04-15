from abc import ABC, abstractmethod

from pomelo_orbit.domain.cd.entities import ApplicationRoute


class ApplicationRouteRepository(ABC):
    @abstractmethod
    def find_by_id(self, route_id: str) -> ApplicationRoute | None: ...

    @abstractmethod
    def find_by_application(self, application_id: str) -> list[ApplicationRoute]: ...

    @abstractmethod
    def save(self, route: ApplicationRoute) -> None: ...

    @abstractmethod
    def delete(self, route: ApplicationRoute) -> None: ...
