"""CI 流水线模板与快照仓储接口"""

from abc import ABC, abstractmethod

from pomelo_orbit.domain.ci.entities import PipelineSnapshot, PipelineTemplate


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
    def is_referenced_by_projects(self, template_id: str) -> bool:
        """检查模板是否有任何快照被项目引用"""
        ...

    @abstractmethod
    def save(self, template: PipelineTemplate) -> None: ...

    @abstractmethod
    def delete(self, template: PipelineTemplate) -> None: ...


class PipelineSnapshotRepository(ABC):
    @abstractmethod
    def find_by_id(self, snapshot_id: str) -> PipelineSnapshot | None: ...

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
