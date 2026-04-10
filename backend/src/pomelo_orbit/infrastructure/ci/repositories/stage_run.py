"""CI StageRun 仓储实现"""

from sqlalchemy.orm import Session

from pomelo_orbit.domain.ci.entities import StageRun
from pomelo_orbit.domain.ci.repositories import StageRunRepository
from pomelo_orbit.infrastructure.ci.mappers import StageRunMapper
from pomelo_orbit.infrastructure.ci.models import StageRunModel
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository


class StageRunRepositoryImpl(BaseRepository[StageRun, StageRunModel], StageRunRepository):
    """Stage 执行记录仓储"""

    def __init__(self, session: Session):
        super().__init__(session, StageRunModel, StageRunMapper())

    def find_by_run(self, run_id: str) -> list[StageRun]:
        orms = self._session.query(StageRunModel).filter(StageRunModel.pipeline_run_id == run_id).all()
        return [self._mapper.to_domain(orm) for orm in orms]
