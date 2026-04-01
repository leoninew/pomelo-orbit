"""CI 领域仓储抽象接口"""

from abc import ABC, abstractmethod

from pomelo_orbit.domain.ci.entities import (
    Artifact,
    Credential,
    Job,
    JobLog,
    PipelineRun,
    PipelineTemplate,
    Project,
)


class CredentialRepository(ABC):
    @abstractmethod
    def find_by_id(self, credential_id: str) -> Credential | None: ...

    @abstractmethod
    def find_all(self) -> list[Credential]: ...

    @abstractmethod
    def find_paginated(self, page: int, per_page: int) -> tuple[list[Credential], int]: ...

    @abstractmethod
    def is_referenced_by_projects(self, credential_id: str) -> bool: ...

    @abstractmethod
    def save(self, credential: Credential) -> None: ...

    @abstractmethod
    def delete(self, credential: Credential) -> None: ...


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


class ProjectRepository(ABC):
    @abstractmethod
    def find_by_id(self, project_id: str) -> Project | None: ...

    @abstractmethod
    def find_paginated(self, page: int, per_page: int) -> tuple[list[Project], int]: ...

    @abstractmethod
    def find_by_repository_url(self, repository_url: str) -> list[Project]: ...

    @abstractmethod
    def has_running_pipelines(self, project_id: str) -> bool: ...

    @abstractmethod
    def save(self, project: Project) -> None: ...

    @abstractmethod
    def delete(self, project: Project) -> None: ...


class PipelineRunRepository(ABC):
    @abstractmethod
    def find_by_id(self, run_id: str) -> PipelineRun | None: ...

    @abstractmethod
    def find_paginated_with_filters(
        self,
        page: int,
        per_page: int,
        project_id: str | None = None,
    ) -> tuple[list[PipelineRun], int]: ...

    @abstractmethod
    def save(self, run: PipelineRun) -> None: ...


class JobRepository(ABC):
    @abstractmethod
    def find_by_id(self, job_id: str) -> Job | None: ...

    @abstractmethod
    def find_by_run(self, run_id: str) -> list[Job]: ...

    @abstractmethod
    def find_by_parent(self, parent_job_id: str) -> list[Job]: ...

    @abstractmethod
    def save(self, job: Job) -> None: ...


class JobLogRepository(ABC):
    @abstractmethod
    def find_by_job(self, job_id: str) -> JobLog | None: ...

    @abstractmethod
    def save(self, job_log: JobLog) -> None: ...


class ArtifactRepository(ABC):
    @abstractmethod
    def find_by_run(self, pipeline_run_id: str) -> list[Artifact]: ...

    @abstractmethod
    def save(self, artifact: Artifact) -> None: ...
