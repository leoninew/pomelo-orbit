"""
日志配置模块
"""

import logging
import time

from fastapi import Request
from fastapi.responses import JSONResponse
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.responses import Response

from pomelo_orbit.infrastructure.config import get_settings

# 日志格式配置
LOG_FORMAT = "%(asctime)s [%(levelname).5s] %(filename)s:%(lineno)d %(message)s"

# Uvicorn 日志配置（日志文件路径将在 main.py 中动态设置）
LOGGING_CONFIG = {
    "version": 1,
    "disable_existing_loggers": False,
    "formatters": {
        "default": {
            "format": LOG_FORMAT,
            "datefmt": "%Y-%m-%d %H:%M:%S",
        },
    },
    "handlers": {
        "console": {
            "class": "logging.StreamHandler",
            "formatter": "default",
        },
        "file": {
            "class": "logging.handlers.TimedRotatingFileHandler",
            "formatter": "default",
            "filename": "logs/pomelo-orbit.log",  # 默认路径，将在 main.py 中覆盖
            "when": "midnight",  # 每天午夜轮换
            "interval": 1,  # 每 1 天
            "backupCount": 7,  # 保留 7 天
            "encoding": "utf-8",
            "delay": True,  # 延迟创建文件，直到第一次写入
            "utc": False,  # 使用本地时间
        },
    },
    "loggers": {
        # 接管 uvicorn 的日志，只记录警告和错误
        "uvicorn": {"handlers": ["console", "file"], "level": "WARNING", "propagate": False},
        "uvicorn.error": {"level": "WARNING"},
        # 禁用 uvicorn.access，由中间件接管
        "uvicorn.access": {"handlers": [], "level": "CRITICAL", "propagate": False},
        # 项目根日志，记录详细的业务信息
        "pomelo_orbit": {"handlers": ["console", "file"], "level": "INFO", "propagate": False},
    },
}


class RequestLoggingMiddleware(BaseHTTPMiddleware):
    """请求日志中间件"""

    def __init__(self, app, logger_name: str = "pomelo_orbit.requests"):
        super().__init__(app)
        self.logger = logging.getLogger(logger_name)
        self.api_prefix = get_settings().logging.api_prefix

    async def _try_get_request_body(self, request: Request) -> str | None:
        """尝试获取请求体"""
        content_type = request.headers.get("content-type", "")
        if "application/json" not in content_type:
            return None

        try:
            body_bytes = await request.body()
            if not body_bytes:
                return None

            # 重建请求体供下游处理
            async def receive():
                return {"type": "http.request", "body": body_bytes}

            request._receive = receive
            return body_bytes.decode("utf-8", errors="replace")
        except Exception as e:
            self.logger.warning(f"Request body read failed: error={e}")
            return None

    async def _try_get_response_body(self, response: Response) -> str | None:
        """尝试获取响应体"""
        content_type = response.headers.get("content-type", "")
        if "application/json" not in content_type:
            return None

        try:
            body = b""
            async for chunk in response.body_iterator:  # type: ignore[attr-defined]
                body += chunk
            if not body:
                return None

            # 恢复 body_iterator 以便响应能正常发送
            async def restore():
                yield body

            response.body_iterator = restore()  # type: ignore[attr-defined]
            return body.decode("utf-8", errors="replace")
        except Exception as e:
            self.logger.warning(f"Response body read failed: error={e}")
            return None

    async def dispatch(self, request: Request, call_next):
        start_time = time.perf_counter()
        method = request.method
        path = request.url.path

        self.logger.info(f"Request: {method} {path}")

        # 尝试读取请求体（空值时跳过）
        request_body = None
        if self.api_prefix and path.startswith(self.api_prefix):
            request_body = await self._try_get_request_body(request)
            if request_body:
                self.logger.info(f"Request body: {request_body[:1024]}")

        try:
            response = await call_next(request)
            duration_ms = (time.perf_counter() - start_time) * 1000

            # 尝试获取响应体（空值时跳过）
            response_body = None
            if self.api_prefix and path.startswith(self.api_prefix):
                response_body = await self._try_get_response_body(response)

            # 记录响应
            if response.status_code >= 500:
                self.logger.error(
                    f"Response: {method} {path} - {response.status_code} ({duration_ms:.2f}ms)", exc_info=True
                )
            elif response.status_code >= 400:
                self.logger.warning(f"Response: {method} {path} - {response.status_code} ({duration_ms:.2f}ms)")
                if response_body:
                    self.logger.warning(f"Response body: {response_body[:1024]}")
            else:
                self.logger.info(f"Response: {method} {path} - {response.status_code} ({duration_ms:.2f}ms)")
                if response_body:
                    self.logger.info(f"Response body: {response_body[:1024]}")

            return response

        except Exception as e:
            duration_ms = (time.perf_counter() - start_time) * 1000
            self.logger.exception(f"Exception: {method} {path} - {type(e).__name__}: {e!s} ({duration_ms:.2f}ms)")
            return JSONResponse(status_code=500, content={"detail": "Internal server error"})
