"""测试 CI 值对象"""

from pomelo_orbit.domain.ci.value_objects import (
    CredentialType,
    PipelineRunStatus,
    PipelineRunTrigger,
    StageDefinition,
    StageStatus,
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

    def test_stage_status(self):
        assert StageStatus.WAITING.value == "waiting"
        assert StageStatus.RUNNING.value == "running"
        assert StageStatus.SUCCESS.value == "success"
        assert StageStatus.FAILED.value == "failed"
        assert StageStatus.FAULTED.value == "faulted"
        assert StageStatus.SKIPPED.value == "skipped"
        assert StageStatus.CANCELED.value == "canceled"


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


class TestStageDefinition:
    def test_create_stage(self):
        stage = StageDefinition(
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
        from pomelo_orbit.domain.ci.value_objects import ArtifactConfig

        stage = StageDefinition(
            name="test",
            image="golang:1.22-alpine",
            script="go test ./...",
            env={"GOFLAGS": "-v"},
            artifacts=[ArtifactConfig(path="coverage.out", name="coverage")],
            depends_on=["clone"],
        )
        assert stage.env == {"GOFLAGS": "-v"}
        assert stage.artifacts is not None
        assert len(stage.artifacts) == 1
        assert stage.depends_on == ["clone"]

    def test_builtin_stage_flags(self):
        stage = StageDefinition(
            name="clone",
            image="alpine/git",
            script="git clone .",
        )
        assert stage.name == "clone"
