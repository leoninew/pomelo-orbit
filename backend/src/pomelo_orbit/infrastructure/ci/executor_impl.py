"""Pipeline 执行器实现"""

import asyncio
import copy
import logging
from pathlib import Path

from pomelo_orbit.domain.ci.entities import Artifact, Job, JobLog
from pomelo_orbit.domain.ci.executor import ExecutionContext, PipelineExecutor
from pomelo_orbit.domain.ci.repositories import (
    ArtifactRepository,
    CredentialRepository,
    JobLogRepository,
    JobRepository,
)
from pomelo_orbit.domain.ci.value_objects import (
    CheckoutStageConfig,
    DockerBuildStageConfig,
    JobStatus,
    RetryPolicy,
    StageDefinition,
    StageType,
    StepDefinition,
    UnitTestStageConfig,
)
from pomelo_orbit.infrastructure.ci.actions.checkout import CheckoutAction
from pomelo_orbit.infrastructure.ci.container import ContainerExecutor
from pomelo_orbit.infrastructure.ci.dependency_graph import CyclicDependencyError, DependencyGraph
from pomelo_orbit.infrastructure.security import SecurityService

logger = logging.getLogger(__name__)


def _stage_to_steps(stage: StageDefinition) -> list[StepDefinition]:
    """将 Stage 转换为可执行的 StepDefinition 列表（内部用）"""
    if stage.type == StageType.CHECKOUT:
        assert isinstance(stage.config, CheckoutStageConfig), f"Stage '{stage.name}' of type checkout is missing config"
        return [
            StepDefinition(
                name=stage.name,
                uses="checkout",
                inputs={"ref": stage.config.ref},
                retry_policy=RetryPolicy.ALWAYS_RERUN,
            )
        ]

    if stage.type == StageType.DOCKER_BUILD:
        assert isinstance(stage.config, DockerBuildStageConfig)
        build_config = stage.config
        return [
            StepDefinition(
                name=stage.name,
                image="docker:latest",
                commands=[
                    f"docker build -t {build_config.image_name} -f {build_config.dockerfile} {build_config.context}",
                    f"docker push {build_config.image_name}",
                ],
                volumes=["/var/run/docker.sock:/var/run/docker.sock"],
                artifacts=[{"type": "docker_image", "name": build_config.image_name}],
            )
        ]

    if stage.type == StageType.UNIT_TEST:
        assert isinstance(stage.config, UnitTestStageConfig)
        test_config = stage.config
        artifacts = [{"type": "file", "path": p, "name": p} for p in test_config.artifact_paths]
        return [
            StepDefinition(
                name=stage.name,
                image=test_config.image,
                commands=test_config.commands,
                artifacts=artifacts if artifacts else None,
            )
        ]

    if stage.type == StageType.CUSTOM:
        return stage.steps or []

    return []  # type: ignore[unreachable]


class PipelineExecutorImpl(PipelineExecutor):
    """Pipeline 执行器实现"""

    def __init__(
        self,
        job_repo: JobRepository,
        job_log_repo: JobLogRepository,
        container_executor: ContainerExecutor,
        artifact_repo: ArtifactRepository,
        credential_repo: CredentialRepository,
        security_service: SecurityService,
    ):
        self.job_repo = job_repo
        self.job_log_repo = job_log_repo
        self.container_executor = container_executor
        self.artifact_repo = artifact_repo
        self.credential_repo = credential_repo
        self.security_service = security_service

    async def execute(self, context: ExecutionContext, stages: list[StageDefinition]) -> bool:
        """按 Stage 依赖关系拓扑排序后并行执行"""
        try:
            # 以 stage.name 为节点，stage.depends_on 为边构建依赖图
            # 将 stages 转换为 DependencyGraph 期望的 StepDefinition 格式
            pseudo_steps = [StepDefinition(name=s.name, depends_on=s.depends_on or []) for s in stages]
            graph = DependencyGraph(pseudo_steps)

            if graph.has_cycle():
                logger.error(f"Cyclic dependency detected: run={context.run_id}")
                return False

            layers = graph.topological_sort()
            stages_map = {s.name: s for s in stages}

            logger.info(
                f"Pipeline execution plan: run={context.run_id}, layers={len(layers)}, total_stages={len(stages)}"
            )

            for layer_idx, layer in enumerate(layers):
                logger.info(f"Executing layer {layer_idx + 1}/{len(layers)}: run={context.run_id}, stages={layer}")
                layer_stages = [stages_map[name] for name in layer]
                results = await self._execute_layer(context, layer_stages)

                failed = [name for name, ok in results.items() if not ok]
                if failed:
                    logger.warning(f"Layer failed: run={context.run_id}, failed_stages={failed}")
                    await self._cancel_remaining(context, layers, layer_idx + 1, stages_map)
                    return False

            logger.info(f"Pipeline execution succeeded: run={context.run_id}")
            return True

        except CyclicDependencyError as e:
            logger.error(f"Cyclic dependency error: run={context.run_id}, error={e}")
            return False
        except Exception as e:
            logger.error(f"Pipeline execution failed: run={context.run_id}, error={e}", exc_info=True)
            return False

    async def _execute_layer(
        self,
        context: ExecutionContext,
        stages: list[StageDefinition],
    ) -> dict[str, bool]:
        """并行执行同一层的所有 Stage"""
        original_jobs_map: dict[str, Job] = {}
        current_jobs_map: dict[str, Job] = {}
        if context.retry_of:
            original_jobs_map = {j.name: j for j in self.job_repo.find_by_run(context.retry_of)}
            current_jobs_map = {j.name: j for j in self.job_repo.find_by_run(context.run_id)}

        tasks = [self._execute_stage(context, stage, original_jobs_map, current_jobs_map) for stage in stages]
        results = await asyncio.gather(*tasks, return_exceptions=True)

        result_map: dict[str, bool] = {}
        for stage, result in zip(stages, results, strict=True):
            if isinstance(result, Exception):
                logger.error(
                    f"Stage raised exception: run={context.run_id}, stage={stage.name}, error={result}", exc_info=result
                )
                result_map[stage.name] = False
            else:
                result_map[stage.name] = bool(result)
        return result_map

    async def _execute_stage(
        self,
        context: ExecutionContext,
        stage: StageDefinition,
        original_jobs_map: dict[str, Job],
        current_jobs_map: dict[str, Job],
    ) -> bool:
        """执行单个 Stage（转换为 Steps 后逐步执行）"""
        steps = _stage_to_steps(stage)
        if not steps:
            logger.warning(f"Stage has no steps: run={context.run_id}, stage={stage.name}")
            return True

        for step in steps:
            # 重试场景：检查是否应跳过
            if context.retry_of and self._should_skip(step, original_jobs_map, current_jobs_map):
                job = Job.create(pipeline_run_id=context.run_id, name=step.name)
                job.status = JobStatus.SKIPPED
                self.job_repo.save(job)
                logger.info(f"Step skipped (retry): run={context.run_id}, step={step.name}")
                continue

            success = await self._execute_step(context, step)
            if not success:
                return False

        return True

    async def _execute_step(self, context: ExecutionContext, step: StepDefinition) -> bool:
        """执行单个 Step"""
        job = Job.create(pipeline_run_id=context.run_id, name=step.name)
        self.job_repo.save(job)

        try:
            job.start()
            self.job_repo.save(job)
            logger.info(f"Step started: run={context.run_id}, step={step.name}")

            if step.uses:
                output, exit_code = await self._execute_action(context, step, job)
            else:
                if not step.image:
                    raise RuntimeError(f"Step '{step.name}' has no image defined")
                exit_code, output = await self.container_executor.run(
                    image=step.image,
                    commands=step.commands or [],
                    environment={k: str(v) for k, v in context.variables.items()},
                    workspace_path=Path(context.workspace_path),
                    artifacts_path=Path(context.artifacts_path),
                    volumes=step.volumes or [],
                )

            log = JobLog.create(job_id=job.id, content=output or "")
            self.job_log_repo.save(log)

            if exit_code == 0:
                job.complete_success(exit_code)
                self.job_repo.save(job)
                self._save_artifacts(context, step, job.name)
                logger.info(f"Step succeeded: run={context.run_id}, step={step.name}")
                return True

            job.complete_failed(exit_code, f"Exit code: {exit_code}")
            self.job_repo.save(job)
            logger.warning(f"Step failed: run={context.run_id}, step={step.name}, exit_code={exit_code}")
            return False

        except Exception as e:
            job.complete_faulted(str(e))
            self.job_repo.save(job)
            logger.error(f"Step faulted: run={context.run_id}, step={step.name}, error={e}", exc_info=True)
            return False

    async def _execute_action(
        self,
        context: ExecutionContext,
        step: StepDefinition,
        job: Job,
    ) -> tuple[str, int]:
        """处理 uses action"""
        action_name = (step.uses or "").split("@")[0].lower()

        if action_name == "checkout":
            if not context.credential_id:
                raise RuntimeError("checkout action requires a credential (set git_credential_id on the project)")
            credential = self.credential_repo.find_by_id(context.credential_id)
            if not credential:
                raise RuntimeError(f"Credential {context.credential_id} not found")
            decrypted = copy.copy(credential)
            decrypted.encrypted_data = self.security_service.decrypt_value(credential.encrypted_data)
            checkout = CheckoutAction(self.container_executor)
            try:
                outputs = await checkout.execute(
                    step=step,
                    repository_url=context.repository_url,
                    credential=decrypted,
                    trigger_ref=context.variables.get("trigger_ref", "master"),
                    workspace_path=Path(context.workspace_path),
                    artifacts_path=Path(context.artifacts_path),
                    run_id=context.run_id,
                )
                return "\n".join(f"{k}={v}" for k, v in outputs.items()), 0
            except Exception as e:
                return str(e), 1
        else:
            logger.warning(f"Unknown action: {step.uses}, skipping")
            return f"Unknown action: {step.uses}", 1

    async def _cancel_remaining(
        self,
        context: ExecutionContext,
        layers: list[list[str]],
        start_layer_idx: int,
        stages_map: dict[str, StageDefinition],
    ) -> None:
        for layer_idx in range(start_layer_idx, len(layers)):
            for stage_name in layers[layer_idx]:
                stage = stages_map.get(stage_name)
                steps = _stage_to_steps(stage) if stage else []
                for step in steps:
                    job = Job.create(pipeline_run_id=context.run_id, name=step.name)
                    job.status = JobStatus.CANCELED
                    self.job_repo.save(job)
                logger.info(f"Stage canceled: run={context.run_id}, stage={stage_name}")

    def _should_skip(
        self,
        step: StepDefinition,
        original_jobs_map: dict[str, Job],
        current_jobs_map: dict[str, Job],
    ) -> bool:
        if step.retry_policy != RetryPolicy.SKIP_IF_SUCCESS:
            return False
        original_job = original_jobs_map.get(step.name)
        if not original_job or original_job.status != JobStatus.SUCCESS:
            return False
        if step.depends_on:
            for dep_name in step.depends_on:
                dep_job = current_jobs_map.get(dep_name)
                if dep_job and dep_job.status != JobStatus.SKIPPED:
                    return False
        return True

    def _save_artifacts(self, context: ExecutionContext, step: StepDefinition, job_name: str) -> None:
        if not step.artifacts:
            return
        for artifact_def in step.artifacts:
            artifact_type = artifact_def.get("type", "file")
            artifact_name = artifact_def.get("name", "")
            artifact_path = artifact_def.get("path")

            if artifact_type not in {"docker_image", "file"}:
                logger.warning(f"Unknown artifact type: run={context.run_id}, job={job_name}, type={artifact_type}")
                continue
            if not artifact_name:
                logger.warning(f"Artifact missing name: run={context.run_id}, job={job_name}")
                continue

            full_path = (
                str(Path(context.artifacts_path) / artifact_path)
                if artifact_type == "file" and artifact_path
                else artifact_path
            )

            artifact = Artifact.create(
                pipeline_run_id=context.run_id,
                job_name=job_name,
                artifact_type=artifact_type,
                name=artifact_name,
                path=str(full_path) if full_path else None,
            )
            self.artifact_repo.save(artifact)
            logger.info(
                f"Artifact saved: run={context.run_id}, job={job_name}, type={artifact_type}, name={artifact_name}"
            )
