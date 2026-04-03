"""
应用相关 Schema 定义
"""

from datetime import datetime

from pydantic import BaseModel, Field

from pomelo_orbit.domain.cd.value_objects import ImagePullPolicy


class ApplicationCreateReq(BaseModel):
    """创建应用"""

    name: str = Field(..., min_length=1, max_length=100)
    code: str = Field(..., min_length=1, max_length=100, pattern="^[a-z][a-z0-9-]*$")
    image_pull_policy: ImagePullPolicy = ImagePullPolicy.MISSING


class ApplicationUpdateReq(BaseModel):
    """更新应用基本信息"""

    name: str | None = Field(None, min_length=1, max_length=100)
    code: str | None = Field(None, min_length=1, max_length=100, pattern="^[a-z0-9-]+$")
    image_pull_policy: ImagePullPolicy | None = None


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

    model_config = {"from_attributes": True}


class ApplicationExportResp(BaseModel):
    """应用导出响应"""

    version: str = "1.0"
    name: str
    code: str
    image_pull_policy: str
    config_files: list[ConfigFileExportReq] = []


class ApplicationImportReq(BaseModel):
    """应用导入请求"""

    version: str = "1.0"
    name: str = Field(..., min_length=1, max_length=100)
    code: str = Field(..., min_length=1, max_length=100, pattern="^[a-z][a-z0-9-]*$")
    image_pull_policy: ImagePullPolicy = ImagePullPolicy.MISSING
    config_files: list[ConfigFileImportReq] = []
