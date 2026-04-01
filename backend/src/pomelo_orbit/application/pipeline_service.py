"""Pipeline 应用服务 - 处理 pipeline 触发和查询"""

import asyncio
import logging
import secrets
from typing import Any

from pomelo_orbit.domain.ci.entities import (
    Artifact,
    Credential,
    PipelineRun,
    PipelineTemplate,
    Project,
)
from pomelo_orbit.domain.ci.executor import ExecutionContext
from pomelo_orbit.domain.ci.value_objects import (
    CredentialType,
    PipelineRunTrigger,
    VariableDeclaration,
)
from pomelo_orbit.domain.exceptions import BusinessError
from pomelo_orbit.infrastructure.ci.container import ContainerExecutor
from pomelo_orbit.infrastructure.ci.executor_impl import PipelineExecutorImpl
from pomelo_orbit.infrastructure.ci.parser import PipelineParseError, parse_pipeline_yaml
from pomelo_orbit.infrastructure.ci.repositories import (
    ArtifactRepository,
    CredentialRepository,
    JobLogRepository,
    JobRepository,
    PipelineRunRepository,
    PipelineTemplateRepository,
    ProjectRepository,
)
from pomelo_orbit.infrastructure.ci.variables import (
    VariableError,
    mask_secrets,
    merge_variables,
    validate_variables,
)
from pomelo_orbit.infrastructure.ci.workspace import cleanup_workspace, create_workspace
from pomelo_orbit.infrastructure.persistence.database import get_session_factory
from pomelo_orbit.infrastructure.time_utils import utc_now

logger = logging.getLogger(__name__)

# 模块级 task 集合，防止后台 task 被 GC 回收
_background_tasks: set[asyncio.Task] = set()


class PipelineService:
    """Pipeline 应用服务"""

    def __init__(
        self,
        project_repo: ProjectRepository,
        credential_repo: CredentialRepository,
        template_repo: PipelineTemplateRepository,
        run_repo: PipelineRunRepository,
        artifact_repo: ArtifactRepository,
        global_variables: dict[str, Any] | None = None,
        session_factory: Any = None,
    ):
        self.project_repo = project_repo
        self.credential_repo = credential_repo
        self.template_repo = template_repo
        self.run_repo = run_repo
        self.artifact_repo = artifact_repo
        self.global_variables = global_variables or {}
        self._session_factory = session_factory

    # -------------------------------------------------------------------------
    # Project CRUD
    # -------------------------------------------------------------------------

    def list_projects(self, page: int = 1, per_page: int = 20) -> tuple[list[Project], int]:
        """列出项目（分页）"""
        return self.project_repo.find_paginated(page=page, per_page=per_page)

    def get_project(self, project_id: str) -> Project:
        """获取项目详情"""
        project = self.project_repo.find_by_id(project_id)
        if not project:
            raise BusinessError(f"Project {project_id} not found", status_code=404)
        return project

    def create_project(
        self,
        name: str,
        repository_url: str,
        pipeline_template_id: str,
        git_credential_id: str,
        variable_overrides: dict[str, Any] | None = None,
        branch_filter: str | None = None,
        enable_webhook: bool = True,
    ) -> Project:
        """创建项目，自动生成 webhook_secret"""
        # 验证模板和凭据存在
        if not self.template_repo.find_by_id(pipeline_template_id):
            raise BusinessError(f"PipelineTemplate {pipeline_template_id} not found", status_code=404)
        if not self.credential_repo.find_by_id(git_credential_id):
            raise BusinessError(f"Credential {git_credential_id} not found", status_code=404)

        webhook_secret = secrets.token_urlsafe(32) if enable_webhook else None

        project = Project.create(
            name=name,
            repository_url=repository_url,
            pipeline_template_id=pipeline_template_id,
            git_credential_id=git_credential_id,
            variable_overrides=variable_overrides,
            webhook_secret=webhook_secret,
            branch_filter=branch_filter,
        )
        self.project_repo.save(project)
        return project

    def update_project(
        self,
        project_id: str,
        name: str | None = None,
        variable_overrides: dict[str, Any] | None = None,
        pipeline_template_id: str | None = None,
        branch_filter: str | None = None,
    ) -> Project:
        """更新项目"""
        project = self.get_project(project_id)
        if pipeline_template_id and not self.template_repo.find_by_id(pipeline_template_id):
            raise BusinessError(f"PipelineTemplate {pipeline_template_id} not found", status_code=404)
        project.update(
            name=name,
            variable_overrides=variable_overrides,
            pipeline_template_id=pipeline_template_id,
            branch_filter=branch_filter,
        )
        self.project_repo.save(project)
        return project

    def delete_project(self, project_id: str) -> None:
        """删除项目"""
        project = self.get_project(project_id)
        if self.project_repo.has_running_pipelines(project_id):
            raise BusinessError("Project has running pipelines, cannot delete", status_code=409)
        self.project_repo.delete(project)

    def get_webhook_config(self, project_id: str, api_base_url: str) -> dict:
        """获取项目 webhook 配置"""
        project = self.get_project(project_id)
        return {
            "url": f"{api_base_url}/api/v1/ci/webhooks/git",
            "secret": project.webhook_secret,
            "events": ["push", "release"],
        }

    def regenerate_webhook_secret(self, project_id: str) -> Project:
        """重新生成 webhook secret"""
        project = self.get_project(project_id)
        project.update(webhook_secret=secrets.token_urlsafe(32))
        self.project_repo.save(project)
        return project

    # -------------------------------------------------------------------------
    # Credential CRUD
    # -------------------------------------------------------------------------

    def list_credentials(self) -> list[Credential]:
        """列出所有凭据"""
        return self.credential_repo.find_all()

    def get_credential(self, credential_id: str) -> Credential:
        """获取凭据"""
        cred = self.credential_repo.find_by_id(credential_id)
        if not cred:
            raise BusinessError(f"Credential {credential_id} not found", status_code=404)
        return cred

    def create_credential(self, name: str, credential_type: str, encrypted_data: str) -> Credential:
        """创建凭据"""
        cred = Credential.create(
            name=name,
            type=CredentialType(credential_type),
            encrypted_data=encrypted_data,
        )
        self.credential_repo.save(cred)
        return cred

    def delete_credential(self, credential_id: str) -> None:
        """删除凭据"""
        cred = self.get_credential(credential_id)
        if self.credential_repo.is_referenced_by_projects(credential_id):
            raise BusinessError("Credential is referenced by projects, cannot delete", status_code=409)
        self.credential_repo.delete(cred)

    # -------------------------------------------------------------------------
    # PipelineTemplate CRUD
    # -------------------------------------------------------------------------

    def list_templates(self) -> list[PipelineTemplate]:
        """列出所有模板"""
        return self.template_repo.find_all()

    def get_template(self, template_id: str) -> PipelineTemplate:
        """获取模板"""
        tmpl = self.template_repo.find_by_id(template_id)
        if not tmpl:
            raise BusinessError(f"PipelineTemplate {template_id} not found", status_code=404)
        return tmpl

    def create_template(
        self,
        name: str,
        content: str,
        description: str = "",
        variable_declarations: list | None = None,
    ) -> PipelineTemplate:
        """创建模板"""
        decls = [VariableDeclaration(**d) for d in (variable_declarations or [])]
        tmpl = PipelineTemplate.create(
            name=name,
            content=content,
            variable_declarations=decls,
            description=description,
        )
        self.template_repo.save(tmpl)
        return tmpl

    def update_template(
        self,
        template_id: str,
        name: str | None = None,
        description: str | None = None,
        content: str | None = None,
        variable_declarations: list | None = None,
    ) -> PipelineTemplate:
        """更新模板"""
        tmpl = self.get_template(template_id)
        decls = None
        if variable_declarations is not None:
            decls = [VariableDeclaration(**d) for d in variable_declarations]
        tmpl.update(name=name, description=description, content=content, variable_declarations=decls)
        self.template_repo.save(tmpl)
        return tmpl

    def delete_template(self, template_id: str) -> None:
        """删除模板"""
        tmpl = self.get_template(template_id)
        if self.template_repo.is_referenced_by_projects(template_id):
            raise BusinessError("Template is referenced by projects, cannot delete", status_code=409)
        self.template_repo.delete(tmpl)

    # -------------------------------------------------------------------------
    # PipelineRun
    # -------------------------------------------------------------------------

    def list_runs(
        self, project_id: str, page: int = 1, per_page: int = 20
    ) -> tuple[list[PipelineRun], int]:
        """列出项目的 pipeline runs"""
        self.get_project(project_id)
        return self.run_repo.find_by_project(project_id, page=page, per_page=per_page)

    def get_run(self, run_id: str) -> PipelineRun:
        """获取 pipeline run 详情"""
        run = self.run_repo.find_by_id(run_id)
        if not run:
            raise BusinessError(f"PipelineRun {run_id} not found", status_code=404)
        return run

    def list_artifacts(self, run_id: str) -> list[Artifact]:
        """列出 pipeline run 的所有制品"""
        self.get_run(run_id)
        return self.artifact_repo.find_by_run(run_id)

    async def trigger_pipeline(
        self,
        project_id: str,
        trigger: PipelineRunTrigger,
        trigger_ref: str,
        runtime_variables: dict[str, Any] | None = None,
    ) -> PipelineRun:
        """
        触发 pipeline 执行

        Args:
            project_id: 项目 ID
            trigger: 触发方式
            trigger_ref: 分支/tag/commit sha
            runtime_variables: 运行时临时变量

        Returns:
            PipelineRun 实例
        """
        project = self.get_project(project_id)
        template = self.get_template(project.pipeline_template_id)

        # 合并变量
        merged_vars = merge_variables(
            global_vars=self.global_variables,
            project_vars=project.variable_overrides,
            runtime_vars=runtime_variables or {},
        )

        # 注入内置变量
        merged_vars.update(
            {
                "trigger_ref": trigger_ref,
                "trigger_type": trigger.value,
                "project_name": project.name,
            }
        )

        # 校验必填变量
        try:
            validate_variables(merged_vars, template.variable_declarations)
        except VariableError as e:
            raise BusinessError(str(e), status_code=400) from e

        # 脱敏快照
        variables_snapshot = mask_secrets(merged_vars, template.variable_declarations)

        # 创建 PipelineRun 记录
        run = PipelineRun.create(
            project_id=project_id,
            trigger=trigger,
            trigger_ref=trigger_ref,
            resolved_pipeline=template.content,
            variables_snapshot=variables_snapshot,
        )
        self.run_repo.save(run)

        logger.info(
            f"Pipeline triggered: project={project.name}, run={run.id}, "
            f"trigger={trigger.value}, ref={trigger_ref}"
        )

        # 异步执行（不等待完成），用模块级集合保存 task 引用防止被 GC
        task = asyncio.create_task(self._execute_run(run, project, merged_vars))
        _background_tasks.add(task)
        task.add_done_callback(_background_tasks.discard)

        return run

    async def _execute_run(
        self,
        run: PipelineRun,
        project: Project,
        variables: dict[str, Any],
    ) -> None:
        """执行 pipeline run（后台任务，使用独立 session 避免请求 session 关闭问题）"""
        session_factory = self._session_factory or get_session_factory()

        try:
            # 解析 pipeline 定义（不需要 DB）
            definition = parse_pipeline_yaml(run.resolved_pipeline)
        except PipelineParseError as e:
            logger.error(f"Pipeline parse failed: run={run.id}, error={e}")
            with session_factory() as session:
                run_repo = PipelineRunRepository(session)
                run.complete_failed()
                run_repo.save(run)
                session.commit()
            return

        # 创建工作目录
        workspace_path, artifacts_path = create_workspace(run.id)

        # 构建执行上下文
        context = ExecutionContext(
            run_id=run.id,
            project_id=project.id,
            repository_url=project.repository_url,
            credential_id=project.git_credential_id,
            variables=variables,
            workspace_path=str(workspace_path),
            artifacts_path=str(artifacts_path),
        )

        with session_factory() as session:
            run_repo = PipelineRunRepository(session)
            job_repo = JobRepository(session)
            job_log_repo = JobLogRepository(session)
            executor = PipelineExecutorImpl(
                job_repo=job_repo,
                job_log_repo=job_log_repo,
                container_executor=ContainerExecutor(),
                artifact_repo=ArtifactRepository(session),
            )

            run.start()
            run_repo.save(run)
            session.commit()

            try:
                success = await executor.execute(context, definition)
                if success:
                    run.complete_success()
                else:
                    run.complete_failed()
            except Exception as e:
                logger.error(f"Pipeline execution error: run={run.id}, error={e}", exc_info=True)
                run.complete_failed()
            finally:
                run_repo.save(run)
                session.commit()
                cleanup_workspace(run.id)
                logger.info(
                    f"Pipeline finished: run={run.id}, status={run.status}, "
                    f"finished_at={utc_now()}"
                )
