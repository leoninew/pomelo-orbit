"""Pipeline 应用服务"""

import logging
from collections.abc import Callable
from typing import Any

from sqlalchemy.orm import Session

from pomelo_orbit.domain.ci.entities import (
    Artifact,
    Credential,
    Job,
    JobLog,
    PipelineRun,
    PipelineSnapshot,
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
    PipelineSnapshotRepository,
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
from pomelo_orbit.infrastructure.ci.repositories import PipelineRunRepositoryImpl
from pomelo_orbit.infrastructure.ci.variables import (
    VariableError,
    extract_variables,
    merge_declarations,
    merge_variables,
    resolve_stage,
    validate_variables,
)
from pomelo_orbit.infrastructure.ci.workspace import cleanup_workspace, create_workspace
from pomelo_orbit.infrastructure.persistence.database import get_session_factory
from pomelo_orbit.infrastructure.time_utils import utc_now

logger = logging.getLogger(__name__)


class PipelineService:
    def __init__(
        self,
        project_repo: ProjectRepository,
        credential_repo: CredentialRepository,
        template_repo: PipelineTemplateRepository,
        snapshot_repo: PipelineSnapshotRepository,
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
        self.snapshot_repo = snapshot_repo
        self.run_repo = run_repo
        self.artifact_repo = artifact_repo
        self.job_repo = job_repo
        self.job_log_repo = job_log_repo
        self.global_variables = global_variables or {}
        self._session_factory = session_factory
        self._executor_factory = executor_factory

    # ── Project CRUD ──────────────────────────────────────────────────────────

    def list_projects(self, page: int = 1, per_page: int = 20) -> tuple[list[Project], int]:
        return self.project_repo.find_paginated(page=page, per_page=per_page)

    def get_project(self, project_id: str) -> Project:
        project = self.project_repo.find_by_id(project_id)
        if not project:
            raise BusinessError(f"Project {project_id} not found", status_code=404)
        return project

    def create_project(
        self,
        name: str,
        repository_url: str,
        pipeline_snapshot_id: str,
        git_credential_id: str | None = None,
        variable_overrides: dict[str, Any] | None = None,
        default_branch: str = "master",
    ) -> Project:
        if not self.snapshot_repo.find_by_id(pipeline_snapshot_id):
            raise BusinessError(f"PipelineSnapshot {pipeline_snapshot_id} not found", status_code=404)
        if git_credential_id and not self.credential_repo.find_by_id(git_credential_id):
            raise BusinessError(f"Credential {git_credential_id} not found", status_code=404)
        project = Project.create(
            name=name,
            repository_url=repository_url,
            pipeline_snapshot_id=pipeline_snapshot_id,
            git_credential_id=git_credential_id,
            variable_overrides=variable_overrides,
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
        pipeline_snapshot_id: str | None = None,
        git_credential_id: str | None = None,
        default_branch: str | None = None,
    ) -> Project:
        project = self.get_project(project_id)
        if pipeline_snapshot_id and not self.snapshot_repo.find_by_id(pipeline_snapshot_id):
            raise BusinessError(f"PipelineSnapshot {pipeline_snapshot_id} not found", status_code=404)
        project.update(
            name=name,
            repository_url=repository_url,
            variable_overrides=variable_overrides,
            pipeline_snapshot_id=pipeline_snapshot_id,
            git_credential_id=git_credential_id,
            default_branch=default_branch,
        )
        self.project_repo.save(project)
        return project

    def delete_project(self, project_id: str) -> None:
        project = self.get_project(project_id)
        if self.project_repo.has_running_pipelines(project_id):
            raise BusinessError("Project has running pipelines, cannot delete", status_code=409)
        self.project_repo.delete(project)

    # ── Credential CRUD ───────────────────────────────────────────────────────

    def list_credentials(self, page: int = 1, per_page: int = 20) -> tuple[list[Credential], int]:
        return self.credential_repo.find_paginated(page=page, per_page=per_page)

    def get_credential(self, credential_id: str) -> Credential:
        cred = self.credential_repo.find_by_id(credential_id)
        if not cred:
            raise BusinessError(f"Credential {credential_id} not found", status_code=404)
        return cred

    def create_credential(self, name: str, credential_type: str, encrypted_data: str) -> Credential:
        cred = Credential.create(name=name, type=CredentialType(credential_type), encrypted_data=encrypted_data)
        self.credential_repo.save(cred)
        return cred

    def update_credential(
        self, credential_id: str, name: str | None = None, encrypted_data: str | None = None
    ) -> Credential:
        cred = self.get_credential(credential_id)
        if name is not None:
            cred.name = name
        if encrypted_data is not None:
            cred.encrypted_data = encrypted_data
        self.credential_repo.save(cred)
        return cred

    def delete_credential(self, credential_id: str) -> None:
        cred = self.get_credential(credential_id)
        if self.credential_repo.is_referenced_by_projects(credential_id):
            raise BusinessError("Credential is referenced by projects, cannot delete", status_code=409)
        self.credential_repo.delete(cred)

    # ── PipelineTemplate CRUD ─────────────────────────────────────────────────

    def list_templates(self, page: int = 1, per_page: int = 20) -> tuple[list[PipelineTemplate], int]:
        return self.template_repo.find_paginated(page=page, per_page=per_page)

    def list_templates_with_latest_version(
        self, page: int = 1, per_page: int = 20
    ) -> tuple[list[tuple[PipelineTemplate, int | None]], int]:
        """列出模板，同时附带每个模板的最新快照版本号（单次批量查询）"""
        templates, total = self.template_repo.find_paginated(page=page, per_page=per_page)
        if not templates:
            return [], total
        latest_versions = self.snapshot_repo.find_latest_versions([t.id for t in templates])
        return [(t, latest_versions.get(t.id)) for t in templates], total

    def get_template(self, template_id: str) -> PipelineTemplate:
        tmpl = self.template_repo.find_by_id(template_id)
        if not tmpl:
            raise BusinessError(f"PipelineTemplate {template_id} not found", status_code=404)
        return tmpl

    def get_template_latest_version(self, template_id: str) -> int | None:
        """获取模板最新快照版本号"""
        versions = self.snapshot_repo.find_latest_versions([template_id])
        return versions.get(template_id)

    def create_template(
        self,
        name: str,
        stages: list,
        description: str = "",
        variable_declarations: list | None = None,
    ) -> PipelineTemplate:
        decls = [VariableDeclaration(**d) if isinstance(d, dict) else d for d in (variable_declarations or [])]
        tmpl = PipelineTemplate.create(
            name=name,
            stages=stages,
            variable_declarations=self._sync_declarations(stages, decls),
            description=description,
        )
        self.template_repo.save(tmpl)
        # 同时创建 version=1 快照
        snapshot = PipelineSnapshot.create(tmpl, version=1)
        self.snapshot_repo.save(snapshot)
        return tmpl

    def update_template(
        self,
        template_id: str,
        name: str | None = None,
        description: str | None = None,
        stages: list | None = None,
        variable_declarations: list | None = None,
    ) -> PipelineTemplate:
        tmpl = self.get_template(template_id)
        decls = None
        if variable_declarations is not None:
            decls = [VariableDeclaration(**d) if isinstance(d, dict) else d for d in variable_declarations]
        tmpl.update(name=name, description=description, stages=stages, variable_declarations=decls)
        # 如果 stages 有更新，重新同步变量声明
        if stages is not None:
            tmpl.variable_declarations = self._sync_declarations(tmpl.stages, tmpl.variable_declarations)
        self.template_repo.save(tmpl)
        # 创建新版本快照
        next_version = self.snapshot_repo.get_next_version(template_id)
        snapshot = PipelineSnapshot.create(tmpl, version=next_version)
        self.snapshot_repo.save(snapshot)
        return tmpl

    def delete_template(self, template_id: str) -> None:
        tmpl = self.get_template(template_id)
        if self.template_repo.is_referenced_by_projects(template_id):
            raise BusinessError("Template is referenced by projects, cannot delete", status_code=409)
        self.template_repo.delete(tmpl)

    def _sync_declarations(self, stages: list, existing: list[VariableDeclaration]) -> list[VariableDeclaration]:
        """从 stages 提取变量占位符，与现有声明合并"""
        extracted = extract_variables(stages)
        return merge_declarations(extracted, existing)

    # ── PipelineSnapshot ──────────────────────────────────────────────────────

    def list_template_snapshots(self, template_id: str) -> list[PipelineSnapshot]:
        self.get_template(template_id)  # 验证模板存在
        return self.snapshot_repo.find_by_template(template_id)

    def get_snapshot(self, snapshot_id: str) -> PipelineSnapshot:
        snapshot = self.snapshot_repo.find_by_id(snapshot_id)
        if not snapshot:
            raise BusinessError(f"PipelineSnapshot {snapshot_id} not found", status_code=404)
        return snapshot

    # ── PipelineRun ───────────────────────────────────────────────────────────

    def list_runs(
        self, page: int = 1, per_page: int = 20, project_id: str | None = None
    ) -> tuple[list[PipelineRun], int]:
        if project_id:
            self.get_project(project_id)
        return self.run_repo.find_paginated_with_filters(page=page, per_page=per_page, project_id=project_id)

    def get_run(self, run_id: str) -> PipelineRun:
        run = self.run_repo.find_by_id(run_id)
        if not run:
            raise BusinessError(f"PipelineRun {run_id} not found", status_code=404)
        return run

    def list_artifacts(self, run_id: str) -> list[Artifact]:
        self.get_run(run_id)
        return self.artifact_repo.find_by_run(run_id)

    def list_jobs(self, run_id: str) -> list[Job]:
        self.get_run(run_id)
        return self.job_repo.find_by_run(run_id)

    def get_job_log(self, job_id: str) -> JobLog | None:
        return self.job_log_repo.find_by_job(job_id)

    def cancel_run(self, run_id: str) -> PipelineRun:
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
        project = self.get_project(project_id)
        snapshot = self.get_snapshot(project.pipeline_snapshot_id)
        declarations = snapshot.variable_declarations_snapshot

        builtin = {
            "REPOSITORY_URL": project.repository_url,
            "DEFAULT_BRANCH": project.default_branch,
            "GIT_CREDENTIAL_ID": project.git_credential_id or "",
            "trigger_ref": trigger_ref,
            "trigger_type": trigger.value,
            "project_name": project.name,
        }

        merged = merge_variables(
            global_vars=self.global_variables,
            project_vars=project.variable_overrides,
            runtime_vars=runtime_variables or {},
            declarations=declarations,
            builtin_vars=builtin,
        )

        try:
            validate_variables(merged, declarations)
        except VariableError as e:
            raise BusinessError(str(e), status_code=400) from e

        run = PipelineRun.create(
            project_id=project_id,
            pipeline_snapshot_id=snapshot.id,
            trigger=trigger,
            trigger_ref=trigger_ref,
            variables_snapshot=merged,  # 存明文，脱敏是展示层的事
        )
        self.run_repo.save(run)
        self.run_repo.commit()

        logger.info(
            f"Pipeline triggered: project={project.name}, run={run.id}, trigger={trigger.value}, ref={trigger_ref}"
        )
        return run, project, merged

    def create_retry_run(self, run_id: str) -> tuple[PipelineRun, Project, dict[str, Any]]:
        original = self.get_run(run_id)
        if original.status not in {PipelineRunStatus.FAILED, PipelineRunStatus.SUCCESS}:
            raise BusinessError(f"Cannot retry run with status {original.status.value}", status_code=400)

        # variables_snapshot 存的是明文，可以直接复用
        new_run = PipelineRun.create(
            project_id=original.project_id,
            pipeline_snapshot_id=original.pipeline_snapshot_id,
            trigger=original.trigger,
            trigger_ref=original.trigger_ref,
            variables_snapshot=original.variables_snapshot,
            retry_of=original.id,
        )
        self.run_repo.save(new_run)
        self.run_repo.commit()

        logger.info(f"Pipeline retry triggered: original_run={run_id}, new_run={new_run.id}")
        project = self.get_project(original.project_id)
        return new_run, project, original.variables_snapshot

    async def execute_run(self, run: PipelineRun, project: Project, variables: dict[str, Any]) -> None:
        """执行 pipeline run（由 BackgroundTasks 调用，使用独立 session）"""
        session_factory = self._session_factory or get_session_factory()

        # 加载快照并解析变量
        snapshot = self.snapshot_repo.find_by_id(run.pipeline_snapshot_id)
        if not snapshot:
            logger.error(f"Snapshot not found: run={run.id}, snapshot_id={run.pipeline_snapshot_id}")
            with session_factory() as session:
                run_repo = PipelineRunRepositoryImpl(session)
                run.complete_failed()
                run_repo.save(run)
                session.commit()
            return

        try:
            resolved_stages = [resolve_stage(s, variables) for s in snapshot.stages_snapshot]
        except Exception as e:
            logger.error(f"Stage resolution failed: run={run.id}, error={e}")
            with session_factory() as session:
                run_repo = PipelineRunRepositoryImpl(session)
                run.complete_failed()
                run_repo.save(run)
                session.commit()
            return

        workspace_path, artifacts_path = create_workspace(run.id)
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
                success = await executor.execute(context, resolved_stages)
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
                logger.info(f"Pipeline finished: run={run.id}, status={run.status}, finished_at={utc_now()}")
