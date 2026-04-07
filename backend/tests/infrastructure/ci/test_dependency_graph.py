"""依赖图测试"""

import pytest

from pomelo_orbit.domain.ci.value_objects import StageDefinition
from pomelo_orbit.infrastructure.ci.dependency_graph import (
    CyclicDependencyError,
    DependencyGraph,
)


def stage(name: str, depends_on: list[str] | None = None) -> StageDefinition:
    return StageDefinition(name=name, id=name, image="alpine", script="echo", depends_on=depends_on or [])


class TestDependencyGraph:
    def test_build_simple_graph(self):
        stages = [stage("a"), stage("b", ["a"]), stage("c", ["b"])]
        graph = DependencyGraph(stages)

        assert graph.in_degree["a"] == 0
        assert graph.in_degree["b"] == 1
        assert graph.in_degree["c"] == 1
        assert graph.graph["a"] == ["b"]
        assert graph.graph["b"] == ["c"]

    def test_build_parallel_graph(self):
        stages = [stage("a"), stage("b"), stage("c", ["a", "b"])]
        graph = DependencyGraph(stages)

        assert graph.in_degree["a"] == 0
        assert graph.in_degree["b"] == 0
        assert graph.in_degree["c"] == 2
        assert "c" in graph.graph["a"]
        assert "c" in graph.graph["b"]

    def test_get_ready_stages(self):
        stages = [stage("a"), stage("b"), stage("c", ["a"])]
        graph = DependencyGraph(stages)

        ready = graph.get_ready_stages()
        assert set(ready) == {"a", "b"}

    def test_mark_completed(self):
        stages = [stage("a"), stage("b", ["a"])]
        graph = DependencyGraph(stages)

        assert graph.in_degree["b"] == 1
        graph.mark_completed("a")
        assert graph.in_degree["b"] == 0

    def test_topological_sort_linear(self):
        stages = [stage("a"), stage("b", ["a"]), stage("c", ["b"])]
        graph = DependencyGraph(stages)

        layers = graph.topological_sort()
        assert len(layers) == 3
        assert layers[0] == ["a"]
        assert layers[1] == ["b"]
        assert layers[2] == ["c"]

    def test_topological_sort_parallel(self):
        stages = [stage("a"), stage("b"), stage("c", ["a", "b"])]
        graph = DependencyGraph(stages)

        layers = graph.topological_sort()
        assert len(layers) == 2
        assert set(layers[0]) == {"a", "b"}
        assert layers[1] == ["c"]

    def test_topological_sort_complex(self):
        stages = [
            stage("checkout"),
            stage("lint", ["checkout"]),
            stage("test", ["checkout"]),
            stage("build", ["lint", "test"]),
        ]
        graph = DependencyGraph(stages)

        layers = graph.topological_sort()
        assert len(layers) == 3
        assert layers[0] == ["checkout"]
        assert set(layers[1]) == {"lint", "test"}
        assert layers[2] == ["build"]

    def test_cyclic_dependency_detection(self):
        stages = [stage("a", ["b"]), stage("b", ["a"])]
        graph = DependencyGraph(stages)

        with pytest.raises(CyclicDependencyError) as exc_info:
            graph.topological_sort()
        assert "Cyclic dependency" in str(exc_info.value)

    def test_self_dependency_detection(self):
        stages = [stage("a", ["a"])]
        graph = DependencyGraph(stages)

        with pytest.raises(CyclicDependencyError):
            graph.topological_sort()

    def test_unknown_dependency(self):
        stages = [stage("a", ["unknown"])]

        with pytest.raises(ValueError) as exc_info:
            DependencyGraph(stages)
        assert "unknown" in str(exc_info.value)

    def test_has_cycle_false(self):
        stages = [stage("a"), stage("b", ["a"])]
        assert not DependencyGraph(stages).has_cycle()

    def test_has_cycle_true(self):
        stages = [stage("a", ["b"]), stage("b", ["a"])]
        assert DependencyGraph(stages).has_cycle()

    def test_remove_job(self):
        stages = [stage("a"), stage("b", ["a"]), stage("c", ["b"])]
        graph = DependencyGraph(stages)

        graph.remove_job("b")

        assert "b" not in graph.in_degree
        assert "b" not in graph.graph
        assert "b" not in graph.graph["a"]
        assert graph.get_ready_stages() == ["a"]

    def test_diamond_dependency(self):
        stages = [stage("a"), stage("b", ["a"]), stage("c", ["a"]), stage("d", ["b", "c"])]
        graph = DependencyGraph(stages)

        layers = graph.topological_sort()
        assert len(layers) == 3
        assert layers[0] == ["a"]
        assert set(layers[1]) == {"b", "c"}
        assert layers[2] == ["d"]
