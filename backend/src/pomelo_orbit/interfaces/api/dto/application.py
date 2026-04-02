"""
应用相关 Schema 定义
"""

from datetime import datetime

from pydantic import BaseModel, Field

from pomelo_orbit.domain.cd.value_objects import ImagePullPolicy


class ImageSourceReq(BaseModel):
    """镜像源配置"""

    image_name: str = Field(..., max_length=500)
    registry_url: str | None = Field(None, max_length=500)


class ApplicationCreateReq(BaseModel):
    """创建应用"""

    name: str = Field(..., min_length=1, max_length=100)
    code: str = Field(..., min_length=1, max_length=100, pattern="^[a-z][a-z0-9-]*$")
    image_pull_policy: ImagePullPolicy = ImagePullPolicy.MISSING

    image_source: ImageSourceReq | None = None


class ApplicationUpdateReq(BaseModel):
    """更新应用基本信息"""

    name: str | None = Field(None, min_length=1, max_length=100)
    code: str | None = Field(None, min_length=1, max_length=100, pattern="^[a-z0-9-]+$")
    image_pull_policy: ImagePullPolicy | None = None

    image_source: ImageSourceReq | None = None


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


class ConfigFileExportReq(BaseModel):
    """配置文件导出/导入请求"""

    path: str
    content: str = ""


class ImageSourceExportResp(BaseModel):
    """镜像源导出响应"""

    image_name: str
    registry_url: str | None


class ImageSourceImportReq(BaseModel):
    """镜像源导入请求"""

    image_name: str = Field(..., max_length=500)
    registry_url: str | None = Field(None, max_length=500)


class ConfigFileImportReq(BaseModel):
    """配置文件导入请求"""

    path: str
    content: str = ""


class ApplicationResp(BaseModel):
    """应用响应"""

    id: str
    name: str
    code: str
    status: str
    image_pull_policy: str
    created_at: datetime
    updated_at: datetime

    image_source: ImageSourceResp | None = None

    model_config = {"from_attributes": True}


class ApplicationExportResp(BaseModel):
    """应用导出响应"""

    version: str = "1.0"
    name: str
    code: str
    image_pull_policy: str
    image_source: ImageSourceExportResp | None = None
    config_files: list[ConfigFileExportReq] = []


class ApplicationImportReq(BaseModel):
    """应用导入请求"""

    version: str = "1.0"
    name: str = Field(..., min_length=1, max_length=100)
    code: str = Field(..., min_length=1, max_length=100, pattern="^[a-z][a-z0-9-]*$")
    image_pull_policy: ImagePullPolicy = ImagePullPolicy.MISSING
    image_source: ImageSourceImportReq | None = None
    config_files: list[ConfigFileImportReq] = []
