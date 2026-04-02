"""CI 制品仓储接口"""

from abc import ABC, abstractmethod

from pomelo_orbit.domain.ci.entities import Artifact


class ArtifactRepository(ABC):
    @abstractmethod
    def find_by_run(self, pipeline_run_id: str) -> list[Artifact]: ...

    @abstractmethod
    def save(self, artifact: Artifact) -> None: ...
