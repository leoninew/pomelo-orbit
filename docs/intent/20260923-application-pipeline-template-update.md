# 应用流水线跟进模板流水线更新
最后修改时间: 2026-09-24 10:20:00

Review status: Accepted

Mode: strict

## Background

Application Pipeline 创建后不会被 Template Pipeline 的后续编辑隐式改变，但当前应用已经保存了 `source_pipeline_id` 和 `source_template_version`，还缺少基于该来源版本进行可靠比较和显式升级的完整路径。现有 `pipeline_snapshot` 已经是项目中的不可变快照机制，不应再引入另一种快照概念。

## Goal

1. 让应用流水线明确知道自己采用的 Template Pipeline ID 和版本。
2. 升级预览读取应用采用版本的 `pipeline_snapshot`，与当前 Template Pipeline 定义比较。
3. 复用同一 `pipeline_snapshot` 机制保存 Template 和 Application 的不可变定义；快照创建只发生在实际采用、成功升级或运行前等明确时机。
4. 保留应用的私有节点配置、绑定和独有变量，无法安全对应的变化明确阻止升级。

## Non-goal

- 不增加 `source_variables_baseline` 或 `source_reference_baseline`。
- 不增加第二张快照表，不创造第二种快照抽象。
- 不因普通模板编辑、独立阶段模板编辑或预览请求无条件创建快照。
- 不回写既有 Run、Artifact、Version 或历史 Snapshot。
- 不在同步 HTTP 用例内自行开启事务；沿用现有 HTTP 切面事务。

## User scenarios

1. 应用流水线显示当前来源 Template Pipeline 的新版本，并可预览差异。
2. 删除并重新引入同一独立阶段模板时，按 `source_template_stage_id` 判断阶段身份，而不是比较引用节点 ID。
3. 模板新增阶段且依赖、变量和制品校验通过时，整体升级自动加入；当前应用阶段被模板删除时，升级阻止提交。
4. 应用已修改的变量、节点名称、DAG、排序和制品 Component 映射不被模板升级静默覆盖。

## Acceptance

- Template 和 Application 均使用现有 `pipeline_snapshot` 保存不可变定义，快照处理不硬编码拒绝某个 `kind`。
- Application Pipeline 的来源关系由 `source_pipeline_id + source_template_version` 持久化。
- 实例化实际采用模板时创建或复用对应模板版本快照；模板普通保存不创建快照。
- 升级预览只读旧来源快照和当前模板；Apply 校验应用版本与模板版本，并在既有 HTTP 切面事务中完成写入。
- 阶段版本只递增；阶段匹配统一使用 `source_template_stage_id`。
- 阶段级“更新至阶段模板”和流水线级“跟进来源 Template Pipeline”保持独立。

## Decisions

- 快照是某个 Pipeline 某个 `pipeline.version` 的不可变定义记录，不是编辑事件，也不是按 kind 分叉的对象。
- Template 快照保存阶段引用和变量声明；Application 运行快照保存执行所需的应用阶段和运行变量。
- 预览不持久化；成功实例化、成功整体升级和实际运行前才允许创建对应快照。

## Risks / assumptions

- 历史应用若缺少对应来源快照，不能把当前应用值冒充旧来源快照，应报告无法安全比较并阻止整体升级。
- 本任务不处理用户直接修改数据库的情况。
- 过程文档与代码冲突时以代码为准，并同步修正文档。

## User review notes

- 用户明确撤销两个 baseline 字段方案。
- 用户明确要求复用当前快照实现，快照不应按 `kind` 拒绝 Template。
- 用户明确要求弄清快照时机：普通模板编辑不能无脑创建快照。
- 用户明确要求同步 HTTP 接口不自行包事务，使用现有事务切面。
