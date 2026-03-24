"""
通用 Schema 定义
"""

from pydantic import BaseModel


class MessageResp(BaseModel):
    """通用消息响应"""

    message: str


class PaginatedResp[T](BaseModel):
    """分页响应"""

    items: list[T]
    total: int
    page: int
    per_page: int
    pages: int
