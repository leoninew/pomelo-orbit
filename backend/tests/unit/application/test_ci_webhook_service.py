"""CI Webhook Service 单元测试"""

from unittest.mock import Mock

from pomelo_orbit.application.ci_webhook_service import CIWebhookService
from pomelo_orbit.domain.ci.entities import PipelineRun, Project
from pomelo_orbit.domain.ci.value_objects import PipelineRunTrigger


def make_service(project_repo=None, pipeline_service=None):
    return CIWebhookService(
        project_repo=project_repo or Mock(),
        pipeline_service=pipeline_service or Mock(),
    )


class TestHandleGithubWebhook:
    def test_no_matching_project(self):
        project_repo = Mock()
        project_repo.find_by_repository_url.return_value = []
        service = make_service(project_repo=project_repo)

        result = service.handle_github_webhook(
            payload_bytes=b"{}",
            payload={
                "ref": "refs/heads/main",
                "repository": {"clone_url": "https://github.com/user/repo.git"},
                "head_commit": {"id": "abc123", "author": {"name": "user"}},
            },
            signature="sha256=test",
        )

        assert result["status"] == "ignored"
        assert result["reason"] == "no matching project"

    def test_signature_verification_failed(self):
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
        service = make_service(project_repo=project_repo, pipeline_service=pipeline_service)

        result = service.handle_github_webhook(
            payload_bytes=b'{"test": "data"}',
            payload={
                "ref": "refs/heads/main",
                "repository": {"clone_url": "https://github.com/user/repo.git"},
                "head_commit": {"id": "abc123", "author": {"name": "user"}},
            },
            signature="sha256=invalid_signature",
        )

        assert result["status"] == "ignored"
        pipeline_service.create_run.assert_not_called()

    def test_branch_filtered(self):
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
        service = make_service(project_repo=project_repo, pipeline_service=pipeline_service)

        result = service.handle_github_webhook(
            payload_bytes=b"{}",
            payload={
                "ref": "refs/heads/feature",
                "repository": {"clone_url": "https://github.com/user/repo.git"},
                "head_commit": {"id": "abc123", "author": {"name": "user"}},
            },
            signature="sha256=test",
        )

        assert result["status"] == "ignored"
        assert result["reason"] == "branch filtered"
        pipeline_service.create_run.assert_not_called()

    def test_parse_error(self):
        service = make_service()
        result = service.handle_github_webhook(
            payload_bytes=b"{}",
            payload={"invalid": "data"},
            signature="sha256=test",
        )
        assert result["status"] == "ignored"
        assert "reason" in result


class TestHandleGitlabWebhook:
    def test_no_matching_project(self):
        project_repo = Mock()
        project_repo.find_by_repository_url.return_value = []
        service = make_service(project_repo=project_repo)

        result = service.handle_gitlab_webhook(
            payload={
                "ref": "refs/heads/main",
                "project": {"git_http_url": "https://gitlab.com/user/repo.git"},
                "checkout_sha": "abc123",
                "user_name": "user",
            },
            token="test-token",
        )

        assert result["status"] == "ignored"
        assert result["reason"] == "no matching project"

    def test_signature_verification_failed(self):
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
        service = make_service(project_repo=project_repo, pipeline_service=pipeline_service)

        result = service.handle_gitlab_webhook(
            payload={
                "ref": "refs/heads/main",
                "project": {"git_http_url": "https://gitlab.com/user/repo.git"},
                "checkout_sha": "abc123",
                "user_name": "user",
            },
            token="wrong-token",
        )

        assert result["status"] == "ignored"
        pipeline_service.create_run.assert_not_called()

    def test_parse_error(self):
        service = make_service()
        result = service.handle_gitlab_webhook(payload={"invalid": "data"}, token="test-token")
        assert result["status"] == "ignored"
        assert "reason" in result


class TestHandleWebhookCore:
    def test_trigger_pipeline_error(self):
        project = Project.create(
            name="test-project",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="template-1",
            git_credential_id="cred-1",
        )
        project_repo = Mock()
        project_repo.find_by_repository_url.return_value = [project]
        pipeline_service = Mock()
        pipeline_service.create_run.side_effect = Exception("Database error")
        service = make_service(project_repo=project_repo, pipeline_service=pipeline_service)

        result = service.handle_github_webhook(
            payload_bytes=b"{}",
            payload={
                "ref": "refs/heads/main",
                "repository": {"clone_url": "https://github.com/user/repo.git"},
                "head_commit": {"id": "abc123", "author": {"name": "user"}},
            },
            signature="sha256=test",
        )

        assert result["status"] == "triggered"
        assert len(result["errors"]) == 1
        assert "Database error" in result["errors"][0]["error"]

    def test_multiple_projects(self):
        project1 = Project.create(
            name="p1",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="t1",
            git_credential_id="c1",
        )
        project2 = Project.create(
            name="p2",
            repository_url="https://github.com/user/repo.git",
            pipeline_template_id="t2",
            git_credential_id="c2",
            branch_filter="main",
        )
        project_repo = Mock()
        project_repo.find_by_repository_url.return_value = [project1, project2]

        run1 = PipelineRun.create(
            project_id=project1.id,
            trigger=PipelineRunTrigger.WEBHOOK,
            trigger_ref="main",
            resolved_pipeline="v: v1",
            variables_snapshot={},
        )
        run2 = PipelineRun.create(
            project_id=project2.id,
            trigger=PipelineRunTrigger.WEBHOOK,
            trigger_ref="main",
            resolved_pipeline="v: v1",
            variables_snapshot={},
        )

        pipeline_service = Mock()
        pipeline_service.create_run.side_effect = [
            (run1, project1, {}),
            (run2, project2, {}),
        ]
        service = make_service(project_repo=project_repo, pipeline_service=pipeline_service)

        result = service.handle_github_webhook(
            payload_bytes=b"{}",
            payload={
                "ref": "refs/heads/main",
                "repository": {"clone_url": "https://github.com/user/repo.git"},
                "head_commit": {"id": "abc123", "author": {"name": "user"}},
            },
            signature="sha256=test",
        )

        assert result["status"] == "triggered"
        assert len(result["triggered"]) == 2
        assert pipeline_service.create_run.call_count == 2
