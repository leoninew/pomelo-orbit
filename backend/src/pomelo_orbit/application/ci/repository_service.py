"""Repository 聚合的应用服务"""

from dataclasses import MISSING

from pomelo_orbit.domain.ci.entities import Repository
from pomelo_orbit.domain.ci.repositories import CredentialRepository, RepositoryRepository
from pomelo_orbit.domain.ci.value_objects import VariableDeclaration
from pomelo_orbit.domain.ci.variable_resolver import VariableResolver
from pomelo_orbit.domain.exceptions import BusinessError
from pomelo_orbit.infrastructure.ci.workspace import cleanup_project


class RepositoryService:
    """Repository 聚合根的应用服务

    职责：
    - Repository 的 CRUD 操作
    - Repository 变量覆盖管理
    - Repository code 唯一性验证
    - 关联 Credential 验证
    """

    def __init__(
        self,
        repository_repo: RepositoryRepository,
        credential_repo: CredentialRepository,
        variable_resolver: VariableResolver,
    ):
        self.repository_repo = repository_repo
        self.credential_repo = credential_repo
        self.variable_resolver = variable_resolver

    def list_repositories(
        self, project_id: str, page: int = 1, per_page: int = 20, search: str | None = None
    ) -> tuple[list[Repository], int]:
        """分页查询项目列表"""
        return self.repository_repo.find_paginated_by_project_id(
            project_id=project_id, page=page, per_page=per_page, search=search
        )

    def get_repository(self, repository_id: str) -> Repository:
        """通过 ID 获取单个 Repository"""
        repository = self.repository_repo.find_by_id(repository_id)
        if not repository:
            raise BusinessError(f"Repository {repository_id} not found", status_code=404)
        return repository

    def create_repository(
        self,
        project_id: str,
        name: str,
        code: str,
        repository_url: str,
        git_credential_id: str | None = None,
        variable_overrides: list[VariableDeclaration] | None = None,
        default_branch: str = "master",
    ) -> Repository:
        """创建项目"""
        if git_credential_id and not self.credential_repo.find_by_id(git_credential_id):
            raise BusinessError(f"Credential {git_credential_id} not found", status_code=404)

        if self.repository_repo.find_by_code(project_id, code):
            raise BusinessError(f"Repository code '{code}' already exists", status_code=409)

        repository = Repository.create(
            project_id=project_id,
            name=name,
            code=code,
            repository_url=repository_url,
            git_credential_id=git_credential_id,
            variable_overrides=self.variable_resolver.sanitize_variable_overrides(variable_overrides),
            default_branch=default_branch,
        )
        self.repository_repo.save(repository)
        return repository

    def update_repository(
        self,
        repository_id: str,
        name: str | None = None,
        repository_url: str | None = None,
        variable_overrides: list[VariableDeclaration] | None = MISSING,  # type: ignore[assignment]
        git_credential_id: str | None = MISSING,  # type: ignore[assignment]
        default_branch: str | None = None,
    ) -> Repository:
        """通过 ID 更新 Repository"""
        repository = self.get_repository(repository_id)

        if (
            git_credential_id is not MISSING  # type: ignore[comparison-overlap]
            and git_credential_id
            and not self.credential_repo.find_by_id(git_credential_id)
        ):
            raise BusinessError(f"Credential {git_credential_id} not found", status_code=404)

        repository.update(
            name=name,
            repository_url=repository_url,
            variable_overrides=(
                self.variable_resolver.sanitize_variable_overrides(variable_overrides)
                if variable_overrides is not MISSING  # type: ignore[comparison-overlap]
                else MISSING
            ),
            git_credential_id=git_credential_id,
            default_branch=default_branch,
        )
        self.repository_repo.save(repository)
        return repository

    def delete_repository(self, repository_id: str, delete_workspace: bool = False) -> None:
        """通过 ID 删除 Repository（检查运行中的流水线）"""
        repository = self.get_repository(repository_id)
        if self.repository_repo.has_running_pipelines(repository.project_id, repository_id):
            raise BusinessError("Repository has running pipelines, cannot delete", status_code=409)
        self.repository_repo.delete(repository)
        if delete_workspace:
            cleanup_project(repository.code)
