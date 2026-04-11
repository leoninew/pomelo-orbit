"""CI 凭据仓储接口"""

from abc import ABC, abstractmethod

from pomelo_orbit.domain.ci.entities import Credential


class CredentialRepository(ABC):
    @abstractmethod
    def find_by_id(self, credential_id: str) -> Credential | None: ...

    @abstractmethod
    def find_by_name(self, name: str) -> Credential | None: ...

    @abstractmethod
    def find_all(self) -> list[Credential]: ...

    @abstractmethod
    def find_paginated(self, page: int, per_page: int) -> tuple[list[Credential], int]: ...

    @abstractmethod
    def is_referenced_by_projects(self, credential_id: str) -> bool: ...

    @abstractmethod
    def save(self, credential: Credential) -> None: ...

    @abstractmethod
    def delete(self, credential: Credential) -> None: ...
