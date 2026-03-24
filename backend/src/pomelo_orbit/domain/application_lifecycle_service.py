"""应用生命周期领域服务 - 封装应用部署、启停、删除的业务规则"""

from ulid import ULID

from pomelo_orbit.domain.entities import Application, Deployment, TriggerType
from pomelo_orbit.domain.exceptions import BusinessError
from pomelo_orbit.domain.value_objects import ApplicationStatus, DeployStatus, OperationType
from pomelo_orbit.infrastructure.time_utils import utc_now


class ApplicationLifecycleDomainService:
    """
    应用生命周期领域服务

    封装应用部署、启停、删除的业务规则，不包含基础设施依赖
    """

    @staticmethod
    def validate_deploy(application: Application) -> None:
        """
        验证应用是否可以部署

        Raises:
            ValueError: 当应用状态不允许部署时
        """
        # 部署没有状态限制，已删除的应用也可以重新部署

    @staticmethod
    def validate_start(application: Application, has_deployment: bool) -> None:
        """
        验证应用是否可以启动

        Args:
            application: 应用实体
            has_deployment: 是否已有部署记录

        Raises:
            ValueError: 当应用状态不允许启动时
        """
        if not has_deployment:
            raise BusinessError("应用目录不存在, 请先部署应用")

        if application.status == ApplicationStatus.STARTED:
            raise BusinessError("应用已在运行中")

    @staticmethod
    def validate_stop(application: Application) -> None:
        """
        验证应用是否可以停止

        Args:
            application: 应用实体

        Raises:
            ValueError: 当应用状态不允许停止时
        """
        if application.status != ApplicationStatus.STARTED:
            raise BusinessError("应用未启动")

    @staticmethod
    def validate_restart(application: Application, has_deployment: bool) -> None:
        """
        验证应用是否可以重启

        Args:
            application: 应用实体
            has_deployment: 是否已有部署记录

        Raises:
            ValueError: 当应用状态不允许重启时
        """
        if application.status != ApplicationStatus.STARTED:
            raise BusinessError("应用未启动")

        if not has_deployment:
            raise BusinessError("应用目录不存在, 请先部署应用")

    @staticmethod
    def validate_delete(application: Application) -> None:
        """
        验证应用是否可以删除

        Args:
            application: 应用实体

        Raises:
            ValueError: 当应用状态不允许删除时
        """
        if application.status == ApplicationStatus.STARTED:
            raise BusinessError("应用正在运行, 请先停止后再删除")

    @staticmethod
    def create_deploy_record(
        application: Application,
        trigger_type: TriggerType = TriggerType.MANUAL,
        env_file: str | None = None,
    ) -> Deployment:
        """
        创建部署记录

        Args:
            application: 应用实体
            trigger_type: 触发类型
            env_file: 环境变量文件名

        Returns:
            新的部署记录
        """
        return Deployment(
            id=str(ULID()),
            application_id=application.id,
            application_name=application.name,
            operation_type=OperationType.DEPLOY,
            trigger_type=trigger_type,
            status=DeployStatus.RUNNING.value,
            is_rollback=False,
            env_file=env_file,
            started_at=utc_now(),
        )

    @staticmethod
    def create_stop_record(application: Application, env_file: str | None = None) -> Deployment:
        """
        创建停止记录

        Args:
            application: 应用实体
            env_file: 环境变量文件名

        Returns:
            新的停止记录
        """
        return Deployment(
            id=str(ULID()),
            application_id=application.id,
            application_name=application.name,
            operation_type=OperationType.STOP,
            trigger_type=TriggerType.MANUAL,
            status=DeployStatus.RUNNING.value,
            is_rollback=False,
            env_file=env_file,
            started_at=utc_now(),
        )

    @staticmethod
    def create_restart_record(application: Application, env_file: str | None = None) -> Deployment:
        """
        创建重启记录

        Args:
            application: 应用实体
            env_file: 环境变量文件名

        Returns:
            新的重启记录
        """
        return Deployment(
            id=str(ULID()),
            application_id=application.id,
            application_name=application.name,
            operation_type=OperationType.RESTART,
            trigger_type=TriggerType.MANUAL,
            status=DeployStatus.RUNNING.value,
            is_rollback=False,
            env_file=env_file,
            started_at=utc_now(),
        )

    @staticmethod
    def mark_deploy_success(deployment: Deployment, application: Application) -> None:
        """标记部署成功并更新应用状态"""
        deployment.status = DeployStatus.SUCCESS.value
        deployment.finished_at = utc_now()
        deployment.duration_ms = int((deployment.finished_at - deployment.started_at).total_seconds() * 1000)
        application.mark_as_started()

    @staticmethod
    def mark_deploy_failure(deployment: Deployment, error_message: str) -> None:
        """标记部署失败"""
        deployment.status = DeployStatus.FAILED.value
        deployment.finished_at = utc_now()
        deployment.duration_ms = int((deployment.finished_at - deployment.started_at).total_seconds() * 1000)
        deployment.error_message = error_message

    @staticmethod
    def mark_operation_success(deployment: Deployment) -> None:
        """标记操作成功（用于停止、重启等）"""
        deployment.status = DeployStatus.SUCCESS.value
        deployment.finished_at = utc_now()
        deployment.duration_ms = int((deployment.finished_at - deployment.started_at).total_seconds() * 1000)

    @staticmethod
    def mark_operation_failure(deployment: Deployment, error_message: str) -> None:
        """标记操作失败"""
        deployment.status = DeployStatus.FAILED.value
        deployment.finished_at = utc_now()
        deployment.duration_ms = int((deployment.finished_at - deployment.started_at).total_seconds() * 1000)
        deployment.error_message = error_message
