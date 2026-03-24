"""Credential application service - handles credential business logic."""

import logging

import ulid

from pomelo_orbit.domain.entities import Credential
from pomelo_orbit.domain.exceptions import BusinessError
from pomelo_orbit.domain.repositories import ApplicationRepository, CredentialRepository
from pomelo_orbit.infrastructure.security import SecurityService

logger = logging.getLogger(__name__)


class CredentialService:
    """凭据应用服务 - 处理凭据的业务逻辑"""

    def __init__(
        self,
        credential_repo: CredentialRepository,
        app_repo: ApplicationRepository,
        security_service: SecurityService,
    ):
        self.credential_repo = credential_repo
        self.app_repo = app_repo
        self.security_service = security_service

    def list_credentials(self, page: int, per_page: int, search: str | None = None) -> tuple[list[Credential], int]:
        """列出所有凭据（分页）"""
        return self.credential_repo.find_paginated(page, per_page, search)

    def get_credential(self, credential_id: str) -> Credential:
        """获取凭据详情（包含解密后的值）"""
        credential = self.credential_repo.find_by_id(credential_id)
        if not credential:
            raise BusinessError(f"Credential {credential_id} not found", status_code=404)
        return credential

    def get_credential_by_application(self, application_id: str) -> Credential | None:
        """获取应用的凭据"""
        return self.credential_repo.find_by_application(application_id)

    def create_credential(
        self,
        application_id: str,
        name: str,
        credential_type: str,
        value: str,
        extra_data: str | None = None,
    ) -> Credential:
        """创建凭据"""
        # 验证应用是否存在
        app = self.app_repo.find_by_id(application_id)
        if not app:
            raise BusinessError(f"Application '{application_id}' not found", status_code=404)

        # 检查名称是否在同一应用内重复
        existing = self.credential_repo.find_by_name_in_app(application_id, name)
        if existing:
            raise BusinessError(
                f"Credential with name '{name}' already exists in this application",
                status_code=400,
            )

        # 加密凭据值
        encrypted_value = self.security_service.encrypt_value(value)

        credential = Credential(
            id=str(ulid.ULID()),
            application_id=application_id,
            name=name,
            type=credential_type,
            value_encrypted=encrypted_value,
            extra_data=extra_data,
        )
        self.credential_repo.save(credential)

        return credential

    def update_credential(
        self,
        credential_id: str,
        name: str | None = None,
        value: str | None = None,
        extra_data: str | None = None,
    ) -> Credential:
        """更新凭据"""
        credential = self.credential_repo.find_by_id(credential_id)
        if not credential:
            raise BusinessError(f"Credential {credential_id} not found", status_code=404)

        if name is not None:
            existing = self.credential_repo.find_by_name_in_app(credential.application_id, name)
            if existing and existing.id != credential_id:
                raise BusinessError(
                    f"Credential with name '{name}' already exists in this application",
                    status_code=400,
                )
            credential.name = name

        if value is not None:
            credential.value_encrypted = self.security_service.encrypt_value(value)

        if extra_data is not None:
            credential.extra_data = extra_data

        self.credential_repo.save(credential)

        return credential

    def delete_credential(self, credential_id: str) -> None:
        """删除凭据"""
        credential = self.credential_repo.find_by_id(credential_id)
        if not credential:
            raise BusinessError(f"Credential {credential_id} not found", status_code=404)

        self.credential_repo.delete(credential)

    def decrypt_credential_value(self, credential: Credential) -> str:
        """解密凭据值"""
        return self.security_service.decrypt_value(credential.value_encrypted)
