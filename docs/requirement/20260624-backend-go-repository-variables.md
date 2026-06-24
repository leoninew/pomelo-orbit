# backend-go 仓库变量配置补齐需求
最后修改时间: 2026-06-24 18:33:49

Review status: Accepted

## Background

`backend-go` 是 Python `backend` 的迁移版本。当前发现 `backend-go` 新建代码仓库后遗漏了 Python backend 已有行为：仓库创建成功后应同步生成一组“变量配置”，供后续 CI/CD 模板、运行时变量解析和页面展示使用。

## Goal

补齐 `backend-go` 新建仓库后的变量配置生成逻辑，使其对齐 Python backend 的核心行为：新仓库创建后自动拥有与仓库自身信息相关的变量配置，至少包含 `repository_id`、`repository_name` 等字段。

## Non-goal

- 不重构整个 repository、CI 或变量配置模块分层。
- 不新增前端交互或页面功能。
- 不引入兼容层、别名或默认值兜底逻辑。
- 不修改已执行数据库迁移，除非实现必需且另行说明。
- 不处理历史仓库缺失变量配置的批量回填，除非现有 backend 行为明确要求且实现范围很小。

## User scenarios

1. 用户在 `backend-go` 中新建代码仓库后，系统立即生成仓库相关变量配置。
2. 后续 CI/CD 运行时变量解析可以读取这些仓库变量，不再因为新建仓库没有变量配置而缺字段。
3. 行为与 Python `backend` 对齐，迁移后不出现功能回退。

## Acceptance

- 对照 Python `backend` 找到新建仓库生成变量配置的字段集合和语义。
- 在 `backend-go` 的仓库创建流程中补齐对应逻辑。
- 新建仓库后至少生成 `repository_id`、`repository_name` 等 Python backend 已有仓库变量配置。
- 变量配置写入应位于合理的 service/repository 分层位置，避免接口层堆业务逻辑。
- 不为业务字段添加无依据默认值，不使用 `getattr`/防御性兼容写法。
- 添加或更新 backend-go 相关测试，覆盖新建仓库后变量配置生成。
- 运行 backend-go 相关格式化/测试/编译检查并记录结果。

## Open questions

- Python backend 的完整变量字段集合待实现前从代码确认。
- backend-go 现有变量配置模型/表结构是否已存在，待实现前确认；若不存在，应优先复用现有 runtime variable/variable override 体系，避免扩大范围。

## Decisions

- 用户已明确要求“轻量模式，大致了解项目分层设计，合理规划，直接实现”，本需求记录直接标记为 `Accepted` 并进入 Implementation / 实现。
- 以 Python `backend` 的实际实现作为字段集合和语义基准，而不是凭名称猜测。

## Risk

- 如果 Python backend 的变量配置不仅服务 CI，还被前端或 CD 模块读取，backend-go 补齐时需要确认写入位置和字段类型，避免只满足单一路径。
- 如果 backend-go 现有 repository 创建流程已有事务边界，新增变量配置写入必须保持原子性，避免仓库创建成功但变量配置缺失。
- 若当前数据库 schema 缺少对应变量配置表或字段，可能需要缩小为现有结构可表达的最小对齐，或向用户报告 schema 差异。
