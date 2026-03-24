"""
Webhook 解析器单元测试
"""

from typing import Any

from pomelo_orbit.application.webhook_parser import (
    parse_github_payload,
    parse_github_ping,
    parse_github_push,
    parse_github_release,
)


class TestParseGithubPush:
    """Github Push 事件解析测试"""

    def test_parse_valid_push_event(self):
        """测试解析有效的 push 事件"""
        payload = {
            "ref": "refs/heads/main",
            "repository": {
                "full_name": "owner/repo",
                "html_url": "https://github.com/owner/repo",
            },
            "sender": {"login": "developer"},
        }

        result = parse_github_push(payload)

        assert result.repository_name == "owner/repo"
        assert result.repository_url == "https://github.com/owner/repo"
        assert result.branch == "main"
        assert result.sender == "developer"
        assert result.event_type == "push"
        assert result.raw_payload == payload

    def test_parse_push_with_feature_branch(self):
        """测试解析特性分支的 push 事件"""
        payload = {
            "ref": "refs/heads/feature/new-feature",
            "repository": {
                "full_name": "owner/repo",
                "html_url": "https://github.com/owner/repo",
            },
            "sender": {"login": "developer"},
        }

        result = parse_github_push(payload)

        assert result.branch == "feature/new-feature"

    def test_parse_push_with_tag_ref(self):
        """测试解析 tag 引用（不是分支）"""
        payload = {
            "ref": "refs/tags/v1.0.0",
            "repository": {
                "full_name": "owner/repo",
                "html_url": "https://github.com/owner/repo",
            },
            "sender": {"login": "developer"},
        }

        result = parse_github_push(payload)

        # tag 不是分支，branch 应该为 None（因为不是 refs/heads/ 开头）
        assert result.branch is None

    def test_parse_push_with_missing_fields(self):
        """测试解析缺少字段的 push 事件"""
        payload = {
            "ref": "refs/heads/main",
            "repository": {},
            "sender": {},
        }

        result = parse_github_push(payload)

        assert result.repository_name is None
        assert result.repository_url is None
        assert result.branch == "main"
        assert result.sender is None

    def test_parse_push_with_empty_ref(self):
        """测试解析空 ref 的 push 事件"""
        payload = {
            "ref": "",
            "repository": {
                "full_name": "owner/repo",
                "html_url": "https://github.com/owner/repo",
            },
            "sender": {"login": "developer"},
        }

        result = parse_github_push(payload)

        assert result.branch is None


class TestParseGithubRelease:
    """Github Release 事件解析测试"""

    def test_parse_valid_release_event(self):
        """测试解析有效的 release 事件"""
        payload = {
            "repository": {
                "full_name": "owner/repo",
                "html_url": "https://github.com/owner/repo",
            },
            "release": {
                "target_commitish": "main",
                "tag_name": "v1.0.0",
            },
            "sender": {"login": "developer"},
        }

        result = parse_github_release(payload)

        assert result.repository_name == "owner/repo"
        assert result.repository_url == "https://github.com/owner/repo"
        assert result.branch == "main"
        assert result.sender == "developer"
        assert result.event_type == "release"

    def test_parse_release_with_missing_fields(self):
        """测试解析缺少字段的 release 事件"""
        payload: dict[str, Any] = {
            "repository": {},
            "release": {},
            "sender": {},
        }

        result = parse_github_release(payload)

        assert result.repository_name is None
        assert result.repository_url is None
        assert result.branch is None
        assert result.sender is None


class TestParseGithubPing:
    """Github Ping 事件解析测试"""

    def test_parse_valid_ping_event(self):
        """测试解析有效的 ping 事件"""
        payload = {
            "repository": {
                "full_name": "owner/repo",
                "html_url": "https://github.com/owner/repo",
            },
            "sender": {"login": "github"},
        }

        result = parse_github_ping(payload)

        assert result.repository_name == "owner/repo"
        assert result.repository_url == "https://github.com/owner/repo"
        assert result.branch is None
        assert result.sender == "github"
        assert result.event_type == "ping"

    def test_parse_ping_with_missing_fields(self):
        """测试解析缺少字段的 ping 事件"""
        payload: dict[str, Any] = {
            "repository": {},
            "sender": {},
        }

        result = parse_github_ping(payload)

        assert result.repository_name is None
        assert result.repository_url is None
        assert result.branch is None
        assert result.sender is None


class TestParseGithubPayload:
    """Github Payload 统一解析测试"""

    def test_parse_push_event_type(self):
        """测试解析 push 事件类型"""
        payload = {
            "ref": "refs/heads/main",
            "repository": {
                "full_name": "owner/repo",
                "html_url": "https://github.com/owner/repo",
            },
            "sender": {"login": "developer"},
        }

        result = parse_github_payload("push", payload)

        assert result is not None
        assert result.event_type == "push"
        assert result.branch == "main"

    def test_parse_release_event_type(self):
        """测试解析 release 事件类型"""
        payload = {
            "repository": {
                "full_name": "owner/repo",
                "html_url": "https://github.com/owner/repo",
            },
            "release": {"target_commitish": "main"},
            "sender": {"login": "developer"},
        }

        result = parse_github_payload("release", payload)

        assert result is not None
        assert result.event_type == "release"

    def test_parse_ping_event_type(self):
        """测试解析 ping 事件类型"""
        payload = {
            "repository": {
                "full_name": "owner/repo",
                "html_url": "https://github.com/owner/repo",
            },
            "sender": {"login": "github"},
        }

        result = parse_github_payload("ping", payload)

        assert result is not None
        assert result.event_type == "ping"

    def test_parse_unsupported_event_type(self):
        """测试解析不支持的事件类型"""
        payload: dict[str, Any] = {"repository": {}, "sender": {}}

        result = parse_github_payload("pull_request", payload)

        assert result is None

    def test_parse_unknown_event_type(self):
        """测试解析未知事件类型"""
        payload: dict[str, Any] = {"repository": {}, "sender": {}}

        result = parse_github_payload("unknown_event", payload)

        assert result is None
