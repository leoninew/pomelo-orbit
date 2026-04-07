"""Pipeline 应用服务"""

import logging
from collections.abc import Callable
from typing import Any

from sqlalchemy.orm import Session

from pomelo_orbit.domain.cd.value_objects import TaskStatus
from pomelo_orbit.domain.ci.entities import (
    Artifact,
    Credential,
    PipelineRun,
    PipelineSnapshot,
    PipelineStage,
    PipelineTemplate,
    Project,
    ProjectWebhook,
    StageLog,
    StageRun,
)
from pomelo_orbit.domain.ci.executor import ExecutionContext, PipelineExecutor
from pomelo_orbit.domain.ci.repositories import (
    ArtifactRepository,
    CredentialRepository,
    PipelineRunRepository,
    PipelineSnapshotRepository,
    PipelineStageRepository,
    PipelineTemplateRepository,
    ProjectRepository,
    ProjectWebhookRepository,
    StageLogRepository,
    StageRunRepository,
)
from pomelo_orbit.domain.ci.value_objects import (
    CredentialType,
    PipelineRunTrigger,
    StageOrchestration,
    VariableDeclaration,
)
from pomelo_orbit.domain.exceptions import BusinessError
from pomelo_orbit.infrastructure.ci.executor_impl import cancel_task
from pomelo_orbit.infrastructure.ci.repositories import PipelineRunRepositoryImpl
from pomelo_orbit.infrastructure.ci.variables import (
    VariableError,
    extract_variables,
    mask_secrets,
    merge_declarations,
    merge_variables,
    resolve_stage,
    validate_variables,
)
from pomelo_orbit.infrastructure.ci.workspace import cleanup_project, cleanup_run, create_workspace
from pomelo_orbit.infrastructure.persistence.database import get_session_factory
from pomelo_orbit.infrastructure.security import SecurityService
from pomelo_orbit.infrastructure.time_utils import utc_now

logger = logging.getLogger(__name__)


class PipelineService:
    def __init__(
        self,
        project_repo: ProjectRepository,
        credential_repo: CredentialRepository,
        template_repo: PipelineTemplateRepository,
        stage_repo: PipelineStageRepository,
        snapshot_repo: PipelineSnapshotRepository,
        run_repo: PipelineRunRepository,
        artifact_repo: ArtifactRepository,
        stage_run_repo: StageRunRepository,
        stage_log_repo: StageLogRepository,
        webhook_repo: ProjectWebhookRepository,
        session_factory: Any,
        executor_factory: Callable[[Session], PipelineExecutor],
        security_service: SecurityService,
        global_variables: dict[str, Any] | None = None,
    ):
        self.project_repo = project_repo
        self.credential_repo = credential_repo
        self.template_repo = template_repo
        self.stage_repo = stage_repo
        self.snapshot_repo = snapshot_repo
        self.run_repo = run_repo
        self.artifact_repo = artifact_repo
        self.stage_run_repo = stage_run_repo
        self.stage_log_repo = stage_log_repo
        self.webhook_repo = webhook_repo
        self.security_service = security_service
        self.global_variables = global_variables or {}
        self._session_factory = session_factory
        self._executor_factory = executor_factory

    # ── Project CRUD ──────────────────────────────────────────────────────────

    def list_projects(
        self, page: int = 1, per_page: int = 20
    ) -> tuple[list[Project], list[str | None], int]:
        """List projects with credential names aligned to the project list."""
        projects, total = self.project_repo.find_paginated(page=page, per_page=per_page)
        credential_names = self._get_credential_names_for_projects(projects)
        return projects, credential_names, total

    def _get_credential_names_for_projects(self, projects: list[Project]) -> list[str | None]:
        """Get credential names for a list of projects, preserving order."""
        credential_ids = [p.git_credential_id for p in projects]
        unique_ids = {cid for cid in credential_ids if cid is not None}
        if not unique_ids:
            return [None] * len(projects)
        credentials = {c.id: c.name for c in self.credential_repo.find_all() if c.id in unique_ids}
        return [credentials.get(cid) if cid is not None else None for cid in credential_ids]

    def _get_credential_name(self, credential_id: str | None) -> str | None:
        """Get credential name by ID."""
        if not credential_id:
            return None
        cred = self.credential_repo.find_by_id(credential_id)
        return cred.name if cred else None

    def get_project(self, project_id: str) -> Project:
        """Get project entity (internal use)."""
        project = self.project_repo.find_by_id(project_id)
        if not project:
            raise BusinessError(f"Project {project_id} not found", status_code=404)
        return project

    def get_project_with_credential_name(self, project_id: str) -> tuple[Project, str | None]:
        """Get project with its credential name (for API responses)."""
        project = self.get_project(project_id)
        credential_name = self._get_credential_name(project.git_credential_id)
        return project, credential_name

    def create_project(
        self,
        name: str,
        code: str,
        repository_url: str,
        git_credential_id: str | None = None,
        variable_overrides: dict[str, Any] | None = None,
        default_branch: str = "master",
    ) -> Project:
        if git_credential_id and not self.credential_repo.find_by_id(git_credential_id):
            raise BusinessError(f"Credential {git_credential_id} not found", status_code=404)
        if self.project_repo.find_by_code(code):
            raise BusinessError(f"Project code '{code}' already exists", status_code=409)

        # 初始化变量覆盖，自动添加内置变量
        overrides = dict(variable_overrides or {})
        overrides["project_repository_url"] = repository_url
        overrides["project_trigger_ref"] = default_branch

        project = Project.create(
            name=name,
            code=code,
            repository_url=repository_url,
            git_credential_id=git_credential_id,
            variable_overrides=overrides,
            default_branch=default_branch,
        )
        self.project_repo.save(project)
        return project

    def create_project_with_credential_name(
        self,
        name: str,
        code: str,
        repository_url: str,
        git_credential_id: str | None = None,
        variable_overrides: dict[str, Any] | None = None,
        default_branch: str = "master",
    ) -> tuple[Project, str | None]:
        """Create project and return with its credential name (for API responses)."""
        project, _ = self._create_project_internal(
            name=name,
            code=code,
            repository_url=repository_url,
            git_credential_id=git_credential_id,
            variable_overrides=variable_overrides,
            default_branch=default_branch,
        )
        credential_name = self._get_credential_name(project.git_credential_id)
        return project, credential_name

    def _create_project_internal(
        self,
        name: str,
        code: str,
        repository_url: str,
        git_credential_id: str | None = None,
        variable_overrides: dict[str, Any] | None = None,
        default_branch: str = "master",
    ) -> tuple[Project, str | None]:
        """Internal implementation of create_project (returns credential name for convenience)."""
        if git_credential_id and not self.credential_repo.find_by_id(git_credential_id):
            raise BusinessError(f"Credential {git_credential_id} not found", status_code=404)
        if self.project_repo.find_by_code(code):
            raise BusinessError(f"Project code '{code}' already exists", status_code=409)

        # 初始化变量覆盖，自动添加内置变量
        overrides = dict(variable_overrides or {})
        overrides["project_repository_url"] = repository_url
        overrides["project_trigger_ref"] = default_branch

        project = Project.create(
            name=name,
            code=code,
            repository_url=repository_url,
            git_credential_id=git_credential_id,
            variable_overrides=overrides,
            default_branch=default_branch,
        )
        self.project_repo.save(project)
        credential_name = self._get_credential_name(git_credential_id)
        return project, credential_name

    def update_project(
        self,
        project_id: str,
        name: str | None = None,
        repository_url: str | None = None,
        variable_overrides: dict[str, Any] | None = None,
        git_credential_id: str | None = None,
        default_branch: str | None = None,
    ) -> Project:
        project = self.get_project(project_id)

        # 合并变量覆盖：保留现有变量，应用用户提供的变量，更新内置变量
        final_overrides = dict(project.variable_overrides)
        if variable_overrides is not None:
            final_overrides.update(variable_overrides)

        # 自动更新内置变量
        if repository_url is not None:
            final_overrides["project_repository_url"] = repository_url
        if default_branch is not None:
            final_overrides["project_trigger_ref"] = default_branch

        project.update(
            name=name,
            repository_url=repository_url,
            variable_overrides=final_overrides,
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
        cleanup_project(project.code)

    # ── ProjectWebhook CRUD ───────────────────────────────────────────────────

    def list_webhooks(self, project_id: str) -> list[ProjectWebhook]:
        self.get_project(project_id)
        return self.webhook_repo.find_by_project(project_id)

    def get_webhook(self, webhook_id: str) -> ProjectWebhook:
        wh = self.webhook_repo.find_by_id(webhook_id)
        if not wh:
            raise BusinessError(f"Webhook {webhook_id} not found", status_code=404)
        return wh

    def create_webhook(
        self,
        project_id: str,
        name: str,
        template_id: str,
        plain_secret: str,
        branch_filter: str | None = None,
    ) -> ProjectWebhook:
        self.get_project(project_id)
        if not self.template_repo.find_by_id(template_id):
            raise BusinessError(f"PipelineTemplate {template_id} not found", status_code=404)
        encrypted = self.security_service.encrypt_value(plain_secret)
        wh = ProjectWebhook.create(
            project_id=project_id,
            name=name,
            template_id=template_id,
            encrypted_secret=encrypted,
            branch_filter=branch_filter or None,  # 空字符串统一转为 None，表示拒绝所有分支
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
    ) -> ProjectWebhook:
        wh = self.get_webhook(webhook_id)
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
        wh = self.get_webhook(webhook_id)
        self.webhook_repo.delete(wh)

    def decrypt_webhook_secret(self, webhook: ProjectWebhook) -> str:
        """解密 webhook 签名密钥（封装 security_service，避免接口层直接访问）"""
        return self.security_service.decrypt_value(webhook.encrypted_secret)

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
        versions = self.snapshot_repo.find_latest_versions([template_id])
        return versions.get(template_id)

    def create_template(
        self,
        name: str,
        description: str = "",
        variable_declarations: list | None = None,
    ) -> PipelineTemplate:
        decls = [VariableDeclaration(**d) if isinstance(d, dict) else d for d in (variable_declarations or [])]
        tmpl = PipelineTemplate.create(name=name, variable_declarations=decls, description=description)
        self.template_repo.save(tmpl)
        return tmpl

    def update_template(
        self,
        template_id: str,
        name: str | None = None,
        description: str | None = None,
        orchestration: list | None = None,
        variable_declarations: list | None = None,
    ) -> PipelineTemplate:
        tmpl = self.get_template(template_id)
        decls = None
        if variable_declarations is not None:
            decls = [VariableDeclaration(**d) if isinstance(d, dict) else d for d in variable_declarations]
        tmpl.update(name=name, description=description, variable_declarations=decls)
        if orchestration is not None:
            orch_list = [StageOrchestration(**o) if isinstance(o, dict) else o for o in orchestration]
            stage_ids = [o.stage_id for o in orch_list]
            stages = self.stage_repo.find_by_ids(stage_ids)
            if len(stages) != len(stage_ids):
                found = {s.id for s in stages}
                missing = [sid for sid in stage_ids if sid not in found]
                raise BusinessError(f"Stage(s) not found: {missing}", status_code=404)
            self.template_repo.save_orchestration(template_id, orch_list)
            # 重新加载含新编排的模板，确保 stages 已更新
            tmpl = self.get_template(template_id)
            if decls is not None:
                tmpl.variable_declarations = decls
        self.template_repo.save(tmpl)
        return self.get_template(template_id)

    def delete_template(self, template_id: str) -> None:
        tmpl = self.get_template(template_id)
        webhooks = self.webhook_repo.find_by_template(template_id)
        if webhooks:
            raise BusinessError("Template is referenced by webhooks, cannot delete", status_code=409)
        # 历史 PipelineRun 通过 pipeline_snapshot_id 关联快照，快照独立存储不受影响。
        # pipeline_template_stages 有 ON DELETE CASCADE，随模板自动删除。
        self.template_repo.delete(tmpl)

    # ── PipelineStage CRUD ────────────────────────────────────────────────────

    def list_stages(self) -> list[PipelineStage]:
        return self.stage_repo.find_all()

    def get_stage(self, stage_id: str) -> PipelineStage:
        stage = self.stage_repo.find_by_id(stage_id)
        if not stage:
            raise BusinessError(f"Stage {stage_id} not found", status_code=404)
        return stage

    def create_stage(
        self,
        name: str,
        image: str,
        script: str,
        env: dict[str, str] | None = None,
        artifacts: list | None = None,
        description: str = "",
    ) -> PipelineStage:
        if self.stage_repo.find_by_name(name):
            raise BusinessError(f"Stage '{name}' already exists", status_code=409)
        stage = PipelineStage.create(
            name=name,
            image=image,
            script=script,
            env=env,
            artifacts=artifacts,
            description=description,
        )
        self.stage_repo.save(stage)
        return stage

    def update_stage(
        self,
        stage_id: str,
        name: str | None = None,
        image: str | None = None,
        script: str | None = None,
        env: dict[str, str] | None = None,
        artifacts: list | None = None,
        description: str | None = None,
    ) -> PipelineStage:
        stage = self.get_stage(stage_id)
        if name is not None and name != stage.name and self.stage_repo.find_by_name(name):
            raise BusinessError(f"Stage '{name}' already exists", status_code=409)
        stage.update(name=name, image=image, script=script, env=env, artifacts=artifacts, description=description)
        self.stage_repo.save(stage)
        # 更新所有引用此 Stage 的模板的 updated_at，触发快照版本检测
        self._touch_templates_referencing(stage_id)
        return stage

    def delete_stage(self, stage_id: str) -> None:
        stage = self.get_stage(stage_id)
        if self.stage_repo.is_referenced_by_templates(stage_id):
            raise BusinessError("Stage is referenced by templates, cannot delete", status_code=409)
        self.stage_repo.delete(stage)

    # ── 模板编排 ──────────────────────────────────────────────────────────────

    def _touch_templates_referencing(self, stage_id: str) -> None:
        """Stage 更新后，touch 所有引用它的模板"""
        # 通过 find_all 过滤（数量有限，可接受）
        templates = self.template_repo.find_all()
        for tmpl in templates:
            if any(o.stage_id == stage_id for o in tmpl.orchestration):
                tmpl.updated_at = utc_now()
                self.template_repo.save(tmpl)

    def _sync_declarations(
        self, stages: list[PipelineStage], existing: list[VariableDeclaration]
    ) -> list[VariableDeclaration]:
        stage_defs = [s.to_stage_definition() for s in stages]
        extracted = extract_variables(stage_defs)
        return merge_declarations(extracted, existing)

    # ── PipelineSnapshot ──────────────────────────────────────────────────────

    def list_template_snapshots(self, template_id: str) -> list[PipelineSnapshot]:
        self.get_template(template_id)
        return self.snapshot_repo.find_by_template(template_id)

    def get_snapshot(self, snapshot_id: str) -> PipelineSnapshot:
        snapshot = self.snapshot_repo.find_by_id(snapshot_id)
        if not snapshot:
            raise BusinessError(f"PipelineSnapshot {snapshot_id} not found", status_code=404)
        return snapshot

    def _get_or_create_snapshot(self, template: PipelineTemplate) -> PipelineSnapshot:
        """获取或创建快照：仅当模板有变更时创建新快照。

        快照在模板更新之后创建，所以 created_at >= updated_at 时说明快照已是最新。
        用 >= 而非 > 是因为同秒内创建的快照 created_at == updated_at 也应复用。
        """
        latest = self.snapshot_repo.find_latest(template.id)
        if latest and latest.created_at >= template.updated_at:
            return latest
        next_version = self.snapshot_repo.get_next_version(template.id)
        snapshot = PipelineSnapshot.create(template, version=next_version)
        self.snapshot_repo.save(snapshot)
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

    def list_stage_runs(self, run_id: str) -> list[StageRun]:
        self.get_run(run_id)
        return self.stage_run_repo.find_by_run(run_id)

    def get_stage_log(self, stage_run_id: str) -> StageLog | None:
        # 日志可能尚未写入（stage 还在运行），返回 None 而非 404
        return self.stage_log_repo.find_by_stage_run(stage_run_id)

    def cancel_run(self, run_id: str) -> PipelineRun:
        run = self.get_run(run_id)
        try:
            run.cancel()
        except ValueError as e:
            raise BusinessError(str(e), status_code=400) from e
        cancel_task(run_id)  # 取消 asyncio task（如果还在运行）
        self.run_repo.save(run)
        self.run_repo.commit()
        return run

    def create_run(
        self,
        project_id: str,
        template_id: str,
        trigger: PipelineRunTrigger,
        trigger_ref: str,
        runtime_variables: dict[str, Any] | None = None,
    ) -> tuple[PipelineRun, Project, dict[str, Any], PipelineSnapshot]:
        project = self.get_project(project_id)
        template = self.get_template(template_id)

        # 按需创建快照（模板有变更才创建新版本）
        snapshot = self._get_or_create_snapshot(template)

        declarations = snapshot.variable_declarations_snapshot
        effective_ref = trigger_ref or project.default_branch
        builtin = {
            "project_repository_url": project.repository_url,
            "DEFAULT_BRANCH": project.default_branch,
            "GIT_CREDENTIAL_ID": project.git_credential_id or "",
            "trigger_ref": effective_ref,
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

        masked = mask_secrets(merged, declarations)
        run = PipelineRun.create(
            project_id=project_id,
            pipeline_snapshot_id=snapshot.id,
            trigger=trigger,
            trigger_ref=effective_ref,
            variables_snapshot=masked,
        )
        self.run_repo.save(run)
        self.run_repo.commit()

        logger.info(
            f"Pipeline triggered: project={project.name}, template={template.name}, run={run.id}, trigger={trigger.value}"
        )
        return run, project, merged, snapshot

    def create_retry_run(self, run_id: str) -> tuple[PipelineRun, Project, dict[str, Any], PipelineSnapshot]:
        original = self.get_run(run_id)
        if original.status not in {TaskStatus.FAULTED, TaskStatus.RAN_TO_COMPLETION}:
            raise BusinessError(f"Cannot retry run with status {original.status.value}", status_code=400)

        snapshot = self.get_snapshot(original.pipeline_snapshot_id)
        project = self.get_project(original.project_id)

        # 重试复用原快照，但需要重新合并变量（secret 变量从 project 重新取）
        declarations = snapshot.variable_declarations_snapshot
        builtin = {
            "project_repository_url": project.repository_url,
            "DEFAULT_BRANCH": project.default_branch,
            "GIT_CREDENTIAL_ID": project.git_credential_id or "",
            "trigger_ref": original.trigger_ref,
            "trigger_type": original.trigger.value,
            "project_name": project.name,
        }
        merged = merge_variables(
            global_vars=self.global_variables,
            project_vars=project.variable_overrides,
            runtime_vars={},
            declarations=declarations,
            builtin_vars=builtin,
        )

        masked = mask_secrets(merged, declarations)
        new_run = PipelineRun.create(
            project_id=original.project_id,
            pipeline_snapshot_id=original.pipeline_snapshot_id,
            trigger=original.trigger,
            trigger_ref=original.trigger_ref,
            variables_snapshot=masked,
            retry_of=original.id,
        )
        self.run_repo.save(new_run)
        self.run_repo.commit()

        logger.info(f"Pipeline retry: original={run_id}, new={new_run.id}")
        return new_run, project, merged, snapshot

    async def execute_run(
        self, run: PipelineRun, project: Project, variables: dict[str, Any], snapshot: PipelineSnapshot
    ) -> None:
        """执行 pipeline run（由 BackgroundTasks 调用，使用独立 session）"""
        session_factory = self._session_factory or get_session_factory()

        try:
            resolved_stages = [resolve_stage(s, variables) for s in snapshot.stages_snapshot]
        except Exception as e:
            logger.error(f"Stage resolution failed: run={run.id}, error={e}", exc_info=True)
            with session_factory() as session:
                run_repo = PipelineRunRepositoryImpl(session)
                run.complete_failed()
                run_repo.save(run)
                session.commit()
            return

        workspace_path, artifacts_path = create_workspace(project.code, run.id)
        context = ExecutionContext(
            run_id=run.id,
            project_id=project.id,
            project_code=project.code,
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
                cleanup_run(project.code, run.id)
                logger.info(f"Pipeline finished: run={run.id}, status={run.status}")
