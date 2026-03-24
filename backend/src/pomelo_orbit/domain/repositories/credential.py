from abc import ABC, abstractmethod

from pomelo_orbit.domain.entities import Credential


class CredentialRepository(ABC):
    @abstractmethod
    def find_by_id(self, credential_id: str) -> Credential | None: ...

    @abstractmethod
    def find_by_application(self, app_id: str) -> Credential | None: ...

    @abstractmethod
    def find_paginated(self, page: int, per_page: int, app_id: str | None) -> tuple[list[Credential], int]: ...

    @abstractmethod
    def find_by_name_in_app(self, app_id: str, name: str) -> Credential | None: ...

    @abstractmethod
    def save(self, credential: Credential) -> None: ...

    @abstractmethod
    def delete(self, credential: Credential) -> None: ...
