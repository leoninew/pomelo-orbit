"""Pipeline 执行器实现"""

import asyncio
import logging
from pathlib import Path

from pomelo_orbit.domain.ci.entities import Artifact, Job, JobLog
from pomelo_orbit.domain.ci.executor import ExecutionContext, PipelineExecutor
from pomelo_orbit.domain.ci.value_objects import JobStatus, PipelineDefinition, RetryPolicy, StepDefinition
from pomelo_orbit.infrastructure.ci.container import ContainerExecutor
from pomelo_orbit.infrastructure.ci.dependency_graph import CyclicDependencyError, DependencyGraph
from pomelo_orbit.infrastructure.ci.repositories import ArtifactRepository, JobLogRepository, JobRepository

logger = logging.getLogger(__name__)


class PipelineExecutorImpl(PipelineExecutor):
    """Pipeline 执行器实现"""

    def __init__(
        self,
        job_repo: JobRepository,
        job_log_repo: JobLogRepository,
        container_executor: ContainerExecutor,
        artifact_repo: ArtifactRepository | None = None,
    ):
        self.job_repo = job_repo
        self.job_log_repo = job_log_repo
        self.container_executor = container_executor
        self.artifact_repo = artifact_repo

    async def execute(
        self, context: ExecutionContext, definition: PipelineDefinition
    ) -> bool:
        """
        执行 pipeline

        Args:
            context: 执行上下文
            definition: Pipeline 定义

        Returns:
            是否执行成功
        """
        try:
            # 构建依赖图
            graph = DependencyGraph(definition.steps)

            # 检查循环依赖
            if graph.has_cycle():
                logger.error(f"Cyclic dependency detected: run={context.run_id}")
                return False

            # 拓扑排序获取执行层级
            layers = graph.topological_sort()
            logger.info(
                f"Pipeline execution plan: run={context.run_id}, "
                f"layers={len(layers)}, total_jobs={len(definition.steps)}"
            )

            # 按层级并行执行
            for layer_idx, layer in enumerate(layers):
                logger.info(
                    f"Executing layer {layer_idx + 1}/{len(layers)}: "
                    f"run={context.run_id}, jobs={layer}"
                )

                # 并行执行该层所有 Job
                results = await self._execute_layer(context, definition, layer)

                # 检查是否有失败的 Job（Fail-fast）
                failed_jobs = [name for name, success in results.items() if not success]
                if failed_jobs:
                    logger.warning(
                        f"Layer execution failed: run={context.run_id}, "
                        f"failed_jobs={failed_jobs}"
                    )
                    # 取消后续所有 Job
                    await self._cancel_remaining_jobs(context, layers, layer_idx + 1)
                    return False

            logger.info(f"Pipeline execution succeeded: run={context.run_id}")
            return True

        except CyclicDependencyError as e:
            logger.error(f"Cyclic dependency error: run={context.run_id}, error={e}")
            return False
        except Exception as e:
            logger.error(
                f"Pipeline execution failed: run={context.run_id}, error={e}",
                exc_info=True,
            )
            return False

    async def _execute_layer(
        self,
        context: ExecutionContext,
        definition: PipelineDefinition,
        job_names: list[str],
    ) -> dict[str, bool]:
        """
        并行执行一层 Job

        Args:
            context: 执行上下文
            definition: Pipeline 定义
            job_names: 该层的 Job 名称列表

        Returns:
            Job 名称 -> 是否成功的映射
        """
        # 找到对应的 StepDefinition
        steps_map = {step.name: step for step in definition.steps}
        steps = [steps_map[name] for name in job_names]

        # 预加载重试场景所需的数据（避免在每个 job 中重复查询）
        original_jobs_map: dict[str, Job] = {}
        current_jobs_map: dict[str, Job] = {}
        if context.retry_of:
            original_jobs = self.job_repo.find_by_run(context.retry_of)
            original_jobs_map = {job.name: job for job in original_jobs}
            current_jobs = self.job_repo.find_by_run(context.run_id)
            current_jobs_map = {job.name: job for job in current_jobs}

        # 并行执行
        tasks = [
            self._execute_step(context, step, original_jobs_map, current_jobs_map)
            for step in steps
        ]
        results = await asyncio.gather(*tasks, return_exceptions=True)

        # 构建结果映射
        result_map: dict[str, bool] = {}
        for step, result in zip(steps, results, strict=True):
            if isinstance(result, Exception):
                logger.error(
                    f"Job execution raised exception: run={context.run_id}, "
                    f"job={step.name}, error={result}",
                    exc_info=result,
                )
                result_map[step.name] = False
            elif isinstance(result, bool):
                result_map[step.name] = result

        return result_map

    async def _execute_step(
        self,
        context: ExecutionContext,
        step: StepDefinition,
        original_jobs_map: dict[str, Job] | None = None,
        current_jobs_map: dict[str, Job] | None = None,
    ) -> bool:
        """
        执行单个 Step

        Args:
            context: 执行上下文
            step: Step 定义
            original_jobs_map: 原 Run 的 jobs 映射（重试场景，可选）
            current_jobs_map: 当前 Run 的 jobs 映射（重试场景，可选）

        Returns:
            是否执行成功
        """
        # 检查是否应该跳过（重试场景）
        if context.retry_of and await self._should_skip_job(
            context, step, original_jobs_map or {}, current_jobs_map or {}
        ):
            job = Job.create(
                pipeline_run_id=context.run_id,
                name=step.name,
            )
            job.status = JobStatus.SKIPPED
            self.job_repo.save(job)
            logger.info(
                f"Job skipped (retry): run={context.run_id}, job={step.name}, "
                f"original_run={context.retry_of}"
            )
            return True

        # 创建 Job 记录
        job = Job.create(
            pipeline_run_id=context.run_id,
            name=step.name,
        )
        self.job_repo.save(job)

        try:
            job.start()
            self.job_repo.save(job)

            logger.info(f"Job started: run={context.run_id}, job={step.name}")

            # 执行容器
            exit_code, output = await self.container_executor.run(
                image=step.image or "alpine:latest",
                commands=step.commands or [],
                environment={k: str(v) for k, v in context.variables.items()},
                workspace_path=Path(context.workspace_path),
                artifacts_path=Path(context.artifacts_path),
                volumes=step.volumes or [],
            )

            # 保存日志
            if output:
                log = JobLog.create(job_id=job.id, content=output)
                self.job_log_repo.save(log)

            # 更新 Job 状态
            if exit_code == 0:
                job.complete_success(exit_code)
                self.job_repo.save(job)
                # 记录制品
                self._save_artifacts(context, step, job.name)
                logger.info(
                    f"Job succeeded: run={context.run_id}, job={step.name}, "
                    f"exit_code={exit_code}"
                )
                return True
            job.complete_failed(exit_code, f"Exit code: {exit_code}")
            self.job_repo.save(job)
            logger.warning(
                f"Job failed: run={context.run_id}, job={step.name}, "
                f"exit_code={exit_code}"
            )
            return False

        except Exception as e:
            job.complete_faulted(str(e))
            self.job_repo.save(job)
            logger.error(
                f"Job faulted: run={context.run_id}, job={step.name}, error={e}",
                exc_info=True,
            )
            return False

    async def _cancel_remaining_jobs(
        self,
        context: ExecutionContext,
        layers: list[list[str]],
        start_layer_idx: int,
    ) -> None:
        """
        取消剩余未执行的 Job

        Args:
            context: 执行上下文
            layers: 所有层级
            start_layer_idx: 开始取消的层级索引
        """
        for layer_idx in range(start_layer_idx, len(layers)):
            for job_name in layers[layer_idx]:
                job = Job.create(
                    pipeline_run_id=context.run_id,
                    name=job_name,
                )
                job.status = JobStatus.CANCELED
                self.job_repo.save(job)
                logger.info(
                    f"Job canceled: run={context.run_id}, job={job_name}"
                )

    async def _should_skip_job(
        self,
        context: ExecutionContext,
        step: StepDefinition,
        original_jobs_map: dict[str, Job],
        current_jobs_map: dict[str, Job],
    ) -> bool:
        """
        判断 Job 是否应该跳过（重试场景）

        跳过条件：
        1. retry_policy 为 skip_if_success
        2. 原 Run 中该 Job 执行成功
        3. 该 Job 的所有依赖都被跳过（没有重新执行）

        Args:
            context: 执行上下文
            step: Step 定义
            original_jobs_map: 原 Run 的 jobs 映射
            current_jobs_map: 当前 Run 的 jobs 映射

        Returns:
            是否应该跳过
        """
        # 没有 retry_of，不跳过
        if not context.retry_of:
            return False

        # retry_policy 不是 skip_if_success，不跳过
        if step.retry_policy != RetryPolicy.SKIP_IF_SUCCESS:
            return False

        # 查询原 Run 中该 Job 的状态
        original_job = original_jobs_map.get(step.name)
        if not original_job or original_job.status != JobStatus.SUCCESS:
            # 原 Job 不存在或未成功，不跳过
            return False

        # 检查依赖：如果有任何依赖被重新执行，则不能跳过
        if step.depends_on:
            for dep_name in step.depends_on:
                dep_job = current_jobs_map.get(dep_name)
                # 依赖已执行且不是 SKIPPED 状态，说明依赖被重新执行了
                if dep_job and dep_job.status != JobStatus.SKIPPED:
                    logger.info(
                        f"Job cannot be skipped: run={context.run_id}, job={step.name}, "
                        f"dependency={dep_name} was re-executed"
                    )
                    return False

        return True

    def _save_artifacts(
        self,
        context: ExecutionContext,
        step: StepDefinition,
        job_name: str,
    ) -> None:
        """job 成功后保存制品记录"""
        if not step.artifacts or not self.artifact_repo:
            return

        for artifact_def in step.artifacts:
            artifact_type = artifact_def.get("type", "file")
            artifact_name = artifact_def.get("name", "")
            artifact_path = artifact_def.get("path")

            if artifact_type not in {"docker_image", "file"}:
                logger.warning(
                    f"Unknown artifact type, skipping: run={context.run_id}, "
                    f"job={job_name}, type={artifact_type}"
                )
                continue

            if not artifact_name:
                logger.warning(
                    f"Artifact declaration missing name, skipping: "
                    f"run={context.run_id}, job={job_name}"
                )
                continue

            # file 类型：path 相对于 artifacts_path
            if artifact_type == "file" and artifact_path:
                full_path: str | None = str(Path(context.artifacts_path) / artifact_path)
            else:
                full_path = str(artifact_path) if artifact_path else None

            artifact = Artifact.create(
                pipeline_run_id=context.run_id,
                job_name=job_name,
                artifact_type=artifact_type,
                name=artifact_name,
                path=full_path,
            )
            self.artifact_repo.save(artifact)
            logger.info(
                f"Artifact saved: run={context.run_id}, job={job_name}, "
                f"type={artifact_type}, name={artifact_name}"
            )
