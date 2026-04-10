"""Route API DTOs"""

from datetime import datetime

from pydantic import BaseModel, ConfigDict, Field


class RouteResp(BaseModel):
    """路由响应"""

    model_config = ConfigDict(from_attributes=True)

    id: str
    name: str
    domain: str
    path_prefix: str
    target_url: str
    enabled: bool
    https_enabled: bool = False
    cert_type: str = "manual"  # 证书类型：manual / letsencrypt / mkcert
    created_at: datetime
    updated_at: datetime


class RouteCreateReq(BaseModel):
    """创建路由请求"""

    name: str = Field(..., pattern=r"^[a-z][a-z0-9._-]*$")
    domain: str
    path_prefix: str = "/"
    target_url: str = Field(..., pattern=r"^https?://[a-zA-Z0-9.-]+:\d+$")
    enabled: bool = False


class RouteUpdateReq(BaseModel):
    """更新路由请求"""

    name: str | None = Field(None, pattern=r"^[a-z][a-z0-9._-]*$")
    domain: str | None = None
    path_prefix: str | None = None
    target_url: str | None = Field(None, pattern=r"^https?://[a-zA-Z0-9.-]+:\d+$")
    enabled: bool | None = None
