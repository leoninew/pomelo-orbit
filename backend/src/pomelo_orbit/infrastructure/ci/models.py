"""CI ORM 模型"""

from datetime import datetime

import ulid
from sqlalchemy import DateTime, ForeignKey, Integer, String, Text
from sqlalchemy.orm import Mapped, mapped_column

from pomelo_orbit.infrastructure.persistence.models import Base
from pomelo_orbit.infrastructure.time_utils import utc_now


class CredentialModel(Base):
    """凭据模型"""

    __tablename__ = "credentials"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    name: Mapped[str] = mapped_column(String(255), nullable=False, index=True)
    type: Mapped[str] = mapped_column(String(50), nullable=False)
    encrypted_data: Mapped[str] = mapped_column(Text, nullable=False)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)


class PipelineTemplateModel(Base):
    """Pipeline 模板模型"""

    __tablename__ = "pipeline_templates"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    name: Mapped[str] = mapped_column(String(255), nullable=False, index=True)
    description: Mapped[str] = mapped_column(Text, nullable=False, default="")
    content: Mapped[str] = mapped_column(Text, nullable=False)
    variable_declarations: Mapped[str] = mapped_column(Text, nullable=False, default="[]")  # JSON
    is_builtin: Mapped[bool] = mapped_column(Integer, nullable=False, default=0)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, onupdate=utc_now, nullable=False)


class ProjectModel(Base):
    """项目模型"""

    __tablename__ = "projects"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    name: Mapped[str] = mapped_column(String(255), nullable=False, index=True)
    repository_url: Mapped[str] = mapped_column(Text, nullable=False)
    pipeline_template_id: Mapped[str] = mapped_column(
        String(26), ForeignKey("pipeline_templates.id"), nullable=False, index=True
    )
    git_credential_id: Mapped[str] = mapped_column(
        String(26), ForeignKey("credentials.id"), nullable=False, index=True
    )
    variable_overrides: Mapped[str] = mapped_column(Text, nullable=False, default="{}")  # JSON
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, onupdate=utc_now, nullable=False)


class PipelineRunModel(Base):
    """Pipeline 运行模型"""

    __tablename__ = "pipeline_runs"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    project_id: Mapped[str] = mapped_column(String(26), ForeignKey("projects.id"), nullable=False, index=True)
    trigger: Mapped[str] = mapped_column(String(50), nullable=False)
    trigger_ref: Mapped[str] = mapped_column(String(255), nullable=False)
    resolved_pipeline: Mapped[str] = mapped_column(Text, nullable=False)
    variables_snapshot: Mapped[str] = mapped_column(Text, nullable=False, default="{}")  # JSON
    status: Mapped[str] = mapped_column(String(50), nullable=False, index=True)
    started_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)
    finished_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False, index=True)


class JobModel(Base):
    """Job 模型"""

    __tablename__ = "jobs"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    pipeline_run_id: Mapped[str] = mapped_column(
        String(26), ForeignKey("pipeline_runs.id"), nullable=False, index=True
    )
    name: Mapped[str] = mapped_column(String(255), nullable=False)
    parent_job_id: Mapped[str | None] = mapped_column(String(26), ForeignKey("jobs.id"), nullable=True, index=True)
    status: Mapped[str] = mapped_column(String(50), nullable=False, index=True)
    started_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)
    finished_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)
    exit_code: Mapped[int | None] = mapped_column(Integer, nullable=True)
    error_message: Mapped[str | None] = mapped_column(Text, nullable=True)


class JobLogModel(Base):
    """Job 日志模型"""

    __tablename__ = "job_logs"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    job_id: Mapped[str] = mapped_column(String(26), ForeignKey("jobs.id"), nullable=False, index=True)
    content: Mapped[str] = mapped_column(Text, nullable=False)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)
