"""
时间处理工具模块

统一管理所有时间相关的操作：
- 数据库存储：UTC naive datetime
- API 传输：ISO 8601 格式（带 Z 后缀）
- 内部逻辑：UTC datetime
"""

from datetime import UTC, datetime, timedelta


def utc_now() -> datetime:
    """
    获取当前 UTC 时间（naive datetime）

    用于数据库存储和内部逻辑
    """
    return datetime.now(UTC).replace(tzinfo=None)


def utc_from_timestamp(timestamp: float) -> datetime:
    """
    从时间戳转换为 UTC naive datetime
    """
    return datetime.fromtimestamp(timestamp, tz=UTC).replace(tzinfo=None)


def to_iso8601(dt: datetime | None) -> str | None:
    """
    将 datetime 转换为 ISO 8601 格式（带 Z 后缀）

    用于 API 响应
    """
    if dt is None:
        return None
    return dt.isoformat() + "Z"


def from_iso8601(iso_string: str) -> datetime:
    """
    从 ISO 8601 格式解析为 UTC naive datetime

    用于 API 请求
    """
    dt = datetime.fromisoformat(iso_string.replace("Z", "+00:00"))
    return dt.astimezone(UTC).replace(tzinfo=None)


def add_minutes(dt: datetime, minutes: int) -> datetime:
    """
    给 datetime 增加分钟数
    """
    return dt + timedelta(minutes=minutes)


def add_hours(dt: datetime, hours: int) -> datetime:
    """
    给 datetime 增加小时数
    """
    return dt + timedelta(hours=hours)


def add_days(dt: datetime, days: int) -> datetime:
    """
    给 datetime 增加天数
    """
    return dt + timedelta(days=days)
