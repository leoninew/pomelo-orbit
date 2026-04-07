"""CI 流水线模板与快照仓储接口"""

from abc import ABC, abstractmethod

from pomelo_orbit.domain.ci.entities import PipelineSnapshot, PipelineStage, PipelineTemplate


class PipelineStageRepository(ABC):
    @abstractmethod
    def find_by_id(self, stage_id: str) -> PipelineStage | None: ...

    @abstractmethod
    def find_by_name(self, name: str) -> PipelineStage | None: ...

    @abstractmethod
    def find_all(self) -> list[PipelineStage]: ...

    @abstractmethod
    def find_by_ids(self, stage_ids: list[str]) -> list[PipelineStage]: ...

    @abstractmethod
    def save(self, stage: PipelineStage) -> None: ...

    @abstractmethod
    def delete(self, stage: PipelineStage) -> None: ...

    @abstractmethod
    def is_referenced_by_templates(self, stage_id: str) -> bool:
        """检查 Stage 是否被任何模板编排引用"""
        ...


class PipelineTemplateRepository(ABC):
    @abstractmethod
    def find_by_id(self, template_id: str) -> PipelineTemplate | None: ...

    @abstractmethod
    def find_all(self) -> list[PipelineTemplate]: ...

    @abstractmethod
    def find_paginated(self, page: int, per_page: int) -> tuple[list[PipelineTemplate], int]: ...

    @abstractmethod
    def save(self, template: PipelineTemplate) -> None: ...

    @abstractmethod
    def save_orchestration(self, template_id: str, orchestration: list) -> None:
        """整体替换模板编排"""
        ...

    @abstractmethod
    def delete(self, template: PipelineTemplate) -> None: ...


class PipelineSnapshotRepository(ABC):
    @abstractmethod
    def find_by_id(self, snapshot_id: str) -> PipelineSnapshot | None: ...

    @abstractmethod
    def find_latest(self, template_id: str) -> PipelineSnapshot | None:
        """获取模板最新快照（version 最大），无快照时返回 None"""
        ...

    @abstractmethod
    def find_by_template(self, template_id: str) -> list[PipelineSnapshot]:
        """按模板 ID 查询所有快照，结果按 version 降序排列"""
        ...

    @abstractmethod
    def get_next_version(self, template_id: str) -> int:
        """返回该模板下一个可用版本号（max(version)+1，无快照时返回 1）"""
        ...

    @abstractmethod
    def save(self, snapshot: PipelineSnapshot) -> None: ...

    @abstractmethod
    def find_latest_versions(self, template_ids: list[str]) -> dict[str, int]:
        """批量查询多个模板的最新快照版本号，返回 {template_id: version}"""
        ...
