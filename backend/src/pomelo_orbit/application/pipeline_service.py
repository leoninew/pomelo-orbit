"""Pipeline 应用服务 - 处理 pipeline 触发和查询"""

import logging
import secrets
from collections.abc import Callable
from typing import Any

from sqlalchemy.orm import Session

from pomelo_orbit.domain.ci.entities import (
    Artifact,
    Credential,
    Job,
    JobLog,
    PipelineRun,
    PipelineTemplate,
    Project,
)
from pomelo_orbit.domain.ci.executor import ExecutionContext, PipelineExecutor
from pomelo_orbit.domain.ci.repositories import (
    ArtifactRepository,
    CredentialRepository,
    JobLogRepository,
    JobRepository,
    PipelineRunRepository,
    PipelineTemplateRepository,
    ProjectRepository,
)
from pomelo_orbit.domain.ci.value_objects import (
    CredentialType,
    PipelineRunStatus,
    PipelineRunTrigger,
    VariableDeclaration,
)
from pomelo_orbit.domain.exceptions import BusinessError
from pomelo_orbit.infrastructure.ci.parser import PipelineParseError, parse_pipeline_yaml
from pomelo_orbit.infrastructure.ci.repositories import PipelineRunRepositoryImpl
from pomelo_orbit.infrastructure.ci.variables import (
    VariableError,
    mask_secrets,
    merge_variables,
    render_template,
    validate_variables,
)
from pomelo_orbit.infrastructure.ci.workspace import cleanup_workspace, create_workspace
from pomelo_orbit.infrastructure.persistence.database import get_session_factory
from pomelo_orbit.infrastructure.time_utils import utc_now

logger = logging.getLogger(__name__)


class PipelineService:
    """Pipeline 应用服务"""

    def __init__(
        self,
        project_repo: ProjectRepository,
        credential_repo: CredentialRepository,
        template_repo: PipelineTemplateRepository,
        run_repo: PipelineRunRepository,
        artifact_repo: ArtifactRepository,
        job_repo: JobRepository,
        job_log_repo: JobLogRepository,
        session_factory: Any,
        executor_factory: Callable[[Session], PipelineExecutor],
        global_variables: dict[str, Any] | None = None,
    ):
        self.project_repo = project_repo
        self.credential_repo = credential_repo
        self.template_repo = template_repo
        self.run_repo = run_repo
        self.artifact_repo = artifact_repo
        self.job_repo = job_repo
        self.job_log_repo = job_log_repo
        self.global_variables = global_variables or {}
        self._session_factory = session_factory
        self._executor_factory = executor_factory

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
        default_branch: str = "master",
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
            default_branch=default_branch,
        )
        self.project_repo.save(project)
        return project

    def update_project(
        self,
        project_id: str,
        name: str | None = None,
        repository_url: str | None = None,
        variable_overrides: dict[str, Any] | None = None,
        pipeline_template_id: str | None = None,
        git_credential_id: str | None = None,
        branch_filter: str | None = None,
        default_branch: str | None = None,
    ) -> Project:
        """更新项目"""
        project = self.get_project(project_id)
        if pipeline_template_id and not self.template_repo.find_by_id(pipeline_template_id):
            raise BusinessError(f"PipelineTemplate {pipeline_template_id} not found", status_code=404)
        project.update(
            name=name,
            repository_url=repository_url,
            variable_overrides=variable_overrides,
            pipeline_template_id=pipeline_template_id,
            git_credential_id=git_credential_id,
            branch_filter=branch_filter,
            default_branch=default_branch,
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

    def list_credentials(self, page: int = 1, per_page: int = 20) -> tuple[list[Credential], int]:
        """列出凭据（分页）"""
        return self.credential_repo.find_paginated(page=page, per_page=per_page)

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

    def list_templates(self, page: int = 1, per_page: int = 20) -> tuple[list[PipelineTemplate], int]:
        """列出模板（分页）"""
        return self.template_repo.find_paginated(page=page, per_page=per_page)

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
        self, page: int = 1, per_page: int = 20, project_id: str | None = None
    ) -> tuple[list[PipelineRun], int]:
        """列出 pipeline runs（可按项目过滤，有 project_id 时校验项目存在）"""
        if project_id:
            self.get_project(project_id)
        return self.run_repo.find_paginated_with_filters(page=page, per_page=per_page, project_id=project_id)

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

    def list_jobs(self, run_id: str) -> list[Job]:
        """列出 pipeline run 的所有 jobs"""
        self.get_run(run_id)
        return self.job_repo.find_by_run(run_id)

    def get_job_log(self, job_id: str) -> JobLog | None:
        """获取 job 日志"""
        return self.job_log_repo.find_by_job(job_id)

    def cancel_run(self, run_id: str) -> PipelineRun:
        """取消 pipeline run（waiting 或 running 状态）"""
        run = self.get_run(run_id)
        try:
            run.cancel()
        except ValueError as e:
            raise BusinessError(str(e), status_code=400) from e
        self.run_repo.save(run)
        self.run_repo.commit()
        return run

    def create_run(
        self,
        project_id: str,
        trigger: PipelineRunTrigger,
        trigger_ref: str,
        runtime_variables: dict[str, Any] | None = None,
    ) -> tuple[PipelineRun, Project, dict[str, Any]]:
        """
        校验、合并变量、创建 PipelineRun 记录并持久化。
        返回 (run, project, merged_vars) 供后续 execute_run 使用。
        """
        project = self.get_project(project_id)
        template = self.get_template(project.pipeline_template_id)

        merged_vars = merge_variables(
            global_vars=self.global_variables,
            project_vars=project.variable_overrides,
            runtime_vars=runtime_variables or {},
        )
        merged_vars.update({
            "trigger_ref": trigger_ref,
            "trigger_type": trigger.value,
            "project_name": project.name,
        })

        try:
            validate_variables(merged_vars, template.variable_declarations)
        except VariableError as e:
            raise BusinessError(str(e), status_code=400) from e

        variables_snapshot = mask_secrets(merged_vars, template.variable_declarations)

        run = PipelineRun.create(
            project_id=project_id,
            trigger=trigger,
            trigger_ref=trigger_ref,
            resolved_pipeline=template.content,
            variables_snapshot=variables_snapshot,
        )
        self.run_repo.save(run)
        self.run_repo.commit()

        logger.info(f"Pipeline triggered: project={project.name}, run={run.id}, trigger={trigger.value}, ref={trigger_ref}")
        return run, project, merged_vars

    def create_retry_run(self, run_id: str) -> tuple[PipelineRun, Project, dict[str, Any]]:
        """
        基于已有 run 创建重试 run 并持久化。
        返回 (new_run, project, variables) 供后续 execute_run 使用。
        """
        original_run = self.get_run(run_id)

        if original_run.status not in {PipelineRunStatus.FAILED, PipelineRunStatus.SUCCESS}:
            raise BusinessError(
                f"Cannot retry run with status {original_run.status.value}",
                status_code=400,
            )

        new_run = PipelineRun.create(
            project_id=original_run.project_id,
            trigger=original_run.trigger,
            trigger_ref=original_run.trigger_ref,
            resolved_pipeline=original_run.resolved_pipeline,
            variables_snapshot=original_run.variables_snapshot,
            retry_of=original_run.id,
        )
        self.run_repo.save(new_run)
        self.run_repo.commit()

        logger.info(f"Pipeline retry triggered: original_run={run_id}, new_run={new_run.id}")
        project = self.get_project(original_run.project_id)
        return new_run, project, original_run.variables_snapshot

    async def execute_run(
        self,
        run: PipelineRun,
        project: Project,
        variables: dict[str, Any],
    ) -> None:
        """执行 pipeline run（由 BackgroundTasks 调用，使用独立 session）"""
        session_factory = self._session_factory or get_session_factory()

        try:
            # 先渲染变量，再解析 YAML
            rendered = render_template(run.resolved_pipeline, variables)
            definition = parse_pipeline_yaml(rendered)
        except (PipelineParseError, VariableError) as e:
            logger.error(f"Pipeline parse failed: run={run.id}, error={e}")
            with session_factory() as session:
                run_repo = PipelineRunRepositoryImpl(session)
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
            retry_of=run.retry_of,
        )

        with session_factory() as session:
            run_repo = PipelineRunRepositoryImpl(session)
            executor = self._executor_factory(session)

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
