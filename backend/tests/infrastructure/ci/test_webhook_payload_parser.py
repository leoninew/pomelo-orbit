"""CI Webhook Payload 解析器测试"""

import pytest

from pomelo_orbit.infrastructure.ci.webhook_payload_parser import (
    WebhookPayloadParseError,
    parse_github_webhook,
    parse_gitlab_webhook,
)


class TestParseGitHubWebhook:
    """GitHub webhook payload 解析测试"""

    def test_parse_push_event(self):
        """解析 push 事件"""
        payload = {
            "ref": "refs/heads/main",
            "after": "abc123",
            "repository": {
                "clone_url": "https://github.com/owner/repo.git",
                "html_url": "https://github.com/owner/repo",
            },
            "sender": {"login": "octocat"},
        }
        result = parse_github_webhook(payload)

        assert result.repository_url == "https://github.com/owner/repo.git"
        assert result.branch == "main"
        assert result.commit_sha == "abc123"
        assert result.author == "octocat"
        assert result.event_type == "push"

    def test_parse_tag_event(self):
        """解析 tag push 事件"""
        payload = {
            "ref": "refs/tags/v1.0.0",
            "after": "def456",
            "repository": {
                "clone_url": "https://github.com/owner/repo.git",
            },
            "sender": {"login": "octocat"},
        }
        result = parse_github_webhook(payload)

        assert result.branch == "v1.0.0"
        assert result.event_type == "tag"

    def test_parse_release_event(self):
        """解析 release 事件"""
        payload = {
            "ref": "",
            "release": {
                "target_commitish": "main",
                "tag_name": "v1.0.0",
            },
            "after": "ghi789",
            "repository": {
                "clone_url": "https://github.com/owner/repo.git",
            },
            "sender": {"login": "octocat"},
        }
        result = parse_github_webhook(payload)

        assert result.branch == "main"
        assert result.event_type == "release"

    def test_parse_uses_pusher_as_fallback_author(self):
        """sender 不存在时使用 pusher"""
        payload = {
            "ref": "refs/heads/main",
            "after": "abc123",
            "repository": {"clone_url": "https://github.com/owner/repo.git"},
            "pusher": {"name": "pusher-user"},
        }
        result = parse_github_webhook(payload)
        assert result.author == "pusher-user"

    def test_parse_missing_repository_url_raises(self):
        """缺少 repository URL 时抛出异常"""
        payload = {
            "ref": "refs/heads/main",
            "repository": {},
        }
        with pytest.raises(WebhookPayloadParseError):
            parse_github_webhook(payload)

    def test_parse_uses_head_commit_sha(self):
        """使用 head_commit.id 作为 commit sha"""
        payload = {
            "ref": "refs/heads/main",
            "head_commit": {"id": "headsha"},
            "repository": {"clone_url": "https://github.com/owner/repo.git"},
        }
        result = parse_github_webhook(payload)
        assert result.commit_sha == "headsha"


class TestParseGitLabWebhook:
    """GitLab webhook payload 解析测试"""

    def test_parse_push_event(self):
        """解析 push 事件"""
        payload = {
            "object_kind": "push",
            "ref": "refs/heads/develop",
            "after": "abc123",
            "project": {
                "http_url": "https://gitlab.com/owner/repo.git",
            },
            "user_name": "gitlab-user",
        }
        result = parse_gitlab_webhook(payload)

        assert result.repository_url == "https://gitlab.com/owner/repo.git"
        assert result.branch == "develop"
        assert result.commit_sha == "abc123"
        assert result.author == "gitlab-user"
        assert result.event_type == "push"

    def test_parse_tag_push_event(self):
        """解析 tag_push 事件"""
        payload = {
            "object_kind": "tag_push",
            "ref": "refs/tags/v2.0.0",
            "after": "tagsha",
            "project": {"http_url": "https://gitlab.com/owner/repo.git"},
            "user_name": "tagger",
        }
        result = parse_gitlab_webhook(payload)

        assert result.branch == "v2.0.0"
        assert result.event_type == "tag"

    def test_parse_uses_checkout_sha_fallback(self):
        """使用 checkout_sha 作为 commit sha 备选"""
        payload = {
            "object_kind": "push",
            "ref": "refs/heads/main",
            "checkout_sha": "checkoutsha",
            "project": {"http_url": "https://gitlab.com/owner/repo.git"},
        }
        result = parse_gitlab_webhook(payload)
        assert result.commit_sha == "checkoutsha"

    def test_parse_missing_repository_url_raises(self):
        """缺少 repository URL 时抛出异常"""
        payload = {
            "object_kind": "push",
            "ref": "refs/heads/main",
            "project": {},
            "repository": {},
        }
        with pytest.raises(WebhookPayloadParseError):
            parse_gitlab_webhook(payload)

    def test_parse_uses_user_username_fallback(self):
        """user_name 不存在时使用 user_username"""
        payload = {
            "object_kind": "push",
            "ref": "refs/heads/main",
            "project": {"http_url": "https://gitlab.com/owner/repo.git"},
            "user_username": "username-fallback",
        }
        result = parse_gitlab_webhook(payload)
        assert result.author == "username-fallback"
