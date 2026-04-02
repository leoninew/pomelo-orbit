"""
领域实体与 ORM 模型之间的映射器
"""

from pomelo_orbit.domain.cd.entities import (
    Application,
    ApplicationConfigFile,
    CertType,
    Deployment,
    GitSource,
    ImageSource,
    Route,
    TriggerType,
)
from pomelo_orbit.domain.cd.value_objects import OperationType
from pomelo_orbit.domain.shared.entities import LoginHistory, User
from pomelo_orbit.infrastructure.persistence.models import (
    ApplicationConfigFileModel,
    ApplicationModel,
    DeploymentModel,
    GitSourceModel,
    ImageSourceModel,
    LoginHistoryModel,
    RouteModel,
    UserModel,
)


class UserMapper:
    """用户实体映射器"""

    @staticmethod
    def to_domain(model: UserModel) -> User:
        """ORM 模型转领域实体"""
        return User(
            id=model.id,
            username=model.username,
            password_hash=model.password_hash,
            created_at=model.created_at,
            updated_at=model.updated_at,
            last_login_at=model.last_login_at,
        )

    @staticmethod
    def to_orm(entity: User) -> UserModel:
        """领域实体转 ORM 模型"""
        return UserModel(
            id=entity.id,
            username=entity.username,
            password_hash=entity.password_hash,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
            last_login_at=entity.last_login_at,
        )


class LoginHistoryMapper:
    """登录历史映射器"""

    @staticmethod
    def to_domain(model: LoginHistoryModel) -> LoginHistory:
        """ORM 模型转领域实体"""
        return LoginHistory(
            id=model.id,
            user_id=model.user_id,
            username=model.username,
            ip_address=model.ip_address,
            user_agent=model.user_agent,
            login_at=model.login_at,
            success=model.success,
        )

    @staticmethod
    def to_orm(entity: LoginHistory) -> LoginHistoryModel:
        """领域实体转 ORM 模型"""
        return LoginHistoryModel(
            id=entity.id,
            user_id=entity.user_id,
            username=entity.username,
            ip_address=entity.ip_address,
            user_agent=entity.user_agent,
            login_at=entity.login_at,
            success=entity.success,
        )


class GitSourceMapper:
    """Git 源映射器"""

    @staticmethod
    def to_domain(model: GitSourceModel) -> GitSource:
        """ORM 模型转领域实体"""
        return GitSource(
            id=model.id,
            application_id=model.application_id,
            repository_url=model.repository_url,
            deploy_branches=model.deploy_branches,
            auto_deploy=model.auto_deploy,
            created_at=model.created_at,
            updated_at=model.updated_at,
        )

    @staticmethod
    def to_orm(entity: GitSource) -> GitSourceModel:
        """领域实体转 ORM 模型"""
        return GitSourceModel(
            id=entity.id,
            application_id=entity.application_id,
            repository_url=entity.repository_url,
            deploy_branches=entity.deploy_branches,
            auto_deploy=entity.auto_deploy,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )


class ImageSourceMapper:
    """镜像源映射器"""

    @staticmethod
    def to_domain(model: ImageSourceModel) -> ImageSource:
        """ORM 模型转领域实体"""
        return ImageSource(
            id=model.id,
            application_id=model.application_id,
            image_name=model.image_name,
            registry_url=model.registry_url,
            created_at=model.created_at,
            updated_at=model.updated_at,
        )

    @staticmethod
    def to_orm(entity: ImageSource) -> ImageSourceModel:
        """领域实体转 ORM 模型"""
        return ImageSourceModel(
            id=entity.id,
            application_id=entity.application_id,
            image_name=entity.image_name,
            registry_url=entity.registry_url,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )


class ApplicationConfigFileMapper:
    """应用配置文件映射器"""

    @staticmethod
    def to_domain(model: ApplicationConfigFileModel) -> ApplicationConfigFile:
        """ORM 模型转领域实体"""
        return ApplicationConfigFile(
            id=model.id,
            application_id=model.application_id,
            path=model.path,
            content=model.content,
            created_at=model.created_at,
            updated_at=model.updated_at,
        )

    @staticmethod
    def to_orm(entity: ApplicationConfigFile) -> ApplicationConfigFileModel:
        """领域实体转 ORM 模型"""
        return ApplicationConfigFileModel(
            id=entity.id,
            application_id=entity.application_id,
            path=entity.path,
            content=entity.content,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )


class ApplicationMapper:
    """应用映射器"""

    @staticmethod
    def to_domain(model: ApplicationModel) -> Application:
        """ORM 模型转领域实体"""
        return Application(
            id=model.id,
            name=model.name,
            code=model.code,
            image_pull_policy=model.image_pull_policy,
            enabled=model.enabled,
            status=model.status,
            created_at=model.created_at,
            updated_at=model.updated_at,
            git_source=GitSourceMapper.to_domain(model.git_source) if model.git_source else None,
            image_source=ImageSourceMapper.to_domain(model.image_source) if model.image_source else None,
            config_files=[ApplicationConfigFileMapper.to_domain(cf) for cf in model.config_files],
        )

    @staticmethod
    def to_orm(entity: Application) -> ApplicationModel:
        """领域实体转 ORM 模型"""
        model = ApplicationModel(
            id=entity.id,
            name=entity.name,
            code=entity.code,
            enabled=entity.enabled,
            image_pull_policy=entity.image_pull_policy,
            status=entity.status,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )

        # 设置关联实体
        if entity.git_source:
            model.git_source = GitSourceMapper.to_orm(entity.git_source)
        if entity.image_source:
            model.image_source = ImageSourceMapper.to_orm(entity.image_source)
        if entity.config_files:
            model.config_files = [ApplicationConfigFileMapper.to_orm(cf) for cf in entity.config_files]

        return model


class DeploymentMapper:
    """部署记录映射器"""

    @staticmethod
    def to_domain(model: DeploymentModel) -> Deployment:
        """ORM 模型转领域实体"""
        return Deployment(
            id=model.id,
            application_id=model.application_id,
            application_name=model.application_name,
            trigger_type=TriggerType(model.trigger_type),
            status=model.status,
            operation_type=OperationType(model.operation_type),
            trigger_ref=model.trigger_ref,
            image_name=model.image_name,
            env_file=model.env_file,
            started_at=model.started_at,
            finished_at=model.finished_at,
            duration_ms=model.duration_ms,
            log_text=model.log_text,
            error_message=model.error_message,
            is_rollback=model.is_rollback,
            rollback_from_deployment_id=model.rollback_from_deployment_id,
        )

    @staticmethod
    def to_orm(entity: Deployment) -> DeploymentModel:
        """领域实体转 ORM 模型"""
        return DeploymentModel(
            id=entity.id,
            application_id=entity.application_id,
            application_name=entity.application_name,
            trigger_type=entity.trigger_type.value,
            operation_type=entity.operation_type.value,
            status=entity.status,
            trigger_ref=entity.trigger_ref,
            image_name=entity.image_name,
            env_file=entity.env_file,
            started_at=entity.started_at,
            finished_at=entity.finished_at,
            duration_ms=entity.duration_ms,
            log_text=entity.log_text,
            error_message=entity.error_message,
            is_rollback=entity.is_rollback,
            rollback_from_deployment_id=entity.rollback_from_deployment_id,
        )


class RouteMapper:
    """路由配置映射器"""

    @staticmethod
    def to_domain(model: RouteModel) -> Route:
        return Route(
            id=model.id,
            name=model.name,
            domain=model.domain,
            path_prefix=model.path_prefix,
            target_url=model.target_url,
            enabled=model.enabled,
            https_enabled=model.https_enabled,
            cert_pem=model.cert_pem,
            cert_key=model.cert_key,
            cert_type=CertType(model.cert_type),
            created_at=model.created_at,
            updated_at=model.updated_at,
        )

    @staticmethod
    def to_orm(entity: Route) -> RouteModel:
        return RouteModel(
            id=entity.id,
            name=entity.name,
            domain=entity.domain,
            path_prefix=entity.path_prefix,
            target_url=entity.target_url,
            enabled=entity.enabled,
            https_enabled=entity.https_enabled,
            cert_pem=entity.cert_pem,
            cert_key=entity.cert_key,
            cert_type=str(entity.cert_type),
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )


__all__ = [
    "ApplicationConfigFileMapper",
    "ApplicationMapper",
    "DeploymentMapper",
    "GitSourceMapper",
    "ImageSourceMapper",
    "LoginHistoryMapper",
    "RouteMapper",
    "UserMapper",
]
