"""依赖图构建和拓扑排序"""

from collections import defaultdict, deque

from pomelo_orbit.domain.ci.value_objects import StageDefinition


class CyclicDependencyError(Exception):
    pass


class DependencyGraph:
    """Stage 依赖图，用于拓扑排序和循环检测"""

    def __init__(self, stages: list[StageDefinition]):
        self.stages_by_id = {s.id: s for s in stages}
        self.graph: defaultdict[str, list[str]] = defaultdict(list)
        self.in_degree: dict[str, int] = {}
        self._build()

    def _build(self) -> None:
        for stage_id in self.stages_by_id:
            self.in_degree[stage_id] = 0

        for stage_id, stage in self.stages_by_id.items():
            for dep_id in stage.depends_on or []:
                if dep_id not in self.stages_by_id:
                    raise ValueError(f"Stage '{stage.name}' depends on unknown stage id '{dep_id}'")
                self.graph[dep_id].append(stage_id)
                self.in_degree[stage_id] += 1

    def topological_sort(self) -> list[list[str]]:
        """返回按层分组的 Stage id 列表，同层可并行执行"""
        layers: list[list[str]] = []
        in_degree_copy = self.in_degree.copy()
        queue: deque[str] = deque(stage_id for stage_id, d in in_degree_copy.items() if d == 0)
        processed = 0

        while queue:
            layer_size = len(queue)
            current_layer: list[str] = []
            for _ in range(layer_size):
                node = queue.popleft()
                current_layer.append(node)
                processed += 1
                for dependent in self.graph[node]:
                    in_degree_copy[dependent] -= 1
                    if in_degree_copy[dependent] == 0:
                        queue.append(dependent)
            layers.append(current_layer)

        if processed != len(self.stages_by_id):
            unprocessed = [n for n, d in in_degree_copy.items() if d > 0]
            raise CyclicDependencyError(f"Cyclic dependency involving: {', '.join(unprocessed)}")

        return layers

    def get_ready_stages(self) -> list[str]:
        """返回当前入度为 0 的所有 stage id（可立即执行）"""
        return [stage_id for stage_id, degree in self.in_degree.items() if degree == 0]

    def mark_completed(self, stage_id: str) -> None:
        """标记某个 stage 完成，更新其依赖者的入度"""
        for dependent in self.graph.get(stage_id, []):
            self.in_degree[dependent] -= 1

    def remove_job(self, stage_id: str) -> None:
        """从图中移除某个 stage 及其所有边"""
        self.in_degree.pop(stage_id, None)
        self.graph.pop(stage_id, None)
        for node in list(self.graph.keys()):
            if stage_id in self.graph[node]:
                self.graph[node].remove(stage_id)

    def has_cycle(self) -> bool:
        try:
            self.topological_sort()
            return False
        except CyclicDependencyError:
            return True
