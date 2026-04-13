"""CI 模块公共 DTO"""

from pydantic import BaseModel, Field

from pomelo_orbit.domain.ci.value_objects import ArtifactType


class ArtifactConfigDto(BaseModel):
    type: ArtifactType = ArtifactType.BINARY
    path: str = Field(min_length=1)
    name: str = Field(min_length=1)
    model_config = {"from_attributes": True}
