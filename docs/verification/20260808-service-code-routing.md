# 不可变 Service Code 与组件级网关域名验证
最后修改时间: 2026-08-08 15:59:39

Review status: Draft

流程模式: 标准 / standard

## 需求对齐

- Service 持久化、SQLC 投影、领域模型、HTTP/Proto、MCP 与 Web 均传递只读 `code`；创建请求要求提供 code，基础信息更新契约未新增该字段。
- 创建页按 `<application_code>-<instance_key>` 预填，允许提交前覆盖；前后端均校验小写 DNS label，并将常规重复映射到 code 控件错误。
- SQLite/MySQL `000032` 为既有 Service 确定性回填 `<application_code>-<instance_key>`，并以全局唯一索引约束 code。
- `gateway_http`、TLS `gateway_tcp`、Gateway 公开暴露投影均使用 `{component_name}.{service_code}.{base_domain}`；无 TLS TCP 仍使用 `HostSNI(*)`。
- 渲染器拒绝相同 Host、entrypoint、归一化 path prefix 的重复 HTTP 路由。Traefik 连接错误判断同步移除冗余的 `net.OpError` 分支。

## 计划对齐

- 已完成迁移、SQLC、仓储端口、应用层校验、HTTP/MCP 契约、Web 展示、Host 派生、Compose/Gateway 投影和活文档更新。
- SQLC 查询改为具名参数，消除了 Service 列表/计数生成结构中的 `interface{}` 绑定。
- Gateway Dashboard 保持既有非 endpoint 域名规则，未扩展到组件级域名。

## 实际差异

暂存内容为 47 个文件，涵盖 `000032` 迁移、Service code 领域与 API、组件路由、MCP、Web、测试与文档。未包含并行的异步操作状态机、Pipeline Stage 或其生成物。

验证期间发现此前为拆分并行改动而制作的零上下文暂存补丁有 4 处插入偏移：schema、Gateway 创建参数、Service DTO/usecase 和 SQLC `Service` 模型。已按 `HEAD` 基线与 SQLC 输出重新构造索引内容；工作区业务源码未被改写。

## 验收清单

- [x] Service code 是创建后不变的全局唯一 DNS-label 身份。
- [x] 创建、列表、详情、MCP 和只读编辑窗均可见 code；更新契约不接收 code。
- [x] Compose 与 Gateway 暴露投影共享组件级 Host 派生。
- [x] 重复 HTTP 路由与非法 Host 输入有回归测试。
- [x] SQLC 和 Proto 生成物与当前暂存输入精确一致。
- [x] 活文档说明两级子域名和 TLS 证书覆盖限制。
- [ ] 未对目标运行数据库执行历史 Service 候选 code 审计；迁移执行前仍需按计划处理非法或重复历史数据。

## 检查结果

| 检查 | 结果 |
| --- | --- |
| SQLC generate（隔离暂存副本） | 通过，Service 生成物哈希一致 |
| Buf Go/TypeScript generate（隔离暂存副本） | 通过，生成物哈希一致 |
| `yarn --cwd web lint:fix`（隔离暂存副本） | 通过 |
| `yarn --cwd web typecheck`（隔离暂存副本） | 通过 |
| `go fmt ./cmd/... ./internal/...`（隔离暂存副本） | 通过 |
| `go vet ./cmd/... ./internal/...`（隔离暂存副本） | 通过 |
| Service code 定向 Go 测试 | 通过：model、Service/Gateway/Deployment usecase、HTTP handler、MCP、SQLC repository、Traefik |
| SQLite `TestMigrateUpSQLiteCreatesPipelineSchema` | 通过 |
| `go test -count=1 ./cmd/... ./internal/...`（隔离暂存副本） | 未通过：`000041_pipeline-demo.sqlite.sql` 依赖未暂存的并行 `000030/000031` Pipeline 迁移，缺少 `pipeline_stage_legacy_000030`；Service code 相关包均通过 |
| `git diff --cached --check` | 通过 |

## 风险与未完成项

- 当前全量 Go 测试的失败来自并行 Pipeline 迁移与 seed fixture 的跨任务依赖，不能作为 Service code 回归失败处理；合并该迁移任务后应再次运行全量测试。
- `000032` 对历史数据的回填不做截断、自动改名或后缀补偿。执行前必须盘点真实目标数据库，非法或重复候选 code 需要人工修复或重建开发数据。
- 两级子域名不匹配 `*.base_domain`；部署者仍需准备实际 Host 的 ACME/SAN 或 `*.service_code.base_domain` 证书覆盖。

## 结论

暂存的 Service code 变更满足已接受的需求与计划，定向回归、生成物一致性、前端检查、Go 格式和静态检查均通过。等待处理并行 Pipeline 迁移依赖后复跑全量 Go 测试，再进行提交。
