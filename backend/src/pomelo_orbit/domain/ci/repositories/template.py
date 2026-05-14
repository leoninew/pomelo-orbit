"""CI 流水线模板与快照仓储接口"""

from abc import ABC, abstractmethod

from pomelo_orbit.domain.ci.entities import BuildStage, PipelineSnapshot, PipelineTemplate


class BuildStageRepository(ABC):
    @abstractmethod
    def find_by_id_in_project(self, project_id: str, stage_id: str) -> BuildStage | None: ...

    @abstractmethod
    def find_by_name(self, project_id: str, name: str) -> BuildStage | None: ...

    @abstractmethod
    def find_paginated_by_project_id(
        self, project_id: str, page: int, per_page: int, search: str | None = None
    ) -> tuple[list[BuildStage], int]: ...

    @abstractmethod
    def find_by_ids(self, project_id: str, stage_ids: list[str]) -> list[BuildStage]: ...

    @abstractmethod
    def save(self, stage: BuildStage) -> None: ...

    @abstractmethod
    def delete(self, stage: BuildStage) -> None: ...

    @abstractmethod
    def is_referenced_by_templates(self, project_id: str, stage_id: str) -> bool: ...


class PipelineTemplateRepository(ABC):
    @abstractmethod
    def find_by_id_in_project(self, project_id: str, template_id: str) -> PipelineTemplate | None: ...

    @abstractmethod
    def find_by_name(self, project_id: str, name: str) -> PipelineTemplate | None: ...

    @abstractmethod
    def find_paginated_by_project_id(
        self, project_id: str, page: int, per_page: int, search: str | None = None
    ) -> tuple[list[PipelineTemplate], int]: ...

    @abstractmethod
    def save(self, template: PipelineTemplate) -> None: ...

    @abstractmethod
    def save_orchestration(self, template_id: str, orchestration: list) -> None: ...

    @abstractmethod
    def delete(self, template: PipelineTemplate) -> None: ...


class PipelineSnapshotRepository(ABC):
    @abstractmethod
    def find_by_id_in_project(self, project_id: str, snapshot_id: str) -> PipelineSnapshot | None: ...

    @abstractmethod
    def find_latest(self, project_id: str, template_id: str) -> PipelineSnapshot | None: ...

    @abstractmethod
    def save(self, snapshot: PipelineSnapshot) -> None: ...
