"""测试 Pipeline YAML 解析器"""

import pytest

from pomelo_orbit.infrastructure.ci.parser import PipelineParseError, parse_pipeline_yaml


class TestPipelineParser:
    """测试 Pipeline 解析器"""

    def test_parse_simple_pipeline(self):
        """测试解析简单 pipeline"""
        yaml_content = """
version: v1
steps:
  - name: build
    image: python:3.12
    commands:
      - pip install -r requirements.txt
      - python setup.py build
"""
        pipeline = parse_pipeline_yaml(yaml_content)

        assert pipeline.version == "v1"
        assert len(pipeline.steps) == 1
        assert pipeline.steps[0].name == "build"
        assert pipeline.steps[0].image == "python:3.12"
        assert pipeline.steps[0].commands is not None
        assert len(pipeline.steps[0].commands) == 2

    def test_parse_pipeline_with_timeout(self):
        """测试解析带超时的 pipeline"""
        yaml_content = """
version: v1
timeout: 3600
steps:
  - name: build
    image: python:3.12
    commands:
      - python setup.py build
"""
        pipeline = parse_pipeline_yaml(yaml_content)

        assert pipeline.timeout == 3600

    def test_parse_nested_steps(self):
        """测试解析嵌套 steps"""
        yaml_content = """
version: v1
steps:
  - name: test
    steps:
      - name: unit-test
        image: python:3.12
        commands:
          - pytest tests/unit
      - name: lint
        image: python:3.12
        commands:
          - ruff check .
"""
        pipeline = parse_pipeline_yaml(yaml_content)

        assert len(pipeline.steps) == 1
        assert pipeline.steps[0].name == "test"
        assert pipeline.steps[0].steps is not None
        assert len(pipeline.steps[0].steps) == 2
        assert pipeline.steps[0].steps[0].name == "unit-test"
        assert pipeline.steps[0].steps[1].name == "lint"

    def test_parse_checkout_step(self):
        """测试解析 checkout step"""
        yaml_content = """
version: v1
steps:
  - name: checkout
    uses: checkout
    with:
      depth: 1
      ref: main
    outputs:
      - commit_sha
      - commit_message
"""
        pipeline = parse_pipeline_yaml(yaml_content)

        step = pipeline.steps[0]
        assert step.name == "checkout"
        assert step.uses == "checkout"
        assert step.inputs is not None
        assert step.inputs["depth"] == 1
        assert step.inputs["ref"] == "main"
        assert step.outputs is not None
        assert len(step.outputs) == 2

    def test_parse_invalid_yaml(self):
        """测试解析无效 YAML"""
        yaml_content = """
version: v1
steps:
  - name: build
    image: python:3.12
    commands
      - invalid yaml
"""
        with pytest.raises(PipelineParseError, match="YAML 格式错误"):
            parse_pipeline_yaml(yaml_content)

    def test_parse_missing_version(self):
        """测试缺少 version 字段"""
        yaml_content = """
steps:
  - name: build
    image: python:3.12
"""
        with pytest.raises(PipelineParseError, match="缺少 version 字段"):
            parse_pipeline_yaml(yaml_content)

    def test_parse_missing_steps(self):
        """测试缺少 steps 字段"""
        yaml_content = """
version: v1
"""
        with pytest.raises(PipelineParseError, match="缺少 steps 字段"):
            parse_pipeline_yaml(yaml_content)

    def test_parse_missing_step_name(self):
        """测试缺少 step name"""
        yaml_content = """
version: v1
steps:
  - image: python:3.12
    commands:
      - echo hello
"""
        with pytest.raises(PipelineParseError, match="Step 缺少 name 字段"):
            parse_pipeline_yaml(yaml_content)

    def test_parse_step_with_depends_on(self):
        """测试解析带 depends_on 的 step"""
        yaml_content = """
version: v1
steps:
  - name: checkout
    uses: checkout
  - name: build
    depends_on: [checkout]
    image: python:3.12
    commands:
      - python setup.py build
"""
        pipeline = parse_pipeline_yaml(yaml_content)

        assert len(pipeline.steps) == 2
        assert pipeline.steps[1].depends_on == ["checkout"]
