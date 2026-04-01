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


class CredentialRepository(BaseRepository[Credential, CredentialModel]):
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


class PipelineTemplateRepository(BaseRepository[PipelineTemplate, PipelineTemplateModel]):
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


class ProjectRepository(BaseRepository[Project, ProjectModel]):
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
        """
        根据仓库 URL 查找项目

        Args:
            repository_url: 仓库 URL（精确匹配）

        Returns:
            匹配的项目列表，可能为空。多个项目可能使用同一个仓库 URL。
        """
        orms = self._session.query(ProjectModel).filter(ProjectModel.repository_url == repository_url).all()
        return [self._mapper.to_domain(orm) for orm in orms]


class PipelineRunRepository(BaseRepository[PipelineRun, PipelineRunModel]):
    """Pipeline 运行仓储"""

    def __init__(self, session: Session):
        super().__init__(session, PipelineRunModel, PipelineRunMapper())

    def find_by_project(
        self, project_id: str, page: int = 1, per_page: int = 20
    ) -> tuple[list[PipelineRun], int]:
        """按项目查询运行列表"""
        query = (
            self._session.query(PipelineRunModel)
            .filter(PipelineRunModel.project_id == project_id)
            .order_by(PipelineRunModel.created_at.desc())
        )

        total = query.count()
        orms = query.offset((page - 1) * per_page).limit(per_page).all()
        return [self._mapper.to_domain(orm) for orm in orms], total


class JobRepository(BaseRepository[Job, JobModel]):
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


class JobLogRepository(BaseRepository[JobLog, JobLogModel]):
    """Job 日志仓储"""

    def __init__(self, session: Session):
        super().__init__(session, JobLogModel, JobLogMapper())

    def find_by_job(self, job_id: str) -> JobLog | None:
        """查询 job 的日志"""
        orm = self._session.query(JobLogModel).filter(JobLogModel.job_id == job_id).first()
        return self._mapper.to_domain(orm) if orm else None


class ArtifactRepository(BaseRepository[Artifact, ArtifactModel]):
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
