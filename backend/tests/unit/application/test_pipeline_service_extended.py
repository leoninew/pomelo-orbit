"""Pipeline Service 扩展单元测试 - 覆盖更多功能"""

from unittest.mock import Mock

import pytest

from pomelo_orbit.application.pipeline_service import PipelineService
from pomelo_orbit.domain.ci.entities import (
    Credential,
    PipelineTemplate,
    Project,
)
from pomelo_orbit.domain.ci.value_objects import CredentialType
from pomelo_orbit.domain.exceptions import BusinessError


class TestProjectWebhook:
    """测试 Project Webhook 相关功能"""

    def test_get_webhook_config(self):
        """测试获取 webhook 配置"""
        project = Project.create(
            name="test-project",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="template-1",
            git_credential_id="cred-1",
        )
        project.webhook_secret = "secret123"

        project_repo = Mock()
        project_repo.find_by_id.return_value = project

        service = PipelineService(
            project_repo=project_repo,
            credential_repo=Mock(),
            template_repo=Mock(),
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        config = service.get_webhook_config("project-1", "https://api.example.com")

        assert config["url"] == "https://api.example.com/api/v1/ci/webhooks/git"
        assert config["secret"] == "secret123"
        assert "push" in config["events"]
        assert "release" in config["events"]

    def test_regenerate_webhook_secret(self):
        """测试重新生成 webhook secret"""
        project = Project.create(
            name="test-project",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="template-1",
            git_credential_id="cred-1",
        )
        old_secret = project.webhook_secret

        project_repo = Mock()
        project_repo.find_by_id.return_value = project

        service = PipelineService(
            project_repo=project_repo,
            credential_repo=Mock(),
            template_repo=Mock(),
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        updated_project = service.regenerate_webhook_secret("project-1")

        assert updated_project.webhook_secret != old_secret
        assert len(updated_project.webhook_secret) > 0
        project_repo.save.assert_called_once()


class TestTemplateUpdate:
    """测试 Template 更新功能"""

    def test_update_template_name_only(self):
        """测试只更新模板名称"""
        template = PipelineTemplate.create(
            name="old-name",
            content="version: '1.0'\nsteps: []",
            variable_declarations=[],
        )

        template_repo = Mock()
        template_repo.find_by_id.return_value = template

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=Mock(),
            template_repo=template_repo,
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        updated = service.update_template("template-1", name="new-name")

        assert updated.name == "new-name"
        template_repo.save.assert_called_once()

    def test_update_template_content(self):
        """测试更新模板内容"""
        template = PipelineTemplate.create(
            name="template",
            content="version: '1.0'\nsteps: []",
            variable_declarations=[],
        )

        template_repo = Mock()
        template_repo.find_by_id.return_value = template

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=Mock(),
            template_repo=template_repo,
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        new_content = "version: '1.0'\nsteps:\n  - name: build"
        updated = service.update_template("template-1", content=new_content)

        assert updated.content == new_content
        template_repo.save.assert_called_once()

    def test_update_template_variable_declarations(self):
        """测试更新模板变量声明"""
        template = PipelineTemplate.create(
            name="template",
            content="version: '1.0'\nsteps: []",
            variable_declarations=[],
        )

        template_repo = Mock()
        template_repo.find_by_id.return_value = template

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=Mock(),
            template_repo=template_repo,
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        new_vars = [
            {"name": "VAR1", "required": True, "secret": False},
            {"name": "VAR2", "required": False, "secret": True},
        ]
        updated = service.update_template("template-1", variable_declarations=new_vars)

        assert len(updated.variable_declarations) == 2
        assert updated.variable_declarations[0].name == "VAR1"
        assert updated.variable_declarations[1].secret is True
        template_repo.save.assert_called_once()

    def test_update_template_all_fields(self):
        """测试更新模板所有字段"""
        template = PipelineTemplate.create(
            name="old-name",
            content="old content",
            variable_declarations=[],
            description="old description",
        )

        template_repo = Mock()
        template_repo.find_by_id.return_value = template

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=Mock(),
            template_repo=template_repo,
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        new_vars = [{"name": "VAR1", "required": True, "secret": False}]
        updated = service.update_template(
            "template-1",
            name="new-name",
            description="new description",
            content="new content",
            variable_declarations=new_vars,
        )

        assert updated.name == "new-name"
        assert updated.description == "new description"
        assert updated.content == "new content"
        assert len(updated.variable_declarations) == 1
        template_repo.save.assert_called_once()


class TestProjectUpdate:
    """测试 Project 更新功能"""

    def test_update_project_name(self):
        """测试更新项目名称"""
        project = Project.create(
            name="old-name",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="template-1",
            git_credential_id="cred-1",
        )

        project_repo = Mock()
        project_repo.find_by_id.return_value = project
        template_repo = Mock()

        service = PipelineService(
            project_repo=project_repo,
            credential_repo=Mock(),
            template_repo=template_repo,
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        updated = service.update_project("project-1", name="new-name")

        assert updated.name == "new-name"
        project_repo.save.assert_called_once()

    def test_update_project_template(self):
        """测试更新项目模板"""
        project = Project.create(
            name="project",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="template-1",
            git_credential_id="cred-1",
        )

        project_repo = Mock()
        project_repo.find_by_id.return_value = project
        template_repo = Mock()
        template_repo.find_by_id.return_value = Mock()  # 新模板存在

        service = PipelineService(
            project_repo=project_repo,
            credential_repo=Mock(),
            template_repo=template_repo,
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        updated = service.update_project("project-1", pipeline_template_id="template-2")

        assert updated.pipeline_template_id == "template-2"
        project_repo.save.assert_called_once()

    def test_update_project_template_not_found(self):
        """测试更新项目模板时模板不存在"""
        project = Project.create(
            name="project",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="template-1",
            git_credential_id="cred-1",
        )

        project_repo = Mock()
        project_repo.find_by_id.return_value = project
        template_repo = Mock()
        template_repo.find_by_id.return_value = None  # 模板不存在

        service = PipelineService(
            project_repo=project_repo,
            credential_repo=Mock(),
            template_repo=template_repo,
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        with pytest.raises(BusinessError) as exc_info:
            service.update_project("project-1", pipeline_template_id="nonexistent")

        assert "not found" in str(exc_info.value).lower()
        assert exc_info.value.status_code == 404

    def test_update_project_variable_overrides(self):
        """测试更新项目变量覆盖"""
        project = Project.create(
            name="project",
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

        new_vars = {"VAR1": "value1", "VAR2": "value2"}
        updated = service.update_project("project-1", variable_overrides=new_vars)

        assert updated.variable_overrides == new_vars
        project_repo.save.assert_called_once()

    def test_update_project_branch_filter(self):
        """测试更新项目分支过滤"""
        project = Project.create(
            name="project",
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

        updated = service.update_project("project-1", branch_filter="main,develop")

        assert updated.branch_filter == "main,develop"
        project_repo.save.assert_called_once()


class TestCredentialOperations:
    """测试 Credential 操作"""

    def test_list_credentials(self):
        """测试列出所有凭据"""
        creds = [
            Credential.create(name="cred1", type=CredentialType.GIT_SSH, encrypted_data="data1"),
            Credential.create(name="cred2", type=CredentialType.GIT_TOKEN, encrypted_data="data2"),
        ]

        credential_repo = Mock()
        credential_repo.find_all.return_value = creds

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=credential_repo,
            template_repo=Mock(),
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        result = service.list_credentials()

        assert len(result) == 2
        assert result[0].name == "cred1"
        assert result[1].name == "cred2"
        assert result[0].type == CredentialType.GIT_SSH
        assert result[1].type == CredentialType.GIT_TOKEN

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

        result = service.get_credential("cred-1")

        assert result.name == "test-cred"
        assert result.type == CredentialType.GIT_SSH

    def test_get_credential_not_found(self):
        """测试获取不存在的凭据"""
        credential_repo = Mock()
        credential_repo.find_by_id.return_value = None

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=credential_repo,
            template_repo=Mock(),
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        with pytest.raises(BusinessError) as exc_info:
            service.get_credential("nonexistent")

        assert "not found" in str(exc_info.value).lower()
        assert exc_info.value.status_code == 404

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
            encrypted_data="encrypted_data",
        )

        assert cred.name == "new-cred"
        assert cred.type == CredentialType.GIT_SSH
        credential_repo.save.assert_called_once()


class TestTemplateOperations:
    """测试 Template 操作"""

    def test_list_templates(self):
        """测试列出所有模板"""
        templates = [
            PipelineTemplate.create(name="tmpl1", content="content1", variable_declarations=[]),
            PipelineTemplate.create(name="tmpl2", content="content2", variable_declarations=[]),
        ]

        template_repo = Mock()
        template_repo.find_all.return_value = templates

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=Mock(),
            template_repo=template_repo,
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        result = service.list_templates()

        assert len(result) == 2
        assert result[0].name == "tmpl1"
        assert result[1].name == "tmpl2"

    def test_get_template_success(self):
        """测试获取模板成功"""
        template = PipelineTemplate.create(
            name="test-template",
            content="version: '1.0'\nsteps: []",
            variable_declarations=[],
        )

        template_repo = Mock()
        template_repo.find_by_id.return_value = template

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=Mock(),
            template_repo=template_repo,
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        result = service.get_template("template-1")

        assert result.name == "test-template"

    def test_get_template_not_found(self):
        """测试获取不存在的模板"""
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
            service.get_template("nonexistent")

        assert "not found" in str(exc_info.value).lower()
        assert exc_info.value.status_code == 404

    def test_create_template_with_variables(self):
        """测试创建带变量声明的模板"""
        template_repo = Mock()

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=Mock(),
            template_repo=template_repo,
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        var_decls = [
            {"name": "VAR1", "required": True, "secret": False},
            {"name": "SECRET1", "required": True, "secret": True},
        ]

        template = service.create_template(
            name="new-template",
            content="version: '1.0'\nsteps: []",
            description="Test template",
            variable_declarations=var_decls,
        )

        assert template.name == "new-template"
        assert template.description == "Test template"
        assert len(template.variable_declarations) == 2
        assert template.variable_declarations[0].name == "VAR1"
        assert template.variable_declarations[1].secret is True
        template_repo.save.assert_called_once()

    def test_create_template_without_variables(self):
        """测试创建不带变量声明的模板"""
        template_repo = Mock()

        service = PipelineService(
            project_repo=Mock(),
            credential_repo=Mock(),
            template_repo=template_repo,
            run_repo=Mock(),
            artifact_repo=Mock(),
        )

        template = service.create_template(
            name="simple-template",
            content="version: '1.0'\nsteps: []",
        )

        assert template.name == "simple-template"
        assert len(template.variable_declarations) == 0
        template_repo.save.assert_called_once()
