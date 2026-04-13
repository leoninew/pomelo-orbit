"""PipelineRun 聚合的应用服务"""

import asyncio
import logging
from collections.abc import Callable
from dataclasses import dataclass
from typing import Any

from sqlalchemy.orm import Session, sessionmaker

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
from pomelo_orbit.infrastructure.ci.repositories import PipelineRunRepositoryImpl
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
from pomelo_orbit.infrastructure.persistence.database import get_session_factory

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
    """PipelineRun 聚合根的应用服务

    职责：
    - PipelineRun 的创建、查询、取消
    - PipelineRun 重试
    - PipelineRun 执行编排
    - StageRun 查询
    - Artifact 查询
    - Stage 日志读取

    依赖：
    - RepositoryRepository: 获取项目信息
    - PipelineTemplateRepository: 获取模板
    - PipelineSnapshotRepository: 获取快照
    - SnapshotManager: 快照管理领域服务
    - VariableResolver: 变量解析领域服务
    - PipelineExecutor: 执行引擎
    """

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
        session_factory: sessionmaker[Session],
        executor_factory: Callable[[Session], PipelineExecutor],
    ):
        self.run_repo = run_repo
        self.artifact_repo = artifact_repo
        self.stage_run_repo = stage_run_repo
        self.repository_repo = repository_repo
        self.template_repo = template_repo
        self.snapshot_repo = snapshot_repo
        self.snapshot_manager = snapshot_manager
        self.variable_resolver = variable_resolver
        self._session_factory = session_factory
        self._executor_factory = executor_factory

    # ── 查询方法 ──────────────────────────────────────────────────────────────

    def list_runs(self, page: int = 1, per_page: int = 20, repository_id: str | None = None) -> PaginatedRuns:
        """分页查询运行列表"""
        if repository_id:
            repository = self.repository_repo.find_by_id(repository_id)
            if not repository:
                raise BusinessError(f"Repository {repository_id} not found", status_code=404)
        runs, total = self.run_repo.find_paginated_with_filters(
            page=page, per_page=per_page, repository_id=repository_id
        )
        return PaginatedRuns(runs=runs, total=total)

    def get_run(self, run_id: str) -> PipelineRun:
        """获取单个运行"""
        run = self.run_repo.find_by_id(run_id)
        if not run:
            raise BusinessError(f"PipelineRun {run_id} not found", status_code=404)
        return run

    def list_artifacts(self, run_id: str) -> list[Artifact]:
        """查询运行的制品列表"""
        return self.artifact_repo.find_by_run(run_id)

    def list_all_artifacts(
        self,
        repository_id: str | None = None,
        page: int = 1,
        per_page: int = 20,
    ) -> tuple[list[Artifact], int]:
        """分页查询所有制品"""
        return self.artifact_repo.find_paginated(page=page, per_page=per_page, repository_id=repository_id)

    def list_stage_runs(self, run_id: str) -> list[StageRun]:
        """查询运行的 stage 列表"""
        return self.stage_run_repo.find_by_run(run_id)

    def read_stage_log(self, run_id: str, stage_run_id: str, offset: int = 0) -> StageLogResult:
        """读取 stage 日志（增量）"""
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

    # ── 命令方法 ──────────────────────────────────────────────────────────────

    def create_run(
        self,
        repository_id: str,
        template_id: str,
        trigger: PipelineRunTrigger,
        trigger_ref: str,
        runtime_variables: dict[str, Any] | None = None,
    ) -> RunCreationResult:
        """创建运行"""
        repository = self.repository_repo.find_by_id(repository_id)
        if not repository:
            raise BusinessError(f"Repository {repository_id} not found", status_code=404)

        template = self.template_repo.find_by_id(template_id)
        if not template:
            raise BusinessError(f"Template {template_id} not found", status_code=404)

        # 按需创建快照（模板有变更才创建新版本）
        # 获取完整的变量声明列表（内置 + stage + 自定义）
        complete_variable_declarations = self.variable_resolver.resolve_template_variables(
            template.stages,
            template.variable_declarations,
        )
        snapshot = self.snapshot_manager.get_or_create_snapshot(template, complete_variable_declarations)
        self.run_repo.commit()

        # 构建运行时变量（合并所有来源）
        merged = self.variable_resolver.build_runtime_variables(
            repository=repository,
            template=template,
            trigger_ref=trigger_ref,
            runtime_overrides=runtime_variables,
            stage_declarations=complete_variable_declarations,
        )

        # 验证变量
        try:
            validate_variables(merged, template.variable_declarations)
        except VariableError as e:
            raise BusinessError(str(e), status_code=400) from e

        # 使用快照中的变量声明作为基准，只填充实际值
        # 快照已经过滤了只在 stage 中使用的变量
        runtime_declarations = []
        for decl in snapshot.variables_snapshot:
            # 从 merged 中获取实际值，并更新 source 和 description
            value = merged.get(decl.name)

            # 判断变量来源并设置正确的 source 和 description
            if decl.name in self.variable_resolver._get_repository_builtin_specs():
                # 仓库内置变量
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
                # 检查是否是仓库自定义变量
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
                    # 保持原有的声明，只更新值
                    runtime_declarations.append(
                        VariableDeclaration(
                            name=decl.name,
                            value=value,
                            source=decl.source,
                            description=decl.description,
                            secret=decl.secret,
                        )
                    )

        # 脱敏后保存到快照
        masked = mask_secrets(merged, runtime_declarations)
        run = PipelineRun.create(
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

    def create_retry_run(self, run_id: str) -> RunCreationResult:
        """创建重试运行"""
        original = self.get_run(run_id)
        if original.status not in {TaskStatus.FAULTED, TaskStatus.RAN_TO_COMPLETION}:
            raise BusinessError(f"Cannot retry run with status {original.status.value}", status_code=400)

        snapshot = self.snapshot_repo.find_by_id(original.snapshot_id)
        if not snapshot:
            raise BusinessError(f"Snapshot {original.snapshot_id} not found", status_code=404)

        repository = self.repository_repo.find_by_id(original.repository_id)
        if not repository:
            raise BusinessError(f"Repository {original.repository_id} not found", status_code=404)

        # 重试复用原快照，快照中已经存储了完整的变量声明列表
        template_info = _TemplateInfo(
            id=original.template_id,
            name=original.template_name,
            version=original.template_version,
            variable_declarations=snapshot.variables_snapshot,
        )

        # 构建运行时变量（重试时不传入 runtime_overrides，但需要重新获取 secret 变量）
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

    def cancel_run(self, run_id: str) -> PipelineRun:
        """取消运行"""
        run = self.get_run(run_id)
        try:
            run.cancel()
        except ValueError as e:
            raise BusinessError(str(e), status_code=400) from e

        cancel_task(run_id)  # 取消 asyncio task（如果还在运行）
        self.run_repo.save(run)
        self.run_repo.commit()
        return run

    async def execute_run(
        self, run: PipelineRun, repository: Repository, variables: dict[str, Any], snapshot: PipelineSnapshot
    ) -> None:
        """执行 pipeline run（由 BackgroundTasks 调用，使用独立 session）"""
        session_factory = self._session_factory or get_session_factory()

        try:
            resolved_stages = [resolve_stage(s, variables) for s in snapshot.stages_snapshot]
        except Exception as e:
            logger.error(f"Stage resolution failed: run={run.id}, error={e}", exc_info=True)
            with session_factory() as session:
                run_repo = PipelineRunRepositoryImpl(session)
                run.complete_failed(f"Stage resolution failed: {e}")
                run_repo.save(run)
                session.commit()
            return

        workspace_path, artifacts_path = create_workspace(repository.code, run.id)
        context = ExecutionContext(
            run_id=run.id,
            repository_id=repository.id,
            repository_name=repository.name,
            template_id=run.template_id,
            template_name=run.template_name,
            project_code=repository.code,
            repository_url=repository.repository_url,
            credential_id=repository.git_credential_id,
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
                    run.complete_failed(context.error_message)
            except asyncio.CancelledError:
                run.cancel()
            except Exception as e:
                logger.error(f"Pipeline execution error: run={run.id}, error={e}", exc_info=True)
                run.complete_failed(f"Unexpected error: {e}")
            finally:
                run_repo.save(run)
                session.commit()
                cleanup_run_secrets(run.id)
                logger.info(f"Pipeline finished: run={run.id}, status={run.status}")
