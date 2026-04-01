"""测试 CI 值对象"""

import pytest
from pydantic import ValidationError

from pomelo_orbit.domain.ci.value_objects import (
    CredentialType,
    JobStatus,
    PipelineDefinition,
    PipelineRunStatus,
    PipelineRunTrigger,
    StepDefinition,
    VariableDeclaration,
)


class TestEnums:
    """测试枚举类型"""

    def test_credential_type(self):
        """测试凭据类型枚举"""
        assert CredentialType.GIT_SSH == "git_ssh"
        assert CredentialType.GIT_TOKEN == "git_token"

    def test_pipeline_run_status(self):
        """测试 pipeline run 状态枚举"""
        assert PipelineRunStatus.WAITING == "waiting"
        assert PipelineRunStatus.RUNNING == "running"
        assert PipelineRunStatus.SUCCESS == "success"
        assert PipelineRunStatus.FAILED == "failed"

    def test_pipeline_run_trigger(self):
        """测试 pipeline run 触发方式枚举"""
        assert PipelineRunTrigger.MANUAL == "manual"
        assert PipelineRunTrigger.WEBHOOK == "webhook"

    def test_job_status(self):
        """测试 job 状态枚举"""
        assert JobStatus.WAITING == "waiting"
        assert JobStatus.RUNNING == "running"
        assert JobStatus.SUCCESS == "success"
        assert JobStatus.FAILED == "failed"
        assert JobStatus.FAULTED == "faulted"
        assert JobStatus.SKIPPED == "skipped"
        assert JobStatus.CANCELED == "canceled"


class TestVariableDeclaration:
    """测试变量声明"""

    def test_create_variable_declaration(self):
        """测试创建变量声明"""
        var_decl = VariableDeclaration(
            name="IMAGE_NAME",
            description="Docker image name",
            required=True,
            default=None,
            secret=False,
        )

        assert var_decl.name == "IMAGE_NAME"
        assert var_decl.description == "Docker image name"
        assert var_decl.required is True
        assert var_decl.default is None
        assert var_decl.secret is False

    def test_variable_declaration_defaults(self):
        """测试变量声明默认值"""
        var_decl = VariableDeclaration(name="VAR")

        assert var_decl.description == ""
        assert var_decl.required is False
        assert var_decl.default is None
        assert var_decl.secret is False


class TestStepDefinition:
    """测试 Step 定义"""

    def test_create_leaf_step(self):
        """测试创建叶子 step"""
        step = StepDefinition(
            name="build",
            image="python:3.12",
            commands=["pip install -r requirements.txt", "python setup.py build"],
        )

        assert step.name == "build"
        assert step.image == "python:3.12"
        assert len(step.commands) == 2
        assert step.steps is None

    def test_create_orchestration_step(self):
        """测试创建编排 step"""
        child_step = StepDefinition(
            name="unit-test",
            image="python:3.12",
            commands=["pytest tests/unit"],
        )

        parent_step = StepDefinition(
            name="test",
            steps=[child_step],
        )

        assert parent_step.name == "test"
        assert parent_step.image is None
        assert len(parent_step.steps) == 1
        assert parent_step.steps[0].name == "unit-test"

    def test_create_checkout_step(self):
        """测试创建 checkout step"""
        step = StepDefinition(
            name="checkout",
            uses="checkout",
            **{"with": {"depth": 1, "ref": "main"}},
            outputs=["commit_sha", "commit_message"],
        )

        assert step.name == "checkout"
        assert step.uses == "checkout"
        assert step.with_["depth"] == 1
        assert step.with_["ref"] == "main"
        assert len(step.outputs) == 2


class TestPipelineDefinition:
    """测试 Pipeline 定义"""

    def test_create_pipeline_definition(self):
        """测试创建 pipeline 定义"""
        step = StepDefinition(
            name="build",
            image="python:3.12",
            commands=["python setup.py build"],
        )

        pipeline = PipelineDefinition(
            version="v1",
            timeout=3600,
            steps=[step],
        )

        assert pipeline.version == "v1"
        assert pipeline.timeout == 3600
        assert len(pipeline.steps) == 1
        assert pipeline.steps[0].name == "build"

    def test_pipeline_definition_without_timeout(self):
        """测试创建没有超时的 pipeline 定义"""
        step = StepDefinition(
            name="build",
            image="python:3.12",
            commands=["python setup.py build"],
        )

        pipeline = PipelineDefinition(
            version="v1",
            steps=[step],
        )

        assert pipeline.timeout is None
