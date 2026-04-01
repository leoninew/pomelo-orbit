"""Pipeline Service 单元测试"""

from unittest.mock import AsyncMock, Mock, patch

import pytest

from pomelo_orbit.application.pipeline_service import PipelineService
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


class TestProjectCRUD:
    """测试 Project CRUD 操作"""

    def test_list_projects(self):
        """测试列出项目"""
        project_repo = Mock()
        project_repo.find_paginated.return_value = ([], 0)

        service = PipelineService(
            project_repo=project_repo,
            credential_repo=Mock(),
            template_repo=Mock(),
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        projects, total = service.list_projects(page=1, per_page=10)

        assert projects == []
        assert total == 0
        project_repo.find_paginated.assert_called_once_with(page=1, per_page=10)

    def test_get_project_success(self):
        """测试获取项目成功"""
        project = Project.create(
            name="test-project",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="template-1",
            git_credential_id="cred-1",
        )
        project_repo = Mock()
        project_repo.find_by_id.return_value = project

        service = PipelineService(
            project_repo=project_repo,
            credential_repo=Mock(),
            template_repo=Mock(),
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        result = service.get_project(project.id)

        assert result == project
        project_repo.find_by_id.assert_called_once_with(project.id)

    def test_get_project_not_found(self):
        """测试获取不存在的项目"""
        project_repo = Mock()
        project_repo.find_by_id.return_value = None

        service = PipelineService(
            project_repo=project_repo,
            credential_repo=Mock(),
            template_repo=Mock(),
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        with pytest.raises(BusinessError) as exc_info:
            service.get_project("nonexistent-id")

        assert exc_info.value.status_code == 404
        assert "not found" in str(exc_info.value)

    def test_create_project_success(self):
        """测试创建项目成功"""
        template_repo = Mock()
        template_repo.find_by_id.return_value = PipelineTemplate.create(
            name="test-template",
            content="version: v1\nsteps: []",
            variable_declarations=[],
        )

        credential_repo = Mock()
        credential_repo.find_by_id.return_value = Credential.create(
            name="test-cred",
            type=CredentialType.GIT_SSH,
            encrypted_data="encrypted",
        )

        project_repo = Mock()

        service = PipelineService(
            project_repo=project_repo,
            credential_repo=credential_repo,
            template_repo=template_repo,
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        project = service.create_project(
            name="new-project",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="template-1",
            git_credential_id="cred-1",
            variable_overrides={"VAR1": "value1"},
            branch_filter="main,develop",
            enable_webhook=True,
        )

        assert project.name == "new-project"
        assert project.webhook_secret is not None
        assert project.branch_filter == "main,develop"
        project_repo.save.assert_called_once_with(project)

    def test_create_project_template_not_found(self):
        """测试创建项目时模板不存在"""
        template_repo = Mock()
        template_repo.find_by_id.return_value = None

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=Mock(),
            template_repo=template_repo,
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        with pytest.raises(BusinessError) as exc_info:
            service.create_project(
                name="new-project",
                repository_url="https://github.com/user/repo.git",
                pipeline_template_id="nonexistent-template",
                git_credential_id="cred-1",
            )

        assert exc_info.value.status_code == 404
        assert "PipelineTemplate" in str(exc_info.value)

    def test_create_project_credential_not_found(self):
        """测试创建项目时凭据不存在"""
        template_repo = Mock()
        template_repo.find_by_id.return_value = PipelineTemplate.create(
            name="test-template",
            content="version: v1\nsteps: []",
            variable_declarations=[],
        )

        credential_repo = Mock()
        credential_repo.find_by_id.return_value = None

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=credential_repo,
            template_repo=template_repo,
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        with pytest.raises(BusinessError) as exc_info:
            service.create_project(
                name="new-project",
                repository_url="https://github.com/user/repo.git",
                pipeline_template_id="template-1",
                git_credential_id="nonexistent-cred",
            )

        assert exc_info.value.status_code == 404
        assert "Credential" in str(exc_info.value)

    def test_update_project_success(self):
        """测试更新项目成功"""
        project = Project.create(
            name="old-name",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="template-1",
            git_credential_id="cred-1",
        )

        project_repo = Mock()
        project_repo.find_by_id.return_value = project

        template_repo = Mock()
        template_repo.find_by_id.return_value = PipelineTemplate.create(
            name="new-template",
            content="version: v1\nsteps: []",
            variable_declarations=[],
        )

        service = PipelineService(
            project_repo=project_repo,
            credential_repo=Mock(),
            template_repo=template_repo,
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        updated = service.update_project(
            project_id=project.id,
            name="new-name",
            variable_overrides={"VAR1": "new-value"},
            pipeline_template_id="template-2",
            branch_filter="main",
        )

        assert updated.name == "new-name"
        assert updated.variable_overrides == {"VAR1": "new-value"}
        assert updated.branch_filter == "main"
        project_repo.save.assert_called_once_with(project)

    def test_delete_project_success(self):
        """测试删除项目成功"""
        project = Project.create(
            name="test-project",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="template-1",
            git_credential_id="cred-1",
        )

        project_repo = Mock()
        project_repo.find_by_id.return_value = project
        project_repo.has_running_pipelines.return_value = False

        service = PipelineService(
            project_repo=project_repo,
            credential_repo=Mock(),
            template_repo=Mock(),
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        service.delete_project(project.id)

        project_repo.delete.assert_called_once_with(project)

    def test_delete_project_with_running_pipelines(self):
        """测试删除有运行中 pipeline 的项目"""
        project = Project.create(
            name="test-project",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="template-1",
            git_credential_id="cred-1",
        )

        project_repo = Mock()
        project_repo.find_by_id.return_value = project
        project_repo.has_running_pipelines.return_value = True

        service = PipelineService(
            project_repo=project_repo,
            credential_repo=Mock(),
            template_repo=Mock(),
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        with pytest.raises(BusinessError) as exc_info:
            service.delete_project(project.id)

        assert exc_info.value.status_code == 409
        assert "running pipelines" in str(exc_info.value)


class TestCredentialCRUD:
    """测试 Credential CRUD 操作"""

    def test_list_credentials(self):
        """测试列出凭据"""
        credential_repo = Mock()
        credential_repo.find_all.return_value = []

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=credential_repo,
            template_repo=Mock(),
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        credentials = service.list_credentials()

        assert credentials == []
        credential_repo.find_all.assert_called_once()

    def test_get_credential_success(self):
        """测试获取凭据成功"""
        cred = Credential.create(
            name="test-cred",
            type=CredentialType.GIT_SSH,
            encrypted_data="encrypted",
        )

        credential_repo = Mock()
        credential_repo.find_by_id.return_value = cred

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=credential_repo,
            template_repo=Mock(),
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        result = service.get_credential(cred.id)

        assert result == cred

    def test_create_credential(self):
        """测试创建凭据"""
        credential_repo = Mock()

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=credential_repo,
            template_repo=Mock(),
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        cred = service.create_credential(
            name="new-cred",
            credential_type="git_ssh",
            encrypted_data="encrypted-data",
        )

        assert cred.name == "new-cred"
        assert cred.type == CredentialType.GIT_SSH
        credential_repo.save.assert_called_once_with(cred)

    def test_delete_credential_success(self):
        """测试删除凭据成功"""
        cred = Credential.create(
            name="test-cred",
            type=CredentialType.GIT_SSH,
            encrypted_data="encrypted",
        )

        credential_repo = Mock()
        credential_repo.find_by_id.return_value = cred
        credential_repo.is_referenced_by_projects.return_value = False

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=credential_repo,
            template_repo=Mock(),
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        service.delete_credential(cred.id)

        credential_repo.delete.assert_called_once_with(cred)

    def test_delete_credential_referenced_by_projects(self):
        """测试删除被项目引用的凭据"""
        cred = Credential.create(
            name="test-cred",
            type=CredentialType.GIT_SSH,
            encrypted_data="encrypted",
        )

        credential_repo = Mock()
        credential_repo.find_by_id.return_value = cred
        credential_repo.is_referenced_by_projects.return_value = True

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=credential_repo,
            template_repo=Mock(),
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        with pytest.raises(BusinessError) as exc_info:
            service.delete_credential(cred.id)

        assert exc_info.value.status_code == 409
        assert "referenced by projects" in str(exc_info.value)


class TestTemplateCRUD:
    """测试 Template CRUD 操作"""

    def test_list_templates(self):
        """测试列出模板"""
        template_repo = Mock()
        template_repo.find_all.return_value = []

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=Mock(),
            template_repo=template_repo,
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        templates = service.list_templates()

        assert templates == []
        template_repo.find_all.assert_called_once()

    def test_create_template(self):
        """测试创建模板"""
        template_repo = Mock()

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=Mock(),
            template_repo=template_repo,
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        template = service.create_template(
            name="new-template",
            content="version: v1\nsteps: []",
            description="Test template",
            variable_declarations=[
                {"name": "VAR1", "description": "Variable 1", "required": True}
            ],
        )

        assert template.name == "new-template"
        assert len(template.variable_declarations) == 1
        template_repo.save.assert_called_once_with(template)

    def test_delete_template_referenced_by_projects(self):
        """测试删除被项目引用的模板"""
        template = PipelineTemplate.create(
            name="test-template",
            content="version: v1\nsteps: []",
            variable_declarations=[],
        )

        template_repo = Mock()
        template_repo.find_by_id.return_value = template
        template_repo.is_referenced_by_projects.return_value = True

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=Mock(),
            template_repo=template_repo,
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        with pytest.raises(BusinessError) as exc_info:
            service.delete_template(template.id)

        assert exc_info.value.status_code == 409
        assert "referenced by projects" in str(exc_info.value)


class TestPipelineRun:
    """测试 Pipeline Run 操作"""

    def test_list_runs(self):
        """测试列出 runs"""
        project = Project.create(
            name="test-project",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="template-1",
            git_credential_id="cred-1",
        )

        project_repo = Mock()
        project_repo.find_by_id.return_value = project

        run_repo = Mock()
        run_repo.find_by_project.return_value = ([], 0)

        service = PipelineService(
            project_repo=project_repo,
            credential_repo=Mock(),
            template_repo=Mock(),
            run_repo=run_repo,
            artifact_repo=Mock(),
        )

        runs, total = service.list_runs(project.id, page=1, per_page=10)

        assert runs == []
        assert total == 0
        run_repo.find_by_project.assert_called_once_with(project.id, page=1, per_page=10)

    def test_get_run_success(self):
        """测试获取 run 成功"""
        run = PipelineRun.create(
            project_id="project-1",
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            resolved_pipeline="version: v1\nsteps: []",
            variables_snapshot={},
        )

        run_repo = Mock()
        run_repo.find_by_id.return_value = run

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=Mock(),
            template_repo=Mock(),
            run_repo=run_repo,
            artifact_repo=Mock(),
        )

        result = service.get_run(run.id)

        assert result == run

    def test_get_run_not_found(self):
        """测试获取不存在的 run"""
        run_repo = Mock()
        run_repo.find_by_id.return_value = None

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=Mock(),
            template_repo=Mock(),
            run_repo=run_repo,
            artifact_repo=Mock(),
        )

        with pytest.raises(BusinessError) as exc_info:
            service.get_run("nonexistent-id")

        assert exc_info.value.status_code == 404

    def test_list_artifacts(self):
        """测试列出制品"""
        run = PipelineRun.create(
            project_id="project-1",
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            resolved_pipeline="version: v1\nsteps: []",
            variables_snapshot={},
        )

        run_repo = Mock()
        run_repo.find_by_id.return_value = run

        artifact_repo = Mock()
        artifact_repo.find_by_run.return_value = []

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=Mock(),
            template_repo=Mock(),
            run_repo=run_repo,
            artifact_repo=artifact_repo,
        )

        artifacts = service.list_artifacts(run.id)

        assert artifacts == []
        artifact_repo.find_by_run.assert_called_once_with(run.id)

    @pytest.mark.asyncio
    async def test_retry_pipeline_success(self):
        """测试重试 pipeline 成功"""
        original_run = PipelineRun.create(
            project_id="project-1",
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            resolved_pipeline="version: v1\nsteps: []",
            variables_snapshot={"VAR1": "value1"},
        )
        original_run.complete_failed()

        project = Project.create(
            name="test-project",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="template-1",
            git_credential_id="cred-1",
        )

        run_repo = Mock()
        run_repo.find_by_id.return_value = original_run

        project_repo = Mock()
        project_repo.find_by_id.return_value = project

        service = PipelineService(
            project_repo=project_repo,
            credential_repo=Mock(),
            template_repo=Mock(),
            run_repo=run_repo,
            artifact_repo=Mock(),
        )

        with patch.object(service, "_execute_run", new_callable=AsyncMock):
            new_run = await service.retry_pipeline(original_run.id)

            assert new_run.retry_of == original_run.id
            assert new_run.project_id == original_run.project_id
            assert new_run.status == PipelineRunStatus.WAITING
            run_repo.save.assert_called()

    @pytest.mark.asyncio
    async def test_retry_pipeline_invalid_status(self):
        """测试重试运行中的 pipeline"""
        running_run = PipelineRun.create(
            project_id="project-1",
            trigger=PipelineRunTrigger.MANUAL,
            trigger_ref="main",
            resolved_pipeline="version: v1\nsteps: []",
            variables_snapshot={},
        )
        running_run.start()

        run_repo = Mock()
        run_repo.find_by_id.return_value = running_run

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=Mock(),
            template_repo=Mock(),
            run_repo=run_repo,
            artifact_repo=Mock(),
        )

        with pytest.raises(BusinessError) as exc_info:
            await service.retry_pipeline(running_run.id)

        assert exc_info.value.status_code == 400
        assert "Cannot retry" in str(exc_info.value)
