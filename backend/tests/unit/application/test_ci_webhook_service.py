"""CI Webhook Service 单元测试"""

from unittest.mock import AsyncMock, Mock

import pytest

from pomelo_orbit.application.ci_webhook_service import CIWebhookService
from pomelo_orbit.domain.ci.entities import Project
from pomelo_orbit.domain.ci.value_objects import PipelineRunTrigger


class TestHandleGithubWebhook:
    """测试 GitHub Webhook 处理"""

    @pytest.mark.asyncio
    async def test_handle_github_webhook_no_matching_project(self):
        """测试没有匹配项目的 webhook"""
        project_repo = Mock()
        project_repo.find_by_repository_url.return_value = []

        pipeline_service = Mock()

        service = CIWebhookService(
            project_repo=project_repo,
            pipeline_service=pipeline_service,
        )

        payload = {
            "ref": "refs/heads/main",
            "repository": {"clone_url": "https://github.com/user/repo.git"},
            "head_commit": {"id": "abc123", "author": {"name": "user"}},
        }

        result = await service.handle_github_webhook(
            payload_bytes=b"{}",
            payload=payload,
            signature="sha256=test",
        )

        assert result["status"] == "ignored"
        assert result["reason"] == "no matching project"
        pipeline_service.trigger_pipeline.assert_not_called()

    @pytest.mark.asyncio
    async def test_handle_github_webhook_signature_verification_failed(self):
        """测试签名验证失败"""
        project = Project.create(
            name="test-project",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="template-1",
            git_credential_id="cred-1",
            webhook_secret="secret123",
        )

        project_repo = Mock()
        project_repo.find_by_repository_url.return_value = [project]

        pipeline_service = Mock()

        service = CIWebhookService(
            project_repo=project_repo,
            pipeline_service=pipeline_service,
        )

        payload = {
            "ref": "refs/heads/main",
            "repository": {"clone_url": "https://github.com/user/repo.git"},
            "head_commit": {"id": "abc123", "author": {"name": "user"}},
        }

        result = await service.handle_github_webhook(
            payload_bytes=b'{"test": "data"}',
            payload=payload,
            signature="sha256=invalid_signature",
        )

        # 签名验证失败，不触发 pipeline
        assert result["status"] == "ignored"
        pipeline_service.trigger_pipeline.assert_not_called()

    @pytest.mark.asyncio
    async def test_handle_github_webhook_branch_filtered(self):
        """测试分支过滤"""
        project = Project.create(
            name="test-project",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="template-1",
            git_credential_id="cred-1",
            branch_filter="main,develop",
        )

        project_repo = Mock()
        project_repo.find_by_repository_url.return_value = [project]

        pipeline_service = Mock()

        service = CIWebhookService(
            project_repo=project_repo,
            pipeline_service=pipeline_service,
        )

        payload = {
            "ref": "refs/heads/feature",  # 不在允许列表中
            "repository": {"clone_url": "https://github.com/user/repo.git"},
            "head_commit": {"id": "abc123", "author": {"name": "user"}},
        }

        result = await service.handle_github_webhook(
            payload_bytes=b"{}",
            payload=payload,
            signature="sha256=test",
        )

        assert result["status"] == "ignored"
        assert result["reason"] == "branch filtered"
        pipeline_service.trigger_pipeline.assert_not_called()

    @pytest.mark.asyncio
    async def test_handle_github_webhook_parse_error(self):
        """测试 payload 解析错误"""
        project_repo = Mock()
        pipeline_service = Mock()

        service = CIWebhookService(
            project_repo=project_repo,
            pipeline_service=pipeline_service,
        )

        # 缺少必要字段的 payload
        payload = {"invalid": "data"}

        result = await service.handle_github_webhook(
            payload_bytes=b"{}",
            payload=payload,
            signature="sha256=test",
        )

        assert result["status"] == "ignored"
        assert "reason" in result
        pipeline_service.trigger_pipeline.assert_not_called()


class TestHandleGitlabWebhook:
    """测试 GitLab Webhook 处理"""

    @pytest.mark.asyncio
    async def test_handle_gitlab_webhook_no_matching_project(self):
        """测试没有匹配项目的 webhook"""
        project_repo = Mock()
        project_repo.find_by_repository_url.return_value = []

        pipeline_service = Mock()

        service = CIWebhookService(
            project_repo=project_repo,
            pipeline_service=pipeline_service,
        )

        payload = {
            "ref": "refs/heads/main",
            "project": {"git_http_url": "https://gitlab.com/user/repo.git"},
            "checkout_sha": "abc123",
            "user_name": "user",
        }

        result = await service.handle_gitlab_webhook(
            payload=payload,
            token="test-token",
        )

        assert result["status"] == "ignored"
        assert result["reason"] == "no matching project"
        pipeline_service.trigger_pipeline.assert_not_called()

    @pytest.mark.asyncio
    async def test_handle_gitlab_webhook_signature_verification_failed(self):
        """测试 GitLab token 验证失败"""
        project = Project.create(
            name="test-project",
            repository_url="https://gitlab.com/user/repo.git",
            pipeline_template_id="template-1",
            git_credential_id="cred-1",
            webhook_secret="correct-token",
        )

        project_repo = Mock()
        project_repo.find_by_repository_url.return_value = [project]

        pipeline_service = Mock()

        service = CIWebhookService(
            project_repo=project_repo,
            pipeline_service=pipeline_service,
        )

        payload = {
            "ref": "refs/heads/main",
            "project": {"git_http_url": "https://gitlab.com/user/repo.git"},
            "checkout_sha": "abc123",
            "user_name": "user",
        }

        result = await service.handle_gitlab_webhook(
            payload=payload,
            token="wrong-token",
        )

        # Token 验证失败，不触发 pipeline
        assert result["status"] == "ignored"
        pipeline_service.trigger_pipeline.assert_not_called()

    @pytest.mark.asyncio
    async def test_handle_gitlab_webhook_parse_error(self):
        """测试 GitLab payload 解析错误"""
        project_repo = Mock()
        pipeline_service = Mock()

        service = CIWebhookService(
            project_repo=project_repo,
            pipeline_service=pipeline_service,
        )

        # 缺少必要字段的 payload
        payload = {"invalid": "data"}

        result = await service.handle_gitlab_webhook(
            payload=payload,
            token="test-token",
        )

        assert result["status"] == "ignored"
        assert "reason" in result
        pipeline_service.trigger_pipeline.assert_not_called()


class TestHandleWebhookCore:
    """测试核心 webhook 处理逻辑"""

    @pytest.mark.asyncio
    async def test_handle_webhook_trigger_pipeline_error(self):
        """测试触发 pipeline 时发生错误"""
        project = Project.create(
            name="test-project",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="template-1",
            git_credential_id="cred-1",
        )

        project_repo = Mock()
        project_repo.find_by_repository_url.return_value = [project]

        pipeline_service = AsyncMock()
        pipeline_service.trigger_pipeline.side_effect = Exception("Database error")

        service = CIWebhookService(
            project_repo=project_repo,
            pipeline_service=pipeline_service,
        )

        payload = {
            "ref": "refs/heads/main",
            "repository": {"clone_url": "https://github.com/user/repo.git"},
            "head_commit": {"id": "abc123", "author": {"name": "user"}},
        }

        result = await service.handle_github_webhook(
            payload_bytes=b"{}",
            payload=payload,
            signature="sha256=test",
        )

        # 应该记录错误但不抛出异常
        assert result["status"] == "triggered"
        assert len(result["errors"]) == 1
        assert result["errors"][0]["project_id"] == project.id
        assert "Database error" in result["errors"][0]["error"]

    @pytest.mark.asyncio
    async def test_handle_webhook_multiple_projects(self):
        """测试多个项目匹配同一个仓库"""
        project1 = Project.create(
            name="project-1",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="template-1",
            git_credential_id="cred-1",
        )

        project2 = Project.create(
            name="project-2",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="template-2",
            git_credential_id="cred-2",
            branch_filter="main",
        )

        project_repo = Mock()
        project_repo.find_by_repository_url.return_value = [project1, project2]

        from pomelo_orbit.domain.ci.entities import PipelineRun

        run1 = PipelineRun.create(
            project_id=project1.id,
            trigger=PipelineRunTrigger.WEBHOOK,
            trigger_ref="main",
            resolved_pipeline="version: v1\nsteps: []",
            variables_snapshot={},
        )

        run2 = PipelineRun.create(
            project_id=project2.id,
            trigger=PipelineRunTrigger.WEBHOOK,
            trigger_ref="main",
            resolved_pipeline="version: v1\nsteps: []",
            variables_snapshot={},
        )

        pipeline_service = AsyncMock()
        pipeline_service.trigger_pipeline.side_effect = [run1, run2]

        service = CIWebhookService(
            project_repo=project_repo,
            pipeline_service=pipeline_service,
        )

        payload = {
            "ref": "refs/heads/main",
            "repository": {"clone_url": "https://github.com/user/repo.git"},
            "head_commit": {"id": "abc123", "author": {"name": "user"}},
        }

        result = await service.handle_github_webhook(
            payload_bytes=b"{}",
            payload=payload,
            signature="sha256=test",
        )

        # 两个项目都应该被触发
        assert result["status"] == "triggered"
        assert len(result["triggered"]) == 2
        assert result["triggered"][0]["project_id"] == project1.id
        assert result["triggered"][1]["project_id"] == project2.id
        assert pipeline_service.trigger_pipeline.call_count == 2
