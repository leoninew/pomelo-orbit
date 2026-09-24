# 全局流水线模板与构建阶段
最后修改时间: 2026-09-24 08:49:42

Review status: Accepted

Mode: standard

## Background

当前 Template Pipeline 和 PipelineStage(kind=template) 都绑定 Project，切换 Project 后无法复用同一套构建编排与构建阶段。Application Pipeline 仍然需要固定绑定所属 Project 的 Repository、Application 和运行数据。

## Goal

- Template Pipeline 与构建阶段模板改为全局资源，所有有权访问项目流水线的成员都能看到并复用同一份数据。
- Application Pipeline、Application Stage、Snapshot、Run 和制品继续按 Project 隔离。
- 既有模板数据迁移为全局记录，模板实例化后仍把阶段内容复制为项目内独立的 Application Stage。

## Non-goal

- 不引入跨项目共享 Application、Repository、Version 或 Pipeline Run。
- 不让模板资源在未通过当前 Project 成员校验时被 API 访问。
- 不改变已实例化 Application Pipeline 的阶段快照和运行语义。

## Acceptance

- 任意已加入的 Project 查询 Pipeline 时都能看到相同的 Template Pipeline 和构建阶段模板。
- 创建、更新、删除模板资源后，其他 Project 的查询结果立即反映相同全局数据。
- Application Pipeline 只返回当前 Project 的记录，Application Stage 只返回所属 Pipeline 的当前 Project 记录。
- 迁移后既有模板资源变为全局记录，模板引用和 Application Pipeline 历史快照不丢失。

## Decisions

- 使用 `pipeline.project_id IS NULL` 标识全局 Template Pipeline。
- 使用 `pipeline_stage.project_id IS NULL` 标识全局构建阶段模板；Application Stage 保持非空 Project ID。
- API 保留 `project_id` 作为当前 Project 成员资格和 Application 资源校验上下文；数据库查询对全局模板忽略项目归属。

## Risks / assumptions

- 模板资源的名称冲突由 UseCase 按全局范围校验；Application Pipeline 名称仍按 Project 校验。
- 本次开发按用户授权就地修改三种数据库的 pipeline schema 定义，并直接归一化开发库 `DBTALK_DSN_APP`；不新增兼容层或新旧两套模板归属逻辑。
