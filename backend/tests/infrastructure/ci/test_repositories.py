"""测试 CI Repositories"""

import ulid
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker

from pomelo_orbit.domain.ci.entities import Project
from pomelo_orbit.infrastructure.ci.models import Base
from pomelo_orbit.infrastructure.ci.repositories import ProjectRepositoryImpl


class TestProjectRepository:
    """测试 ProjectRepository"""

    def setup_method(self):
        """每个测试前设置"""
        # 使用内存数据库
        self.engine = create_engine("sqlite:///:memory:")
        Base.metadata.create_all(self.engine)
        session_local = sessionmaker(bind=self.engine)
        self.session = session_local()
        self.repo = ProjectRepositoryImpl(self.session)

    def teardown_method(self):
        """每个测试后清理"""
        self.session.close()
        Base.metadata.drop_all(self.engine)

    def test_save_and_find_by_id(self):
        """测试保存和按 ID 查找"""
        project = Project.create(
            name="test-project",
            repository_url="https://github.com/test/repo.git",
            pipeline_template_id=str(ulid.ULID()),
            git_credential_id=str(ulid.ULID()),
            webhook_secret="my-secret",
            branch_filter="main,develop",
        )

        self.repo.save(project)
        self.session.commit()

        found = self.repo.find_by_id(project.id)

        assert found is not None
        assert found.id == project.id
        assert found.name == project.name
        assert found.webhook_secret == "my-secret"
        assert found.branch_filter == "main,develop"

    def test_find_by_repository_url(self):
        """测试按仓库 URL 查找"""
        repo_url = "https://github.com/test/repo.git"

        # 创建多个项目，其中两个使用相同的仓库 URL
        project1 = Project.create(
            name="project1",
            repository_url=repo_url,
            pipeline_template_id=str(ulid.ULID()),
            git_credential_id=str(ulid.ULID()),
            webhook_secret="secret1",
        )
        project2 = Project.create(
            name="project2",
            repository_url=repo_url,
            pipeline_template_id=str(ulid.ULID()),
            git_credential_id=str(ulid.ULID()),
            webhook_secret="secret2",
        )
        project3 = Project.create(
            name="project3",
            repository_url="https://github.com/other/repo.git",
            pipeline_template_id=str(ulid.ULID()),
            git_credential_id=str(ulid.ULID()),
        )

        self.repo.save(project1)
        self.repo.save(project2)
        self.repo.save(project3)
        self.session.commit()

        # 查找使用相同仓库 URL 的项目
        found = self.repo.find_by_repository_url(repo_url)

        assert len(found) == 2
        assert {p.name for p in found} == {"project1", "project2"}

    def test_find_by_repository_url_not_found(self):
        """测试按仓库 URL 查找（未找到）"""
        found = self.repo.find_by_repository_url("https://github.com/nonexistent/repo.git")

        assert len(found) == 0

    def test_update_webhook_fields(self):
        """测试更新 webhook 字段"""
        project = Project.create(
            name="test-project",
            repository_url="https://github.com/test/repo.git",
            pipeline_template_id=str(ulid.ULID()),
            git_credential_id=str(ulid.ULID()),
        )

        self.repo.save(project)
        self.session.commit()

        # 更新 webhook 字段
        project.update(webhook_secret="new-secret", branch_filter="main")
        self.repo.save(project)
        self.session.commit()

        # 重新查询验证
        found = self.repo.find_by_id(project.id)

        assert found is not None
        assert found.webhook_secret == "new-secret"
        assert found.branch_filter == "main"
