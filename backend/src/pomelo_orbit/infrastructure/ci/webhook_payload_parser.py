"""CI Webhook Payload 解析器"""

import logging
from dataclasses import dataclass

logger = logging.getLogger(__name__)


class WebhookPayloadParseError(Exception):
    """Webhook payload 解析错误"""


@dataclass
class CIWebhookPayload:
    """解析后的 CI Webhook 数据"""

    repository_url: str
    branch: str | None
    commit_sha: str
    author: str
    event_type: str  # push | release | tag


def parse_github_webhook(payload: dict) -> CIWebhookPayload:
    """
    解析 GitHub webhook payload

    Args:
        payload: GitHub webhook JSON payload

    Returns:
        CIWebhookPayload

    Raises:
        WebhookPayloadParseError: 解析失败
    """
    try:
        repo = payload.get("repository", {})
        repository_url = repo.get("clone_url") or repo.get("html_url") or repo.get("ssh_url")
        if not repository_url:
            raise WebhookPayloadParseError("缺少 repository URL")

        # 判断事件类型
        ref = payload.get("ref", "")
        if ref.startswith("refs/tags/"):
            event_type = "tag"
            branch = ref.removeprefix("refs/tags/")
        elif ref.startswith("refs/heads/"):
            event_type = "push"
            branch = ref.removeprefix("refs/heads/")
        elif "release" in payload:
            event_type = "release"
            release = payload.get("release", {})
            branch = release.get("target_commitish")
        else:
            event_type = "push"
            branch = ref or None

        # 提取 commit sha
        commit_sha = payload.get("after") or payload.get("head_commit", {}).get("id") or ""

        author = payload.get("sender", {}).get("login") or payload.get("pusher", {}).get("name") or ""

        return CIWebhookPayload(
            repository_url=repository_url,
            branch=branch,
            commit_sha=commit_sha,
            author=author,
            event_type=event_type,
        )
    except WebhookPayloadParseError:
        raise
    except Exception as e:
        raise WebhookPayloadParseError(f"GitHub payload 解析失败: {e}") from e


def parse_gitlab_webhook(payload: dict) -> CIWebhookPayload:
    """
    解析 GitLab webhook payload

    Args:
        payload: GitLab webhook JSON payload

    Returns:
        CIWebhookPayload

    Raises:
        WebhookPayloadParseError: 解析失败
    """
    try:
        project = payload.get("project", {})
        repository_url = (
            project.get("http_url") or project.get("git_http_url") or payload.get("repository", {}).get("homepage")
        )
        if not repository_url:
            raise WebhookPayloadParseError("缺少 repository URL")

        ref = payload.get("ref", "")
        object_kind = payload.get("object_kind", "push")

        if object_kind == "tag_push" or ref.startswith("refs/tags/"):
            event_type = "tag"
            branch = ref.removeprefix("refs/tags/")
        else:
            event_type = "push"
            branch = ref.removeprefix("refs/heads/") if ref.startswith("refs/heads/") else ref or None

        commit_sha = payload.get("after") or payload.get("checkout_sha") or ""
        author = payload.get("user_name") or payload.get("user_username") or ""

        return CIWebhookPayload(
            repository_url=repository_url,
            branch=branch,
            commit_sha=commit_sha,
            author=author,
            event_type=event_type,
        )
    except WebhookPayloadParseError:
        raise
    except Exception as e:
        raise WebhookPayloadParseError(f"GitLab payload 解析失败: {e}") from e
