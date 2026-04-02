"""Pipeline Service 单元测试"""

from unittest.mock import Mock

import pytest

from pomelo_orbit.application.ci.pipeline_service import PipelineService
from pomelo_orbit.domain.ci.entities import (
    Credential,
    PipelineRun,
    PipelineTemplate,
    Project,
)
from pomelo_orbit.domain.ci.value_objects import (
    CredentialType,
    PipelineRunStatus,
    PipelineRunTrigger,
)
from pomelo_orbit.domain.exceptions import BusinessError


def make_service(**overrides) -> PipelineService:
    defaults: dict = {
        "project_repo": Mock(),
        "credential_repo": Mock(),
        "template_repo": Mock(),
        "run_repo": Mock(),
        "artifact_repo": Mock(),
        "job_repo": Mock(),
        "job_log_repo": Mock(),
        "session_factory": Mock(),
        "executor_factory": Mock(),
    }
    defaults.update(overrides)
    return PipelineService(**defaults)


class TestProjectCRUD:
    def test_list_projects(self):
        project_repo = Mock()
        project_repo.find_paginated.return_value = ([], 0)
        service = make_service(project_repo=project_repo)
        projects, total = service.list_projects(page=1, per_page=10)
        assert projects == [] and total == 0
        project_repo.find_paginated.assert_called_once_with(page=1, per_page=10)

    def test_get_project_success(self):
        project = Project.create(
            name="p", repository_url="https://x.git", pipeline_template_id="t", git_credential_id="c"
        )
        project_repo = Mock()
        project_repo.find_by_id.return_value = project
        assert make_service(project_repo=project_repo).get_project(project.id) == project

    def test_get_project_not_found(self):
        project_repo = Mock()
        project_repo.find_by_id.return_value = None
        with pytest.raises(BusinessError) as exc:
            make_service(project_repo=project_repo).get_project("x")
        assert exc.value.status_code == 404

    def test_create_project_success(self):
        template_repo = Mock()
        template_repo.find_by_id.return_value = PipelineTemplate.create(
            name="t", content="v: v1", variable_declarations=[]
        )
        credential_repo = Mock()
        credential_repo.find_by_id.return_value = Credential.create(
            name="c", type=CredentialType.GIT_SSH, encrypted_data="x"
        )
        project_repo = Mock()
        service = make_service(project_repo=project_repo, credential_repo=credential_repo, template_repo=template_repo)
        project = service.create_project(
            name="p",
            repository_url="https://x.git",
            pipeline_template_id="t",
            git_credential_id="c",
            branch_filter="main",
            enable_webhook=True,
        )
        assert project.name == "p" and project.webhook_secret is not None
        project_repo.save.assert_called_once_with(project)

    def test_create_project_template_not_found(self):
        template_repo = Mock()
        template_repo.find_by_id.return_value = None
        with pytest.raises(BusinessError) as exc:
            make_service(template_repo=template_repo).create_project(
                name="p", repository_url="https://x.git", pipeline_template_id="t", git_credential_id="c"
            )
        assert exc.value.status_code == 404 and "PipelineTemplate" in str(exc.value)

    def test_create_project_credential_not_found(self):
        template_repo = Mock()
        template_repo.find_by_id.return_value = PipelineTemplate.create(
            name="t", content="v: v1", variable_declarations=[]
        )
        credential_repo = Mock()
        credential_repo.find_by_id.return_value = None
        with pytest.raises(BusinessError) as exc:
            make_service(template_repo=template_repo, credential_repo=credential_repo).create_project(
                name="p", repository_url="https://x.git", pipeline_template_id="t", git_credential_id="c"
            )
        assert exc.value.status_code == 404 and "Credential" in str(exc.value)

    def test_update_project_success(self):
        project = Project.create(
            name="old", repository_url="https://x.git", pipeline_template_id="t1", git_credential_id="c"
        )
        project_repo = Mock()
        project_repo.find_by_id.return_value = project
        template_repo = Mock()
        template_repo.find_by_id.return_value = Mock()
        service = make_service(project_repo=project_repo, template_repo=template_repo)
        updated = service.update_project(
            project.id, name="new", variable_overrides={"K": "V"}, pipeline_template_id="t2", branch_filter="main"
        )
        assert updated.name == "new" and updated.branch_filter == "main"
        project_repo.save.assert_called_once()

    def test_update_project_template_not_found(self):
        project = Project.create(
            name="p", repository_url="https://x.git", pipeline_template_id="t1", git_credential_id="c"
        )
        project_repo = Mock()
        project_repo.find_by_id.return_value = project
        template_repo = Mock()
        template_repo.find_by_id.return_value = None
        with pytest.raises(BusinessError) as exc:
            make_service(project_repo=project_repo, template_repo=template_repo).update_project(
                project.id, pipeline_template_id="nonexistent"
            )
        assert exc.value.status_code == 404

    def test_delete_project_success(self):
        project = Project.create(
            name="p", repository_url="https://x.git", pipeline_template_id="t", git_credential_id="c"
        )
        project_repo = Mock()
        project_repo.find_by_id.return_value = project
        project_repo.has_running_pipelines.return_value = False
        service = make_service(project_repo=project_repo)
        service.delete_project(project.id)
        project_repo.delete.assert_called_once_with(project)

    def test_delete_project_with_running_pipelines(self):
        project = Project.create(
            name="p", repository_url="https://x.git", pipeline_template_id="t", git_credential_id="c"
        )
        project_repo = Mock()
        project_repo.find_by_id.return_value = project
        project_repo.has_running_pipelines.return_value = True
        with pytest.raises(BusinessError) as exc:
            make_service(project_repo=project_repo).delete_project(project.id)
        assert exc.value.status_code == 409

    def test_get_webhook_config(self):
        project = Project.create(
            name="p", repository_url="https://x.git", pipeline_template_id="t", git_credential_id="c"
        )
        project.webhook_secret = "secret123"
        project_repo = Mock()
        project_repo.find_by_id.return_value = project
        config = make_service(project_repo=project_repo).get_webhook_config(project.id, "https://api.example.com")
        assert config["url"] == "https://api.example.com/api/v1/ci/webhooks/git"
        assert config["secret"] == "secret123"

    def test_regenerate_webhook_secret(self):
        project = Project.create(
            name="p", repository_url="https://x.git", pipeline_template_id="t", git_credential_id="c"
        )
        old_secret = project.webhook_secret
        project_repo = Mock()
        project_repo.find_by_id.return_value = project
        updated = make_service(project_repo=project_repo).regenerate_webhook_secret(project.id)
        assert updated.webhook_secret != old_secret
        project_repo.save.assert_called_once()


class TestCredentialCRUD:
    def test_list_credentials(self):
        creds = [
            Credential.create(name="c1", type=CredentialType.GIT_SSH, encrypted_data="d1"),
            Credential.create(name="c2", type=CredentialType.GIT_TOKEN, encrypted_data="d2"),
        ]
        credential_repo = Mock()
        credential_repo.find_paginated.return_value = (creds, 2)
        result, total = make_service(credential_repo=credential_repo).list_credentials()
        assert len(result) == 2 and total == 2

    def test_get_credential_success(self):
        cred = Credential.create(name="c", type=CredentialType.GIT_SSH, encrypted_data="x")
        credential_repo = Mock()
        credential_repo.find_by_id.return_value = cred
        assert make_service(credential_repo=credential_repo).get_credential(cred.id) == cred

    def test_get_credential_not_found(self):
        credential_repo = Mock()
        credential_repo.find_by_id.return_value = None
        with pytest.raises(BusinessError) as exc:
            make_service(credential_repo=credential_repo).get_credential("x")
        assert exc.value.status_code == 404

    def test_create_credential(self):
        credential_repo = Mock()
        cred = make_service(credential_repo=credential_repo).create_credential(
            name="c", credential_type="git_ssh", encrypted_data="x"
        )
        assert cred.name == "c" and cred.type == CredentialType.GIT_SSH
        credential_repo.save.assert_called_once_with(cred)

    def test_delete_credential_success(self):
        cred = Credential.create(name="c", type=CredentialType.GIT_SSH, encrypted_data="x")
        credential_repo = Mock()
        credential_repo.find_by_id.return_value = cred
        credential_repo.is_referenced_by_projects.return_value = False
        make_service(credential_repo=credential_repo).delete_credential(cred.id)
        credential_repo.delete.assert_called_once_with(cred)

    def test_delete_credential_referenced(self):
        cred = Credential.create(name="c", type=CredentialType.GIT_SSH, encrypted_data="x")
        credential_repo = Mock()
        credential_repo.find_by_id.return_value = cred
        credential_repo.is_referenced_by_projects.return_value = True
        with pytest.raises(BusinessError) as exc:
            make_service(credential_repo=credential_repo).delete_credential(cred.id)
        assert exc.value.status_code == 409


class TestTemplateCRUD:
    def test_list_templates(self):
        templates = [
            PipelineTemplate.create(name="t1", content="c1", variable_declarations=[]),
            PipelineTemplate.create(name="t2", content="c2", variable_declarations=[]),
        ]
        template_repo = Mock()
        template_repo.find_paginated.return_value = (templates, 2)
        result, total = make_service(template_repo=template_repo).list_templates()
        assert len(result) == 2 and total == 2

    def test_get_template_success(self):
        tmpl = PipelineTemplate.create(name="t", content="c", variable_declarations=[])
        template_repo = Mock()
        template_repo.find_by_id.return_value = tmpl
        assert make_service(template_repo=template_repo).get_template(tmpl.id) == tmpl

    def test_get_template_not_found(self):
        template_repo = Mock()
        template_repo.find_by_id.return_value = None
        with pytest.raises(BusinessError) as exc:
            make_service(template_repo=template_repo).get_template("x")
        assert exc.value.status_code == 404

    def test_create_template_with_variables(self):
        template_repo = Mock()
        var_decls = [
            {"name": "V1", "required": True, "secret": False},
            {"name": "S1", "required": True, "secret": True},
        ]
        tmpl = make_service(template_repo=template_repo).create_template(
            name="t", content="c", description="d", variable_declarations=var_decls
        )
        assert len(tmpl.variable_declarations) == 2 and tmpl.variable_declarations[1].secret is True
        template_repo.save.assert_called_once()

    def test_create_template_without_variables(self):
        template_repo = Mock()
        tmpl = make_service(template_repo=template_repo).create_template(name="t", content="c")
        assert len(tmpl.variable_declarations) == 0
        template_repo.save.assert_called_once()

    def test_update_template(self):
        tmpl = PipelineTemplate.create(name="old", content="old", variable_declarations=[])
        template_repo = Mock()
        template_repo.find_by_id.return_value = tmpl
        updated = make_service(template_repo=template_repo).update_template(
            tmpl.id,
            name="new",
            content="new content",
            variable_declarations=[{"name": "V", "required": True, "secret": False}],
        )
        assert updated.name == "new" and updated.content == "new content" and len(updated.variable_declarations) == 1
        template_repo.save.assert_called_once()

    def test_delete_template_referenced(self):
        tmpl = PipelineTemplate.create(name="t", content="c", variable_declarations=[])
        template_repo = Mock()
        template_repo.find_by_id.return_value = tmpl
        template_repo.is_referenced_by_projects.return_value = True
        with pytest.raises(BusinessError) as exc:
            make_service(template_repo=template_repo).delete_template(tmpl.id)
        assert exc.value.status_code == 409


class TestPipelineRun:
    def test_list_runs(self):
        project = Project.create(
            name="p", repository_url="https://x.git", pipeline_template_id="t", git_credential_id="c"
        )
        project_repo = Mock()
        project_repo.find_by_id.return_value = project
        run_repo = Mock()
        run_repo.find_paginated_with_filters.return_value = ([], 0)
        runs, total = make_service(project_repo=project_repo, run_repo=run_repo).list_runs(
            project_id=project.id, page=1, per_page=10
        )
        assert runs == [] and total == 0
        run_repo.find_paginated_with_filters.assert_called_once_with(page=1, per_page=10, project_id=project.id)

    def test_get_run_success(self):
        run = PipelineRun.create(
            project_id="p",
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            resolved_pipeline="v: v1",
            variables_snapshot={},
        )
        run_repo = Mock()
        run_repo.find_by_id.return_value = run
        assert make_service(run_repo=run_repo).get_run(run.id) == run

    def test_get_run_not_found(self):
        run_repo = Mock()
        run_repo.find_by_id.return_value = None
        with pytest.raises(BusinessError) as exc:
            make_service(run_repo=run_repo).get_run("x")
        assert exc.value.status_code == 404

    def test_list_artifacts(self):
        run = PipelineRun.create(
            project_id="p",
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            resolved_pipeline="v: v1",
            variables_snapshot={},
        )
        run_repo = Mock()
        run_repo.find_by_id.return_value = run
        artifact_repo = Mock()
        artifact_repo.find_by_run.return_value = []
        artifacts = make_service(run_repo=run_repo, artifact_repo=artifact_repo).list_artifacts(run.id)
        assert artifacts == []
        artifact_repo.find_by_run.assert_called_once_with(run.id)

    @pytest.mark.asyncio
    async def test_retry_pipeline_success(self):
        original = PipelineRun.create(
            project_id="p",
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            resolved_pipeline="v: v1",
            variables_snapshot={"K": "V"},
        )
        original.complete_failed()
        project = Project.create(
            name="p", repository_url="https://x.git", pipeline_template_id="t", git_credential_id="c"
        )
        run_repo = Mock()
        run_repo.find_by_id.return_value = original
        project_repo = Mock()
        project_repo.find_by_id.return_value = project
        service = make_service(project_repo=project_repo, run_repo=run_repo)
        new_run, _, _ = service.create_retry_run(original.id)
        assert new_run.retry_of == original.id and new_run.status == PipelineRunStatus.WAITING
        run_repo.save.assert_called()

    def test_retry_pipeline_invalid_status(self):
        run = PipelineRun.create(
            project_id="p",
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            resolved_pipeline="v: v1",
            variables_snapshot={},
        )
        run.start()
        run_repo = Mock()
        run_repo.find_by_id.return_value = run
        with pytest.raises(BusinessError) as exc:
            make_service(run_repo=run_repo).create_retry_run(run.id)
        assert exc.value.status_code == 400
