"""Credential 聚合的应用服务"""

from pomelo_orbit.domain.ci.entities import Credential
from pomelo_orbit.domain.ci.repositories import CredentialRepository
from pomelo_orbit.domain.ci.value_objects import CredentialType
from pomelo_orbit.domain.exceptions import BusinessError
from pomelo_orbit.infrastructure.security import SecurityService


class CredentialService:
    """Credential 聚合根的应用服务

    职责：
    - Credential 的 CRUD 操作
    - 业务规则验证（如删除时检查引用）
    """

    def __init__(self, credential_repo: CredentialRepository, security_service: SecurityService):
        self.credential_repo = credential_repo
        self.security_service = security_service

    def list_credentials(self, page: int = 1, per_page: int = 20) -> tuple[list[Credential], int]:
        """分页查询凭据列表"""
        return self.credential_repo.find_paginated(page=page, per_page=per_page)

    def get_credential(self, credential_id: str) -> Credential:
        """获取单个凭据"""
        cred = self.credential_repo.find_by_id(credential_id)
        if not cred:
            raise BusinessError(f"Credential {credential_id} not found", status_code=404)
        return cred

    def create_credential(self, name: str, credential_type: str, data: str) -> Credential:
        """创建凭据"""
        encrypted = self.security_service.encrypt_value(data)
        cred = Credential.create(name=name, type=CredentialType(credential_type), encrypted_data=encrypted)
        self.credential_repo.save(cred)
        return cred

    def update_credential(self, credential_id: str, name: str | None = None, data: str | None = None) -> Credential:
        """更新凭据"""
        cred = self.get_credential(credential_id)
        if name is not None:
            cred.name = name
        if data is not None:
            cred.encrypted_data = self.security_service.encrypt_value(data)
        self.credential_repo.save(cred)
        return cred

    def delete_credential(self, credential_id: str) -> None:
        """删除凭据（检查引用）"""
        cred = self.get_credential(credential_id)
        if self.credential_repo.is_referenced_by_projects(credential_id):
            raise BusinessError("Credential is referenced by projects, cannot delete", status_code=409)
        self.credential_repo.delete(cred)

    def export_credential(self, credential_id: str) -> dict:
        """导出凭据（解密数据）"""
        cred = self.get_credential(credential_id)
        return {
            "name": cred.name,
            "type": cred.type.value,
            "data": self.security_service.decrypt_value(cred.encrypted_data),
        }

    def import_credential(self, name: str, credential_type: str, data: str) -> Credential:
        """导入凭据"""
        return self.create_credential(name=name, credential_type=credential_type, data=data)
