"""
SQLAlchemy ORM 模型定义
"""

from datetime import datetime

import ulid
from sqlalchemy import Boolean, DateTime, ForeignKey, Integer, String, Text
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column, relationship

from pomelo_orbit.domain.cd.value_objects import ApplicationStatus, ImagePullPolicy, OperationType
from pomelo_orbit.infrastructure.time_utils import utc_now


class Base(DeclarativeBase):
    """ORM 基类"""


class UserModel(Base):
    """用户模型"""

    __tablename__ = "user"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    username: Mapped[str] = mapped_column(String(50), unique=True, nullable=False, index=True)
    password_hash: Mapped[str] = mapped_column(String(255), nullable=False)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, onupdate=utc_now, nullable=False)
    last_login_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)


class LoginHistoryModel(Base):
    """登录历史模型"""

    __tablename__ = "login_history"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    user_id: Mapped[str] = mapped_column(String(26), ForeignKey("user.id", ondelete="CASCADE"), nullable=False)
    username: Mapped[str] = mapped_column(String(50), nullable=False)
    ip_address: Mapped[str | None] = mapped_column(String(50), nullable=True)
    user_agent: Mapped[str | None] = mapped_column(String(500), nullable=True)
    login_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)
    success: Mapped[bool] = mapped_column(Boolean, default=True, nullable=False)


class ApplicationModel(Base):
    """应用模型（聚合根）"""

    __tablename__ = "application"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    name: Mapped[str] = mapped_column(String(100), unique=True, nullable=False, index=True)
    code: Mapped[str] = mapped_column(String(100), unique=True, nullable=False, index=True)

    # 部署配置
    image_pull_policy: Mapped[str] = mapped_column(String(20), default=ImagePullPolicy.MISSING, nullable=False)
    status: Mapped[str] = mapped_column(String(20), default=ApplicationStatus.UNDEPLOYED, nullable=False)

    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, onupdate=utc_now, nullable=False)

    # 关系
    image_source: Mapped["ImageSourceModel | None"] = relationship(
        "ImageSourceModel", back_populates="application", uselist=False, cascade="all, delete-orphan"
    )
    config_files: Mapped[list["ApplicationConfigFileModel"]] = relationship(
        "ApplicationConfigFileModel", back_populates="application", cascade="all, delete-orphan"
    )


class ImageSourceModel(Base):
    """镜像源"""

    __tablename__ = "image_source"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    application_id: Mapped[str] = mapped_column(String(26), ForeignKey("application.id"), nullable=False, unique=True)
    image_name: Mapped[str] = mapped_column(String(500), nullable=False)
    registry_url: Mapped[str | None] = mapped_column(String(500), nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, onupdate=utc_now, nullable=False)

    # 关系
    application: Mapped["ApplicationModel"] = relationship("ApplicationModel", back_populates="image_source")


class DeploymentModel(Base):
    """部署记录"""

    __tablename__ = "deployment"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    application_id: Mapped[str | None] = mapped_column(String(26), nullable=True, index=True)
    application_name: Mapped[str] = mapped_column(String(100), nullable=False, index=True)
    operation_type: Mapped[str] = mapped_column(String(20), default=OperationType.DEPLOY, nullable=False)
    trigger_type: Mapped[str] = mapped_column(String(20), nullable=False)  # webhook | manual
    trigger_ref: Mapped[str | None] = mapped_column(String(255), nullable=True)
    # 逻辑外键：通过 webhook_event_id 关联到 webhook_event，应用层维护一致性
    webhook_event_id: Mapped[str | None] = mapped_column(String(26), nullable=True, index=True)
    image_name: Mapped[str | None] = mapped_column(String(500), nullable=True)
    env_file: Mapped[str | None] = mapped_column(String(255), nullable=True)
    status: Mapped[str] = mapped_column(String(20), nullable=False, index=True)  # queued | running | success | failed
    started_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False, index=True)
    finished_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)
    duration_ms: Mapped[int | None] = mapped_column(Integer, nullable=True)
    log_text: Mapped[str | None] = mapped_column(Text, nullable=True)
    error_message: Mapped[str | None] = mapped_column(Text, nullable=True)
    is_rollback: Mapped[bool] = mapped_column(Boolean, default=False, nullable=False)
    rollback_from_deployment_id: Mapped[str | None] = mapped_column(
        String(26), ForeignKey("deployment.id"), nullable=True
    )

    # 关系（仅自引用）
    # webhook_event_id 为逻辑外键，通过应用层维护一致性


class ApplicationConfigFileModel(Base):
    """应用配置文件"""

    __tablename__ = "application_config_file"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    application_id: Mapped[str] = mapped_column(
        String(26), ForeignKey("application.id", ondelete="CASCADE"), nullable=False, index=True
    )
    path: Mapped[str] = mapped_column(String(500), nullable=False)
    content: Mapped[str] = mapped_column(Text, nullable=False)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, onupdate=utc_now, nullable=False)

    # 关系
    application: Mapped["ApplicationModel"] = relationship("ApplicationModel", back_populates="config_files")


class RouteModel(Base):
    """路由配置"""

    __tablename__ = "route"

    id: Mapped[str] = mapped_column(String(26), primary_key=True, default=lambda: str(ulid.ULID()))
    name: Mapped[str] = mapped_column(String(100), nullable=False)
    domain: Mapped[str] = mapped_column(String(255), nullable=False, index=True)
    path_prefix: Mapped[str] = mapped_column(String(255), default="/", nullable=False)
    target_url: Mapped[str] = mapped_column(String(500), nullable=False)
    enabled: Mapped[bool] = mapped_column(Boolean, nullable=False, index=True)
    https_enabled: Mapped[bool] = mapped_column(Boolean, default=False, nullable=False)
    cert_pem: Mapped[str | None] = mapped_column(Text, nullable=True)
    cert_key: Mapped[str | None] = mapped_column(Text, nullable=True)
    cert_type: Mapped[str] = mapped_column(String(20), default="manual", nullable=False)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, nullable=False)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=utc_now, onupdate=utc_now, nullable=False)
