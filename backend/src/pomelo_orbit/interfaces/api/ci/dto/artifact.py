"""制品 DTO"""

from datetime import datetime

from pydantic import BaseModel

from pomelo_orbit.domain.ci.value_objects import ArtifactType


class ArtifactResp(BaseModel):
    id: str
    pipeline_run_id: str
    repository_id: str
    repository_name: str
    template_id: str
    template_name: str
    stage_name: str
    type: ArtifactType
    name: str
    path: str | None
    created_at: datetime

    model_config = {"from_attributes": True}
