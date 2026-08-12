# 服务组件运行时基础配置覆盖验证记录
最后修改时间: 2026-08-12 23:01:52

Review status: Accepted

Mode: standard

## Requirement Alignment

当前实现与已接受的 Requirement 对齐：

- `service_component` 新增 `entrypoint_json`、`command_json`、`pull_policy`、`restart_policy`；SQLite 与 MySQL 均新增 `000032` 迁移，未修改既有迁移。
- Service Component 仍由 `source_version_component_id` 映射 Version Component；镜像继续仅来自 Version 声明。
- `MergeServiceComponent` 合并四个运行时覆盖；有效计划哈希和 Compose 渲染读取该有效组件。
- HTTP 和 MCP 均将四个字段映射至同一 `ServiceComponentOverlayInput`，响应也返回对应覆盖；MCP mapper 同步保留挂载的 `source_is_host_path`。
- 服务组件详情接口现在分别返回 `version_component` 和 `service_component`，不再返回合并后的 `effective` 组件；运行配置卡片直接使用两组源值。字段有实际 Service 覆盖且与 Version 默认不同时使用差异样式，并显示“重置”按钮；重置会清空 Service 字段。

## Spec Alignment

不适用。本任务采用 standard 模式，未创建独立 Spec；按 Requirement 与 Plan 核对。

## Plan Alignment

已实现 Plan 中的迁移、SQLC、领域模型、存储、有效计划、HTTP/MCP 和前端修改；根据用户反馈补充了详情来源契约调整。

尚未达到 Plan 的以下验证深度：

- 没有为新增四个 Service Component 直接覆盖补充完整的定向 Go 测试，覆盖继承、非空覆盖、显式空 argv、策略校验、哈希变化、HTTP/MCP 往返及 SQLite 存储往返。
- 没有为 `ServiceComponentDetail.vue` 的草稿初始化、完整 payload、字段重置和显式空 argv 添加前端单元测试。

## Actual Diff Summary

- 数据库：新增 SQLite/MySQL `000032_service_component_runtime_overlays` 上下迁移，并更新 schema、查询与 SQLC 生成物。
- 后端：扩展 `model.ServiceComponent`、Repository 读写、Usecase 校验/归一化、有效服务计划合并、HTTP Proto 映射和 MCP 输出。
- 前端：新增服务组件运行配置编辑卡片和本地化文本；运行配置及既有覆盖卡片的恢复文案统一为“重置”，并按 Version 默认值与 Service 当前值显示差异。
- 详情契约：Service Component 详情由 declaration/component/effective 三组改为 `version_component`/`service_component` 两组，生成 Go/TypeScript Proto 同步更新。
- 测试：补充来源值 mapper/usecase 断言；`internal/bootstrap/app_test.go` 的 SQLite 迁移断言更新到版本 `32`。

## Expected Vs Actual Changed Files

| 预期范围 | 实际情况 |
| --- | --- |
| SQLite/MySQL 迁移、schema、SQLC | 已修改；SQLite 启动迁移测试通过。 |
| 领域、仓储、有效计划与 Compose | 已修改；相关 Go 包和全量 Go 测试通过。 |
| HTTP/MCP/proto | 已修改；HTTP 详情 mapper 和生成协议已验证，MCP 更新契约保持共用输入并覆盖运行字段与挂载宿主路径模式；没有完整字段级往返测试。 |
| 服务组件详情与本地化 | 已修改；Version/Service 两组来源、差异样式和重置载荷已实现；没有本功能的组件单元测试。 |
| 文档与验证记录 | Requirement、Plan、Verification 与 MCP 操作指南已按本次来源契约和覆盖语义调整并更新。 |

当前 diff 未发现与本功能无关的业务源代码改动；SQLC 多个 domain 的 model 文件同步变化来自共享 `service_component` 表结构生成。

## Acceptance Checklist

- [x] 新四字段可持久化并读取，未覆盖时可继承 Version；镜像未进入 Service 覆盖。
- [x] `pull_policy` 与 `restart_policy` 在 Service Usecase 中按约定枚举校验。
- [x] `entrypoint` 与 `command` 在领域/存储层以 nil 与空 argv 区分继承和显式清空。
- [x] 命令文本经 `commandline.Parse` 解析，禁用环境变量与反引号展开。
- [x] 有效计划、哈希和 Compose 使用 `MergeServiceComponent` 的运行字段。
- [x] 保存本身不触发部署；服务计划变更仍由既有 pending-deploy 哈希比较表示。
- [x] 服务组件详情返回 `version_component` 默认值和 `service_component` 当前稀疏值，不返回 `effective` 详情；运行配置差异显示 amber 样式和“重置”，重置后请求清空对应 Service 字段并保留其他 overlay 组。
- [x] HTTP 与 MCP 使用同一输入字段，并在组件输出中包含新增覆盖。
- [x] MCP Service Component overlay 保留 `source_is_host_path`，不会在更新挂载时丢失宿主路径模式。
- [x] Version 切换仍经过 `normalizeOverlay` 校验新增覆盖。
- [x] SQLite 与 MySQL 新迁移均已新增，既有迁移未修改。
- [x] MySQL E2E 的 migration version 断言已更新至 `32`；该环境依赖检查未配置，按用户确认不阻塞本次验收。

## Test Results

| 命令 | 结果 |
| --- | --- |
| `go fmt ./cmd/... ./internal/...` | 通过 |
| `go vet ./cmd/... ./internal/...` | 通过 |
| `go test ./cmd/... ./internal/...` | 通过 |
| `go test ./internal/application/service/usecase ./internal/api/http/handler/service` | 通过 |
| `go test ./internal/api/mcp/delivery -count=1` | 通过；覆盖运行字段与 `source_is_host_path` 映射。 |
| `yarn --cwd web lint:fix` | 通过 |
| `yarn --cwd web typecheck` | 通过 |
| `yarn --cwd web test` | 11 个测试文件、64 个测试通过 |
| `yarn --cwd web build` | 本次未运行 |
| `git diff HEAD --check` | 通过 |
| MySQL E2E | 未运行：需要 `BACKEND_GO_E2E_CONFIG`；迁移版本断言已更新至 `32`。 |
| 浏览器人工交互 | 未运行：仓库约定不主动启动开发服务器。 |

## Scope Deviations

存在一项经用户明确要求的契约调整：详情接口移除 `effective`，改为显式返回 Version Component 与 Service Component 两组来源值；这属于原计划详情数据结构的修正，不是新增业务能力。

## Risks And Incomplete Items

1. MySQL E2E 未运行：需要 `BACKEND_GO_E2E_CONFIG`；迁移版本断言已同步至 `32`。
2. 未运行浏览器人工交互：遵守仓库约束，不主动启动开发服务器。
3. 新增运行字段尚未增加完整的 Usecase、Repository、HTTP/MCP 和前端草稿/载荷/重置单元测试；现有全量 Go、前端测试、类型检查和静态检查均通过。

## Conclusion

本次变更已完成标准 Verification：详情接口明确区分 Version 默认值与 Service 当前稀疏值，移除 `effective` 详情返回；前端差异样式和“重置”行为已同步。Go 格式化、vet、全量测试，以及 Web lint、typecheck、测试均通过。MySQL E2E 和浏览器人工交互未运行，前者依赖未配置，后者受仓库约束限制。
