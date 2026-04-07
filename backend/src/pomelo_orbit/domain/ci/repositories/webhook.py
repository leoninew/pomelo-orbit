"""CI ProjectWebhook 仓储接口"""

from abc import ABC, abstractmethod

from pomelo_orbit.domain.ci.entities import ProjectWebhook


class ProjectWebhookRepository(ABC):
    @abstractmethod
    def find_by_id(self, webhook_id: str) -> ProjectWebhook | None: ...

    @abstractmethod
    def find_by_project(self, project_id: str) -> list[ProjectWebhook]: ...

    @abstractmethod
    def find_by_template(self, template_id: str) -> list[ProjectWebhook]: ...

    @abstractmethod
    def save(self, webhook: ProjectWebhook) -> None: ...

    @abstractmethod
    def delete(self, webhook: ProjectWebhook) -> None: ...
