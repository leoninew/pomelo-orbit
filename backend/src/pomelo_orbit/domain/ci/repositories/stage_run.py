"""CI StageRun 仓储接口"""

from abc import ABC, abstractmethod

from pomelo_orbit.domain.ci.entities import StageRun


class StageRunRepository(ABC):
    @abstractmethod
    def find_by_id(self, stage_run_id: str) -> StageRun | None: ...

    @abstractmethod
    def find_by_run(self, run_id: str) -> list[StageRun]: ...

    @abstractmethod
    def save(self, stage_run: StageRun) -> None: ...

    @abstractmethod
    def commit(self) -> None: ...
