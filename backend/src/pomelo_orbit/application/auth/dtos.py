"""认证应用服务的数据模型"""

from pydantic import BaseModel


class LoginReq(BaseModel):
    """登录请求"""

    username: str
    password: str
    csrf_token: str
    ip_address: str
    user_agent: str | None = None
    captcha_token: str
    captcha_answer: str


class LoginResp(BaseModel):
    """登录响应"""

    access_token: str
    token_type: str = "bearer"
