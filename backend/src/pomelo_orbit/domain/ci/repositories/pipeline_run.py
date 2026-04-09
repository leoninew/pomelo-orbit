"""CI Pipeline 运行仓储接口"""

from abc import ABC, abstractmethod

from pomelo_orbit.domain.ci.entities import PipelineRun


class PipelineRunRepository(ABC):
    @abstractmethod
    def find_by_id(self, run_id: str) -> PipelineRun | None: ...

    @abstractmethod
    def find_paginated_with_filters(
        self,
        page: int,
        per_page: int,
        repository_id: str | None = None,
    ) -> tuple[list[PipelineRun], int]: ...

    @abstractmethod
    def save(self, run: PipelineRun) -> None: ...

    @abstractmethod
    def commit(self) -> None: ...
