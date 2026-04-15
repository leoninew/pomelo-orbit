"""CI 模块公共 DTO"""

from pydantic import BaseModel, Field

from pomelo_orbit.domain.ci.value_objects import ArtifactConfig, ArtifactType


class ArtifactConfigDto(BaseModel):
    type: ArtifactType
    path: str = Field(min_length=1)
    name: str = Field(min_length=1)
    model_config = {"from_attributes": True}

    def to_artifact_config(self) -> ArtifactConfig:
        return ArtifactConfig(type=self.type, path=self.path, name=self.name)
