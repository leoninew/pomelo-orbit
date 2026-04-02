"""CI Job 仓储接口"""

from abc import ABC, abstractmethod

from pomelo_orbit.domain.ci.entities import Job, JobLog


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
