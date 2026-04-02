"""CI 流水线模板仓储接口"""

from abc import ABC, abstractmethod

from pomelo_orbit.domain.ci.entities import PipelineTemplate


class PipelineTemplateRepository(ABC):
    @abstractmethod
    def find_by_id(self, template_id: str) -> PipelineTemplate | None: ...

    @abstractmethod
    def find_all(self) -> list[PipelineTemplate]: ...

    @abstractmethod
    def find_paginated(self, page: int, per_page: int) -> tuple[list[PipelineTemplate], int]: ...

    @abstractmethod
    def find_builtin(self) -> list[PipelineTemplate]: ...

    @abstractmethod
    def is_referenced_by_projects(self, template_id: str) -> bool: ...

    @abstractmethod
    def save(self, template: PipelineTemplate) -> None: ...

    @abstractmethod
    def delete(self, template: PipelineTemplate) -> None: ...
