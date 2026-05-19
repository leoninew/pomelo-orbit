"""
领域实体与 ORM 模型之间的映射器
"""

from pomelo_orbit.domain.auth.entities import LoginAttempt, LoginHistory, Permission, Role, User
from pomelo_orbit.domain.cd.entities import (
    Application,
    ApplicationConfigFile,
    ApplicationRoute,
    ApplicationServiceConfig,
    CertType,
    Deployment,
    Route,
    TriggerType,
)
from pomelo_orbit.domain.cd.value_objects import OperationType
from pomelo_orbit.infrastructure.persistence.models import (
    ApplicationConfigFileModel,
    ApplicationModel,
    ApplicationRouteModel,
    ApplicationServiceConfigModel,
    DeploymentModel,
    LoginAttemptModel,
    LoginHistoryModel,
    PermissionModel,
    RoleModel,
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
            status=model.status,
            oauth_provider=model.oauth_provider,
            oauth_provider_id=model.oauth_provider_id,
            email=model.email,
            auth_source=model.auth_source,
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
            status=entity.status,
            oauth_provider=entity.oauth_provider,
            oauth_provider_id=entity.oauth_provider_id,
            email=entity.email,
            auth_source=entity.auth_source,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
            last_login_at=entity.last_login_at,
        )


class RoleMapper:
    @staticmethod
    def to_domain(model: RoleModel) -> Role:
        return Role(
            id=model.id,
            code=model.code,
            name=model.name,
            description=model.description,
            created_at=model.created_at,
            updated_at=model.updated_at,
        )

    @staticmethod
    def to_orm(entity: Role) -> RoleModel:
        return RoleModel(
            id=entity.id,
            code=entity.code,
            name=entity.name,
            description=entity.description,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )


class PermissionMapper:
    @staticmethod
    def to_domain(model: PermissionModel) -> Permission:
        return Permission(
            id=model.id,
            code=model.code,
            name=model.name,
            description=model.description,
            created_at=model.created_at,
            updated_at=model.updated_at,
        )

    @staticmethod
    def to_orm(entity: Permission) -> PermissionModel:
        return PermissionModel(
            id=entity.id,
            code=entity.code,
            name=entity.name,
            description=entity.description,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
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


class LoginAttemptMapper:
    """登录尝试记录映射器"""

    @staticmethod
    def to_domain(model: LoginAttemptModel) -> LoginAttempt:
        """ORM 模型转领域实体"""
        return LoginAttempt(
            id=model.id,
            username=model.username,
            ip_address=model.ip_address,
            user_agent=model.user_agent,
            success=model.success,
            created_at=model.created_at,
        )

    @staticmethod
    def to_orm(entity: LoginAttempt) -> LoginAttemptModel:
        """领域实体转 ORM 模型"""
        return LoginAttemptModel(
            id=entity.id,
            username=entity.username,
            ip_address=entity.ip_address,
            user_agent=entity.user_agent,
            success=entity.success,
            created_at=entity.created_at,
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


class ApplicationRouteMapper:
    """应用路由映射器"""

    @staticmethod
    def to_domain(model: ApplicationRouteModel) -> ApplicationRoute:
        return ApplicationRoute(
            id=model.id,
            application_id=model.application_id,
            service_name=model.service_name,
            domain=model.domain,
            port=model.port,
            created_at=model.created_at,
            updated_at=model.updated_at,
        )

    @staticmethod
    def to_orm(entity: ApplicationRoute) -> ApplicationRouteModel:
        return ApplicationRouteModel(
            id=entity.id,
            application_id=entity.application_id,
            service_name=entity.service_name,
            domain=entity.domain,
            port=entity.port,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )


class ApplicationServiceConfigMapper:
    """应用 service 配置映射器"""

    @staticmethod
    def to_domain(model: ApplicationServiceConfigModel) -> ApplicationServiceConfig:
        return ApplicationServiceConfig(
            id=model.id,
            application_id=model.application_id,
            service_name=model.service_name,
            image=model.image,
            environment=model.environment,
            volumes=model.volumes,
            created_at=model.created_at,
            updated_at=model.updated_at,
        )

    @staticmethod
    def to_orm(entity: ApplicationServiceConfig) -> ApplicationServiceConfigModel:
        return ApplicationServiceConfigModel(
            id=entity.id,
            application_id=entity.application_id,
            service_name=entity.service_name,
            image=entity.image,
            environment=entity.environment,
            volumes=entity.volumes,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )


class ApplicationMapper:
    """应用映射器"""

    @staticmethod
    def to_domain(model: ApplicationModel) -> Application:
        return Application(
            id=model.id,
            project_id=model.project_id,
            name=model.name,
            code=model.code,
            image_pull_policy=model.image_pull_policy,
            status=model.status,
            route_managed=model.route_managed,
            created_at=model.created_at,
            updated_at=model.updated_at,
            config_files=[ApplicationConfigFileMapper.to_domain(cf) for cf in model.config_files],
            routes=[ApplicationRouteMapper.to_domain(r) for r in model.app_routes],
        )

    @staticmethod
    def to_orm(entity: Application) -> ApplicationModel:
        model = ApplicationModel(
            id=entity.id,
            project_id=entity.project_id,
            name=entity.name,
            code=entity.code,
            image_pull_policy=entity.image_pull_policy,
            status=entity.status,
            route_managed=entity.route_managed,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )
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
            project_id=model.project_id,
            application_id=model.application_id,
            application_name=model.application_name,
            trigger_type=TriggerType(model.trigger_type),
            status=model.status,
            operation_type=OperationType(model.operation_type),
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
            project_id=entity.project_id,
            application_id=entity.application_id,
            application_name=entity.application_name,
            trigger_type=entity.trigger_type.value,
            operation_type=entity.operation_type.value,
            status=entity.status,
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
    "ApplicationRouteMapper",
    "ApplicationServiceConfigMapper",
    "DeploymentMapper",
    "LoginHistoryMapper",
    "PermissionMapper",
    "RoleMapper",
    "RouteMapper",
    "UserMapper",
]
