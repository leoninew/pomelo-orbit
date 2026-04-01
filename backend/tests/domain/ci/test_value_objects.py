"""测试 CI 值对象"""

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

    def test_credential_type(self):
        assert CredentialType.GIT_SSH.value == "git_ssh"
        assert CredentialType.GIT_TOKEN.value == "git_token"

    def test_pipeline_run_status(self):
        assert PipelineRunStatus.WAITING.value == "waiting"
        assert PipelineRunStatus.RUNNING.value == "running"
        assert PipelineRunStatus.SUCCESS.value == "success"
        assert PipelineRunStatus.FAILED.value == "failed"

    def test_pipeline_run_trigger(self):
        assert PipelineRunTrigger.MANUAL.value == "manual"
        assert PipelineRunTrigger.WEBHOOK.value == "webhook"

    def test_job_status(self):
        assert JobStatus.WAITING.value == "waiting"
        assert JobStatus.RUNNING.value == "running"
        assert JobStatus.SUCCESS.value == "success"
        assert JobStatus.FAILED.value == "failed"
        assert JobStatus.FAULTED.value == "faulted"
        assert JobStatus.SKIPPED.value == "skipped"
        assert JobStatus.CANCELED.value == "canceled"


class TestVariableDeclaration:

    def test_create_variable_declaration(self):
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
        var_decl = VariableDeclaration(name="VAR")
        assert var_decl.description == ""
        assert var_decl.required is False
        assert var_decl.default is None
        assert var_decl.secret is False


class TestStepDefinition:

    def test_create_leaf_step(self):
        step = StepDefinition(
            name="build",
            image="python:3.12",
            commands=["pip install -r requirements.txt", "python setup.py build"],
        )
        assert step.name == "build"
        assert step.image == "python:3.12"
        assert step.commands is not None
        assert len(step.commands) == 2
        assert step.steps is None

    def test_create_orchestration_step(self):
        child_step = StepDefinition(name="unit-test", image="python:3.12", commands=["pytest tests/unit"])
        parent_step = StepDefinition(name="test", steps=[child_step])
        assert parent_step.name == "test"
        assert parent_step.image is None
        assert parent_step.steps is not None
        assert len(parent_step.steps) == 1
        assert parent_step.steps[0].name == "unit-test"

    def test_create_checkout_step(self):
        step = StepDefinition(
            name="checkout",
            uses="checkout",
            inputs={"depth": 1, "ref": "main"},
            outputs=["commit_sha", "commit_message"],
        )
        assert step.name == "checkout"
        assert step.uses == "checkout"
        assert step.inputs is not None
        assert step.inputs["depth"] == 1
        assert step.inputs["ref"] == "main"
        assert step.outputs is not None
        assert len(step.outputs) == 2


class TestPipelineDefinition:

    def test_create_pipeline_definition(self):
        step = StepDefinition(name="build", image="python:3.12", commands=["python setup.py build"])
        pipeline = PipelineDefinition(version="v1", timeout=3600, steps=[step])
        assert pipeline.version == "v1"
        assert pipeline.timeout == 3600
        assert len(pipeline.steps) == 1
        assert pipeline.steps[0].name == "build"

    def test_pipeline_definition_without_timeout(self):
        step = StepDefinition(name="build", image="python:3.12", commands=["python setup.py build"])
        pipeline = PipelineDefinition(version="v1", steps=[step])
        assert pipeline.timeout is None
