# 不可变 Service Code 与组件级网关域名计划
最后修改时间: 2026-08-08 15:16:30

Review status: Accepted

流程模式: 标准 / standard

## 需求依据

- [需求](../requirement/20260808-service-code-routing.md) 已接受。
- `service_code` 在 Service 创建窗以 `<application_code>-<instance_key>` 预填，创建时由用户提交，是全局唯一、创建后不可编辑的 DNS-label 路由身份。
- 标准 Service 的 `gateway_http` Host 为 `{component_name}.{service_code}.{gateway.base_domain}`；TLS `gateway_tcp` 的 SNI 也使用同一组件级 Host。
- 路由继续由部署渲染的 Traefik Docker labels 下发；平台 `Route` 与 Gateway Dashboard 不在本次变更范围。
- 不修改已执行迁移文件、不引入旧域名兼容、别名或自动后缀。

## 实施步骤

### 1. 确认迁移链与历史数据，再新增 Service code 迁移

1. 在写入迁移前，对目标数据库做只读盘点：按 `application.code || '-' || service.instance_key` 计算候选 code，列出空值、非 DNS label、超过 63 字符和重复值；同时确认现有 Service 数量与目标数据库类型。
2. 本任务迁移版本固定为用户确认的 `000032`。现有 `000030` 迁移不属于本任务，不重命名或修改对方文件。
3. 新增 SQLite/MySQL 成对迁移，为 `service` 增加非空 `code`，按原始 `<application_code>-<instance_key>` 回填，并创建全局唯一约束/索引。迁移前盘点发现异常时停止实施并由用户显式修复或重建开发数据，不对 code 小写化、截断或追加序号。
4. 将 `sql/schema/schema.sql` 同步为迁移后的终态，使 SQLC 的 SQLite schema 与运行时数据库一致；不改写已执行的 `000026_service`。

### 2. 扩展 Service 领域模型、仓储端口与 SQLC 读写

1. 在 `internal/model/service.go` 的 `Service` 与 `ServiceListItem` 增加 `Code`，并在 `ServiceListItem.Service()` 保持该字段。
2. 更新 `sql/query/service/service.sql` 的所有 Service 查询、写入和列表投影，使 `code` 随 Service 完整读取，并新增按 code 的读取。运行 `task sqlc`，仅提交服务查询受影响的 `internal/gen/sqlc/service/**` 生成物。
3. 在 `internal/repository/service.go` 维持由仓储负责持久化和读取的边界；`internal/repository/impl/sqlc/service/repository.go` 将 SQLC row 映射到 `Code`。不引入 SQLite/MySQL 专属错误码或错误文本解析；`CreateServiceWithComponents` 的事务继续保证 Service 与初始组件映射原子创建。
4. 不让 HTTP handler、Gateway 或 renderer 直接使用 SQLC 行、数据库错误或生成代码。

### 3. 在 Service 应用层校验并冻结 code

1. 在 `internal/application/service/usecase` 增加专用的 code 规范化/校验辅助逻辑：trim 用户提交的 code，严格验证为长度不超过 63 的小写 DNS label，首尾不得为 `-`。
2. `CreateService` 接收并持久化 `model.Service.Code`，将格式错误映射为 validation，并通过 `ServiceByCode` 把常规重复提交映射为 conflict。保留数据库唯一索引防止全局 code 并发冲突，但并发穿透仅按通用创建失败处理。
3. `UpdateServiceBasic` 仅更新 Version、instance key 与组件映射；不重新生成、不校验为新 code，也不写入 Service code。Application code 更新同样不得级联 Service code。
4. 扩展 Service usecase fake/store 测试，覆盖合法创建、非法提交编码、长度边界、已存在 code 的预检冲突、修改 instance key 后 code 不变，以及 Service view/list 保留 code。

### 4. 发布只读 API 契约和前端展示

1. 在 `proto/orbit/v1/service/service.proto` 为 `ServiceResp` 和 `ServiceCreateReq` 添加 `code`；不要向 `ServiceBasicUpdateReq` 添加 code。执行 `task proto` 更新 Go/TypeScript 生成物。
2. 更新 `internal/api/http/handler/service/service_mapper.go` 与 mapper 测试，使 list、get、create、update 返回同一只读 code；创建 handler 映射请求中的 code。
3. MCP 的 `orbit_create_service` 不接收 `code`，而是按 Application code 与 `instance_key` 生成 `<application-code>-<instance-key>`；`orbit_update_service_basic` 不接收 code。创建及 Service 更新工具的输出、`orbit_list_application_services` 的摘要都应返回只读 code，便于后续部署和路由工具使用稳定身份。
4. 在 `web/src/views/service/components/ServiceBasicInfoCard.vue` 显示 Service code；`ServiceDetail.vue` 的基本信息弹窗只读展示 code，仅保留 Version 和 instance key 可编辑，code 不进入响应式编辑表单或保存请求。
5. 在 `web/src/views/service/ServicePage.vue` 的创建窗展示可编辑 code，选中 Application 或修改 instance key 时以 `<application_code>-<instance_key>` 预填；将后端 code 校验或冲突错误归属到编码控件。列表/卡片继续呈现 code，并补充中英文 `service.fields.code` 文案。

### 5. 收敛组件级 Host 派生、渲染与公开暴露投影

1. 在模型层新增纯 Host 派生辅助逻辑，接收 `GatewayConfig`、不可变 `Service.Code` 与组件 name，集中校验各 DNS label 和完整域名；避免 `deployment/usecase` 与 `gateway/usecase` 各自字符串拼接、或形成应用用例之间的反向依赖。
2. 在 `internal/application/deployment/usecase/compose_renderer.go` 将 `gateway_http` 的 `Host(...)`、TLS `gateway_tcp` 的 `HostSNI(...)` 和相关 router 名称改为消费组件级 Host/Service code。缺失或不合法的 code、组件 name、base domain 必须使 preview/deploy 明确失败，不使用 Application code 或 instance key 回退。
3. 在渲染阶段校验同一有效 `(host, entrypoint, normalized_path_prefix)` 只对应一个 `gateway_http` endpoint，拒绝重复 Traefik rule；`""` 与 `"/"` 归一为根路径。
4. 在 `internal/application/gateway/usecase/service.go` 让 `buildGatewayExposureItem` 接收 Service，并使用同一辅助逻辑生成 `public_host`、HTTP `client_hint` 和 TLS TCP 提示；调用方在遍历运行中的 Service 时传递该 Service。
5. 保持 Gateway Dashboard 的 `injectGatewayDashboardLabels` 使用现有规则，直到用户明确决定是否把该非 endpoint 路由纳入组件级域名方案。

### 6. 回归测试、活文档与验证准备

1. 扩展 `compose_renderer_entrypoint_test.go` 或新增定向 renderer 测试：同一 Service 的 `minio`、`ragflow` 产生不同 Host；不同 Service 产生不同 Host；TLS TCP 使用组件级 SNI；重复 HTTP rule 与不合法 label 被拒绝。
2. 扩展 Gateway usecase 测试，断言公开暴露的 `public_host` 和 `client_hint` 与 Compose label 相同；扩展 HTTP mapper/proto 测试，断言 code 仅输出不接收。
3. 对新迁移做 SQLite/MySQL 的迁移或仓储集成覆盖，验证回填、非空和唯一约束；异常历史数据的执行前审计结果须记录在实施报告，不能以测试数据掩盖。
4. 更新 `docs/product/cd-model.md`、`docs/architecture/cd-runtime.md` 与 `docs/guides/routing-and-certificates.md`：Service 的路由身份、组件级 Host、TLS 通配证书不匹配两层标签的限制。只修订本任务相关段落，不顺带重写其他历史漂移。
5. 代码实施后按仓库约束运行：`task sqlc`、`task proto`、`go fmt ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`。任何因并行迁移冲突而无法执行的项目须如实记录。
6. 清理 SQLC 列表和计数查询中由 SQLite 未定类型参数产生的 `interface{}`：用显式类型转换保持具名参数、可选过滤和分页绑定语义，并更新所有仓储调用与覆盖查询绑定的集成测试。

## 预计涉及文件

| 区域 | 路径 |
| --- | --- |
| 迁移与 Schema | `sql/migration/{sqlite,mysql}/000032_service_code.*.sql`、`sql/schema/schema.sql` |
| SQLC 与仓储 | `sql/query/service/service.sql`、`internal/gen/sqlc/service/**`、`internal/repository/service.go`、`internal/repository/impl/sqlc/service/repository.go` |
| 领域与用例 | `internal/model/service.go`、`internal/model/<service-host>.go`、`internal/application/service/{dto,usecase}/**` |
| 部署与 Gateway | `internal/application/deployment/usecase/compose_renderer.go`、相关 renderer tests、`internal/application/gateway/usecase/service.go` 与 tests |
| API 契约 | `proto/orbit/v1/service/service.proto`、`internal/gen/proto/orbit/v1/service/**`、`internal/api/http/handler/service/**` |
| MCP 契约 | `internal/api/mcp/delivery/{service_tools,server,mapper,server_test}.go` |
| Web | `web/src/gen/proto/orbit/v1/service/**`、`web/src/views/service/{ServicePage,ServiceDetail}.vue`、`web/src/views/service/components/ServiceBasicInfoCard.vue`、`web/src/i18n/locales/{zh-CN,en-US}.ts` |
| 活文档 | `docs/product/cd-model.md`、`docs/architecture/cd-runtime.md`、`docs/guides/routing-and-certificates.md` |

## 验证计划

- 创建：`ragflow` + `default` 的创建窗预填 `ragflow-default`；用户提交的合法编码出现在 HTTP 的 create/get/list、MCP 的 create/update/list 响应及只读 UI 中。
- 不可变性：更新 instance key、Version、Application code 后，既有 Service code 和由其派生的组件 Host 保持不变。
- 约束：非法生成结果、超过 63 字符、常规全局重复 code、重复有效 HTTP route 都得到确定的 validation/conflict 错误；并发创建最终由唯一索引保障，但不承诺精确的重复 code 文案。
- 路由：`minio.ragflow-default.example.com` 与 `ragflow.ragflow-default.example.com` 分别进入对应容器端口；Gateway 暴露投影与 Compose labels 精确一致。
- TLS：验证 `*.example.com` 不作为满足两级子域名证书的依据；`letsencrypt` 或手工 SAN/二级通配配置由部署者提供。
- 质量：运行第 6 节列出的 SQLC、Proto、Go 与 Web 检查，并核对生成物和活文档只包含本任务必要变更。

## Blockers、假设与风险

- **Blocker**：历史 Service 候选 code 的盘点必须通过。若存在非法或重复值，不能自动改名；需要用户决定修复数据还是重建开发库后才能执行迁移。
- **Assumption**：`<application_code>-<instance_key>` 中的连字符是创建窗默认编码的分隔符；用户可在新建 Service 时人工覆盖。
- **Assumption**：组件 code 指现有 `VersionComponent.name`。使用 `gateway_http` 的组件 name 不合法时由 Host 派生校验拒绝 preview/deploy；不新增 Component code 字段或旧域名别名。
- **Scope boundary**：Gateway Dashboard 不是普通 `gateway_http` endpoint，本计划保持它的现有 Host；如需改为 `traefik.<gateway_service_code>.<base_domain>`，应在本计划接受前确认并单列测试与迁移影响。
- **Risk**：两级子域名不受 `*.base_domain` TLS 证书覆盖。未准备逐 Host SAN、`*.service_code.base_domain` 或 ACME 发放策略时，HTTPS 会因证书名不匹配失败。
- **Scope boundary**：不为罕见的 Service code 并发创建竞争引入数据库驱动专属错误码和错误文本匹配；唯一索引保证数据正确性。

## Rollback

- 迁移执行前保留由用户负责的数据库备份；回退仅使用新增迁移的 down 文件，不修改旧迁移或运行时业务逻辑回填旧 Host。
- 代码回退会恢复旧 Application-level Host，但本任务不维护两个 Host 同时可用；已部署服务需要通过明确的重新部署恢复旧 labels。
- 若历史数据盘点失败，在新增迁移前停止，不进行半完成 Schema 写入。

## User Review Notes

- 2026-08-08：用户要求开始计划，Requirement 自动标记为 Accepted。
- 2026-08-08：计划将 `service_code` 作为创建后只读、不可变的路由身份；创建窗预填拼接编码并允许用户修改。
- 2026-08-08：用户要求开始实现，并确认本任务使用 `000032` 迁移版本；计划视为接受。
- 2026-08-08：用户要求一并清理既有 `CountApplications`、`CountRepositories`、`ListPipelineRuns` 及相关 SQLC 参数结构中的 `interface{}`。
- 2026-08-08：用户接受常规 `service_code` 重复通过 `ServiceByCode` 预检处理，不保留跨 SQLite/MySQL 的精确唯一约束错误解析；同时简化 Traefik 连接错误的冗余 `net.OpError` 判断。
