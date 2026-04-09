"""Pipeline Service 单元测试"""

from datetime import datetime
from unittest.mock import Mock

import pytest

from pomelo_orbit.application.ci.pipeline_service import PipelineService
from pomelo_orbit.domain.cd.value_objects import TaskStatus
from pomelo_orbit.domain.ci.entities import (
    Credential,
    PipelineRun,
    PipelineTemplate,
    Repository,
)
from pomelo_orbit.domain.ci.value_objects import (
    CredentialType,
    PipelineRunTrigger,
)
from pomelo_orbit.domain.exceptions import BusinessError


def make_service(**overrides) -> PipelineService:
    defaults: dict = {
        "repository_repo": Mock(),
        "credential_repo": Mock(),
        "template_repo": Mock(),
        "stage_repo": Mock(),
        "snapshot_repo": Mock(),
        "run_repo": Mock(),
        "artifact_repo": Mock(),
        "stage_run_repo": Mock(),
        "webhook_repo": Mock(),
        "session_factory": Mock(),
        "executor_factory": Mock(),
        "security_service": Mock(),
        "global_variables": {},
    }
    defaults.update(overrides)
    return PipelineService(**defaults)


def make_project(**kwargs) -> Repository:
    defaults: dict = {"name": "p", "code": "p", "repository_url": "https://x.git"}
    defaults.update(kwargs)
    return Repository.create(**defaults)


class TestRepositoryCRUD:
    def test_list_repositories(self):
        repository_repo = Mock()
        repository_repo.find_paginated.return_value = ([], 0)
        service = make_service(repository_repo=repository_repo)
        projects, _, total = service.list_repository(page=1, per_page=10)
        assert projects == [] and total == 0
        repository_repo.find_paginated.assert_called_once_with(page=1, per_page=10)

    def test_get_project_success(self):
        project = make_project()
        repository_repo = Mock()
        repository_repo.find_by_id.return_value = project
        assert make_service(repository_repo=repository_repo).get_repository(project.id) == project

    def test_get_project_not_found(self):
        repository_repo = Mock()
        repository_repo.find_by_id.return_value = None
        with pytest.raises(BusinessError) as exc:
            make_service(repository_repo=repository_repo).get_repository("x")
        assert exc.value.status_code == 404

    def test_create_project_success(self):
        credential_repo = Mock()
        credential_repo.find_by_id.return_value = Credential.create(
            name="c", type=CredentialType.GIT_SSH, encrypted_data="x"
        )
        template_repo = Mock()
        template_repo.find_by_id.return_value = Mock()
        repository_repo = Mock()
        repository_repo.find_by_code.return_value = None
        service = make_service(
            repository_repo=repository_repo, credential_repo=credential_repo, template_repo=template_repo
        )
        project = service.create_project(
            name="p",
            code="p",
            repository_url="https://x.git",
            git_credential_id="c",
            default_branch="main",
        )
        assert project.name == "p"
        assert project.variable_overrides["repository_repository_url"] == "https://x.git"
        assert project.variable_overrides["repository_trigger_ref"] == "main"
        repository_repo.save.assert_called_once_with(project)

    def test_create_project_credential_not_found(self):
        credential_repo = Mock()
        credential_repo.find_by_id.return_value = None
        repository_repo = Mock()
        repository_repo.find_by_code.return_value = None
        with pytest.raises(BusinessError) as exc:
            make_service(credential_repo=credential_repo).create_project(
                name="p", code="p", repository_url="https://x.git", git_credential_id="c"
            )
        assert exc.value.status_code == 404

    def test_update_project_success(self):
        project = make_project()
        project.variable_overrides = {"USER_VAR": "value"}
        repository_repo = Mock()
        repository_repo.find_by_id.return_value = project
        service = make_service(repository_repo=repository_repo)
        updated = service.update_project(
            project.id,
            name="new",
            repository_url="https://new.git",
            default_branch="develop",
            variable_overrides={"K": "V"},
        )
        assert updated.name == "new"
        # 验证用户变量被保留和更新
        assert updated.variable_overrides["USER_VAR"] == "value"
        assert updated.variable_overrides["K"] == "V"
        # 验证内置变量自动更新
        assert updated.variable_overrides["repository_repository_url"] == "https://new.git"
        assert updated.variable_overrides["repository_trigger_ref"] == "develop"
        repository_repo.save.assert_called_once()

    def test_delete_project_success(self):
        project = make_project()
        repository_repo = Mock()
        repository_repo.find_by_id.return_value = project
        repository_repo.has_running_pipelines.return_value = False
        service = make_service(repository_repo=repository_repo)
        service.delete_project(project.id)
        repository_repo.delete.assert_called_once_with(project)

    def test_delete_project_with_running_pipelines(self):
        project = make_project()
        repository_repo = Mock()
        repository_repo.find_by_id.return_value = project
        repository_repo.has_running_pipelines.return_value = True
        with pytest.raises(BusinessError) as exc:
            make_service(repository_repo=repository_repo).delete_project(project.id)
        assert exc.value.status_code == 409


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
            PipelineTemplate.create(name="t1", variable_declarations=[]),
            PipelineTemplate.create(name="t2", variable_declarations=[]),
        ]
        template_repo = Mock()
        template_repo.find_paginated.return_value = (templates, 2)
        result, total = make_service(template_repo=template_repo).list_templates()
        assert len(result) == 2 and total == 2

    def test_get_template_success(self):
        tmpl = PipelineTemplate.create(name="t", variable_declarations=[])
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
        tmpl = make_service(template_repo=template_repo).create_template(
            name="t",
            variable_declarations=[{"name": "IMAGE", "required": True}],
        )
        assert len(tmpl.variable_declarations) == 1
        assert tmpl.variable_declarations[0].name == "IMAGE"
        template_repo.save.assert_called_once()

    def test_create_template_without_variables(self):
        template_repo = Mock()
        tmpl = make_service(template_repo=template_repo).create_template(name="t")
        assert len(tmpl.variable_declarations) == 0
        template_repo.save.assert_called_once()

    def test_update_template(self):
        tmpl = PipelineTemplate.create(name="old", variable_declarations=[])
        template_repo = Mock()
        template_repo.find_by_id.return_value = tmpl
        updated = make_service(template_repo=template_repo).update_template(
            tmpl.id,
            name="new",
        )
        assert updated.name == "new"
        template_repo.save.assert_called_once()

    def test_delete_template_referenced(self):
        tmpl = PipelineTemplate.create(name="t", variable_declarations=[])
        template_repo = Mock()
        template_repo.find_by_id.return_value = tmpl
        webhook_repo = Mock()
        webhook_repo.find_by_template.return_value = [Mock()]
        with pytest.raises(BusinessError) as exc:
            make_service(template_repo=template_repo, webhook_repo=webhook_repo).delete_template(tmpl.id)
        assert exc.value.status_code == 409


class TestPipelineRun:
    def test_list_runs(self):
        project = make_project()
        repository_repo = Mock()
        repository_repo.find_by_id.return_value = project
        run_repo = Mock()
        run_repo.find_paginated_with_filters.return_value = ([], 0)
        runs, total = make_service(repository_repo=repository_repo, run_repo=run_repo).list_runs(
            repository_id=project.id, page=1, per_page=10
        )
        assert runs == [] and total == 0

    def test_get_run_success(self):
        run = PipelineRun.create(
            repository_id="p",
            repository_name="proj",
            pipeline_snapshot_id="snap-1",
            template_id="tpl-1",
            template_name="tpl",
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
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
            repository_id="p",
            repository_name="proj",
            pipeline_snapshot_id="snap-1",
            template_id="tpl-1",
            template_name="tpl",
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            variables_snapshot={},
        )
        run_repo = Mock()
        run_repo.find_by_id.return_value = run
        artifact_repo = Mock()
        artifact_repo.find_by_run.return_value = []
        artifacts = make_service(run_repo=run_repo, artifact_repo=artifact_repo).list_artifacts(run.id)
        assert artifacts == []

    @pytest.mark.asyncio
    async def test_retry_pipeline_success(self):
        original = PipelineRun.create(
            repository_id="p",
            repository_name="proj",
            pipeline_snapshot_id="snap-1",
            template_id="tpl-1",
            template_name="tpl",
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            variables_snapshot={"K": "V"},
        )
        original.complete_failed()
        project = make_project()
        snapshot = Mock()
        snapshot.variable_declarations_snapshot = []

        run_repo = Mock()
        run_repo.find_by_id.return_value = original
        repository_repo = Mock()
        repository_repo.find_by_id.return_value = project
        snapshot_repo = Mock()
        snapshot_repo.find_by_id.return_value = snapshot

        service = make_service(repository_repo=repository_repo, run_repo=run_repo, snapshot_repo=snapshot_repo)
        new_run, _, _, snap = service.create_retry_run(original.id)
        assert new_run.retry_of == original.id and new_run.status == TaskStatus.WAITING_TO_RUN
        assert snap is snapshot
        run_repo.save.assert_called()

    def test_retry_pipeline_invalid_status(self):
        run = PipelineRun.create(
            repository_id="p",
            repository_name="proj",
            pipeline_snapshot_id="snap-1",
            template_id="tpl-1",
            template_name="tpl",
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            variables_snapshot={},
        )
        run.start()
        run_repo = Mock()
        run_repo.find_by_id.return_value = run
        with pytest.raises(BusinessError) as exc:
            make_service(run_repo=run_repo).create_retry_run(run.id)
        assert exc.value.status_code == 400


class TestWebhookCRUD:
    def test_list_webhooks(self):
        project = make_project()
        repository_repo = Mock()
        repository_repo.find_by_id.return_value = project
        webhook_repo = Mock()
        webhook_repo.find_by_repository.return_value = []
        webhooks = make_service(repository_repo=repository_repo, webhook_repo=webhook_repo).list_webhooks(project.id)
        assert webhooks == []
        webhook_repo.find_by_repository.assert_called_once_with(project.id)

    def test_get_webhook_success(self):
        from pomelo_orbit.domain.ci.entities import RepositoryWebhook

        webhook = RepositoryWebhook.create(
            repository_id="p",
            name="w",
            template_id="t",
            encrypted_secret="secret",
        )
        webhook_repo = Mock()
        webhook_repo.find_by_id.return_value = webhook
        result = make_service(webhook_repo=webhook_repo).get_webhook(webhook.id)
        assert result == webhook

    def test_get_webhook_not_found(self):
        webhook_repo = Mock()
        webhook_repo.find_by_id.return_value = None
        with pytest.raises(BusinessError) as exc:
            make_service(webhook_repo=webhook_repo).get_webhook("x")
        assert exc.value.status_code == 404

    def test_create_webhook_success(self):

        project = make_project()
        repository_repo = Mock()
        repository_repo.find_by_id.return_value = project
        template_repo = Mock()
        template_repo.find_by_id.return_value = Mock()
        webhook_repo = Mock()
        service = make_service(repository_repo=repository_repo, template_repo=template_repo, webhook_repo=webhook_repo)
        webhook = service.create_webhook(
            project.id,
            name="w",
            template_id="t",
            branch_filter="main",
            plain_secret="secret",
        )
        assert webhook.name == "w"
        assert webhook.branch_filter == "main"
        webhook_repo.save.assert_called_once()

    def test_update_webhook_success(self):
        from pomelo_orbit.domain.ci.entities import RepositoryWebhook

        webhook = RepositoryWebhook.create(
            repository_id="p",
            name="w",
            template_id="t",
            encrypted_secret="secret",
        )
        webhook_repo = Mock()
        webhook_repo.find_by_id.return_value = webhook
        service = make_service(webhook_repo=webhook_repo)
        updated = service.update_webhook(webhook.id, branch_filter="develop")
        assert updated.branch_filter == "develop"
        webhook_repo.save.assert_called_once()

    def test_delete_webhook_success(self):
        from pomelo_orbit.domain.ci.entities import RepositoryWebhook

        webhook = RepositoryWebhook.create(
            repository_id="p",
            name="w",
            template_id="t",
            encrypted_secret="secret",
        )
        webhook_repo = Mock()
        webhook_repo.find_by_id.return_value = webhook
        service = make_service(webhook_repo=webhook_repo)
        service.delete_webhook(webhook.id)
        webhook_repo.delete.assert_called_once_with(webhook)

    def test_decrypt_webhook_secret(self):
        from pomelo_orbit.domain.ci.entities import RepositoryWebhook

        webhook = RepositoryWebhook.create(
            repository_id="p",
            name="w",
            template_id="t",
            encrypted_secret="encrypted",
        )
        security_service = Mock()
        security_service.decrypt_value.return_value = "decrypted"
        webhook_repo = Mock()
        webhook_repo.find_by_id.return_value = webhook
        service = make_service(webhook_repo=webhook_repo, security_service=security_service)
        secret = service.decrypt_webhook_secret(webhook)
        assert secret == "decrypted"
        security_service.decrypt_value.assert_called_once_with("encrypted")


class TestStageCRUD:
    def test_list_stages(self):
        from pomelo_orbit.domain.ci.entities import PipelineStage

        stages = [
            PipelineStage.create(name="s1", image="alpine", script="echo"),
            PipelineStage.create(name="s2", image="golang", script="build"),
        ]
        stage_repo = Mock()
        stage_repo.find_all.return_value = stages
        result = make_service(stage_repo=stage_repo).list_stages()
        assert result == stages

    def test_get_stage_success(self):
        from pomelo_orbit.domain.ci.entities import PipelineStage

        stage = PipelineStage.create(
            name="s",
            image="alpine",
            script="echo",
        )
        stage_repo = Mock()
        stage_repo.find_by_id.return_value = stage
        result = make_service(stage_repo=stage_repo).get_stage(stage.id)
        assert result == stage

    def test_get_stage_not_found(self):
        stage_repo = Mock()
        stage_repo.find_by_id.return_value = None
        with pytest.raises(BusinessError) as exc:
            make_service(stage_repo=stage_repo).get_stage("x")
        assert exc.value.status_code == 404

    def test_create_stage_success(self):

        stage_repo = Mock()
        stage_repo.find_by_name.return_value = None
        service = make_service(stage_repo=stage_repo)
        stage = service.create_stage(
            name="s",
            image="alpine",
            script="echo",
        )
        assert stage.name == "s"
        stage_repo.save.assert_called_once()

    def test_create_stage_already_exists(self):
        from pomelo_orbit.domain.ci.entities import PipelineStage

        existing = PipelineStage.create(name="s", image="alpine", script="echo")
        stage_repo = Mock()
        stage_repo.find_by_name.return_value = existing
        service = make_service(stage_repo=stage_repo)
        with pytest.raises(BusinessError) as exc:
            service.create_stage(name="s", image="alpine", script="echo")
        assert exc.value.status_code == 409

    def test_update_stage_success(self):
        from pomelo_orbit.domain.ci.entities import PipelineStage

        stage = PipelineStage.create(
            name="s",
            image="alpine",
            script="echo",
        )
        stage_repo = Mock()
        stage_repo.find_by_id.return_value = stage
        template_repo = Mock()
        template_repo.find_all.return_value = []
        service = make_service(stage_repo=stage_repo, template_repo=template_repo)
        updated = service.update_stage(stage.id, image="new_image")
        assert updated.image == "new_image"
        stage_repo.save.assert_called_once()

    def test_delete_stage_success(self):
        from pomelo_orbit.domain.ci.entities import PipelineStage

        stage = PipelineStage.create(
            name="s",
            image="alpine",
            script="echo",
        )
        template_repo = Mock()
        stage_repo = Mock()
        stage_repo.find_by_id.return_value = stage
        stage_repo.is_referenced_by_templates.return_value = False
        service = make_service(stage_repo=stage_repo, template_repo=template_repo)
        service.delete_stage(stage.id)
        stage_repo.delete.assert_called_once_with(stage)

    def test_delete_stage_referenced(self):
        from pomelo_orbit.domain.ci.entities import PipelineStage

        stage = PipelineStage.create(
            name="s",
            image="alpine",
            script="echo",
        )
        template_repo = Mock()
        template_repo.find_by_stage.return_value = [Mock()]
        stage_repo = Mock()
        stage_repo.find_by_id.return_value = stage
        service = make_service(stage_repo=stage_repo, template_repo=template_repo)
        with pytest.raises(BusinessError) as exc:
            service.delete_stage(stage.id)
        assert exc.value.status_code == 409


class TestPipelineTrigger:
    def test_create_run_success(self):
        from pomelo_orbit.domain.ci.entities import PipelineSnapshot

        project = make_project()
        repository_repo = Mock()
        repository_repo.find_by_id.return_value = project
        template = PipelineTemplate.create(name="t", variable_declarations=[])
        template_repo = Mock()
        template_repo.find_by_id.return_value = template
        snapshot = PipelineSnapshot(
            id="snap-1",
            template_id=template.id,
            version=1,
            stages_snapshot=[],
            variable_declarations_snapshot=[],
            created_at=datetime.now(),
        )
        snapshot_repo = Mock()
        snapshot_repo.find_by_id.return_value = snapshot
        snapshot_repo.find_latest.return_value = None

        run_repo = Mock()
        executor_factory = Mock()
        executor = Mock()
        executor.execute.return_value = True
        executor_factory.create.return_value = executor

        service = make_service(
            repository_repo=repository_repo,
            template_repo=template_repo,
            snapshot_repo=snapshot_repo,
            run_repo=run_repo,
            executor_factory=executor_factory,
        )
        run, _, _, _ = service.create_run(
            project.id,
            template.id,
            PipelineRunTrigger.MANUAL,
            "main",
        )
        assert run.repository_id == project.id
        run_repo.save.assert_called_once()

    def test_cancel_run_success(self):
        from pomelo_orbit.domain.ci.entities import PipelineSnapshot

        snapshot = PipelineSnapshot(
            id="snap-1",
            template_id="t",
            version=1,
            stages_snapshot=[],
            variable_declarations_snapshot=[],
            created_at=datetime.now(),
        )
        run = PipelineRun.create(
            repository_id="p",
            repository_name="proj",
            pipeline_snapshot_id=snapshot.id,
            template_id="tpl-1",
            template_name="tpl",
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            variables_snapshot={},
        )
        run.start()
        run_repo = Mock()
        run_repo.find_by_id.return_value = run
        run_repo.commit = Mock()
        service = make_service(run_repo=run_repo)
        service.cancel_run(run.id)
        assert run.status == TaskStatus.CANCELED
        run_repo.save.assert_called_once()
        run_repo.commit.assert_called_once()

    def test_cancel_run_invalid_status(self):
        from pomelo_orbit.domain.ci.entities import PipelineSnapshot

        snapshot = PipelineSnapshot(
            id="snap-1",
            template_id="t",
            version=1,
            stages_snapshot=[],
            variable_declarations_snapshot=[],
            created_at=datetime.now(),
        )
        run = PipelineRun.create(
            repository_id="p",
            repository_name="proj",
            pipeline_snapshot_id=snapshot.id,
            template_id="tpl-1",
            template_name="tpl",
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            variables_snapshot={},
        )
        run.complete_success()
        run_repo = Mock()
        run_repo.find_by_id.return_value = run
        run_repo.commit = Mock()
        service = make_service(run_repo=run_repo)
        with pytest.raises(BusinessError) as exc:
            service.cancel_run(run.id)
        assert exc.value.status_code == 400


class TestSnapshot:
    def test_get_snapshot_success(self):
        from pomelo_orbit.domain.ci.entities import PipelineSnapshot

        snapshot = PipelineSnapshot(
            id="snap-1",
            template_id="t",
            version=1,
            stages_snapshot=[],
            variable_declarations_snapshot=[],
            created_at=datetime.now(),
        )
        snapshot_repo = Mock()
        snapshot_repo.find_by_id.return_value = snapshot
        service = make_service(snapshot_repo=snapshot_repo)
        result = service.get_snapshot(snapshot.id)
        assert result == snapshot

    def test_get_snapshot_not_found(self):
        snapshot_repo = Mock()
        snapshot_repo.find_by_id.return_value = None
        with pytest.raises(BusinessError) as exc:
            make_service(snapshot_repo=snapshot_repo).get_snapshot("x")
        assert exc.value.status_code == 404
