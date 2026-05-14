"""CI 凭据仓储接口"""

from abc import ABC, abstractmethod

from pomelo_orbit.domain.ci.entities import Credential


class CredentialRepository(ABC):
    @abstractmethod
    def find_by_id(self, credential_id: str) -> Credential | None: ...

    @abstractmethod
    def find_by_id_in_project(self, project_id: str, credential_id: str) -> Credential | None: ...

    @abstractmethod
    def find_by_name(self, project_id: str, name: str) -> Credential | None: ...

    @abstractmethod
    def find_paginated_by_project_id(
        self, project_id: str, page: int, per_page: int
    ) -> tuple[list[Credential], int]: ...

    @abstractmethod
    def is_referenced_by_repositories(self, project_id: str, credential_id: str) -> bool: ...

    @abstractmethod
    def save(self, credential: Credential) -> None: ...

    @abstractmethod
    def delete(self, credential: Credential) -> None: ...
