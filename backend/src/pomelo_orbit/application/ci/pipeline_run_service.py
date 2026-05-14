"""PipelineRun 聚合的应用服务"""

import asyncio
import logging
from dataclasses import dataclass
from typing import Any

from pomelo_orbit.domain.cd.value_objects import TaskStatus
from pomelo_orbit.domain.ci.entities import (
    Artifact,
    PipelineRun,
    PipelineSnapshot,
    Repository,
    StageRun,
)
from pomelo_orbit.domain.ci.executor import ExecutionContext, PipelineExecutor
from pomelo_orbit.domain.ci.repositories import (
    ArtifactRepository,
    PipelineRunRepository,
    PipelineSnapshotRepository,
    PipelineTemplateRepository,
    RepositoryRepository,
    StageRunRepository,
)
from pomelo_orbit.domain.ci.snapshot_manager import SnapshotManager
from pomelo_orbit.domain.ci.value_objects import PipelineRunTrigger, VariableDeclaration, VariableSource
from pomelo_orbit.domain.ci.variable_resolver import VariableResolver
from pomelo_orbit.domain.exceptions import BusinessError
from pomelo_orbit.infrastructure.ci.executor_impl import cancel_task
from pomelo_orbit.infrastructure.ci.variables import (
    VariableError,
    mask_secrets,
    resolve_stage,
    validate_variables,
)
from pomelo_orbit.infrastructure.ci.workspace import (
    cleanup_run_secrets,
    create_workspace,
    get_stage_log_path,
)
from pomelo_orbit.infrastructure.time_utils import from_iso8601

logger = logging.getLogger(__name__)


@dataclass
class PaginatedRuns:
    runs: list[PipelineRun]
    total: int


@dataclass
class StageLogResult:
    logs: str
    offset: int
    is_complete: bool


@dataclass
class RunCreationResult:
    run: PipelineRun
    repository: Repository
    merged_variables: dict[str, Any]
    snapshot: PipelineSnapshot


@dataclass
class _TemplateInfo:
    """重试时用于变量解析的轻量模板信息"""

    id: str
    name: str
    version: int
    variable_declarations: list


class PipelineRunService:
    """PipelineRun 聚合根的应用服务"""

    def __init__(
        self,
        run_repo: PipelineRunRepository,
        artifact_repo: ArtifactRepository,
        stage_run_repo: StageRunRepository,
        repository_repo: RepositoryRepository,
        template_repo: PipelineTemplateRepository,
        snapshot_repo: PipelineSnapshotRepository,
        snapshot_manager: SnapshotManager,
        variable_resolver: VariableResolver,
        pipeline_executor: PipelineExecutor,
    ):
        self.run_repo = run_repo
        self.artifact_repo = artifact_repo
        self.stage_run_repo = stage_run_repo
        self.repository_repo = repository_repo
        self.template_repo = template_repo
        self.snapshot_repo = snapshot_repo
        self.snapshot_manager = snapshot_manager
        self.variable_resolver = variable_resolver
        self._pipeline_executor = pipeline_executor

    def list_runs(
        self,
        project_id: str,
        page: int = 1,
        per_page: int = 20,
        repository_id: str | None = None,
        template_id: str | None = None,
        date_from: str | None = None,
        date_to: str | None = None,
    ) -> PaginatedRuns:
        """分页查询运行列表"""
        if repository_id and not self.repository_repo.find_by_id_in_project(project_id, repository_id):
            raise BusinessError(f"Repository {repository_id} not found", status_code=404)
        if template_id and not self.template_repo.find_by_id_in_project(project_id, template_id):
            raise BusinessError(f"Template {template_id} not found", status_code=404)
        date_from_dt = from_iso8601(date_from) if date_from else None
        date_to_dt = from_iso8601(date_to) if date_to else None
        runs, total = self.run_repo.find_paginated_with_filters(
            project_id=project_id,
            page=page,
            per_page=per_page,
            repository_id=repository_id,
            template_id=template_id,
            date_from=date_from_dt,
            date_to=date_to_dt,
        )
        return PaginatedRuns(runs=runs, total=total)

    def get_run(self, project_id: str, run_id: str) -> PipelineRun:
        """获取单个运行"""
        run = self.run_repo.find_by_id_in_project(project_id, run_id)
        if not run:
            raise BusinessError(f"PipelineRun {run_id} not found", status_code=404)
        return run

    def list_artifacts(self, project_id: str, run_id: str) -> list[Artifact]:
        """查询运行的制品列表"""
        self.get_run(project_id, run_id)
        return self.artifact_repo.find_by_run(project_id, run_id)

    def list_all_artifacts(
        self,
        project_id: str,
        repository_id: str | None = None,
        template_id: str | None = None,
        search: str | None = None,
        page: int = 1,
        per_page: int = 20,
    ) -> tuple[list[Artifact], int]:
        """分页查询所有制品"""
        if repository_id and not self.repository_repo.find_by_id_in_project(project_id, repository_id):
            raise BusinessError(f"Repository {repository_id} not found", status_code=404)
        if template_id and not self.template_repo.find_by_id_in_project(project_id, template_id):
            raise BusinessError(f"Template {template_id} not found", status_code=404)
        return self.artifact_repo.find_paginated_by_project_id(
            project_id=project_id,
            page=page,
            per_page=per_page,
            repository_id=repository_id,
            template_id=template_id,
            search=search,
        )

    def list_stage_runs(self, project_id: str, run_id: str) -> list[StageRun]:
        """查询运行的 stage 列表"""
        self.get_run(project_id, run_id)
        return self.stage_run_repo.find_by_run(run_id)

    def read_stage_log(self, project_id: str, run_id: str, stage_run_id: str, offset: int = 0) -> StageLogResult:
        """读取 stage 日志（增量）"""
        self.get_run(project_id, run_id)
        stage_run = self.stage_run_repo.find_by_id(stage_run_id)
        if not stage_run or stage_run.pipeline_run_id != run_id:
            return StageLogResult(logs="", offset=offset, is_complete=True)

        log_path = get_stage_log_path(run_id, stage_run_id)
        if not log_path.exists():
            is_complete = stage_run.status in (
                TaskStatus.RAN_TO_COMPLETION,
                TaskStatus.FAULTED,
                TaskStatus.CANCELED,
            )
            return StageLogResult(logs="", offset=offset, is_complete=is_complete)

        with log_path.open(encoding="utf-8") as f:
            f.seek(offset)
            content = f.read()
            new_offset = f.tell()

        is_complete = stage_run.status in (
            TaskStatus.RAN_TO_COMPLETION,
            TaskStatus.FAULTED,
            TaskStatus.CANCELED,
        )
        return StageLogResult(logs=content, offset=new_offset, is_complete=is_complete)

    def create_run(
        self,
        project_id: str,
        repository_id: str,
        template_id: str,
        trigger: PipelineRunTrigger,
        trigger_ref: str,
        runtime_variables: dict[str, Any] | None = None,
    ) -> RunCreationResult:
        """创建运行"""
        repository = self.repository_repo.find_by_id_in_project(project_id, repository_id)
        if not repository:
            raise BusinessError(f"Repository {repository_id} not found", status_code=404)

        template = self.template_repo.find_by_id_in_project(project_id, template_id)
        if not template:
            raise BusinessError(f"Template {template_id} not found", status_code=404)

        complete_variable_declarations = self.variable_resolver.resolve_template_variables(
            template.stages,
            template.variable_declarations,
        )
        snapshot = self.snapshot_manager.get_or_create_snapshot(template, complete_variable_declarations)
        self.run_repo.commit()

        merged = self.variable_resolver.build_runtime_variables(
            repository=repository,
            template=template,
            trigger_ref=trigger_ref,
            runtime_overrides=runtime_variables,
            stage_declarations=complete_variable_declarations,
        )

        try:
            validate_variables(merged, template.variable_declarations)
        except VariableError as e:
            raise BusinessError(str(e), status_code=400) from e

        runtime_declarations = []
        for decl in snapshot.variables_snapshot:
            value = merged.get(decl.name)
            if decl.name in self.variable_resolver._get_repository_builtin_specs():
                runtime_declarations.append(
                    VariableDeclaration(
                        name=decl.name,
                        value=value,
                        source=VariableSource.REPOSITORY,
                        description=decl.description or f"运行时注入: {decl.name}",
                        secret=decl.secret,
                    )
                )
            else:
                repo_custom_var = next(
                    (v for v in (repository.variable_overrides or []) if v.name == decl.name),
                    None,
                )
                if repo_custom_var:
                    runtime_declarations.append(
                        VariableDeclaration(
                            name=decl.name,
                            value=value,
                            source=VariableSource.REPOSITORY_CUSTOM,
                            description=repo_custom_var.description or "",
                            secret=repo_custom_var.secret,
                        )
                    )
                else:
                    runtime_declarations.append(
                        VariableDeclaration(
                            name=decl.name,
                            value=value,
                            source=decl.source,
                            description=decl.description,
                            secret=decl.secret,
                        )
                    )

        masked = mask_secrets(merged, runtime_declarations)
        run = PipelineRun.create(
            project_id=project_id,
            repository_id=repository_id,
            repository_name=repository.name,
            snapshot_id=snapshot.id,
            template_id=template.id,
            template_name=template.name,
            template_version=snapshot.version,
            trigger=trigger,
            trigger_ref=trigger_ref,
            variables_snapshot=masked,
        )
        self.run_repo.save(run)
        self.run_repo.commit()

        logger.info(
            f"Pipeline triggered: repository={repository.name}, template={template.name}, run={run.id}, trigger={trigger.value}"
        )
        return RunCreationResult(run=run, repository=repository, merged_variables=merged, snapshot=snapshot)

    def create_retry_run(self, project_id: str, run_id: str) -> RunCreationResult:
        """创建重试运行"""
        original = self.get_run(project_id, run_id)
        if original.status not in {TaskStatus.FAULTED, TaskStatus.RAN_TO_COMPLETION}:
            raise BusinessError(f"Cannot retry run with status {original.status.value}", status_code=400)

        snapshot = self.snapshot_repo.find_by_id_in_project(project_id, original.snapshot_id)
        if not snapshot:
            raise BusinessError(f"Snapshot {original.snapshot_id} not found", status_code=404)

        repository = self.repository_repo.find_by_id_in_project(project_id, original.repository_id)
        if not repository:
            raise BusinessError(f"Repository {original.repository_id} not found", status_code=404)

        template_info = _TemplateInfo(
            id=original.template_id,
            name=original.template_name,
            version=original.template_version,
            variable_declarations=snapshot.variables_snapshot,
        )

        merged = self.variable_resolver.build_runtime_variables(
            repository=repository,
            template=template_info,
            trigger_ref=original.trigger_ref,
            runtime_overrides=None,
        )

        try:
            validate_variables(merged, snapshot.variables_snapshot)
        except VariableError as e:
            raise BusinessError(str(e), status_code=400) from e

        masked = mask_secrets(merged, snapshot.variables_snapshot)
        new_run = PipelineRun.create(
            project_id=project_id,
            repository_id=original.repository_id,
            repository_name=repository.name,
            snapshot_id=original.snapshot_id,
            template_id=original.template_id,
            template_name=original.template_name,
            template_version=original.template_version,
            trigger=original.trigger,
            trigger_ref=original.trigger_ref,
            variables_snapshot=masked,
            retry_of=original.id,
        )
        self.run_repo.save(new_run)
        self.run_repo.commit()

        logger.info(f"Pipeline retry: original={original.id}, new={new_run.id}")
        return RunCreationResult(run=new_run, repository=repository, merged_variables=merged, snapshot=snapshot)

    def cancel_run(self, project_id: str, run_id: str) -> PipelineRun:
        """取消运行"""
        run = self.get_run(project_id, run_id)
        try:
            run.cancel()
        except ValueError as e:
            raise BusinessError(str(e), status_code=400) from e

        cancel_task(run_id)
        self.run_repo.save(run)
        self.run_repo.commit()
        return run

    async def execute_run(
        self, run: PipelineRun, repository: Repository, variables: dict[str, Any], snapshot: PipelineSnapshot
    ) -> None:
        """执行 pipeline run（由 Dishka 容器在 BackgroundTasks 中调用，拥有独立 session）"""
        try:
            resolved_stages = [resolve_stage(s, variables) for s in snapshot.stages_snapshot]
        except Exception as e:
            logger.error(f"Stage resolution failed: run={run.id}, error={e}", exc_info=True)
            run.complete_failed(f"Stage resolution failed: {e}")
            self.run_repo.save(run)
            self.run_repo.commit()
            return

        create_workspace(repository.code, run.id)

        context = ExecutionContext(
            run_id=run.id,
            project_id=run.project_id,
            repository_id=repository.id,
            repository_name=repository.name,
            template_id=run.template_id,
            template_name=run.template_name,
            project_code=repository.code,
            repository_url=repository.repository_url,
            credential_id=repository.git_credential_id,
            variables=variables,
            retry_of=run.retry_of,
        )

        run.start()
        self.run_repo.save(run)
        self.run_repo.commit()

        try:
            success = await self._pipeline_executor.execute(context, resolved_stages)
            if success:
                run.complete_success()
            else:
                run.complete_failed(context.error_message)
        except asyncio.CancelledError:
            run.cancel()
        except Exception as e:
            logger.error(f"Pipeline execution error: run={run.id}, error={e}", exc_info=True)
            run.complete_failed(f"Unexpected error: {e}")
        finally:
            self.run_repo.save(run)
            self.run_repo.commit()
            cleanup_run_secrets(run.id)
            logger.info(f"Pipeline finished: run={run.id}, status={run.status}")
