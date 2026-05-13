"""
认证相关 Schema 定义
"""

from datetime import datetime

from pydantic import BaseModel, Field


class LoginReq(BaseModel):
    """登录请求"""

    username: str = Field(..., min_length=1, max_length=50)
    password: str = Field(..., min_length=1)
    csrf_token: str = Field(..., min_length=1, description="CSRF Token")
    captcha_token: str = Field(..., min_length=1, description="验证码 Token")
    captcha_answer: str = Field(..., min_length=1, description="验证码答案")


class CsrfTokenResp(BaseModel):
    """CSRF Token 响应"""

    token: str


class CaptchaResp(BaseModel):
    """验证码响应"""

    token: str
    image: str  # base64 编码的图片


class TokenResp(BaseModel):
    """Token 响应"""

    access_token: str
    token_type: str = "bearer"


class UserInfo(BaseModel):
    """用户信息"""

    id: str
    username: str
    email: str | None = None
    auth_source: str
    created_at: datetime
    last_login_at: datetime | None = None


class PasswordChangeReq(BaseModel):
    """修改密码请求"""

    old_password: str = Field(..., min_length=1)
    new_password: str = Field(..., min_length=6)


class LoginHistoryResp(BaseModel):
    """登录历史响应"""

    id: str
    user_id: str
    username: str
    ip_address: str | None
    user_agent: str | None
    login_at: datetime
    success: bool

    model_config = {"from_attributes": True}


class GoogleCallbackReq(BaseModel):
    """Google OAuth 回调请求"""

    code: str
