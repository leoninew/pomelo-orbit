# backend-go cd application 接口迁移需求

Review status: Accepted

## Background

backend-go 是现有 backend 的 Go 版本实现和改良，但 `/api/cd/application` 相关接口尚未完整迁移。需要在不一次性展开全部代码的前提下，逐个 API 端点检查现有 Python 后端能力、backend-go 当前状态，并补齐缺失的 application CRUD 接口。

## Goal

- 检查并补齐 backend-go 中 `/api/cd/application` 相关接口。
- 覆盖 application 的 CRUD 操作，以及与这些操作直接相关的请求、响应、服务、仓储、路由和测试。
- 逐个 API 端点处理：处理到哪个端点才读取该端点所需的相关文件，避免无范围地一次性读取所有文件。
- 最后更新 `docs/todo.md` 中 backend-go cd application 迁移相关内容。

## Non-goal

- 不迁移与 `/api/cd/application` 无直接关系的 CD 接口。
- 不做兼容层、旧字段别名或额外默认值处理。
- 不主动重构无关模块。
- 不执行 git 写操作（commit、push、merge 等）。

## User scenarios

- 调用方可以通过 backend-go 的 `/api/cd/application` 接口完成应用列表查询、详情查询、创建、更新和删除。
- 调用方得到的请求校验、响应结构和错误语义应尽量与现有 backend 对应接口保持一致，除非 Go 版本已有明确改良设计。

## Acceptance

- backend-go 存在并注册 `/api/cd/application` CRUD 相关路由。
- CRUD 请求/响应模型、service 和 repository 行为与现有 backend 对应接口对齐。
- 为新增或修复的 application 接口补充必要测试。
- 相关 Go 测试或项目约定验证命令通过；如无法运行或失败，需记录原因和输出。
- `docs/todo.md` 已更新迁移进度。

## Open questions

暂无必须阻塞实现的问题。实现过程中如果发现 Python 后端接口语义与 backend-go 既有架构冲突，应先报告并更新需求/范围。

## Decisions

- 使用 light / 轻量模式。
- 按用户要求逐个 API 端点处理，不预先读取所有文件。
- 以现有 backend 行为为迁移参考，以 backend-go 已有代码风格和架构为落地约束。

## Risk

- Python 后端与 backend-go 的模型、命名或错误处理可能已经出现差异，需要在实现时逐项确认。
- CRUD 之外可能存在 application 相关辅助接口；本次只覆盖与 `/api/cd/application` CRUD 直接相关的接口，除非检查过程中发现 CRUD 必须依赖它们。
- 当前工作区已有用户未提交改动，实施时需避免覆盖无关变更。
