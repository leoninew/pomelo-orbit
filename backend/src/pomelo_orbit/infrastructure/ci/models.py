"""CI ORM 模型"""

from datetime import datetime

import ulid
from sqlalchemy import DateTime, ForeignKey, Integer, String, Text, UniqueConstraint
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
    """流水线模板模型"""

    __tablename__ = "pipeline_templates"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    name: Mapped[str] = mapped_column(String(255), nullable=False, index=True)
    description: Mapped[str] = mapped_column(Text, nullable=False, default="")
    variable_declarations: Mapped[str] = mapped_column(Text, nullable=False, default="[]")  # JSON
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, onupdate=utc_now, nullable=False)


class PipelineStageModel(Base):
    """流水线 Stage 模型：执行最小单元，不含编排属性"""

    __tablename__ = "pipeline_stages"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    name: Mapped[str] = mapped_column(String(255), nullable=False, index=True)
    image: Mapped[str] = mapped_column(String(255), nullable=False)
    script: Mapped[str] = mapped_column(Text, nullable=False, default="")
    env: Mapped[str] = mapped_column(Text, nullable=False, default="{}")  # JSON
    artifacts: Mapped[str | None] = mapped_column(Text, nullable=True)  # JSON | NULL
    description: Mapped[str] = mapped_column(Text, nullable=False, default="")
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, onupdate=utc_now, nullable=False)


class PipelineTemplateStageModel(Base):
    """模板编排表：模板对 Stage 的引用 + 依赖 + 顺序"""

    __tablename__ = "pipeline_template_stages"
    __table_args__ = (UniqueConstraint("template_id", "stage_id", name="uq_template_stage"),)

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    template_id: Mapped[str] = mapped_column(
        String(26), ForeignKey("pipeline_templates.id", ondelete="CASCADE"), nullable=False, index=True
    )
    stage_id: Mapped[str] = mapped_column(String(26), ForeignKey("pipeline_stages.id"), nullable=False)
    depends_on: Mapped[str] = mapped_column(Text, nullable=False, default="[]")  # JSON array of stage_id
    sort_order: Mapped[int] = mapped_column(Integer, nullable=False, default=0)


class PipelineSnapshotModel(Base):
    """流水线快照模型（模板的不可变版本副本）"""

    __tablename__ = "pipeline_snapshots"
    __table_args__ = (UniqueConstraint("template_id", "version", name="uq_snapshot_template_version"),)

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    template_id: Mapped[str] = mapped_column(
        String(26), ForeignKey("pipeline_templates.id"), nullable=False, index=True
    )
    version: Mapped[int] = mapped_column(Integer, nullable=False)
    stages_snapshot: Mapped[str] = mapped_column(Text, nullable=False, default="[]")  # JSON
    variable_declarations_snapshot: Mapped[str] = mapped_column(Text, nullable=False, default="[]")  # JSON
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)


class ProjectModel(Base):
    """项目模型"""

    __tablename__ = "projects"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    name: Mapped[str] = mapped_column(String(255), nullable=False, index=True)
    code: Mapped[str] = mapped_column(String(100), nullable=False, unique=True, index=True)
    repository_url: Mapped[str] = mapped_column(Text, nullable=False)
    git_credential_id: Mapped[str | None] = mapped_column(
        String(26), ForeignKey("credentials.id"), nullable=True, index=True
    )
    variable_overrides: Mapped[str] = mapped_column(Text, nullable=False, default="{}")  # JSON
    default_branch: Mapped[str] = mapped_column(String(255), nullable=False, default="master")
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, onupdate=utc_now, nullable=False)


class ProjectWebhookModel(Base):
    """项目 Webhook 配置模型"""

    __tablename__ = "project_webhooks"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    project_id: Mapped[str] = mapped_column(
        String(26), ForeignKey("projects.id", ondelete="CASCADE"), nullable=False, index=True
    )
    name: Mapped[str] = mapped_column(String(255), nullable=False)
    template_id: Mapped[str] = mapped_column(String(26), ForeignKey("pipeline_templates.id"), nullable=False)
    branch_filter: Mapped[str | None] = mapped_column(String(255), nullable=True)
    encrypted_secret: Mapped[str] = mapped_column(Text, nullable=False)
    enabled: Mapped[bool] = mapped_column(Integer, nullable=False, default=1)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, onupdate=utc_now, nullable=False)


class PipelineRunModel(Base):
    """Pipeline 运行模型"""

    __tablename__ = "pipeline_runs"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    project_id: Mapped[str] = mapped_column(String(26), ForeignKey("projects.id"), nullable=False, index=True)
    pipeline_snapshot_id: Mapped[str] = mapped_column(
        String(26), ForeignKey("pipeline_snapshots.id"), nullable=False, index=True
    )
    trigger: Mapped[str] = mapped_column(String(50), nullable=False)
    trigger_ref: Mapped[str] = mapped_column(String(255), nullable=False)
    variables_snapshot: Mapped[str] = mapped_column(Text, nullable=False, default="{}")  # JSON
    status: Mapped[str] = mapped_column(String(50), nullable=False, index=True)
    retry_of: Mapped[str | None] = mapped_column(String(26), ForeignKey("pipeline_runs.id"), nullable=True, index=True)
    started_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)
    finished_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False, index=True)


class StageRunModel(Base):
    """Stage 执行记录模型"""

    __tablename__ = "stage_runs"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    pipeline_run_id: Mapped[str] = mapped_column(String(26), ForeignKey("pipeline_runs.id"), nullable=False, index=True)
    name: Mapped[str] = mapped_column(String(255), nullable=False)
    status: Mapped[str] = mapped_column(String(50), nullable=False, index=True)
    started_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)
    finished_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)
    exit_code: Mapped[int | None] = mapped_column(Integer, nullable=True)
    error_message: Mapped[str | None] = mapped_column(Text, nullable=True)


class StageLogModel(Base):
    """Stage 执行日志模型"""

    __tablename__ = "stage_logs"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    stage_run_id: Mapped[str] = mapped_column(String(26), ForeignKey("stage_runs.id"), nullable=False, index=True)
    content: Mapped[str] = mapped_column(Text, nullable=False)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)


class ArtifactModel(Base):
    """制品模型"""

    __tablename__ = "artifacts"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    pipeline_run_id: Mapped[str] = mapped_column(String(26), ForeignKey("pipeline_runs.id"), nullable=False, index=True)
    stage_name: Mapped[str] = mapped_column(String(255), nullable=False)
    type: Mapped[str] = mapped_column(String(50), nullable=False)
    name: Mapped[str] = mapped_column(String(255), nullable=False)
    path: Mapped[str | None] = mapped_column(Text, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)
