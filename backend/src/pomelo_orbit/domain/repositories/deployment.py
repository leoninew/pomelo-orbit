from abc import ABC, abstractmethod
from datetime import datetime

from pomelo_orbit.domain.entities import Deployment


class DeploymentRepository(ABC):
    @abstractmethod
    def find_by_id(self, deployment_id: str) -> Deployment | None: ...

    @abstractmethod
    def find_all(self) -> list[Deployment]: ...

    @abstractmethod
    def find_by_application(self, app_id: str, page: int, per_page: int) -> tuple[list[Deployment], int]: ...

    @abstractmethod
    def find_last_successful_deploy(self, app_id: str) -> Deployment | None: ...

    @abstractmethod
    def find_paginated_with_filters(
        self,
        page: int,
        per_page: int,
        application_id: str | None = None,
        status: str | None = None,
        search: str | None = None,
        date_from: datetime | None = None,
        date_to: datetime | None = None,
    ) -> tuple[list[Deployment], int]: ...

    @abstractmethod
    def save(self, deployment: Deployment) -> None: ...

    @abstractmethod
    def commit(self) -> None: ...
