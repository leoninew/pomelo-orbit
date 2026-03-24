from abc import ABC, abstractmethod

from pomelo_orbit.domain.entities import ApplicationConfigFile


class ConfigFileRepository(ABC):
    @abstractmethod
    def find_by_id(self, file_id: str) -> ApplicationConfigFile | None: ...

    @abstractmethod
    def find_by_application(self, app_id: str) -> list[ApplicationConfigFile]: ...

    @abstractmethod
    def save(self, config_file: ApplicationConfigFile) -> None: ...

    @abstractmethod
    def delete(self, config_file: ApplicationConfigFile) -> None: ...
