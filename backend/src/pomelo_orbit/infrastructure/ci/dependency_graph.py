"""依赖图构建和拓扑排序"""

from collections import defaultdict, deque

from pomelo_orbit.domain.ci.value_objects import StepDefinition


class CyclicDependencyError(Exception):
    """循环依赖错误"""


class DependencyGraph:
    """Job 依赖图"""

    def __init__(self, steps: list[StepDefinition]):
        self.steps = {step.name: step for step in steps}
        self.graph: defaultdict[str, list[str]] = defaultdict(list)  # name -> [依赖的 name]
        self.in_degree: dict[str, int] = {}  # name -> 入度
        self._build()

    def _build(self) -> None:
        """构建依赖图"""
        # 初始化所有节点
        for name in self.steps:
            self.in_degree[name] = 0

        # 构建边和入度
        for name, step in self.steps.items():
            if step.depends_on:
                for dep in step.depends_on:
                    if dep not in self.steps:
                        raise ValueError(f"Job '{name}' depends on unknown job '{dep}'")
                    self.graph[dep].append(name)  # dep -> name 的边
                    self.in_degree[name] += 1

    def get_ready_jobs(self) -> list[str]:
        """获取所有入度为 0 的 Job（可以立即执行）"""
        return [name for name, degree in self.in_degree.items() if degree == 0]

    def mark_completed(self, job_name: str) -> None:
        """标记 Job 完成，更新依赖它的 Job 的入度"""
        for dependent in self.graph[job_name]:
            self.in_degree[dependent] -= 1

    def remove_job(self, job_name: str) -> None:
        """从图中移除 Job（用于跳过或取消）"""
        self.in_degree.pop(job_name, None)
        # 清理 graph 中的引用
        self.graph.pop(job_name, None)
        for deps in self.graph.values():
            if job_name in deps:
                deps.remove(job_name)

    def topological_sort(self) -> list[list[str]]:
        """拓扑排序，返回按层级分组的 Job 列表

        Returns:
            list[list[str]]: 每个子列表是一层，同一层的 Job 可以并行执行

        Raises:
            CyclicDependencyError: 如果存在循环依赖
        """
        layers: list[list[str]] = []
        in_degree_copy = self.in_degree.copy()
        queue: deque[str] = deque()

        # 找出所有入度为 0 的节点
        for name, degree in in_degree_copy.items():
            if degree == 0:
                queue.append(name)

        processed_count = 0

        while queue:
            # 当前层的所有节点
            layer_size = len(queue)
            current_layer: list[str] = []

            for _ in range(layer_size):
                node = queue.popleft()
                current_layer.append(node)
                processed_count += 1

                # 更新依赖当前节点的节点的入度
                for dependent in self.graph[node]:
                    in_degree_copy[dependent] -= 1
                    if in_degree_copy[dependent] == 0:
                        queue.append(dependent)

            layers.append(current_layer)

        # 检查是否所有节点都被处理（检测循环依赖）
        if processed_count != len(self.steps):
            unprocessed = [name for name, degree in in_degree_copy.items() if degree > 0]
            raise CyclicDependencyError(
                f"Cyclic dependency detected involving jobs: {', '.join(unprocessed)}"
            )

        return layers

    def has_cycle(self) -> bool:
        """检查是否存在循环依赖"""
        try:
            self.topological_sort()
            return False
        except CyclicDependencyError:
            return True
