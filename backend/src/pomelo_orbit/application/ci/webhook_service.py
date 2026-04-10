"""Webhook 聚合的应用服务"""

from pomelo_orbit.domain.ci.entities import RepositoryWebhook
from pomelo_orbit.domain.ci.repositories import (
    PipelineTemplateRepository,
    RepositoryRepository,
    RepositoryWebhookRepository,
)
from pomelo_orbit.domain.exceptions import BusinessError
from pomelo_orbit.infrastructure.ci.webhook_verifier import verify_github_signature, verify_gitlab_signature
from pomelo_orbit.infrastructure.security import SecurityService


class WebhookService:
    """RepositoryWebhook 聚合根的应用服务

    职责：
    - Webhook 的 CRUD 操作
    - Webhook 签名加密/解密
    - Webhook 签名验证
    - 关联 Repository 和 Template 验证

    依赖：
    - RepositoryService: 验证项目存在性
    - TemplateService: 验证模板存在性
    - SecurityService: 加密/解密
    """

    def __init__(
        self,
        webhook_repo: RepositoryWebhookRepository,
        repository_repo: RepositoryRepository,
        template_repo: PipelineTemplateRepository,
        security_service: SecurityService,
    ):
        self.webhook_repo = webhook_repo
        self.repository_repo = repository_repo
        self.template_repo = template_repo
        self.security_service = security_service

    def list_webhooks(self, repository_id: str) -> list[RepositoryWebhook]:
        """查询项目的 webhook 列表"""
        # 验证项目存在
        if not self.repository_repo.find_by_id(repository_id):
            raise BusinessError(f"Repository {repository_id} not found", status_code=404)
        return self.webhook_repo.find_by_repository(repository_id)

    def get_webhook(self, webhook_id: str) -> RepositoryWebhook:
        """获取单个 webhook"""
        wh = self.webhook_repo.find_by_id(webhook_id)
        if not wh:
            raise BusinessError(f"Webhook {webhook_id} not found", status_code=404)
        return wh

    def create_webhook(
        self,
        repository_id: str,
        name: str,
        template_id: str,
        plain_secret: str,
        branch_filter: str | None = None,
    ) -> RepositoryWebhook:
        """创建 webhook"""
        # 验证项目存在
        if not self.repository_repo.find_by_id(repository_id):
            raise BusinessError(f"Repository {repository_id} not found", status_code=404)

        # 验证模板存在
        if not self.template_repo.find_by_id(template_id):
            raise BusinessError(f"PipelineTemplate {template_id} not found", status_code=404)

        encrypted = self.security_service.encrypt_value(plain_secret)
        wh = RepositoryWebhook.create(
            repository_id=repository_id,
            name=name,
            template_id=template_id,
            encrypted_secret=encrypted,
            branch_filter=branch_filter,
        )
        self.webhook_repo.save(wh)
        return wh

    def update_webhook(
        self,
        webhook_id: str,
        name: str | None = None,
        template_id: str | None = None,
        branch_filter: str | None = None,
        plain_secret: str | None = None,
        enabled: bool | None = None,
    ) -> RepositoryWebhook:
        """更新 webhook"""
        wh = self.get_webhook(webhook_id)

        # 验证模板存在（如果要更新）
        if template_id and not self.template_repo.find_by_id(template_id):
            raise BusinessError(f"PipelineTemplate {template_id} not found", status_code=404)

        encrypted_secret = self.security_service.encrypt_value(plain_secret) if plain_secret else None
        wh.update(
            name=name,
            template_id=template_id,
            branch_filter=branch_filter,
            encrypted_secret=encrypted_secret,
            enabled=enabled,
        )
        self.webhook_repo.save(wh)
        return wh

    def delete_webhook(self, webhook_id: str) -> None:
        """删除 webhook"""
        wh = self.get_webhook(webhook_id)
        self.webhook_repo.delete(wh)

    def decrypt_webhook_secret(self, webhook: RepositoryWebhook) -> str:
        """解密 webhook 签名密钥"""
        return self.security_service.decrypt_value(webhook.encrypted_secret)

    def verify_webhook_signature(
        self,
        source: str,
        payload: bytes,
        signature: str,
        secret: str,
    ) -> bool:
        """验证 webhook 签名

        Args:
            source: "github" 或 "gitlab"
            payload: 原始请求体 bytes
            signature: 签名（GitHub 的 X-Hub-Signature-256 或 GitLab 的 X-Gitlab-Token）
            secret: 解密后的 webhook secret

        Returns:
            验证是否通过
        """
        if source == "github":
            return verify_github_signature(payload, signature, secret)
        if source == "gitlab":
            return verify_gitlab_signature(signature, secret)
        return False
