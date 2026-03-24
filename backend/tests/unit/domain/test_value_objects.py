"""
值对象测试
"""

import pytest

from pomelo_orbit.domain.value_objects import (
    ApplicationStatus,
    DeployStatus,
    ImagePullPolicy,
    OperationType,
)


class TestApplicationStatus:
    """ApplicationStatus 枚举测试"""

    def test_has_started_status(self):
        """验证 ApplicationStatus 包含 STARTED 枚举值"""
        assert ApplicationStatus.STARTED.value == "started"

    def test_has_stopped_status(self):
        """验证 ApplicationStatus 包含 STOPPED 枚举值"""
        assert ApplicationStatus.STOPPED.value == "stopped"

    def test_invalid_status_raises_error(self):
        """验证使用无效枚举值时抛出 ValueError"""
        with pytest.raises(ValueError):
            ApplicationStatus("invalid_status")


class TestOperationType:
    """OperationType 枚举测试"""

    def test_has_deploy_type(self):
        """验证 OperationType 包含 DEPLOY 枚举值"""
        assert OperationType.DEPLOY.value == "deploy"

    def test_has_stop_type(self):
        """验证 OperationType 包含 STOP 枚举值"""
        assert OperationType.STOP.value == "stop"

    def test_has_restart_type(self):
        """验证 OperationType 包含 RESTART 枚举值"""
        assert OperationType.RESTART.value == "restart"

    def test_invalid_type_raises_error(self):
        """验证使用无效枚举值时抛出 ValueError"""
        with pytest.raises(ValueError):
            OperationType("invalid_type")


class TestDeployStatus:
    """DeployStatus 枚举测试"""

    def test_has_queued_status(self):
        """验证 DeployStatus 包含 QUEUED 枚举值"""
        assert DeployStatus.QUEUED.value == "queued"

    def test_has_running_status(self):
        """验证 DeployStatus 包含 RUNNING 枚举值"""
        assert DeployStatus.RUNNING.value == "running"

    def test_has_success_status(self):
        """验证 DeployStatus 包含 SUCCESS 枚举值"""
        assert DeployStatus.SUCCESS.value == "success"

    def test_has_failed_status(self):
        """验证 DeployStatus 包含 FAILED 枚举值"""
        assert DeployStatus.FAILED.value == "failed"


class TestImagePullPolicy:
    """ImagePullPolicy 枚举测试"""

    def test_has_always_policy(self):
        """验证 ImagePullPolicy 包含 ALWAYS 枚举值"""
        assert ImagePullPolicy.ALWAYS.value == "always"

    def test_has_missing_policy(self):
        """验证 ImagePullPolicy 包含 MISSING 枚举值"""
        assert ImagePullPolicy.MISSING.value == "missing"

    def test_has_never_policy(self):
        """验证 ImagePullPolicy 包含 NEVER 枚举值"""
        assert ImagePullPolicy.NEVER.value == "never"
