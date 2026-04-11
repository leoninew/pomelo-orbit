"""CI 凭据仓储实现"""

from sqlalchemy.orm import Session

from pomelo_orbit.domain.ci.entities import Credential
from pomelo_orbit.domain.ci.repositories import CredentialRepository
from pomelo_orbit.infrastructure.ci.mappers import CredentialMapper
from pomelo_orbit.infrastructure.ci.models import CredentialModel, ProjectModel
from pomelo_orbit.infrastructure.persistence.base_repository import BaseRepository


class CredentialRepositoryImpl(BaseRepository[Credential, CredentialModel], CredentialRepository):
    """凭据仓储"""

    def __init__(self, session: Session):
        super().__init__(session, CredentialModel, CredentialMapper())

    def find_by_name(self, name: str) -> Credential | None:
        """按名称查找凭据"""
        model = self._session.query(CredentialModel).filter(CredentialModel.name == name).first()
        return self._mapper.to_domain(model) if model else None

    def is_referenced_by_projects(self, credential_id: str) -> bool:
        """检查凭据是否被项目引用"""
        return (
            self._session.query(ProjectModel).filter(ProjectModel.git_credential_id == credential_id).first()
            is not None
        )
