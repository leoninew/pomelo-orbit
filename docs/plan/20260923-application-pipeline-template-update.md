# 应用流水线跟进模板流水线更新计划
最后修改时间: 2026-09-24 10:20:00

Review status: Accepted

Mode: strict

## 范围

- 保留现有阶段级“更新至阶段模板”能力。
- Application Pipeline 继续通过 `source_pipeline_id + source_template_version` 记录当前采用的 Template Pipeline 版本。
- 不增加来源比较字段，不增加第二套快照表或新的快照抽象。
- 使用现有 `pipeline_snapshot` 表保存 Pipeline 定义快照；快照本身不按 `kind` 拒绝，`kind` 只由被快照的 Pipeline 决定其取数方式。

## 关键定义

快照是某个 Pipeline 在某个 `pipeline.version` 下的不可变定义记录：阶段集合和变量声明都从当时的 Pipeline 定义复制进去。它不是每次编辑的草稿，也不是每个应用额外保存的一份模板副本。

- Template Pipeline 快照保存 `PipelineStageReference` 和 Template 变量声明。
- Application Pipeline 快照继续保存运行前冻结的应用阶段、运行变量声明、Repository/Application/Version 信息。
- 两者共用 `pipeline_snapshot`，不新增表；快照记录不需要业务分支来区分 `kind`。
- Application Pipeline 的来源指针仍是 `source_pipeline_id + source_template_version`，需要比较时按这两个字段从同一快照表精确读取来源版本。

## 快照时机

- 创建或编辑 Template Pipeline 不自动创建快照。
- 仅修改独立阶段模板、不修改 Template Pipeline 引用，不创建 Pipeline 快照。
- Application Pipeline 实例化实际采用模板时，复用或创建对应 Template Pipeline 版本的快照。
- Application Pipeline 预览升级只读旧来源快照和当前 Template Pipeline 定义，不写快照。
- Application Pipeline 升级成功后，在同一请求的 HTTP 切面事务中复用或创建目标模板版本的快照。
- Application Pipeline 运行前继续按应用 Pipeline 版本复用或创建运行快照；旧 Snapshot、Run、Artifact 和 Version 不回写。

## 升级流程

1. 加载应用 Pipeline、`source_pipeline_id` 指向的 Template Pipeline、应用当前采用版本的 `pipeline_snapshot` 和当前模板引用。
2. 以 `source_template_stage_id` 匹配阶段；引用节点 ID 和应用节点 ID 只用于 DAG/实例节点，不作为阶段模板身份。
3. 比较来源快照与当前模板定义，目标阶段版本高于应用阶段版本时只更新模板来源内容，应用节点私有配置保留；阶段版本不回退。
4. 当前应用阶段在目标模板中不存在、目标引用内容无法确定、变量作用域无法对应或现有配置校验失败时，预览报告冲突并禁止 Apply。
5. 模板新增且能完成依赖、变量和制品校验的阶段自动加入；模板变量按应用节点 ID 重映射，应用独有变量保留。
6. Apply 再次校验应用版本和来源模板版本，更新应用 Pipeline、阶段和变量；不在 Service 内开启事务，依赖既有 HTTP 切面事务。

## 文件与实现步骤

1. 删除已撤销的两个来源比较字段及其 SQL/生成代码/未提交迁移残留。
2. 扩展现有 `pipeline_snapshot` 读取入口，支持 `pipeline_id + pipeline_version` 精确读取，并允许 Template 与 Application 使用同一快照机制。
3. 在实例化、模板升级预览/Apply、运行前快照路径中复用既有快照创建逻辑；模板编辑路径不调用快照创建。
4. 实现来源快照与当前模板定义的升级预览和合并，复用现有 DAG、变量、制品绑定校验。
5. 注册 Pipeline 级预览/Apply HTTP/Proto 契约，阶段级更新路由保持不变。
6. 更新活文档和测试，确认快照复用、按需创建、升级比较、并发校验和历史记录不变。

## 验收

- 两个废弃字段不存在，不增加第二套快照表或第二种快照概念。
- Template 和 Application 均可通过 `pipeline_snapshot` 保存不可变定义；快照创建不因普通模板编辑触发。
- 应用实例化后有可按 `source_pipeline_id + source_template_version` 读取的模板定义快照。
- 升级预览不写库，比较来源快照与当前模板；Apply 成功后更新来源版本并固化目标模板版本快照。
- HTTP 同步接口不自行创建事务；聚合写入由既有切面统一保证原子性。
- 阶段级更新、应用运行快照和历史 Run 行为保持不变。

## 验证计划

- 运行 Pipeline 用例、Pipeline Repository 和 Pipeline Run 相关测试。
- 执行 SQLC/Proto 生成后检查生成物无废弃字段残留。
- 按仓库约定运行 `task check` 和 `go test ./cmd/... ./internal/...`。
- 修改前端契约后运行 `yarn --cwd web lint:fix` 与 `yarn --cwd web typecheck`。
