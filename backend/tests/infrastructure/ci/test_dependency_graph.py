"""依赖图测试"""

import pytest

from pomelo_orbit.domain.ci.value_objects import StepDefinition
from pomelo_orbit.infrastructure.ci.dependency_graph import (
    CyclicDependencyError,
    DependencyGraph,
)


class TestDependencyGraph:
    """依赖图测试"""

    def test_build_simple_graph(self):
        """测试构建简单依赖图"""
        steps = [
            StepDefinition(name="a", image="alpine"),
            StepDefinition(name="b", image="alpine", depends_on=["a"]),
            StepDefinition(name="c", image="alpine", depends_on=["b"]),
        ]
        graph = DependencyGraph(steps)

        assert graph.in_degree["a"] == 0
        assert graph.in_degree["b"] == 1
        assert graph.in_degree["c"] == 1
        assert graph.graph["a"] == ["b"]
        assert graph.graph["b"] == ["c"]

    def test_build_parallel_graph(self):
        """测试构建并行依赖图"""
        steps = [
            StepDefinition(name="a", image="alpine"),
            StepDefinition(name="b", image="alpine"),
            StepDefinition(name="c", image="alpine", depends_on=["a", "b"]),
        ]
        graph = DependencyGraph(steps)

        assert graph.in_degree["a"] == 0
        assert graph.in_degree["b"] == 0
        assert graph.in_degree["c"] == 2
        assert "c" in graph.graph["a"]
        assert "c" in graph.graph["b"]

    def test_get_ready_jobs(self):
        """测试获取可执行的 Job"""
        steps = [
            StepDefinition(name="a", image="alpine"),
            StepDefinition(name="b", image="alpine"),
            StepDefinition(name="c", image="alpine", depends_on=["a"]),
        ]
        graph = DependencyGraph(steps)

        ready = graph.get_ready_jobs()
        assert set(ready) == {"a", "b"}

    def test_mark_completed(self):
        """测试标记 Job 完成"""
        steps = [
            StepDefinition(name="a", image="alpine"),
            StepDefinition(name="b", image="alpine", depends_on=["a"]),
        ]
        graph = DependencyGraph(steps)

        assert graph.in_degree["b"] == 1
        graph.mark_completed("a")
        assert graph.in_degree["b"] == 0

    def test_topological_sort_linear(self):
        """测试线性依赖的拓扑排序"""
        steps = [
            StepDefinition(name="a", image="alpine"),
            StepDefinition(name="b", image="alpine", depends_on=["a"]),
            StepDefinition(name="c", image="alpine", depends_on=["b"]),
        ]
        graph = DependencyGraph(steps)

        layers = graph.topological_sort()
        assert len(layers) == 3
        assert layers[0] == ["a"]
        assert layers[1] == ["b"]
        assert layers[2] == ["c"]

    def test_topological_sort_parallel(self):
        """测试并行依赖的拓扑排序"""
        steps = [
            StepDefinition(name="a", image="alpine"),
            StepDefinition(name="b", image="alpine"),
            StepDefinition(name="c", image="alpine", depends_on=["a", "b"]),
        ]
        graph = DependencyGraph(steps)

        layers = graph.topological_sort()
        assert len(layers) == 2
        assert set(layers[0]) == {"a", "b"}
        assert layers[1] == ["c"]

    def test_topological_sort_complex(self):
        """测试复杂依赖的拓扑排序"""
        steps = [
            StepDefinition(name="checkout", image="alpine"),
            StepDefinition(name="lint", image="alpine", depends_on=["checkout"]),
            StepDefinition(name="test", image="alpine", depends_on=["checkout"]),
            StepDefinition(name="build", image="alpine", depends_on=["lint", "test"]),
        ]
        graph = DependencyGraph(steps)

        layers = graph.topological_sort()
        assert len(layers) == 3
        assert layers[0] == ["checkout"]
        assert set(layers[1]) == {"lint", "test"}
        assert layers[2] == ["build"]

    def test_cyclic_dependency_detection(self):
        """测试循环依赖检测"""
        steps = [
            StepDefinition(name="a", image="alpine", depends_on=["b"]),
            StepDefinition(name="b", image="alpine", depends_on=["a"]),
        ]
        graph = DependencyGraph(steps)

        with pytest.raises(CyclicDependencyError) as exc_info:
            graph.topological_sort()
        assert "Cyclic dependency" in str(exc_info.value)

    def test_self_dependency_detection(self):
        """测试自依赖检测"""
        steps = [
            StepDefinition(name="a", image="alpine", depends_on=["a"]),
        ]
        graph = DependencyGraph(steps)

        with pytest.raises(CyclicDependencyError):
            graph.topological_sort()

    def test_unknown_dependency(self):
        """测试依赖不存在的 Job"""
        steps = [
            StepDefinition(name="a", image="alpine", depends_on=["unknown"]),
        ]

        with pytest.raises(ValueError) as exc_info:
            DependencyGraph(steps)
        assert "unknown job" in str(exc_info.value)

    def test_has_cycle(self):
        """测试 has_cycle 方法"""
        # 无循环
        steps = [
            StepDefinition(name="a", image="alpine"),
            StepDefinition(name="b", image="alpine", depends_on=["a"]),
        ]
        graph = DependencyGraph(steps)
        assert not graph.has_cycle()

        # 有循环
        steps = [
            StepDefinition(name="a", image="alpine", depends_on=["b"]),
            StepDefinition(name="b", image="alpine", depends_on=["a"]),
        ]
        graph = DependencyGraph(steps)
        assert graph.has_cycle()

    def test_remove_job(self):
        """测试移除 Job"""
        steps = [
            StepDefinition(name="a", image="alpine"),
            StepDefinition(name="b", image="alpine", depends_on=["a"]),
            StepDefinition(name="c", image="alpine", depends_on=["b"]),
        ]
        graph = DependencyGraph(steps)

        # 移除中间节点 b
        graph.remove_job("b")
        
        # 验证 in_degree 中已移除
        assert "b" not in graph.in_degree
        
        # 验证 graph 中已清理
        assert "b" not in graph.graph
        assert "b" not in graph.graph["a"]
        
        # 验证可执行的 Job
        ready = graph.get_ready_jobs()
        assert ready == ["a"]

    def test_diamond_dependency(self):
        """测试菱形依赖"""
        steps = [
            StepDefinition(name="a", image="alpine"),
            StepDefinition(name="b", image="alpine", depends_on=["a"]),
            StepDefinition(name="c", image="alpine", depends_on=["a"]),
            StepDefinition(name="d", image="alpine", depends_on=["b", "c"]),
        ]
        graph = DependencyGraph(steps)

        layers = graph.topological_sort()
        assert len(layers) == 3
        assert layers[0] == ["a"]
        assert set(layers[1]) == {"b", "c"}
        assert layers[2] == ["d"]
