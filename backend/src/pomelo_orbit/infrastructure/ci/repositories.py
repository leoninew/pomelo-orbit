"""CI Repository 实现"""

from sqlalchemy.orm import Session

from pomelo_orbit.domain.ci.entities import (
    Artifact,
    Credential,
    Job,
    JobLog,
    PipelineRun,
    PipelineTemplate,
    Project,
)
from pomelo_orbit.domain.ci.repositories import (
    ArtifactRepository,
    CredentialRepository,
    JobLogRepository,
    JobRepository,
    PipelineRunRepository,
    PipelineTemplateRepository,
    ProjectRepository,
)
from pomelo_orbit.infrastructure.ci.mappers import (
    ArtifactMapper,
    CredentialMapper,
    JobLogMapper,
    JobMapper,
    PipelineRunMapper,
    PipelineTemplateMapper,
    ProjectMapper,
)
from pomelo_orbit.infrastructure.ci.models import (
    ArtifactModel,
    CredentialModel,
    JobLogModel,
    JobModel,
    PipelineRunModel,
    PipelineTemplateModel,
    ProjectModel,
)
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository


class CredentialRepositoryImpl(BaseRepository[Credential, CredentialModel], CredentialRepository):
    """凭据仓储"""

    def __init__(self, session: Session):
        super().__init__(session, CredentialModel, CredentialMapper())

    def is_referenced_by_projects(self, credential_id: str) -> bool:
        """检查凭据是否被项目引用"""
        return (
            self._session.query(ProjectModel)
            .filter(ProjectModel.git_credential_id == credential_id)
            .first()
            is not None
        )


class PipelineTemplateRepositoryImpl(BaseRepository[PipelineTemplate, PipelineTemplateModel], PipelineTemplateRepository):
    """Pipeline 模板仓储"""

    def __init__(self, session: Session):
        super().__init__(session, PipelineTemplateModel, PipelineTemplateMapper())

    def is_referenced_by_projects(self, template_id: str) -> bool:
        """检查模板是否被项目引用"""
        return (
            self._session.query(ProjectModel)
            .filter(ProjectModel.pipeline_template_id == template_id)
            .first()
            is not None
        )

    def find_builtin(self) -> list[PipelineTemplate]:
        """查询内置模板"""
        orms = self._session.query(PipelineTemplateModel).filter(PipelineTemplateModel.is_builtin == 1).all()
        return [self._mapper.to_domain(orm) for orm in orms]


class ProjectRepositoryImpl(BaseRepository[Project, ProjectModel], ProjectRepository):
    """项目仓储"""

    def __init__(self, session: Session):
        super().__init__(session, ProjectModel, ProjectMapper())

    def has_running_pipelines(self, project_id: str) -> bool:
        """检查项目是否有运行中的 pipeline"""
        return (
            self._session.query(PipelineRunModel)
            .filter(
                PipelineRunModel.project_id == project_id,
                PipelineRunModel.status.in_(["waiting", "running"]),
            )
            .first()
            is not None
        )

    def find_by_repository_url(self, repository_url: str) -> list[Project]:
        """根据仓库 URL 查找项目（精确匹配，多个项目可能共用同一仓库）"""
        orms = self._session.query(ProjectModel).filter(ProjectModel.repository_url == repository_url).all()
        return [self._mapper.to_domain(orm) for orm in orms]


class PipelineRunRepositoryImpl(BaseRepository[PipelineRun, PipelineRunModel], PipelineRunRepository):
    """Pipeline 运行仓储"""

    def __init__(self, session: Session):
        super().__init__(session, PipelineRunModel, PipelineRunMapper())

    def find_paginated_with_filters(
        self,
        page: int = 1,
        per_page: int = 20,
        project_id: str | None = None,
    ) -> tuple[list[PipelineRun], int]:
        """分页查询运行列表（支持按项目过滤）"""
        query = self._session.query(PipelineRunModel).order_by(PipelineRunModel.created_at.desc())
        if project_id:
            query = query.filter(PipelineRunModel.project_id == project_id)
        total = query.count()
        orms = query.offset((page - 1) * per_page).limit(per_page).all()
        return [self._mapper.to_domain(orm) for orm in orms], total


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


class ArtifactRepositoryImpl(BaseRepository[Artifact, ArtifactModel], ArtifactRepository):
    """制品仓储"""

    def __init__(self, session: Session):
        super().__init__(session, ArtifactModel, ArtifactMapper())

    def find_by_run(self, pipeline_run_id: str) -> list[Artifact]:
        """查询 pipeline run 的所有制品"""
        orms = (
            self._session.query(ArtifactModel)
            .filter(ArtifactModel.pipeline_run_id == pipeline_run_id)
            .order_by(ArtifactModel.created_at)
            .all()
        )
        return [self._mapper.to_domain(orm) for orm in orms]
