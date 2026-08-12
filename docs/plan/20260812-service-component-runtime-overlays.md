# 服务组件运行时基础配置覆盖实施计划
最后修改时间: 2026-08-12 20:01:34

Review status: Accepted

Mode: standard

## Requirement Basis

依据 [服务组件运行时基础配置覆盖需求](../requirement/20260812-service-component-runtime-overlays.md)：Service Component 在保持 Version 组件映射、镜像来源及接口契约不变的前提下，可覆盖 `entrypoint`、`command`、`pull_policy` 和 `restart_policy`。保存不自动部署，所有生效读取、预览、哈希与渲染使用同一有效计划。

## Design Summary

在 `service_component` 上直接增加与 `version_component` 对应的稀疏覆盖字段：`entrypoint_json`、`command_json`、`pull_policy`、`restart_policy`。不建立 runtime 子表或嵌套对象，不存储或暴露镜像覆盖。所有字段仍由同一 `ServiceComponent` 映射、同一 `UpdateServiceComponentOverlay` 替换事务和同一有效计划合并逻辑处理。

| 字段 | 持久化语义 | 有效值语义 |
| --- | --- | --- |
| `entrypoint_json` | `NULL` 表示继承；`[]` 或非空 JSON argv 表示覆盖 | `NULL` 继承 Version；`[]` 使 Compose 省略 `entrypoint` |
| `command_json` | `NULL` 表示继承；`[]` 或非空 JSON argv 表示覆盖 | `NULL` 继承 Version；`[]` 使 Compose 省略 `command` |
| `pull_policy` | `NULL` 表示继承；否则为受限枚举 | 使用覆盖值或 Version 值 |
| `restart_policy` | `NULL` 表示继承；否则为 Version 已支持的策略值 | 使用覆盖值或 Version 值；`no` 不输出 Compose restart |

服务层输入与对外响应保持命令文本格式，使用既有 `commandline.Parse` / `Format` 在文本与 argv 之间转换。`entrypoint_json` 与 `command_json` 的 `NULL`/`[]` 直接表达继承/显式清空；`pull_policy` 与 `restart_policy` 使用空值继承和已有的合法枚举值覆盖，不新增独立状态列。

## Implementation Steps

1. 新增 `service_component` 字段与 SQLC 支持。
   - 新增 `sql/migration/sqlite/000032_service_component_runtime_overlays.up.sql` / `.down.sql` 与 MySQL 对应文件，直接为 `service_component` 添加可空的 `entrypoint_json`、`command_json`、`pull_policy`、`restart_policy` 列，并为两个策略列添加与 Version 相同的有效值约束。
   - 更新 `sql/schema/schema.sql` 与 `sql/query/service/service.sql`，让 Service Component 的查询、插入和 overlay 更新直接读写四个列；不增加读取、删除或插入 runtime 子记录的查询。
   - 运行 `task sqlc`，提交 `internal/gen/sqlc/*` 的必要生成结果。保持其他已存在的工作区改动不变。

2. 扩展 `ServiceComponent` 领域模型与仓储映射。
   - 在 `internal/model/service.go` 的 `ServiceComponent` 直接增加与 Version 对应的 `Entrypoint`、`Command`、`PullPolicy`、`RestartPolicy` 字段。argv 使用 nil/空切片区分继承与显式清空；策略用 nil/合法值区分继承与覆盖。
   - 在 `internal/repository/impl/sqlc/service/repository.go` 的 `serviceComponentFromRow`、`insertServiceComponent` 和 `UpdateServiceComponentOverlay` 事务中直接读取、写入和替换这四个字段；删除子覆盖记录时不影响这些列。
   - 扩展 `internal/application/service/dto/service.go` 的 overlay 输入，避免编辑基础运行配置时丢失已有 env、mount、resource 与 endpoint 覆盖。

3. 集中校验与有效计划合并。
   - 在 `internal/application/service/usecase/service.go` 的 `normalizeOverlay` 中校验 `ServiceComponent` 的四个直接覆盖字段，解析命令文本为 argv、校验 `pull_policy` / `restart_policy`，并消除与 Version 相同的稀疏覆盖值。
   - 扩展 `internal/application/deployment/usecase/effective_service_plan.go` 的 `MergeServiceComponent`：只为四个允许字段覆盖 `EffectiveServiceComponent`；保持 `Image`、名称、依赖、健康检查、设备、`tmpfs`、`ulimits` 及端点契约从 Version 继承。
   - 复用已有 `EffectiveServicePlanHash` 和 `versionComponentFromEffective`，确认 runtime overlay 进入哈希并被 Compose 渲染；不改动版本预览逻辑。
   - 让 Service 切换 Version 继续由 `remapServiceComponents` 调用同一字段校验，确保不兼容的现有策略值阻止整个更新事务。

4. 扩展 HTTP、MCP 与生成协议。
   - 更新 `proto/orbit/v1/service/service.proto` 的 Service Component overlay 请求/响应与声明/有效详情结构，直接增加与 `version_component` 对应的 `entrypoint`、`command`、`pull_policy`、`restart_policy` 字段，不为镜像创建 Service 覆盖字段或独立 runtime 消息。
   - 运行 `task proto`，更新 Go protobuf、Web TypeScript DTO，并调整 HTTP mapper 与 MCP mapper / `orbit_update_service_component_overlay` 输出。
   - 保持 API 路由、更新操作和 MCP 工具名称不变；它们仍替换同一个声明组件的完整稀疏 overlay。

5. 更新服务组件详情交互。
   - 在 `web/src/views/service/ServiceComponentDetail.vue` 增加运行配置编辑分组，显示 Version 默认、Service 当前和继承/覆盖来源。
   - `pull_policy` 使用枚举选择，`restart_policy` 使用 Version 同值集合的枚举选择，其中 `no` 表示显式禁用且“恢复继承”提交空值；`entrypoint` / `command` 使用命令文本输入并支持继承、覆盖和显式清空。
   - 生成保存载荷时合并现有四类 overlay，保存和重新加载后保留未改动的覆盖；镜像只读显示或不在该页面的可编辑表单中出现。
   - 复用项目共享组件、既有表单错误样式与 `useStatusAsync` / Toast 行为；运行配置和既有环境变量、资源、端点、挂载覆盖统一使用“重置为 Version 值”文本和清除覆盖行为，不再由 UI 创建删除覆盖；更新本地化文本。

6. 补充定向测试与运行项目检查。
   - 有效计划测试覆盖四个直接字段的继承、覆盖、argv 显式清空和哈希变化，断言镜像仍来自 Version。
   - Service usecase 测试覆盖文本命令解析、非法输入、重复/无效策略、冗余覆盖消除、详情声明/覆盖/有效值，以及切换 Version 时的校验和原子性。
   - SQLC repository 测试覆盖 `service_component` 四个列的 SQLite 往返读写、`NULL` 与空 argv 区分。
   - HTTP mapper/MCP 测试覆盖请求和响应契约；Web 单元测试覆盖草稿初始化和载荷生成的三态语义。
   - 实现后运行 `go fmt ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`，以及相关前端测试。

## Files To Change

| 范围 | 预期文件 |
| --- | --- |
| 数据库 | `sql/migration/{sqlite,mysql}/000032_service_component_runtime_overlays.*.sql`、`sql/schema/schema.sql`、`sql/query/service/service.sql`、SQLC 生成物 |
| 后端领域与仓储 | `internal/model/service.go`、`internal/repository/impl/sqlc/service/repository.go`、相关 repository / usecase DTO |
| 有效计划与渲染 | `internal/application/deployment/usecase/effective_service_plan.go`、`compose_renderer.go`（仅在测试确认需要时） |
| 传输 | `proto/orbit/v1/service/service.proto`、HTTP service handler mapper、MCP delivery mapper/tool、生成 DTO |
| 前端 | `web/src/views/service/ServiceComponentDetail.vue`、关联 API/生成 DTO、i18n、定向测试 |
| 测试与文档 | Service、deployment、repository、HTTP/MCP、Web 测试；本需求和后续 verification 文档 |

## Verification Plan

1. 迁移测试从空 SQLite 数据库升至 `000032`，确认 `service_component` 新列、约束及已有 Service 迁移链正常。
2. 后端单测验证四字段合并的全部状态、无镜像覆盖、有效计划哈希和 Compose 内容。
3. 服务更新和 Version 切换测试验证失败不产生部分持久化写入。
4. 传输测试验证 HTTP/MCP 均能往返 `service_component` 的直接覆盖字段，且生成协议没有忽略字段。
5. 前端测试及类型检查验证三态编辑、完整 payload 保留和窄屏显示；执行前端 lint fix 与 typecheck。
6. 执行仓库规定的 Go 格式化、vet 与测试命令，并在 Verification 阶段记录实际结果。

## Assumptions

- Version `pull_policy` 合法集合仍为 `always`、`missing`、`never`；Service 复用该集合。
- Version `restart_policy` 当前合法集合仍为 `no`、`unless-stopped`；Service 覆盖不扩大该集合。
- `service_component` 的四个可空直接字段按现有稀疏覆盖方式表达继承；不需要 runtime 子表、状态列或嵌套 runtime DTO。

## Risks

- `nil` argv 与空 argv 的 JSON/Go/TypeScript 表达必须端到端保真；任一层把空数组省略将失去显式清空语义。
- 当前 runtime renderer 只在非空 argv 时输出 `entrypoint` / `command`。实现必须以明确的“字段被覆盖”状态驱动渲染，避免空 argv 被错误地重新继承。
- `restart_policy=no` 与 Compose 省略 restart 字段之间需有一致的有效模型表达，不能只在前端消除显示。
- 数据库与 proto 生成会触及共享生成文件；实现时必须与当前 Pipeline Run 未提交改动共存，避免覆盖或回退它们。

## Rollback

- 新功能未部署前可将新增的 `service_component` 覆盖列置空并回退新代码恢复 Version 继承行为。
- 已保存覆盖的环境中，发布修复版本应优先保留记录并恢复继承解析；不得通过修改已执行迁移或删除 Service 组件数据进行回滚。

## User Review Notes

- 2026-08-12：需求阶段确认 Service 不覆盖镜像，仅覆盖启动命令、拉取策略与重启策略；用户要求进入计划阶段。
- 2026-08-12：用户要求开始实现，计划阶段自动接受。
- 2026-08-12：用户要求所有 Service Component 覆盖卡片统一重置语义和文案，删除入口改为恢复 Version 声明。
- 2026-08-12：用户确认详情沿用既有覆盖卡片的 Version 默认值、当前值和来源表达；MySQL E2E 为环境依赖的非阻塞检查。
