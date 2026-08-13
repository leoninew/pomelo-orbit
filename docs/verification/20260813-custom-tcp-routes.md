# 自定义 TCP 路由验证记录
最后修改时间: 2026-08-13 22:53:46

Review status: Draft

Mode: strict

## Requirement Alignment

实现覆盖已接受 Requirement 的主要功能链路：

- Route 模型、SQL、Proto、HTTP API 和 Web 表单均区分 `http` 与 `tcp`；TCP Route 使用受管 Service Component 的 `(protocol, container_port)` Endpoint，持久化公开 `listen_port`。
- Route usecase 校验跨项目权限、受管 Component/Endpoint 存在性、TCP 的 `internal` mode、保留端口、启用 TCP Route 的端口独占，以及运行中 Service 的 `host` / `local` 宿主机端口冲突。
- Traefik REST 快照同时包含顶层 `http` 与 `tcp` namespace；TCP 生成 `HostSNI(*)`、`tcp<listen_port>` entrypoint 和受管容器别名上游，空 map 会被发送以清理历史动态配置。
- Gateway compile 根据所有启用 TCP Route 收集监听端口，生成静态 entrypoint 和 Compose 端口映射；Route 写入后的全量快照流程会触发该 reconcile。
- Gateway deploy/restart 成功路径在标记部署成功前调用 Route 全量快照发布；发布失败会使部署失败。
- Endpoint mode 已收敛为 `internal`、`local`、`host`、`gateway`；业务代码不接受 `gateway_http` 或 `gateway_tcp`，Endpoint 名称已改为由协议与容器端口派生。

RAGFlow integrated/split 的手工维护 contract 已改为 `gateway`，生成的 primitives 和 `AGENTS.md` 同步更新。对这些非归档可执行输入的旧 mode 搜索无结果，因此不会再被新运行时枚举拒绝。

## Spec Alignment

实现与 Spec 的核心设计一致：TCP Route 一端口一条启用路由、动态 TCP REST 配置与 Gateway 静态监听分离、受管 HTTP/TCP target 以 Service/Component/Endpoint 三级选择、以及 Gateway worker 成功后同步全量 Route。新增 `000033`、`000034`、`000035` SQLite/MySQL 迁移和对应离线转换/清理脚本，没有改动已执行迁移。

未完成真实部署验证，故无法在实际 Traefik/Gateway 环境中确认静态端口切换、REST 快照恢复和端口冲突反馈；代码级测试已覆盖这些行为的主要分支。

## Plan Alignment

Plan 所列的数据库/SQLC、Endpoint mode、Route usecase、Traefik adapter、Gateway compile、部署 worker、HTTP/MCP/Proto、Web 与活文档改动均已出现在实际 diff 中。`task sqlc` 和 `task proto` 成功运行，生成物可由当前 schema 与 proto 重新生成。

Plan 中的交付前人工核验未执行：不主动启动开发服务器，也没有对运行中的 Gateway 做 deploy/restart。MySQL E2E 同样未运行，因为本轮未配置 `BACKEND_GO_E2E_CONFIG`。

## Actual Diff Summary

- 数据库：新增 Route TCP 字段、Endpoint 身份迁移和旧 mode 移除迁移；新增 SQLite/MySQL 离线 mode 转换与 Endpoint 数据清理脚本。
- 后端：扩展 Route、Endpoint 和 Service 模型/仓储/SQLC，增加 TCP target 校验、端口冲突检测、Gateway listener compile 与部署成功后的 Route publish hook。
- Traefik：REST 发布和 Dashboard router 查询支持 HTTP 与 TCP；快照会清空两个协议 namespace 的旧配置。
- API 与生成物：更新 HTTP mapper、MCP endpoint 描述、Go/TypeScript Proto、SQLC 代码。
- 前端：新增 Route 受管目标级联选择器，Route 创建/编辑/详情支持 HTTP 高级自定义下游和 TCP 地址/监听端口；Endpoint 展示改为协议加容器端口。
- 文档：更新产品、运行时、路由、MCP 操作说明和本任务的 Requirement/Spec/Plan。

验证开始前，本功能 diff 涉及 121 个文件、4,187 行新增和 1,109 行删除；随后新增本验证记录。`go fmt` 与 `yarn --cwd web lint:fix` 已按仓库约束执行，最终 diff 无空白错误。

## Expected Vs Actual Changed Files

| 预期范围 | 实际情况 |
| --- | --- |
| SQLite/MySQL 迁移、离线脚本、schema、query、SQLC | 已修改；生成命令成功，SQLite 路由集成测试通过。 |
| Endpoint mode、有效计划与 Compose | 已修改；`gateway_tcp` 业务渲染已删除，Gateway TCP 监听改由 Route 集合驱动。 |
| Route、Traefik 与 Gateway | 已修改；包含 TCP target/端口校验、HTTP/TCP REST 快照、Gateway compile 及部署后同步测试。 |
| HTTP、MCP、Proto | 已修改；Go 与 TypeScript 生成物由 `task proto` 重新生成。 |
| Route Web UI 与 Endpoint 展示 | 已修改；lint、typecheck 与现有前端测试通过。 |
| 活文档与 RAGFlow 契约 | 已更新；手工 contract、生成 primitives 与 `AGENTS.md` 均已使用 `gateway`，旧 mode 搜索无结果。 |

## Acceptance Checklist

- [x] Route 规格、API、持久化和 Web UI 可区分 HTTP 与 TCP，并只显示协议适用字段。
- [x] TCP Route 校验项目内受管 Service Component、TCP 协议、`internal` mode、权限与 endpoint 存在性。
- [x] TCP Route 选择 Endpoint 时以容器端口预填公开监听端口，仍允许用户修改。
- [x] HTTP Route 默认使用受管 HTTP Endpoint，受管引用和高级自定义 URL 互斥。
- [x] Route Web UI 通过项目范围 Service 查询与 Service 详情构建 Service/Component/Endpoint Combobox，而非当前分页列表。
- [x] Endpoint identity 收敛为 `protocol + container_port`，Web 派生 `http<port>` / `tcp<port>` 显示。
- [x] HTTP/TCP Route 都生成对应顶层 Traefik REST namespace；TCP 使用独占 entrypoint 和 `HostSNI(*)`。
- [x] Gateway compile 使用启用 TCP Route 的端口集合，Route 写入会触发全量 snapshot 与 Gateway listener compile。
- [x] Gateway deploy/restart 成功路径会发布全量 Route snapshot，发布错误会使 Deployment 失败。
- [x] 迁移、路由校验、Traefik snapshot、Gateway compile 和 worker hook 已有 Go 自动化覆盖；Gateway 默认配置调用方、测试 fixture 与配置 fixture 已补齐。`TraefikConfig` 已收敛为五项运行时字段，环境变量绑定不再保留已移入代码常量的 Gateway 身份与展示字段；全量 Go 测试通过。
- [x] 旧 `gateway_http` / `gateway_tcp` 已从全部非归档的可执行 RAGFlow 部署输入和仓库约束移除；contract 重渲染与测试通过。
- [ ] 已在真实 Gateway 环境完成 deploy/restart，验证 Route snapshot 恢复、静态 entrypoint/端口映射和业务容器不直接映射 TCP 端口。
- [ ] 已在 MySQL 环境执行迁移/E2E，验证新迁移与离线转换前置条件。

## Test Results

| 命令 | 结果 |
| --- | --- |
| `task sqlc` | 通过 |
| `task proto` | 通过；`buf dep update` 提示没有待更新依赖。 |
| `go fmt ./cmd/... ./internal/...` | 通过 |
| `go vet ./cmd/... ./internal/...` | 通过 |
| `go test ./cmd/... ./internal/...` | 通过；Gateway 默认配置的构造注入、测试 fixture 与配置 fixture 已同步，随后修复的 Traefik 配置绑定收敛已复跑。 |
| `go test ./internal/config -count=1` | 通过；验证 Traefik 五字段环境变量覆盖与配置校验。 |
| `go test ./internal/application/gateway/... -count=1` | 通过；验证 Gateway 默认值与编译用例。 |
| `yarn --cwd web lint:fix` | 通过 |
| `yarn --cwd web typecheck` | 通过 |
| `yarn --cwd web test` | 11 个测试文件、64 个测试通过 |
| RAGFlow contract renderer tests | 通过：integrated/split 共 10 个测试。 |
| `render_contract.py --check` | integrated/split 均通过，生成 primitives 与 contract 一致。 |
| 非归档 RAGFlow 旧 mode 搜索 | 通过：`AGENTS.md` 与两个部署技能中无 `gateway_http` / `gateway_tcp`。 |
| `git diff --check` 与 `git diff --cached --check` | 通过 |
| MySQL E2E | 未运行：需要 `BACKEND_GO_E2E_CONFIG`。 |
| 浏览器/Gateway 人工验证 | 未运行：仓库约束禁止主动启动、停止或重启开发服务器，且本轮没有用户授权操作真实部署。 |

## Scope Deviations

未发现超出 Requirement、Spec 或 Plan 的产品代码扩展。生成的 SQLC model 文件在多个 domain 同步变化，源于共享 Endpoint/Service schema 变化。

## Risks And Incomplete Items

1. 迁移 `000033` 允许旧 mode 仅为离线转换窗口服务；上线前必须先备份并执行 `sql/migration/offline/000033_endpoint_mode_rename.{sqlite,mysql}.sql`，再收紧到 `000035` 的四个现行 mode。
2. MySQL 迁移路径和真实 Gateway 行为没有在本轮环境运行；TCP Route 的外部端口还受宿主机占用与防火墙影响。
3. 当前工作区包含未提交 diff，且验证命令可能进行了格式化写入；未执行任何 Git 写操作或重置。

## Conclusion

Gateway 默认配置调用链和 RAGFlow contract 旧 mode 均已修复，当前全量 Go、前端、生成和 RAGFlow contract 检查全部通过，主要实现与已接受的 Requirement、Spec 和 Plan 对齐。本验证记录保持 `Draft`，仅因 MySQL E2E 与真实 Gateway deploy/restart 尚未在当前环境和授权范围内执行；完成这两项后可标记为完整交付。
