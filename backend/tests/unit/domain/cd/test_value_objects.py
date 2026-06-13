"""
值对象测试
"""

import pytest

from pomelo_orbit.domain.cd.value_objects import (
    ApplicationStatus,
    ImagePullPolicy,
    OperationType,
    TaskStatus,
)


class TestApplicationStatus:
    """ApplicationStatus 枚举测试"""

    def test_has_deployed_status(self):
        assert ApplicationStatus.DEPLOYED.value == "deployed"

    def test_has_undeployed_status(self):
        assert ApplicationStatus.UNDEPLOYED.value == "undeployed"

    def test_has_deploying_status(self):
        assert ApplicationStatus.DEPLOYING.value == "deploying"

    def test_has_deploy_failed_status(self):
        assert ApplicationStatus.DEPLOY_FAILED.value == "deploy_failed"

    def test_invalid_status_raises_error(self):
        with pytest.raises(ValueError):
            ApplicationStatus("invalid_status")


class TestOperationType:
    """OperationType 枚举测试"""

    def test_has_deploy_type(self):
        assert OperationType.DEPLOY.value == "deploy"

    def test_has_stop_type(self):
        assert OperationType.STOP.value == "stop"

    def test_has_restart_type(self):
        assert OperationType.RESTART.value == "restart"

    def test_invalid_type_raises_error(self):
        with pytest.raises(ValueError):
            OperationType("invalid_type")


class TestTaskStatus:
    """TaskStatus 枚举测试"""

    def test_has_waiting_to_run_status(self):
        assert TaskStatus.WAITING_TO_RUN.value == "waiting_to_run"

    def test_has_running_status(self):
        assert TaskStatus.RUNNING.value == "running"

    def test_has_ran_to_completion_status(self):
        assert TaskStatus.RAN_TO_COMPLETION.value == "ran_to_completion"

    def test_has_faulted_status(self):
        assert TaskStatus.FAULTED.value == "faulted"

    def test_has_canceled_status(self):
        assert TaskStatus.CANCELED.value == "canceled"


class TestImagePullPolicy:
    """ImagePullPolicy 枚举测试"""

    def test_has_always_policy(self):
        assert ImagePullPolicy.ALWAYS.value == "always"

    def test_has_missing_policy(self):
        assert ImagePullPolicy.MISSING.value == "missing"

    def test_has_never_policy(self):
        assert ImagePullPolicy.NEVER.value == "never"
