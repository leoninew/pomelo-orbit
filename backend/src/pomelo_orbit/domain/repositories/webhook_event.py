from abc import ABC, abstractmethod

from pomelo_orbit.domain.entities import WebhookEvent


class WebhookEventRepository(ABC):
    @abstractmethod
    def find_by_id(self, event_id: str) -> WebhookEvent | None: ...

    @abstractmethod
    def find_by_application(self, app_id: str, page: int, per_page: int) -> tuple[list[WebhookEvent], int]: ...

    @abstractmethod
    def save(self, event: WebhookEvent) -> None: ...
