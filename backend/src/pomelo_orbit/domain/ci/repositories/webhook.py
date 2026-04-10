"""CI RepositoryWebhook 仓储接口"""

from abc import ABC, abstractmethod

from pomelo_orbit.domain.ci.entities import RepositoryWebhook


class RepositoryWebhookRepository(ABC):
    @abstractmethod
    def find_by_id(self, webhook_id: str) -> RepositoryWebhook | None: ...

    @abstractmethod
    def find_by_repository(self, repository_id: str) -> list[RepositoryWebhook]: ...

    @abstractmethod
    def find_by_template(self, template_id: str) -> list[RepositoryWebhook]: ...

    @abstractmethod
    def save(self, webhook: RepositoryWebhook) -> None: ...

    @abstractmethod
    def delete(self, webhook: RepositoryWebhook) -> None: ...
