"""DeploymentRepository 单元测试"""

from pomelo_orbit.domain.cd.entities import Deployment, TriggerType
from pomelo_orbit.domain.cd.value_objects import OperationType
from pomelo_orbit.infrastructure.cd.repositories.deployment import DeploymentRepositoryImpl
from pomelo_orbit.infrastructure.time_utils import utc_now


class TestDeploymentRepository:
    """部署记录仓储测试"""

    def test_save_and_find_by_id(self, db_session):
        """测试保存部署记录并通过 ID 查找"""
        repo = DeploymentRepositoryImpl(db_session)
        deployment = Deployment(
            id="deploy-1",
            project_id="project-1",
            application_id="app-1",
            application_name="Test App",
            trigger_type=TriggerType.MANUAL,
            status="success",
            operation_type=OperationType.DEPLOY,
            is_rollback=False,
            started_at=utc_now(),
        )

        repo.save(deployment)
        found = repo.find_by_id("deploy-1")

        assert found is not None
        assert found.id == "deploy-1"
        assert found.application_id == "app-1"
        assert found.status == "success"

    def test_find_by_application(self, db_session):
        """测试查询应用的部署记录"""
        repo = DeploymentRepositoryImpl(db_session)
        for i in range(5):
            deployment = Deployment(
                id=f"deploy-app1-{i}",
                project_id="project-1",
                application_id="app-1",
                application_name="App 1",
                trigger_type=TriggerType.MANUAL,
                status="success",
                operation_type=OperationType.DEPLOY,
                is_rollback=False,
                started_at=utc_now(),
            )
            repo.save(deployment)

        for i in range(3):
            deployment = Deployment(
                id=f"deploy-app2-{i}",
                project_id="project-1",
                application_id="app-2",
                application_name="App 2",
                trigger_type=TriggerType.MANUAL,
                status="success",
                operation_type=OperationType.DEPLOY,
                is_rollback=False,
                started_at=utc_now(),
            )
            repo.save(deployment)

        deployments, total = repo.find_by_application("app-1")
        assert total == 5
        assert len(deployments) == 5
        assert all(d.application_id == "app-1" for d in deployments)

    def test_find_by_application_pagination(self, db_session):
        """测试部署记录分页"""
        repo = DeploymentRepositoryImpl(db_session)
        for i in range(25):
            deployment = Deployment(
                id=f"deploy-page-{i}",
                project_id="project-1",
                application_id="app-1",
                application_name="App 1",
                trigger_type=TriggerType.MANUAL,
                status="success",
                operation_type=OperationType.DEPLOY,
                is_rollback=False,
                started_at=utc_now(),
            )
            repo.save(deployment)

        deployments, total = repo.find_by_application("app-1", page=1, per_page=10)
        assert total == 25
        assert len(deployments) == 10

        deployments, total = repo.find_by_application("app-1", page=3, per_page=10)
        assert total == 25
        assert len(deployments) == 5

    def test_update_deployment(self, db_session):
        """测试更新部署记录"""
        repo = DeploymentRepositoryImpl(db_session)
        deployment = Deployment(
            id="deploy-update",
            project_id="project-1",
            application_id="app-1",
            application_name="App 1",
            trigger_type=TriggerType.MANUAL,
            status="pending",
            operation_type=OperationType.DEPLOY,
            is_rollback=False,
            started_at=utc_now(),
        )
        repo.save(deployment)

        deployment.status = "success"
        deployment.finished_at = utc_now()
        repo.save(deployment)

        found = repo.find_by_id("deploy-update")
        assert found is not None
        assert found.status == "success"
        assert found.finished_at is not None

    def test_delete_deployment(self, db_session):
        """测试删除部署记录"""
        repo = DeploymentRepositoryImpl(db_session)
        deployment = Deployment(
            id="deploy-delete",
            project_id="project-1",
            application_id="app-1",
            application_name="App 1",
            trigger_type=TriggerType.MANUAL,
            status="success",
            operation_type=OperationType.DEPLOY,
            is_rollback=False,
            started_at=utc_now(),
        )
        repo.save(deployment)

        repo.delete(deployment)
        found = repo.find_by_id("deploy-delete")
        assert found is None
