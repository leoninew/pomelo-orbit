"""
应用相关 Schema 定义
"""

from datetime import datetime

from pydantic import BaseModel, Field

from pomelo_orbit.domain.value_objects import ImagePullPolicy


class GitSourceReq(BaseModel):
    """Git 源配置"""

    repository_url: str = Field(..., max_length=500)
    deploy_branches: str = "main,master"
    auto_deploy: bool = True


class ImageSourceReq(BaseModel):
    """镜像源配置"""

    image_name: str = Field(..., max_length=500)
    registry_url: str | None = Field(None, max_length=500)


class ApplicationCreateReq(BaseModel):
    """创建应用"""

    name: str = Field(..., min_length=1, max_length=100)
    code: str = Field(..., min_length=1, max_length=100, pattern="^[a-z][a-z0-9-]*$")
    enabled: bool = True
    image_pull_policy: ImagePullPolicy = ImagePullPolicy.MISSING

    git_source: GitSourceReq | None = None
    image_source: ImageSourceReq | None = None


class ApplicationUpdateReq(BaseModel):
    """更新应用基本信息"""

    name: str | None = Field(None, min_length=1, max_length=100)
    code: str | None = Field(None, min_length=1, max_length=100, pattern="^[a-z0-9-]+$")
    enabled: bool | None = None
    image_pull_policy: ImagePullPolicy | None = None

    git_source: GitSourceReq | None = None
    image_source: ImageSourceReq | None = None


class GitSourceResp(BaseModel):
    """Git 源响应"""

    id: str
    repository_url: str
    deploy_branches: str
    auto_deploy: bool

    model_config = {"from_attributes": True}


class ImageSourceResp(BaseModel):
    """镜像源响应"""

    id: str
    image_name: str
    registry_url: str | None

    model_config = {"from_attributes": True}


class ConfigFileReq(BaseModel):
    """配置文件请求"""

    path: str
    content: str = ""


class ConfigFileResp(BaseModel):
    """配置文件响应"""

    id: str
    path: str
    created_at: datetime

    model_config = {"from_attributes": True}


class ApplicationResp(BaseModel):
    """应用响应"""

    id: str
    name: str
    code: str
    enabled: bool
    status: str
    image_pull_policy: str
    created_at: datetime
    updated_at: datetime

    git_source: GitSourceResp | None = None
    image_source: ImageSourceResp | None = None

    model_config = {"from_attributes": True}
