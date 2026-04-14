"""接口层公共工具"""

from collections.abc import Awaitable, Callable
from typing import Any

from dishka import AsyncContainer


async def run_in_new_scope(
    container: AsyncContainer,
    service_type: type,
    coro_fn: Callable[[Any], Awaitable[None]],
) -> None:
    """在新的 Dishka REQUEST scope 中执行异步任务，自动管理 session 生命周期"""
    async with container() as c:
        svc: Any = await c.get(service_type)
        await coro_fn(svc)
