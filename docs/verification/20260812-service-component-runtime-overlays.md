# 服务组件运行时基础配置覆盖验证记录
最后修改时间: 2026-08-12 20:01:34

Review status: Accepted

Mode: standard

## Requirement Alignment

以下实现与已接受的 Requirement 对齐：

- `service_component` 新增 `entrypoint_json`、`command_json`、`pull_policy`、`restart_policy`；SQLite 与 MySQL 均新增 `000032` 迁移，未修改既有迁移。
- Service Component 仍由 `source_version_component_id` 映射 Version Component；镜像继续仅来自 Version 声明。
- `MergeServiceComponent` 合并四个运行时覆盖；有效计划哈希和 Compose 渲染读取该有效组件。
- HTTP 和 MCP 均将四个字段映射至同一 `ServiceComponentOverlayInput`，响应也返回对应覆盖。
- 服务组件详情新增运行配置卡片，按其他覆盖卡片的既有模式展示 Version 默认值、当前 Service 值和覆盖来源；环境变量、资源、端点、挂载的恢复操作统一为“重置为 Version 值”。重置后保存请求不发送对应稀疏覆盖。

## Spec Alignment

不适用。本任务采用 standard 模式，未创建独立 Spec；按 Requirement 与 Plan 核对。

## Plan Alignment

已实现 Plan 中的迁移、SQLC、领域模型、存储、有效计划、HTTP/MCP 和前端修改。

尚未达到 Plan 的以下验证深度：

- 没有为新增四个 Service Component 直接覆盖补充定向 Go 测试，覆盖继承、非空覆盖、显式空 argv、策略校验、哈希变化、HTTP/MCP 往返及 SQLite 存储往返。
- 没有为 `ServiceComponentDetail.vue` 的草稿初始化、完整 payload、字段重置和显式空 argv 添加前端单元测试。

## Actual Diff Summary

- 数据库：新增 SQLite/MySQL `000032_service_component_runtime_overlays` 上下迁移，并更新 schema、查询与 SQLC 生成物。
- 后端：扩展 `model.ServiceComponent`、Repository 读写、Usecase 校验/归一化、有效服务计划合并、HTTP Proto 映射和 MCP 输出。
- 前端：新增服务组件运行配置编辑卡片和本地化文本；覆盖卡片的恢复文案统一为“重置为 Version 值”。
- 测试：`internal/bootstrap/app_test.go` 的 SQLite 迁移断言更新到版本 `32`。

## Expected Vs Actual Changed Files

| 预期范围 | 实际情况 |
| --- | --- |
| SQLite/MySQL 迁移、schema、SQLC | 已修改；SQLite 启动迁移测试通过。 |
| 领域、仓储、有效计划与 Compose | 已修改；相关 Go 包和全量 Go 测试通过。 |
| HTTP/MCP/proto | 已修改；相关包测试通过，但没有新增字段级定向断言。 |
| 服务组件详情与本地化 | 已修改；前端检查、构建和既有测试通过，但没有本功能的组件单元测试。 |
| 文档与验证记录 | Requirement、Plan 已更新；本验证记录新增。 |

当前 diff 未发现与本功能无关的业务源代码改动；SQLC 多个 domain 的 model 文件同步变化来自共享 `service_component` 表结构生成。

## Acceptance Checklist

- [x] 新四字段可持久化并读取，未覆盖时可继承 Version；镜像未进入 Service 覆盖。
- [x] `pull_policy` 与 `restart_policy` 在 Service Usecase 中按约定枚举校验。
- [x] `entrypoint` 与 `command` 在领域/存储层以 nil 与空 argv 区分继承和显式清空。
- [x] 命令文本经 `commandline.Parse` 解析，禁用环境变量与反引号展开。
- [x] 有效计划、哈希和 Compose 使用 `MergeServiceComponent` 的运行字段。
- [x] 保存本身不触发部署；服务计划变更仍由既有 pending-deploy 哈希比较表示。
- [x] 服务组件详情以与既有覆盖卡片一致的方式展示 Version 默认值、当前 Service 值及覆盖来源；可逐字段重置且保留其他 overlay 组。
- [x] HTTP 与 MCP 使用同一输入字段，并在组件输出中包含新增覆盖。
- [x] Version 切换仍经过 `normalizeOverlay` 校验新增覆盖。
- [x] SQLite 与 MySQL 新迁移均已新增，既有迁移未修改。
- [x] MySQL E2E 的 migration version 断言已更新至 `32`；该环境依赖检查未配置，按用户确认不阻塞本次验收。

## Test Results

| 命令 | 结果 |
| --- | --- |
| `go fmt ./cmd/... ./internal/...` | 通过 |
| `go vet ./cmd/... ./internal/...` | 通过 |
| `go test -count=1 ./cmd/... ./internal/...` | 通过 |
| `go test -count=1 ./internal/application/deployment/usecase ./internal/application/service/usecase ./internal/repository/impl/sqlc/service ./internal/api/http/handler/service ./internal/api/mcp/delivery ./internal/bootstrap` | 通过 |
| `yarn --cwd web lint:fix` | 通过 |
| `yarn --cwd web typecheck` | 通过 |
| `yarn --cwd web test` | 11 个测试文件、64 个测试通过 |
| `yarn --cwd web build` | 通过；仅有依赖包既有的 Rollup `#__PURE__` 注释警告 |
| `git diff HEAD --check` | 通过 |
| MySQL E2E | 未运行：需要 `BACKEND_GO_E2E_CONFIG`；迁移版本断言已更新至 `32`。 |
| 浏览器人工交互 | 未运行：仓库约定不主动启动开发服务器。 |

## Scope Deviations

无计划外的业务能力扩展。`internal/bootstrap/app_test.go` 更新为迁移 `32` 是新迁移的必要测试维护。

## Risks And Incomplete Items

1. MySQL E2E 未运行：需要 `BACKEND_GO_E2E_CONFIG`。迁移断言已同步至 `32`，且用户确认该环境依赖检查不阻塞交付。
2. 新增运行字段尚未增加独立的 Usecase、Repository、HTTP/MCP 和前端草稿/载荷/重置单元测试；现有全量 Go、前端测试、类型检查和构建均已通过。

## Conclusion

自动化构建、类型检查、格式检查、静态分析和当前全量测试均通过。运行配置的覆盖、保存和逐字段重置遵循环境变量、资源、端点和挂载的既有行为；MySQL E2E 因未配置环境未运行，但不阻塞本次验收。验证通过。
