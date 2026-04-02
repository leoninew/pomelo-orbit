"""CI Job 仓储实现"""

from sqlalchemy.orm import Session

from pomelo_orbit.domain.ci.entities import Job, JobLog
from pomelo_orbit.domain.ci.repositories import JobLogRepository, JobRepository
from pomelo_orbit.infrastructure.ci.mappers import JobLogMapper, JobMapper
from pomelo_orbit.infrastructure.ci.models import JobLogModel, JobModel
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository


class JobRepositoryImpl(BaseRepository[Job, JobModel], JobRepository):
    """Job 仓储"""

    def __init__(self, session: Session):
        super().__init__(session, JobModel, JobMapper())

    def find_by_run(self, run_id: str) -> list[Job]:
        """查询 pipeline run 的所有 jobs"""
        orms = self._session.query(JobModel).filter(JobModel.pipeline_run_id == run_id).all()
        return [self._mapper.to_domain(orm) for orm in orms]

    def find_by_parent(self, parent_job_id: str) -> list[Job]:
        """查询子 jobs"""
        orms = self._session.query(JobModel).filter(JobModel.parent_job_id == parent_job_id).all()
        return [self._mapper.to_domain(orm) for orm in orms]


class JobLogRepositoryImpl(BaseRepository[JobLog, JobLogModel], JobLogRepository):
    """Job 日志仓储"""

    def __init__(self, session: Session):
        super().__init__(session, JobLogModel, JobLogMapper())

    def find_by_job(self, job_id: str) -> JobLog | None:
        """查询 job 的日志"""
        orm = self._session.query(JobLogModel).filter(JobLogModel.job_id == job_id).first()
        return self._mapper.to_domain(orm) if orm else None
