"""Stage 聚合的应用服务"""

import re
from dataclasses import asdict

from pomelo_orbit.domain.ci.entities import BuildStage
from pomelo_orbit.domain.ci.repositories import BuildStageRepository
from pomelo_orbit.domain.exceptions import BusinessError


class BuildStageService:
    """BuildStage 聚合根的应用服务

    职责：
    - Stage 的 CRUD 操作
    - Stage 名称唯一性验证
    - Stage 复制功能
    """

    def __init__(self, stage_repo: BuildStageRepository):
        self.stage_repo = stage_repo

    def list_stages(self, page: int = 1, per_page: int = 20, search: str | None = None) -> tuple[list[BuildStage], int]:
        """分页查询 Stage"""
        return self.stage_repo.find_paginated(page, per_page, search=search)

    def get_stage(self, stage_id: str) -> BuildStage:
        """获取单个 Stage"""
        stage = self.stage_repo.find_by_id(stage_id)
        if not stage:
            raise BusinessError(f"Stage {stage_id} not found", status_code=404)
        return stage

    def load_stages_by_ids(self, stage_ids: list[str]) -> list[BuildStage]:
        """批量加载 Stage（用于模板编排）"""
        stages = self.stage_repo.find_by_ids(stage_ids)
        if len(stages) != len(stage_ids):
            found = {stage.id for stage in stages}
            missing = [stage_id for stage_id in stage_ids if stage_id not in found]
            raise BusinessError(f"Stage(s) not found: {missing}", status_code=404)
        return stages

    def create_stage(
        self,
        name: str,
        image: str,
        script: str,
        artifacts: list | None = None,
        description: str = "",
    ) -> BuildStage:
        """创建 Stage"""
        if self.stage_repo.find_by_name(name):
            raise BusinessError(f"Stage '{name}' already exists", status_code=409)
        stage = BuildStage.create(
            name=name,
            image=image,
            script=script,
            artifacts=artifacts,
            description=description,
        )
        self.stage_repo.save(stage)
        return stage

    def update_stage(
        self,
        stage_id: str,
        name: str | None = None,
        image: str | None = None,
        script: str | None = None,
        artifacts: list | None = None,
        description: str | None = None,
    ) -> BuildStage:
        """更新 Stage"""
        stage = self.get_stage(stage_id)
        if name is not None and name != stage.name and self.stage_repo.find_by_name(name):
            raise BusinessError(f"Stage '{name}' already exists", status_code=409)
        stage.update(name=name, image=image, script=script, artifacts=artifacts, description=description)
        self.stage_repo.save(stage)
        return stage

    def duplicate_stage(self, stage_id: str) -> BuildStage:
        """复制 Stage（自动生成唯一名称）"""
        stage = self.get_stage(stage_id)

        # 获取基础名称（去掉可能的 " copy" 或 " copy N" 后缀）
        base_name = re.sub(r" copy( \d+)?$", "", stage.name)

        # 从 "原名称 copy" 开始尝试，依次递增
        i = 1
        while True:
            new_name = f"{base_name} copy" if i == 1 else f"{base_name} copy {i}"
            if not self.stage_repo.find_by_name(new_name):
                break
            i += 1

        return self.create_stage(
            name=new_name,
            image=stage.image,
            script=stage.script,
            artifacts=[asdict(a) if not isinstance(a, dict) else a for a in stage.artifacts]
            if stage.artifacts
            else None,
            description=stage.description,
        )

    def delete_stage(self, stage_id: str) -> None:
        """删除 Stage（检查模板引用）"""
        stage = self.get_stage(stage_id)
        if self.stage_repo.is_referenced_by_templates(stage_id):
            raise BusinessError("Stage is referenced by templates, cannot delete", status_code=409)
        self.stage_repo.delete(stage)
