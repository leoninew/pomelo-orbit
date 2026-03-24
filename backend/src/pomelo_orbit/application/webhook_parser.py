"""
Webhook 解析服务
"""

from pomelo_orbit.application.dtos import WebhookPayload


def parse_github_push(payload: dict) -> WebhookPayload:
    """解析 Github push 事件"""
    repo = payload.get("repository", {})
    ref = payload.get("ref", "")
    # ref 格式: refs/heads/main -> main
    branch = ref.replace("refs/heads/", "") if ref.startswith("refs/heads/") else None

    return WebhookPayload(
        repository_name=repo.get("full_name"),
        repository_url=repo.get("html_url"),
        branch=branch,
        sender=payload.get("sender", {}).get("login"),
        event_type="push",
        raw_payload=payload,
    )


def parse_github_release(payload: dict) -> WebhookPayload:
    """解析 Github release 事件"""
    repo = payload.get("repository", {})
    release = payload.get("release", {})

    return WebhookPayload(
        repository_name=repo.get("full_name"),
        repository_url=repo.get("html_url"),
        branch=release.get("target_commitish"),
        sender=payload.get("sender", {}).get("login"),
        event_type="release",
        raw_payload=payload,
    )


def parse_github_ping(payload: dict) -> WebhookPayload:
    """解析 Github ping 事件"""
    repo = payload.get("repository", {})

    return WebhookPayload(
        repository_name=repo.get("full_name"),
        repository_url=repo.get("html_url"),
        branch=None,
        sender=payload.get("sender", {}).get("login"),
        event_type="ping",
        raw_payload=payload,
    )


def parse_github_payload(event_type: str, payload: dict) -> WebhookPayload | None:
    """解析 Github Webhook payload"""
    parsers = {
        "push": parse_github_push,
        "release": parse_github_release,
        "ping": parse_github_ping,
    }
    parser = parsers.get(event_type)
    if parser:
        return parser(payload)
    return None


__all__ = ["parse_github_payload"]
