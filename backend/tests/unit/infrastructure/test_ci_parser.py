"""CI Pipeline Parser 单元测试"""

import pytest

from pomelo_orbit.infrastructure.ci.parser import PipelineParseError, parse_pipeline_yaml


class TestParsePipelineYaml:
    """测试 parse_pipeline_yaml"""

    def test_parse_simple_pipeline(self):
        """测试解析简单 pipeline"""
        yaml_content = """
version: "1"
steps:
  - name: build
    image: python:3.12
    commands:
      - pip install -r requirements.txt
      - python setup.py build
"""
        pipeline = parse_pipeline_yaml(yaml_content)

        assert pipeline.version == "1"
        assert pipeline.timeout is None
        assert len(pipeline.steps) == 1
        assert pipeline.steps[0].name == "build"
        assert pipeline.steps[0].image == "python:3.12"
        assert pipeline.steps[0].commands == [
            "pip install -r requirements.txt",
            "python setup.py build",
        ]

    def test_parse_pipeline_with_timeout(self):
        """测试解析带 timeout 的 pipeline"""
        yaml_content = """
version: "1"
timeout: 3600
steps:
  - name: test
    image: node:18
    commands:
      - npm test
"""
        pipeline = parse_pipeline_yaml(yaml_content)

        assert pipeline.version == "1"
        assert pipeline.timeout == 3600
        assert len(pipeline.steps) == 1

    def test_parse_pipeline_with_dependencies(self):
        """测试解析带依赖的 pipeline"""
        yaml_content = """
version: "1"
steps:
  - name: build
    image: python:3.12
    commands:
      - python setup.py build
  - name: test
    image: python:3.12
    commands:
      - pytest
    depends_on:
      - build
"""
        pipeline = parse_pipeline_yaml(yaml_content)

        assert len(pipeline.steps) == 2
        assert pipeline.steps[1].depends_on == ["build"]

    def test_parse_pipeline_with_artifacts(self):
        """测试解析带 artifacts 的 pipeline"""
        yaml_content = """
version: "1"
steps:
  - name: build
    image: python:3.12
    commands:
      - python setup.py build
    artifacts:
      - path: dist/**
      - path: build/**
"""
        pipeline = parse_pipeline_yaml(yaml_content)

        assert len(pipeline.steps[0].artifacts) == 2
        assert pipeline.steps[0].artifacts[0]["path"] == "dist/**"

    def test_parse_pipeline_with_volumes(self):
        """测试解析带 volumes 的 pipeline"""
        yaml_content = """
version: "1"
steps:
  - name: build
    image: python:3.12
    commands:
      - python setup.py build
    volumes:
      - /tmp/cache:/cache
"""
        pipeline = parse_pipeline_yaml(yaml_content)

        assert pipeline.steps[0].volumes == ["/tmp/cache:/cache"]

    def test_parse_pipeline_with_retry_policy(self):
        """测试解析带 retry_policy 的 pipeline"""
        yaml_content = """
version: "1"
steps:
  - name: flaky-test
    image: python:3.12
    commands:
      - pytest
    retry_policy: always_rerun
"""
        pipeline = parse_pipeline_yaml(yaml_content)

        assert pipeline.steps[0].retry_policy == "always_rerun"

    def test_parse_pipeline_with_nested_steps(self):
        """测试解析嵌套 steps"""
        yaml_content = """
version: "1"
steps:
  - name: parallel-tests
    steps:
      - name: unit-test
        image: python:3.12
        commands:
          - pytest tests/unit
      - name: integration-test
        image: python:3.12
        commands:
          - pytest tests/integration
"""
        pipeline = parse_pipeline_yaml(yaml_content)

        assert len(pipeline.steps) == 1
        assert pipeline.steps[0].name == "parallel-tests"
        assert pipeline.steps[0].steps is not None
        assert len(pipeline.steps[0].steps) == 2
        assert pipeline.steps[0].steps[0].name == "unit-test"
        assert pipeline.steps[0].steps[1].name == "integration-test"

    def test_parse_pipeline_with_action(self):
        """测试解析使用 action 的 pipeline"""
        yaml_content = """
version: "1"
steps:
  - name: checkout
    uses: actions/checkout@v3
    with:
      repository: myorg/myrepo
      ref: main
"""
        pipeline = parse_pipeline_yaml(yaml_content)

        assert pipeline.steps[0].uses == "actions/checkout@v3"
        assert pipeline.steps[0].with_ == {
            "repository": "myorg/myrepo",
            "ref": "main",
        }

    def test_parse_invalid_yaml(self):
        """测试解析无效 YAML"""
        yaml_content = """
version: 1
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
version: "1"
"""
        with pytest.raises(PipelineParseError, match="缺少 steps 字段"):
            parse_pipeline_yaml(yaml_content)

    def test_parse_steps_not_array(self):
        """测试 steps 不是数组"""
        yaml_content = """
version: "1"
steps: not-an-array
"""
        with pytest.raises(PipelineParseError, match="steps 必须是数组"):
            parse_pipeline_yaml(yaml_content)

    def test_parse_step_missing_name(self):
        """测试 step 缺少 name 字段"""
        yaml_content = """
version: "1"
steps:
  - image: python:3.12
    commands:
      - echo hello
"""
        with pytest.raises(PipelineParseError, match="Step 缺少 name 字段"):
            parse_pipeline_yaml(yaml_content)

    def test_parse_nested_steps_not_array(self):
        """测试嵌套 steps 不是数组"""
        yaml_content = """
version: "1"
steps:
  - name: parallel
    steps: not-an-array
"""
        with pytest.raises(PipelineParseError, match="steps 必须是数组"):
            parse_pipeline_yaml(yaml_content)

    def test_parse_not_dict(self):
        """测试 pipeline 不是对象"""
        yaml_content = """
- not a dict
"""
        with pytest.raises(PipelineParseError, match="Pipeline 必须是一个对象"):
            parse_pipeline_yaml(yaml_content)
