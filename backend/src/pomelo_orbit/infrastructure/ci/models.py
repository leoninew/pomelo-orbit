"""CI ORM 模型"""

from datetime import datetime

import ulid
from sqlalchemy import DateTime, ForeignKey, Integer, String, Text, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column

from pomelo_orbit.infrastructure.persistence.models import Base
from pomelo_orbit.infrastructure.time_utils import utc_now


class CredentialModel(Base):
    """凭据模型"""

    __tablename__ = "credential"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    project_id: Mapped[str] = mapped_column(String(26), ForeignKey("project.id"), nullable=True, index=True)
    name: Mapped[str] = mapped_column(String(255), nullable=False, index=True)
    type: Mapped[str] = mapped_column(String(50), nullable=False)
    encrypted_data: Mapped[str] = mapped_column(Text, nullable=False)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)


class PipelineTemplateModel(Base):
    """流水线模板模型"""

    __tablename__ = "pipeline_template"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    project_id: Mapped[str] = mapped_column(String(26), ForeignKey("project.id"), nullable=True, index=True)
    name: Mapped[str] = mapped_column(String(255), nullable=False, index=True)
    description: Mapped[str] = mapped_column(Text, nullable=False, default="")
    variable_declarations: Mapped[str] = mapped_column(Text, nullable=False, default="[]")  # JSON
    version: Mapped[int] = mapped_column(Integer, nullable=False, default=1)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, onupdate=utc_now, nullable=False)


class BuildStageModel(Base):
    """构建 Stage 模型：执行最小单元，不含编排属性"""

    __tablename__ = "build_stage"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    project_id: Mapped[str] = mapped_column(String(26), ForeignKey("project.id"), nullable=True, index=True)
    name: Mapped[str] = mapped_column(String(255), nullable=False, index=True)
    image: Mapped[str] = mapped_column(String(255), nullable=False)
    script: Mapped[str] = mapped_column(Text, nullable=False, default="")
    artifacts: Mapped[str | None] = mapped_column(Text, nullable=True)  # JSON | NULL
    description: Mapped[str] = mapped_column(Text, nullable=False, default="")
    version: Mapped[int] = mapped_column(Integer, nullable=False, default=1)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, onupdate=utc_now, nullable=False)


class PipelineTemplateStageModel(Base):
    """模板编排表：模板对 Stage 的引用 + 依赖 + 顺序"""

    __tablename__ = "pipeline_template_stage"
    __table_args__ = (UniqueConstraint("template_id", "stage_id", name="uq_template_stage"),)

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    template_id: Mapped[str] = mapped_column(
        String(26), ForeignKey("pipeline_template.id", ondelete="CASCADE"), nullable=False, index=True
    )
    stage_id: Mapped[str] = mapped_column(String(26), ForeignKey("build_stage.id"), nullable=False)
    stage_name: Mapped[str] = mapped_column(String(255), nullable=False)  # 模板内唯一标识
    stage_version: Mapped[int] = mapped_column(Integer, nullable=False, default=1)  # 编排时的 stage 版本
    depends_on: Mapped[str] = mapped_column(Text, nullable=False, default="[]")  # JSON array of stage_name
    sort_order: Mapped[int] = mapped_column(Integer, nullable=False, default=0)


class PipelineSnapshotModel(Base):
    """流水线快照模型（模板的不可变版本副本）"""

    __tablename__ = "pipeline_snapshot"
    __table_args__ = (UniqueConstraint("template_id", "version", name="uq_snapshot_template_version"),)

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    project_id: Mapped[str] = mapped_column(String(26), ForeignKey("project.id"), nullable=True, index=True)
    template_id: Mapped[str] = mapped_column(String(26), ForeignKey("pipeline_template.id"), nullable=False, index=True)
    version: Mapped[int] = mapped_column(Integer, nullable=False)
    stages_snapshot: Mapped[str] = mapped_column(Text, nullable=False, default="[]")  # JSON
    variables_snapshot: Mapped[str] = mapped_column(Text, nullable=False, default="[]")  # JSON
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)


class RepositoryModel(Base):
    """代码仓库模型"""

    __tablename__ = "repository"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    project_id: Mapped[str] = mapped_column(String(26), ForeignKey("project.id"), nullable=True, index=True)
    name: Mapped[str] = mapped_column(String(255), nullable=False, index=True)
    code: Mapped[str] = mapped_column(String(100), nullable=False, unique=True, index=True)
    repository_url: Mapped[str] = mapped_column(Text, nullable=False)
    git_credential_id: Mapped[str | None] = mapped_column(
        String(26), ForeignKey("credential.id"), nullable=True, index=True
    )
    variable_overrides: Mapped[str] = mapped_column(Text, nullable=False, default="[]")  # JSON array
    default_branch: Mapped[str] = mapped_column(String(255), nullable=False, default="master")
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, onupdate=utc_now, nullable=False)


class ProjectWebhookModel(Base):
    """项目 Webhook 配置模型"""

    __tablename__ = "repository_webhook"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    repository_id: Mapped[str] = mapped_column(
        String(26), ForeignKey("repository.id", ondelete="CASCADE"), nullable=False, index=True
    )
    name: Mapped[str] = mapped_column(String(255), nullable=False)
    template_id: Mapped[str] = mapped_column(String(26), ForeignKey("pipeline_template.id"), nullable=False)
    branch_filter: Mapped[str | None] = mapped_column(String(255), nullable=True)
    encrypted_secret: Mapped[str] = mapped_column(Text, nullable=False)
    enabled: Mapped[bool] = mapped_column(Integer, nullable=False, default=1)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, onupdate=utc_now, nullable=False)


class PipelineRunModel(Base):
    """Pipeline 运行模型"""

    __tablename__ = "pipeline_run"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    project_id: Mapped[str] = mapped_column(String(26), ForeignKey("project.id"), nullable=True, index=True)
    repository_id: Mapped[str] = mapped_column(String(26), ForeignKey("repository.id"), nullable=False, index=True)
    repository_name: Mapped[str] = mapped_column(String(255), nullable=False, default="")
    snapshot_id: Mapped[str] = mapped_column(String(26), ForeignKey("pipeline_snapshot.id"), nullable=False, index=True)
    template_id: Mapped[str] = mapped_column(String(26), nullable=False, default="")
    template_name: Mapped[str] = mapped_column(String(255), nullable=False, default="")
    template_version: Mapped[int] = mapped_column(nullable=False)
    trigger: Mapped[str] = mapped_column(String(50), nullable=False)
    trigger_ref: Mapped[str] = mapped_column(String(255), nullable=False)
    variables_snapshot: Mapped[str] = mapped_column(Text, nullable=False, default="[]")  # JSON
    status: Mapped[str] = mapped_column(String(50), nullable=False, index=True)
    retry_of: Mapped[str | None] = mapped_column(String(26), ForeignKey("pipeline_run.id"), nullable=True, index=True)
    started_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)
    finished_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)
    error_message: Mapped[str | None] = mapped_column(Text, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False, index=True)


class StageRunModel(Base):
    """Stage 执行记录模型"""

    __tablename__ = "stage_run"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    pipeline_run_id: Mapped[str] = mapped_column(String(26), ForeignKey("pipeline_run.id"), nullable=False, index=True)
    stage_id: Mapped[str] = mapped_column(String(26), nullable=False)
    stage_name: Mapped[str] = mapped_column(String(255), nullable=False)
    status: Mapped[str] = mapped_column(String(50), nullable=False, index=True)
    started_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)
    finished_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)
    exit_code: Mapped[int | None] = mapped_column(Integer, nullable=True)
    error_message: Mapped[str | None] = mapped_column(Text, nullable=True)


class ArtifactModel(Base):
    """制品模型"""

    __tablename__ = "artifact"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    project_id: Mapped[str] = mapped_column(String(26), ForeignKey("project.id"), nullable=True, index=True)
    pipeline_run_id: Mapped[str] = mapped_column(String(26), ForeignKey("pipeline_run.id"), nullable=False, index=True)
    repository_id: Mapped[str] = mapped_column(String(26), nullable=False, default="", index=True)
    repository_name: Mapped[str] = mapped_column(String(255), nullable=False, default="")
    template_id: Mapped[str] = mapped_column(String(26), nullable=False, default="")
    template_name: Mapped[str] = mapped_column(String(255), nullable=False, default="")
    stage_name: Mapped[str] = mapped_column(String(255), nullable=False)
    type: Mapped[str] = mapped_column(String(50), nullable=False)
    name: Mapped[str] = mapped_column(String(255), nullable=False)
    path: Mapped[str | None] = mapped_column(Text, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)
