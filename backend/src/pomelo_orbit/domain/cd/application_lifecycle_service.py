"""应用生命周期领域服务 - 封装应用部署、启停、删除的业务规则"""

from ulid import ULID

from pomelo_orbit.domain.cd.entities import Application, Deployment, TriggerType
from pomelo_orbit.domain.cd.value_objects import ApplicationStatus, OperationType, TaskStatus
from pomelo_orbit.domain.exceptions import BusinessError
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
            BusinessError: 当应用状态不允许部署时
        """
        if not application.can_deploy():
            raise BusinessError("应用正在部署中, 请稍后再试")

    @staticmethod
    def validate_stop(application: Application) -> None:
        """
        验证应用是否可以停止

        Raises:
            BusinessError: 当应用状态不允许停止时
        """
        if not application.can_stop():
            raise BusinessError("应用未在运行中, 无法停止")

    @staticmethod
    def validate_restart(application: Application) -> None:
        """
        验证应用是否可以重启

        Raises:
            BusinessError: 当应用状态不允许重启时
        """
        if not application.can_restart():
            raise BusinessError("应用未在运行中, 无法重启")

    @staticmethod
    def validate_delete(application: Application) -> None:
        """
        验证应用是否可以删除

        Raises:
            BusinessError: 当应用状态不允许删除时
        """
        if application.status == ApplicationStatus.DEPLOYING:
            raise BusinessError("应用正在部署中, 请稍后再试")
        if application.status == ApplicationStatus.DEPLOYED:
            raise BusinessError("应用正在运行中, 请先停止后再删除")

    @staticmethod
    def create_deploy_record(
        application: Application,
        trigger_type: TriggerType = TriggerType.MANUAL,
        env_file: str | None = None,
    ) -> Deployment:
        """创建部署记录"""
        return Deployment(
            id=str(ULID()),
            application_id=application.id,
            application_name=application.name,
            operation_type=OperationType.DEPLOY,
            trigger_type=trigger_type,
            status=TaskStatus.WAITING_TO_RUN.value,
            is_rollback=False,
            env_file=env_file,
            started_at=utc_now(),
        )

    @staticmethod
    def create_stop_record(application: Application, env_file: str | None = None) -> Deployment:
        """创建停止记录"""
        return Deployment(
            id=str(ULID()),
            application_id=application.id,
            application_name=application.name,
            operation_type=OperationType.STOP,
            trigger_type=TriggerType.MANUAL,
            status=TaskStatus.WAITING_TO_RUN.value,
            is_rollback=False,
            env_file=env_file,
            started_at=utc_now(),
        )

    @staticmethod
    def create_restart_record(
        application: Application,
        trigger_type: TriggerType = TriggerType.MANUAL,
        env_file: str | None = None,
    ) -> Deployment:
        """创建重启记录"""
        return Deployment(
            id=str(ULID()),
            application_id=application.id,
            application_name=application.name,
            operation_type=OperationType.RESTART,
            trigger_type=trigger_type,
            status=TaskStatus.WAITING_TO_RUN.value,
            is_rollback=False,
            env_file=env_file,
            started_at=utc_now(),
        )

    @staticmethod
    def mark_deploy_success(deployment: Deployment, application: Application) -> None:
        """标记部署/重启成功，应用进入 deployed 状态"""
        deployment.status = TaskStatus.RAN_TO_COMPLETION.value
        deployment.finished_at = utc_now()
        deployment.duration_ms = int((deployment.finished_at - deployment.started_at).total_seconds() * 1000)
        application.mark_as_deployed()

    @staticmethod
    def mark_deploy_failure(deployment: Deployment, application: Application, error_message: str) -> None:
        """标记部署/重启失败，应用进入 deploy_failed 状态"""
        deployment.status = TaskStatus.FAULTED.value
        deployment.finished_at = utc_now()
        deployment.duration_ms = int((deployment.finished_at - deployment.started_at).total_seconds() * 1000)
        deployment.error_message = error_message
        application.mark_as_deploy_failed()

    @staticmethod
    def mark_deploy_canceled(deployment: Deployment, application: Application) -> None:
        """标记部署取消，应用进入 deploy_failed 状态"""
        deployment.status = TaskStatus.CANCELED.value
        deployment.finished_at = utc_now()
        deployment.duration_ms = int((deployment.finished_at - deployment.started_at).total_seconds() * 1000)
        application.mark_as_deploy_failed()

    @staticmethod
    def mark_stop_success(deployment: Deployment, application: Application) -> None:
        """标记停止成功，应用进入 undeployed 状态"""
        deployment.status = TaskStatus.RAN_TO_COMPLETION.value
        deployment.finished_at = utc_now()
        deployment.duration_ms = int((deployment.finished_at - deployment.started_at).total_seconds() * 1000)
        application.mark_as_undeployed()

    @staticmethod
    def mark_stop_failure(deployment: Deployment, application: Application, error_message: str) -> None:
        """标记停止失败，应用进入 deploy_failed 状态"""
        deployment.status = TaskStatus.FAULTED.value
        deployment.finished_at = utc_now()
        deployment.duration_ms = int((deployment.finished_at - deployment.started_at).total_seconds() * 1000)
        deployment.error_message = error_message
        application.mark_as_deploy_failed()
