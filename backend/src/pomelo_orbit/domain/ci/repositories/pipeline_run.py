"""CI Pipeline 运行仓储接口"""

from abc import ABC, abstractmethod
from datetime import datetime

from pomelo_orbit.domain.ci.entities import PipelineRun


class PipelineRunRepository(ABC):
    @abstractmethod
    def find_by_id(self, run_id: str) -> PipelineRun | None: ...

    @abstractmethod
    def find_paginated_with_filters(
        self,
        project_id: str,
        page: int,
        per_page: int,
        repository_id: str | None = None,
        template_id: str | None = None,
        date_from: datetime | None = None,
        date_to: datetime | None = None,
    ) -> tuple[list[PipelineRun], int]: ...

    @abstractmethod
    def save(self, run: PipelineRun) -> None: ...

    @abstractmethod
    def commit(self) -> None: ...
