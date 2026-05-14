"""CI 制品仓储接口"""

from abc import ABC, abstractmethod

from pomelo_orbit.domain.ci.entities import Artifact


class ArtifactRepository(ABC):
    @abstractmethod
    def find_by_run(self, project_id: str, pipeline_run_id: str) -> list[Artifact]: ...

    @abstractmethod
    def find_paginated_by_project_id(
        self,
        project_id: str,
        page: int = 1,
        per_page: int = 20,
        repository_id: str | None = None,
        template_id: str | None = None,
        search: str | None = None,
    ) -> tuple[list[Artifact], int]: ...

    @abstractmethod
    def save(self, artifact: Artifact) -> None: ...
