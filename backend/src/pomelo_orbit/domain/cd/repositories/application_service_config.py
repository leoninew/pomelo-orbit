from abc import ABC, abstractmethod

from pomelo_orbit.domain.cd.entities import ApplicationServiceConfig


class ApplicationServiceConfigRepository(ABC):
    @abstractmethod
    def find_by_id(self, config_id: str) -> ApplicationServiceConfig | None: ...

    @abstractmethod
    def find_by_application(self, application_id: str) -> list[ApplicationServiceConfig]: ...

    @abstractmethod
    def find_by_application_and_service(
        self, application_id: str, service_name: str
    ) -> ApplicationServiceConfig | None: ...

    @abstractmethod
    def save(self, config: ApplicationServiceConfig) -> None: ...

    @abstractmethod
    def delete(self, config: ApplicationServiceConfig) -> None: ...
