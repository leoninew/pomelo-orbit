"""Pipeline 执行器实现"""

import asyncio
import logging
from copy import copy
from pathlib import Path
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from typing import TextIO

from pomelo_orbit.domain.cd.value_objects import TaskStatus
from pomelo_orbit.domain.ci.entities import Artifact, StageRun
from pomelo_orbit.domain.ci.executor import ExecutionContext, PipelineExecutor
from pomelo_orbit.domain.ci.repositories import (
    ArtifactRepository,
    CredentialRepository,
    StageRunRepository,
)
from pomelo_orbit.domain.ci.value_objects import CredentialType, StageDefinition
from pomelo_orbit.infrastructure.ci.container import ContainerExecutor
from pomelo_orbit.infrastructure.ci.dependency_graph import CyclicDependencyError, DependencyGraph
from pomelo_orbit.infrastructure.ci.workspace import get_secrets_path, get_stage_log_path
from pomelo_orbit.infrastructure.security import SecurityService

logger = logging.getLogger(__name__)

# 全局 task 注册表，用于支持真实取消。
# 注意：此为进程级变量，仅在单进程/单 worker 部署下有效。
# 多 worker 部署时 cancel_task 无法跨进程取消，cancel_run 会静默失败（run 状态
# 仍会被标记为 canceled，但实际执行不会中断）。
_running_tasks: dict[str, asyncio.Task] = {}


def register_task(run_id: str, task: asyncio.Task) -> None:
    _running_tasks[run_id] = task


def cancel_task(run_id: str) -> bool:
    task = _running_tasks.get(run_id)
    if task and not task.done():
        task.cancel()
        return True
    return False


def unregister_task(run_id: str) -> None:
    _running_tasks.pop(run_id, None)


class PipelineExecutorImpl(PipelineExecutor):
    def __init__(
        self,
        stage_run_repo: StageRunRepository,
        container_executor: ContainerExecutor,
        artifact_repo: ArtifactRepository,
        credential_repo: CredentialRepository,
        security_service: SecurityService,
    ):
        self.stage_run_repo = stage_run_repo
        self.container_executor = container_executor
        self.artifact_repo = artifact_repo
        self.credential_repo = credential_repo
        self.security_service = security_service

    async def execute(self, context: ExecutionContext, stages: list[StageDefinition]) -> bool:
        """按 Stage 依赖关系拓扑排序后分层并行执行，1 Stage = 1 Job"""
        task = asyncio.current_task()
        if task:
            # 注册当前 asyncio.Task，使 cancel_task() 能通过 run_id 找到它并调用 task.cancel()。
            # 必须在执行开始前注册，否则用户在极短窗口内取消时会静默失败。
            register_task(context.run_id, task)
        try:
            return await self._execute_internal(context, stages)
        finally:
            # 无论正常结束、异常还是取消，都要清理注册表，避免 run_id 泄漏。
            unregister_task(context.run_id)

    async def _execute_internal(self, context: ExecutionContext, stages: list[StageDefinition]) -> bool:
        try:
            graph = DependencyGraph(stages)
            try:
                layers = graph.topological_sort()
            except CyclicDependencyError as e:
                logger.error(f"Cyclic dependency: run={context.run_id}, error={e}", exc_info=True)
                context.error_message = f"Cyclic dependency detected: {e}"
                return False

            stages_map = {s.id: s for s in stages}
            for layer_idx, layer in enumerate(layers):
                logger.info(
                    f"Executing layer: index={layer_idx + 1}, total={len(layers)}, run={context.run_id}, stages={layer}"
                )
                layer_stages = [stages_map[stage_id] for stage_id in layer]
                results = await self._execute_layer(context, layer_stages)

                failed = [name for name, ok in results.items() if not ok]
                if failed:
                    logger.warning(f"Layer failed: run={context.run_id}, failed={failed}")
                    context.error_message = f"Stage(s) failed: {', '.join(failed)}"
                    await self._cancel_remaining(context, layers, layer_idx + 1, stages_map)
                    return False

            return True

        except asyncio.CancelledError:
            # 取消信号从 _execute_stage 穿透上来，在这里记录日志后继续向上传播。
            # 不能在此处把 run 标记为 canceled，因为 session 在 execute_run 里管理，
            # 这里没有访问权限。run 的最终状态由 execute_run 的 except CancelledError 负责。
            logger.info(f"Pipeline cancelled: run={context.run_id}")
            raise
        except Exception as e:
            # 非取消的意外异常（理论上不应到达这里，因为 _execute_stage 已经把
            # Exception 转换成了 return False）。作为兜底，记录日志并返回失败。
            logger.error(f"Pipeline execution failed: run={context.run_id}, error={e}", exc_info=True)
            context.error_message = f"Pipeline execution error: {e}"
            return False

    async def _execute_layer(self, context: ExecutionContext, stages: list[StageDefinition]) -> dict[str, bool]:
        """并行执行同一层的所有 Stage"""
        tasks = [self._execute_stage(context, stage) for stage in stages]
        # return_exceptions=False：让 CancelledError 不被 gather 吞掉，直接向上传播。
        # 这样外层 _execute_internal 的 except CancelledError 才能感知到取消信号。
        # 代价是同层其他 stage 的 task 会被 gather 自动取消，它们各自的 CancelledError
        # 分支会负责把自己的 StageRun 状态写入 DB。
        results = await asyncio.gather(*tasks, return_exceptions=False)

        result_map: dict[str, bool] = {}
        for stage, result in zip(stages, results, strict=True):
            result_map[stage.name] = bool(result)
        return result_map

    async def _execute_stage(self, context: ExecutionContext, stage: StageDefinition) -> bool:
        """执行单个 Stage，创建 StageRun 记录"""
        stage_run = StageRun.create(pipeline_run_id=context.run_id, stage_id=stage.id, stage_name=stage.name)
        self.stage_run_repo.save(stage_run)

        # 日志写入文件，执行过程中实时可读，不再写 DB。
        log_path = get_stage_log_path(context.run_id, stage_run.id)
        log_path.parent.mkdir(parents=True, exist_ok=True)

        try:
            stage_run.start()
            self._save_stage_run(stage_run)
            logger.info(f"Stage started: run={context.run_id}, stage={stage.name}, image={stage.image}")

            with log_path.open("w", encoding="utf-8") as log_file:
                if context.credential_id:
                    exit_code, output = await self._execute_clone(context, stage, log_file=log_file)
                else:
                    exit_code, output = await self._execute_container(context, stage, log_file=log_file)

            if exit_code == 0:
                stage_run.complete_success(exit_code)
                # artifact 先保存，再 commit stage 状态，保证两者在同一个事务里。
                self._save_artifacts(context, stage, stage_run.stage_name)
                self._save_stage_run(stage_run)
                logger.info(f"Stage succeeded: run={context.run_id}, stage={stage.name}")
                return True

            # 只取最后 3 行作为错误摘要，完整日志已写入日志文件
            if output and output.strip():
                last_lines = [line for line in output.strip().splitlines() if line.strip()][-3:]
                error_message = "\n".join(last_lines)
            else:
                error_message = f"Exit code: {exit_code}"
            stage_run.complete_failed(exit_code, error_message)
            self._save_stage_run(stage_run)
            logger.warning(f"Stage failed: run={context.run_id}, stage={stage.name}, exit_code={exit_code}")
            return False

        except asyncio.CancelledError:
            # 取消不是 stage 自身的失败，而是整个 pipeline 被外部中止。
            # 必须先把 StageRun 状态落库，再 raise，让信号继续向上传播到
            # execute_run，由它负责把 PipelineRun 标记为 canceled。
            # 不能 return False：那会让外层误以为是普通失败，把 run 标记为 faulted。
            stage_run.complete_faulted("Cancelled")
            self._save_stage_run(stage_run)
            raise
        except Exception as e:
            # 意外异常（容器 API 报错、网络中断等）与容器返回非零退出码对 pipeline
            # 来说结果相同：这个 stage 失败了。先落库 StageRun 状态，再 return False
            # 让外层走正常的失败流程（_cancel_remaining + run.complete_failed）。
            # 不 raise：外层 _execute_internal 的 except Exception 不会更新 StageRun，
            # 也不会调用 _cancel_remaining，直接 raise 会导致后续 stage 状态不一致。
            stage_run.complete_faulted(str(e))
            self._save_stage_run(stage_run)
            logger.error(f"Stage faulted: run={context.run_id}, stage={stage.name}, error={e}", exc_info=True)
            return False

    def _save_stage_run(self, stage_run: StageRun) -> None:
        """保存 StageRun 并立即提交。

        每次状态变更后立即 commit，而不是等整个 pipeline 结束再统一提交。
        这样前端轮询时能实时看到各 stage 的进度，而不是等 run 结束后才全部刷新。
        """
        self.stage_run_repo.save(stage_run)
        self.stage_run_repo.commit()

    async def _execute_container(
        self, context: ExecutionContext, stage: StageDefinition, log_file: "TextIO | None" = None
    ) -> tuple[int, str]:
        """普通容器执行：合并 stage.env 和 context.variables 作为环境变量"""
        env = {k: str(v) for k, v in context.variables.items()}
        env.update(stage.env)  # stage 级 env 优先

        commands = [line for line in stage.script.splitlines() if line.strip()]
        exit_code, output = await self.container_executor.run(
            image=stage.image,
            commands=commands,
            environment=env,
            workspace_path=Path(context.workspace_path),
            artifacts_path=Path(context.artifacts_path),
            volumes=[],
            log_file=log_file,
        )
        return exit_code, output

    async def _execute_clone(
        self, context: ExecutionContext, stage: StageDefinition, log_file: "TextIO | None" = None
    ) -> tuple[int, str]:
        """clone stage：处理 SSH key / token 挂载"""
        if not context.credential_id:
            raise RuntimeError("clone stage requires git_credential_id on the project")

        credential = self.credential_repo.find_by_id(context.credential_id)
        if not credential:
            raise RuntimeError(f"Credential {context.credential_id} not found")

        decrypted = copy(credential)
        decrypted.encrypted_data = self.security_service.decrypt_value(credential.encrypted_data)

        env = {k: str(v) for k, v in context.variables.items()}
        env.update(stage.env)
        extra_binds: dict[str, dict[str, str]] = {}
        key_path: Path | None = None

        if decrypted.type == CredentialType.GIT_SSH:
            secrets_path = get_secrets_path(context.run_id)
            secrets_path.mkdir(parents=True, exist_ok=True)
            key_path = secrets_path / "id_rsa"
            key_path.write_text(decrypted.get_private_key())
            try:
                key_path.chmod(0o600)
            except Exception:
                # chmod 失败时立即删除文件，避免以不安全权限留在磁盘
                key_path.unlink(missing_ok=True)
                raise
            extra_binds[str(secrets_path)] = {"bind": "/run/secrets", "mode": "ro"}

        elif decrypted.type == CredentialType.GIT_TOKEN:
            token = decrypted.get_token()
            repo_url = context.repository_url
            if repo_url.startswith("git@"):
                without_prefix = repo_url[len("git@") :]
                host, path = without_prefix.split(":", 1)
                repo_url = f"https://{token}@{host}/{path}"
            elif repo_url.startswith("https://"):
                repo_url = repo_url.replace("https://", f"https://{token}@")
            # 直接替换脚本中已渲染的 SSH URL 为带 token 的 HTTPS URL
            stage = copy(stage)
            stage.script = stage.script.replace(context.repository_url, repo_url)

        # 在容器内修正 SSH key 权限（Windows 宿主机 chmod 不生效）
        script_lines = [line for line in stage.script.splitlines() if line.strip()]
        commands = []
        if decrypted.type == CredentialType.GIT_SSH:
            commands.append("chmod 600 /run/secrets/id_rsa")
        commands.extend(script_lines)

        try:
            exit_code, output = await self.container_executor.run(
                image=stage.image,
                commands=commands,
                environment=env,
                workspace_path=Path(context.workspace_path),
                artifacts_path=Path(context.artifacts_path),
                volumes=[],
                extra_binds=extra_binds,
                log_file=log_file,
            )
        finally:
            # 容器执行完成后立即删除 SSH 私钥文件
            if key_path and key_path.exists():
                key_path.unlink()

        return exit_code, output

    async def _cancel_remaining(
        self,
        context: ExecutionContext,
        layers: list[list[str]],
        start_layer_idx: int,
        stages_map: dict[str, StageDefinition],
    ) -> None:
        # 某一层失败后，后续层的 stage 不会被执行，但它们的 StageRun 记录需要
        # 显式创建并标记为 canceled，否则前端看不到这些 stage，无法展示完整的执行图。
        for layer_idx in range(start_layer_idx, len(layers)):
            for stage_id in layers[layer_idx]:
                stage = stages_map[stage_id]
                stage_run = StageRun.create(pipeline_run_id=context.run_id, stage_id=stage.id, stage_name=stage.name)
                stage_run.status = TaskStatus.CANCELED
                self.stage_run_repo.save(stage_run)
            logger.info(f"Layer canceled: run={context.run_id}, layer={layer_idx}")

    def _save_artifacts(self, context: ExecutionContext, stage: StageDefinition, stage_name: str) -> None:
        if not stage.artifacts:
            return
        for a in stage.artifacts:
            full_path = Path(context.artifacts_path) / a.path
            if not full_path.exists():
                logger.warning(
                    f"Artifact file not found, skipping: run={context.run_id}, stage={stage_name}, path={full_path}"
                )
                continue
            artifact = Artifact.create(
                pipeline_run_id=context.run_id,
                stage_name=stage_name,
                artifact_type="file",
                name=a.name,
                path=str(full_path),
            )
            self.artifact_repo.save(artifact)
            logger.info(f"Artifact saved: run={context.run_id}, stage={stage_name}, name={a.name}")
