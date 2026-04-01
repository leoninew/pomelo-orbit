"""Pipeline 执行器实现"""

import asyncio
import logging
from pathlib import Path

from pomelo_orbit.domain.ci.entities import Artifact, Job, JobLog
from pomelo_orbit.domain.ci.executor import ExecutionContext, PipelineExecutor
from pomelo_orbit.domain.ci.value_objects import JobStatus, PipelineDefinition, StepDefinition
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

        # 并行执行
        tasks = [self._execute_step(context, step) for step in steps]
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
        self, context: ExecutionContext, step: StepDefinition
    ) -> bool:
        """
        执行单个 Step

        Args:
            context: 执行上下文
            step: Step 定义

        Returns:
            是否执行成功
        """
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
