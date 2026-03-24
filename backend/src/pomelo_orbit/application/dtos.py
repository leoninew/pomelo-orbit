"""
应用层数据传输对象
"""

from dataclasses import dataclass


@dataclass
class WebhookPayload:
    """解析后的 Webhook 数据"""

    repository_name: str | None  # 格式: owner/repo
    repository_url: str | None
    branch: str | None
    sender: str | None
    event_type: str
    raw_payload: dict


__all__ = ["WebhookPayload"]
