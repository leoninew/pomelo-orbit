"""构建 Stage DTO"""

from datetime import datetime

from pydantic import BaseModel, Field

from pomelo_orbit.interfaces.api.ci.dto.common import ArtifactConfigDto


class BuildStageResp(BaseModel):
    id: str
    name: str
    image: str
    script: str
    artifacts: list[ArtifactConfigDto] | None
    description: str
    version: int
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class BuildStageCreateReq(BaseModel):
    name: str = Field(min_length=1)
    image: str = Field(min_length=1)
    script: str = Field(min_length=1)
    artifacts: list[ArtifactConfigDto] | None = None
    description: str = ""


class BuildStageUpdateReq(BaseModel):
    name: str | None = Field(default=None, min_length=1)
    image: str | None = Field(default=None, min_length=1)
    script: str | None = Field(default=None, min_length=1)
    artifacts: list[ArtifactConfigDto] | None = None
    description: str | None = None
