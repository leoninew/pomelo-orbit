"""Pipeline YAML 解析器"""

import yaml
from pydantic import ValidationError

from pomelo_orbit.domain.ci.value_objects import PipelineDefinition, StepDefinition


class PipelineParseError(Exception):
    """Pipeline 解析错误"""



def parse_pipeline_yaml(yaml_content: str) -> PipelineDefinition:
    """
    解析 pipeline YAML

    Args:
        yaml_content: YAML 文本

    Returns:
        PipelineDefinition

    Raises:
        PipelineParseError: 解析失败
    """
    try:
        data = yaml.safe_load(yaml_content)
    except yaml.YAMLError as e:
        raise PipelineParseError(f"YAML 格式错误: {e}")

    if not isinstance(data, dict):
        raise PipelineParseError("Pipeline 必须是一个对象")

    if "version" not in data:
        raise PipelineParseError("缺少 version 字段")

    if "steps" not in data:
        raise PipelineParseError("缺少 steps 字段")

    try:
        # 递归解析 steps
        steps_data = data["steps"]
        if not isinstance(steps_data, list):
            raise PipelineParseError("steps 必须是数组")

        steps = [_parse_step(step_data) for step_data in steps_data]

        return PipelineDefinition(
            version=data["version"],
            timeout=data.get("timeout"),
            steps=steps,
        )
    except ValidationError as e:
        raise PipelineParseError(f"Pipeline 定义验证失败: {e}")


def _parse_step(step_data: dict) -> StepDefinition:
    """递归解析 step"""
    if not isinstance(step_data, dict):
        raise PipelineParseError("Step 必须是对象")

    if "name" not in step_data:
        raise PipelineParseError("Step 缺少 name 字段")

    # 递归解析嵌套 steps
    nested_steps = None
    if "steps" in step_data:
        if not isinstance(step_data["steps"], list):
            raise PipelineParseError(f"Step '{step_data['name']}' 的 steps 必须是数组")
        nested_steps = [_parse_step(s) for s in step_data["steps"]]

    try:
        return StepDefinition(
            name=step_data["name"],
            image=step_data.get("image"),
            commands=step_data.get("commands"),
            uses=step_data.get("uses"),
            inputs=step_data.get("with"),
            volumes=step_data.get("volumes"),
            depends_on=step_data.get("depends_on"),
            timeout=step_data.get("timeout"),
            retry_policy=step_data.get("retry_policy"),
            artifacts=step_data.get("artifacts"),
            outputs=step_data.get("outputs"),
            steps=nested_steps,
        )
    except ValidationError as e:
        raise PipelineParseError(f"Step '{step_data['name']}' 验证失败: {e}")
