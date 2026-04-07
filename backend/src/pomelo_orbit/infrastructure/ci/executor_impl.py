"""Pipeline 执行器实现"""

import asyncio
import logging
from copy import copy
from pathlib import Path

from pomelo_orbit.domain.ci.entities import Artifact, StageLog, StageRun
from pomelo_orbit.domain.ci.executor import ExecutionContext, PipelineExecutor
from pomelo_orbit.domain.ci.repositories import (
    ArtifactRepository,
    CredentialRepository,
    StageLogRepository,
    StageRunRepository,
)
from pomelo_orbit.domain.ci.value_objects import CredentialType, StageDefinition, StageStatus
from pomelo_orbit.infrastructure.ci.container import ContainerExecutor
from pomelo_orbit.infrastructure.ci.dependency_graph import CyclicDependencyError, DependencyGraph
from pomelo_orbit.infrastructure.ci.workspace import get_secrets_path
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
        stage_log_repo: StageLogRepository,
        container_executor: ContainerExecutor,
        artifact_repo: ArtifactRepository,
        credential_repo: CredentialRepository,
        security_service: SecurityService,
    ):
        self.stage_run_repo = stage_run_repo
        self.stage_log_repo = stage_log_repo
        self.container_executor = container_executor
        self.artifact_repo = artifact_repo
        self.credential_repo = credential_repo
        self.security_service = security_service

    async def execute(self, context: ExecutionContext, stages: list[StageDefinition]) -> bool:
        """按 Stage 依赖关系拓扑排序后分层并行执行，1 Stage = 1 Job"""
        task = asyncio.current_task()
        if task:
            register_task(context.run_id, task)
        try:
            return await self._execute_internal(context, stages)
        finally:
            unregister_task(context.run_id)

    async def _execute_internal(self, context: ExecutionContext, stages: list[StageDefinition]) -> bool:
        try:
            graph = DependencyGraph(stages)
            try:
                layers = graph.topological_sort()
            except CyclicDependencyError:
                logger.error(f"Cyclic dependency: run={context.run_id}")
                return False

            stages_map = {s.name: s for s in stages}

            for layer_idx, layer in enumerate(layers):
                logger.info(f"Executing layer: index={layer_idx + 1}, total={len(layers)}, run={context.run_id}, stages={layer}")
                layer_stages = [stages_map[name] for name in layer]
                results = await self._execute_layer(context, layer_stages)

                failed = [name for name, ok in results.items() if not ok]
                if failed:
                    logger.warning(f"Layer failed: run={context.run_id}, failed={failed}")
                    await self._cancel_remaining(context, layers, layer_idx + 1, stages_map)
                    return False

            return True

        except asyncio.CancelledError:
            logger.info(f"Pipeline cancelled: run={context.run_id}")
            raise
        except Exception as e:
            logger.error(f"Pipeline execution failed: run={context.run_id}, error={e}", exc_info=True)
            return False

    async def _execute_layer(self, context: ExecutionContext, stages: list[StageDefinition]) -> dict[str, bool]:
        """并行执行同一层的所有 Stage"""
        tasks = [self._execute_stage(context, stage) for stage in stages]
        # 不用 return_exceptions=True，让 CancelledError 正常向上传播
        results = await asyncio.gather(*tasks, return_exceptions=False)

        result_map: dict[str, bool] = {}
        for stage, result in zip(stages, results, strict=True):
            result_map[stage.name] = bool(result)
        return result_map

    async def _execute_stage(self, context: ExecutionContext, stage: StageDefinition) -> bool:
        """执行单个 Stage，创建 StageRun 记录"""
        stage_run = StageRun.create(pipeline_run_id=context.run_id, name=stage.name)
        self.stage_run_repo.save(stage_run)

        try:
            stage_run.start()
            self.stage_run_repo.save(stage_run)
            logger.info(f"Stage started: run={context.run_id}, stage={stage.name}, image={stage.image}")

            if stage.name == "clone":
                exit_code, output = await self._execute_clone(context, stage)
            else:
                exit_code, output = await self._execute_container(context, stage)

            log = StageLog.create(stage_run_id=stage_run.id, content=output or "")
            self.stage_log_repo.save(log)

            if exit_code == 0:
                stage_run.complete_success(exit_code)
                self.stage_run_repo.save(stage_run)
                self._save_artifacts(context, stage, stage_run.name)
                logger.info(f"Stage succeeded: run={context.run_id}, stage={stage.name}")
                return True

            stage_run.complete_failed(exit_code, f"Exit code: {exit_code}")
            self.stage_run_repo.save(stage_run)
            logger.warning(f"Stage failed: run={context.run_id}, stage={stage.name}, exit_code={exit_code}")
            return False

        except asyncio.CancelledError:
            stage_run.complete_faulted("Cancelled")
            self.stage_run_repo.save(stage_run)
            raise
        except Exception as e:
            stage_run.complete_faulted(str(e))
            self.stage_run_repo.save(stage_run)
            logger.error(f"Stage faulted: run={context.run_id}, stage={stage.name}, error={e}", exc_info=True)
            return False

    async def _execute_container(self, context: ExecutionContext, stage: StageDefinition) -> tuple[int, str]:
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
        )
        return exit_code, output

    async def _execute_clone(self, context: ExecutionContext, stage: StageDefinition) -> tuple[int, str]:
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
            secrets_path = get_secrets_path(context.project_code, context.run_id)
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
            # 替换命令中的 project_repository_url
            env["project_repository_url"] = repo_url

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
        for layer_idx in range(start_layer_idx, len(layers)):
            for stage_name in layers[layer_idx]:
                stage_run = StageRun.create(pipeline_run_id=context.run_id, name=stage_name)
                stage_run.status = StageStatus.CANCELED
                self.stage_run_repo.save(stage_run)
            logger.info(f"Layer canceled: run={context.run_id}, layer={layer_idx}")

    def _save_artifacts(self, context: ExecutionContext, stage: StageDefinition, stage_name: str) -> None:
        if not stage.artifacts:
            return
        for a in stage.artifacts:
            full_path = str(Path(context.artifacts_path) / a.path)
            artifact = Artifact.create(
                pipeline_run_id=context.run_id,
                stage_name=stage_name,
                artifact_type="file",
                name=a.name,
                path=full_path,
            )
            self.artifact_repo.save(artifact)
            logger.info(f"Artifact saved: run={context.run_id}, stage={stage_name}, name={a.name}")
