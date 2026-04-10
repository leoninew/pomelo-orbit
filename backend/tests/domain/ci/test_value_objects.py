"""测试 CI 值对象"""

from pomelo_orbit.domain.cd.value_objects import TaskStatus
from pomelo_orbit.domain.ci.value_objects import (
    ArtifactConfig,
    CredentialType,
    PipelineRunTrigger,
    StageDefinition,
    VariableDeclaration,
    VariableSource,
)


class TestEnums:
    def test_credential_type(self):
        assert CredentialType.GIT_SSH.value == "git_ssh"
        assert CredentialType.GIT_TOKEN.value == "git_token"

    def test_task_status(self):
        assert TaskStatus.WAITING_TO_RUN.value == "waiting_to_run"
        assert TaskStatus.RUNNING.value == "running"
        assert TaskStatus.RAN_TO_COMPLETION.value == "ran_to_completion"
        assert TaskStatus.FAULTED.value == "faulted"
        assert TaskStatus.CANCELED.value == "canceled"

    def test_pipeline_run_trigger(self):
        assert PipelineRunTrigger.MANUAL.value == "manual"
        assert PipelineRunTrigger.WEBHOOK.value == "webhook"


class TestVariableDeclaration:
    def test_create_variable_declaration(self):
        var_decl = VariableDeclaration(
            name="IMAGE_NAME",
            description="Docker image name",
            value=None,
            secret=False,
            source=VariableSource.TEMPLATE,
        )
        assert var_decl.name == "IMAGE_NAME"
        assert var_decl.description == "Docker image name"
        assert var_decl.value is None
        assert var_decl.secret is False
        assert var_decl.source == VariableSource.TEMPLATE

    def test_variable_declaration_defaults(self):
        var_decl = VariableDeclaration(name="VAR")
        assert var_decl.description == ""
        assert var_decl.value is None
        assert var_decl.secret is False
        assert var_decl.source == VariableSource.TEMPLATE_CUSTOM


class TestStageDefinition:
    def test_create_stage(self):
        stage = StageDefinition(
            id="build",
            name="build",
            image="python:3.12-slim",
            script="pip install -r requirements.txt\npytest tests/",
        )
        assert stage.name == "build"
        assert stage.image == "python:3.12-slim"
        assert "pytest" in stage.script
        assert stage.depends_on == []
        assert stage.env == {}
        assert stage.artifacts is None

    def test_stage_with_env_and_artifacts(self):
        stage = StageDefinition(
            id="test",
            name="test",
            image="golang:1.22-alpine",
            script="go test ./...",
            env={"GOFLAGS": "-v"},
            artifacts=[ArtifactConfig(path="coverage.out", name="coverage")],
            depends_on=["clone-id"],
        )
        assert stage.env == {"GOFLAGS": "-v"}
        assert stage.artifacts is not None
        assert len(stage.artifacts) == 1
        assert len(stage.depends_on) == 1
        assert stage.depends_on[0] == "clone-id"

    def test_builtin_stage_flags(self):
        stage = StageDefinition(
            id="clone",
            name="clone",
            image="alpine/git",
            script="git clone .",
        )
        assert stage.name == "clone"
