# 持续部署对话记录持久化验证
最后修改时间: 2026-08-15 23:34:00

Review status: Accepted

流程模式: 标准 / standard

## 验证依据

- Requirement: [持续部署对话记录持久化](../requirement/20260815-deployment-dialogue-history.md)，状态 `Accepted`。
- Plan: [持续部署对话记录持久化计划](../plan/20260815-deployment-dialogue-history.md)，状态 `Accepted`。
- Spec: 不适用；本功能采用标准模式，未单独创建 Spec。

## 需求对齐

当前实现建立了项目级 Conversation 与 Message 两个持久化实体，消息按 `(created_at, id)` 稳定排序；新会话、继续会话和删除均由 dialogue application service 先检查项目成员关系，HTTP handler 分别要求 `dialogue:read` 或 `dialogue:write`。

对话执行仍沿用既有 LLM/MCP 与 SSE 工具事件。只有模型产生非空最终回答后，application service 才在事务中创建或更新 Conversation，并写入本轮 user/assistant 两条 Message；模型失败、空响应或工具轮次未完成时不写入伪造的 assistant Message。删除 Conversation 由数据库外键级联删除 Message。

前端将历史放在持续部署对话工作区内：桌面端为可收起的固定宽度侧栏，移动端使用共享 `AppDrawer`，删除使用共享 `AppDialog` 确认。历史读取只展示持久化的 user/assistant Message；实时工具状态仍只存在于当前 SSE 会话状态中。时间展示复用 `web/src/utils/time.ts`。

## Plan 对齐

### 已实现

1. SQLite/MySQL 均新增 `deployment_dialogue_conversation`、`deployment_dialogue_message`、级联外键和排序索引。
2. 数据迁移新增 `dialogue:read`、`dialogue:write`，并授权内置管理员角色。
3. 新增 sqlc query、生成包、Repository、model、application usecase、HTTP API 和 bootstrap 装配。
4. Proto/Go/TypeScript 契约增加 Conversation 列表、详情和可选 Conversation ID，既有 SSE event 字段保持不变。
5. 前端实现新建、历史加载、切换、继续、删除、桌面折叠侧栏与移动抽屉。
6. Repository 测试覆盖按项目列出、稳定消息顺序和删除级联；usecase 测试覆盖首轮创建、继续会话、最终回答后写入、失败不持久化和工具事件。

### 验证范围说明

1. 不为当前实现不存在的业务和场景构造负向测试；既有 RBAC 与项目成员校验链路通过代码审查和现有测试验证。
2. 事务由项目既有面向切面的事务设施统一处理，本功能验证事务边界和调用方式，不重复从零搭建通用事务故障测试；特殊业务分支允许手工验证。
3. 前端交互已完成人工审查；自动化静态检查通过。
4. 历史记录操作已复用共享 `SelectControl`，共享组件通过可选触发器插槽支持紧凑的更多操作入口，业务组件不再直接引用 Reka primitives。

## 实际 diff 摘要

- 过程文档：新增 Requirement、Plan；本文件新增 Verification。
- 数据库：新增双数据库结构迁移、权限数据迁移、schema 快照、dialogue query 与 sqlc 生成物。
- 后端：新增 Conversation/Message model、Repository、application 行为、HTTP API、RBAC 接入与依赖装配。
- 契约：更新 dialogue Proto 及 Go/TypeScript 生成物。
- 前端：新增历史导航组件，改造 store、API、对话页和中英文文案。
- 范围外变更：`.gitignore`、`.dockerignore` 包含本地 Go cache 忽略项，与对话历史功能无直接关系，提交前应确认是否拆分。

## 预期与实际文件对比

| 计划范围 | 实际结果 |
| --- | --- |
| `sql/migration/{sqlite,mysql}/000030_deployment_dialogue_history.*` | 已新增。 |
| `sql/migration/data/{sqlite,mysql}/000030_deployment_dialogue_permission.*` | 已新增。 |
| `sql/schema/**`、`sql/query/dialogue/**`、`internal/gen/sqlc/**` | 已更新；共用 query 支持两种数据库，未单设 `sql/query/mysql/dialogue/`。 |
| model、Repository、application usecase | 已新增并接入。 |
| Proto、Go/TS 生成物、HTTP handler/routes | 已更新并接入。 |
| 前端 API/store/page/i18n | 已更新；额外拆出 `DeploymentDialogueHistory.vue`，符合组件化目标。 |
| handler 与前端行为测试 | 复用既有权限、事务测试体系，并完成人工前端审查。 |

## 验收标准检查

| # | 验收项 | 结果 | 依据 |
| --- | --- | --- | --- |
| 1 | Conversation/Message 独立实体及归属字段 | 通过 | 双数据库迁移、model、Repository。 |
| 2 | 多轮消息稳定排序读取 | 通过 | SQL `(created_at, id)` 排序及 Repository 测试。 |
| 3 | 仅最终回答后原子写入本轮问答 | 通过 | 使用既有事务切面，成功及无最终回答不持久化路径测试通过。 |
| 4 | 同一工作区创建、历史导航、切换和加载 | 通过 | 实现审查与前端人工审查通过。 |
| 5 | 刷新或跨设备后可恢复历史 | 通过 | Conversation/Message 经 API 持久化并可重新加载。 |
| 6 | 既有 SSE 与工具事件保持，持久化插入最终路径 | 通过 | handler/usecase 审查及工具循环测试。 |
| 7 | 首条用户问题自动生成标题且不可编辑 | 通过 | `conversationTitle` 与无重命名 API/UI。 |
| 8 | RBAC 与项目成员关系共同控制访问 | 通过 | handler 权限映射与 usecase 项目成员校验均已接入。 |
| 9 | 删除 Conversation 级联删除 Message | 通过 | 外键定义及 Repository 集成测试。 |

## 空库迁移验证

### SQLite

- 从全新空文件迁移：`version=30 state=clean`。
- 三项数据迁移均执行：系统种子、对话权限、Pipeline 模板库。
- 普通表数量：45。
- 两张对话历史表存在，`dialogue:read` / `dialogue:write` 存在。
- `PRAGMA integrity_check = ok`，`PRAGMA foreign_key_check` 无结果。

### MySQL

- 从全新 `utf8mb4_0900_ai_ci` 数据库迁移：`version=30 state=clean`。
- 三项数据迁移均执行：系统种子、对话权限、Pipeline 模板库。
- 普通表数量：45。
- 两张对话历史表存在，管理员具有两项对话权限。
- 对话表包含 3 个外键；`CHECK TABLE` 两张表均返回 `OK`。

## 命令结果

| 命令 | 结果 |
| --- | --- |
| `task sqlc` | 用户已执行；本地 `./bin/sqlc generate` 复验通过且生成物无差异。 |
| `task proto` | 用户已执行；本地重复 Buf 生成因远程插件服务不可用未完成，已有生成物已恢复且可通过编译/typecheck。 |
| `go fmt ./cmd/... ./internal/...` | 通过。 |
| `go vet ./cmd/... ./internal/...` | 通过。 |
| `go test ./internal/application/dialogue/usecase ./internal/repository/impl/sqlc/dialogue ./internal/infrastructure/database -count=1 -v` | 通过。 |
| `go test ./cmd/... ./internal/... -count=1` | 通过。 |
| `yarn --cwd web lint:fix` | 通过。 |
| `yarn --cwd web typecheck` | 通过。 |
| `yarn --cwd web test` | 未运行成功：沙箱禁止 esbuild 子进程，启动时报 `spawn EPERM`。 |
| `git diff --check` / `git diff --cached --check` | 通过。 |

## 非阻塞说明

1. 本地重复 Buf 生成受远程插件服务影响；用户已执行 `task proto`，现有生成物通过 Go 编译和前端 typecheck。
2. Vitest 在当前沙箱因 `spawn EPERM` 无法启动；前端已完成人工审查，lint 与 typecheck 通过。
3. `.gitignore` / `.dockerignore` 的 cache 忽略改动不属于本功能，提交时应确认其归属。

## 结论

后端持久化主路径、双数据库空库迁移、权限种子、Repository 行为和 Go 回归测试均通过；前端复用共享下拉选择组件，静态检查与人工审查通过。本次 Verification 判定为 `Accepted`。
