# 移除服务实例语义计划
最后修改时间: 2026-09-16 17:26:38

Review status: Accepted

Mode: standard

## Basis

依据已接受的 [Intent](../intent/20260916-remove-service-instance.md) 实施。Service 保留为 Application 可拥有的多条运行绑定，区分标识是当前 Project 唯一的 `service.code`，而不是 `instance_key`。Environment 的历史 `state` 字段不参与生命周期或就绪性判断，环境状态由目标配置和 Probe 结果表达。本任务直接删除旧 schema 与协议字段，不提供兼容 alias、default fallback 或 Proto `reserved`。

## Implementation steps

### 1. 移除实例键持久化与生成契约

- 新增三数据库 `000043` 迁移：移除 `service.instance_key` 与 `(application_id, instance_key)` 唯一约束，保留 `uq_service_project_code`；不合并或删除同一 Application 的既有多条 Service。
- 同步 `sql/schema/{sqlite,mysql}.sql` 的终态定义。SQLite 以表重建方式保留 Service 及其子表关联；MySQL/PostgreSQL 删除相应列和唯一索引。
- 新增三数据库 `000044` 迁移：删除 `environment.state` 列及相关约束；同步终态 schema。SQLite 以表重建保留 Environment 的目标、Probe 和 Gateway 绑定数据，MySQL/PostgreSQL 直接删除列。
- 从 `internal/model.Service`、`ServiceListItem`、Deployment 投影与 deployment `options_json` 中删除 instance-key 字段；历史 `options_json` 中的未知 JSON 属性不进入新模型，也不新增迁移或运行时分支。
- 修改 `sql/query/service`、`deployment` 与 `gateway`：删除按 key 查询、排序和 join 字段；保留并覆盖 `project_id + code` 的唯一性检查。执行 `task sqlc`，只以 SQL 定义更新生成代码。
- 从 environment SQL 查询、SQLC repository、领域模型、DTO、HTTP/Proto/MCP 映射和环境测试中删除 `state`；保留 `last_probe_status` 及其 Revision/诊断字段，执行 `task sqlc` 与 `task proto` 更新生成代码。

### 2. 收敛 Service 与部署用例

- Service create DTO/proto/handler 保留 `application_id`、`version_id`、`code`；删除 `instance_key`。基础更新只更换 Version，不再接受实例键。
- 创建时继续规范化 Service code，并在当前 Project 调用 `ServiceByProjectAndCode` 检查冲突；不以 Application 计数、端口、挂载、网络或运行状态拒绝第二条 Service。
- 删除 `ServiceByKey`、`ServiceIdByKey`、`ensureSingleRuntime` 及其测试。所有运行时解析以 scoped `service_id` 加载 Service，并校验它属于请求 Application（若入口同时含 Application ID）。
- 从 effective plan、plan hash、Compose renderer、Deployment command/worker/verification 中删除 instance-key 字段。由 `service.code` 生成 Compose project、工作目录、日志目录、Traefik router 名、显式 container name 与平台网络 alias，避免多个 Service 因内部命名发生必然冲突；端口及用户声明挂载仍在部署时由 Docker 判定。
- Deployment 列表和详情不再投影 `service_instance_key`。

### 3. Gateway 收敛为受管 Service

- Gateway 创建仍建立一个受管 Service，code 保持 `traefik-default`，但不写入或查找 instance key。
- 将 `DefaultService`、`default_service_*`、`GatewayRuntimeServiceCode` 的 default 排序和 Provision 输入替换为语义中性的 Gateway Service；Gateway domain 检查受管 Application 恰有一个 Service，并以 Service ID 传递部署、停止与初始化状态。
- 直接删除 Gateway、Initialization 和 Route 的 DefaultService/instance-key HTTP 与 Proto 字段，不使用 `reserved`；执行 `task proto` 后同步生成调用点。

### 4. 更新 Web 与 MCP 合同

- `web/src/views/service/ServicePage.vue` 删除实例键控件与验证。Application 选择后预填 code 为 `<application-code>-default`，保留 code 编辑框；提交依赖服务端当前 Project code 唯一校验，冲突显示既有表单错误。
- `ServiceDetail.vue` 基础编辑移除实例键，列表、卡片、标题、面包屑、日志抽屉和删除确认只使用 Application 名称、Service code 与 Version，不再拼接或隐藏 `xxx / default`。
- 逐项更新 Gateway、Route、Application、Deployment 页面与导航测试，使用 `service_id` / `service_code` 而非 default service 或 instance key；删除不再成立的 Service selector 分支。
- 统一更新中英文 i18n：删除“服务实例”“实例键”“默认服务”和 `xxx / default` 的 Service 文案，替换为“服务”及 Service code。只处理 Service 领域含义，保留变量、入口点、network 等无关的 default 文案。
- MCP Service create/update、gateway provision、runtime doctor、compose/runtime 查询的 schema 与输出移除 instance-key。多 Service 下运行时工具要求 `service_id`，不再以 Application ID + key 或空值默认 `default` 定位目标。

### 5. 更新活文档与测试

- 更新 `docs/product/cd-model.md`、`docs/architecture/backend.md`、`docs/guides/deployment.md`、`docs/guides/mcp-direct-operations.md` 中的 Service/默认 Gateway 表述；不修改归档文档。
- Go 测试覆盖：同 Application 多 Service 可创建，Project 内重复 code 冲突，第二条活跃 Service 不受应用级 guard 拒绝，runtime/renderer 使用 Service code，Gateway 仅有受管 Service，以及移除字段后的 HTTP/Proto 映射。
- Web 测试覆盖：Application 选择预填 code、实例字段不存在、Service code 冲突反馈与 `xxx / default` 文案/标题收敛。
- 重新生成后执行 `task sqlc`、`task proto`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`、相关 Vitest、`task check`、`go test ./cmd/... ./internal/... ./sql` 与 `git diff --check`。

## Files to change

| 范围 | 文件/目录 |
| --- | --- |
| 数据定义与迁移 | `sql/migration/{sqlite,mysql,postgres}/000043_*`、`000044_*`、`sql/schema/**`、`sql/query/{service,deployment,gateway,environment}/**`、`internal/gen/sqlc/**` |
| Service/Deployment | `internal/model/{service,deployment,service_routing}.go`、`internal/repository/service.go`、`internal/application/{service,deployment}/**`、相关 SQLC implementation 与测试 |
| Gateway/Route/Initialization | `internal/application/{gateway,route,project_initialization}/**`、对应 HTTP handler/routes、tests |
| HTTP/Proto | `proto/orbit/v1/{service,deployment,gateway,project_initialization}/**`、`internal/api/http/handler/{service,deployment,gateway,application,project_initialization}/**`、生成 Proto |
| MCP | `internal/api/mcp/delivery/**` |
| Environment | `internal/model/environment.go`、`internal/application/environment/**`、`internal/repository/impl/sqlc/environment/**`、`internal/api/http/handler/environment/**`、`proto/orbit/v1/environment/**` |
| Web | `web/src/api/service/**`、`web/src/views/{service,gateway,route}/**`、相关 Application/Deployment 视图、`web/src/i18n/locales/**`、相关 tests |
| 活文档 | `docs/product/cd-model.md`、`docs/architecture/backend.md`、`docs/guides/{deployment,mcp-direct-operations}.md` 与本任务过程文档 |

## Verification plan

1. 迁移测试在 SQLite、MySQL、PostgreSQL 验证 Service 行及子表保留、instance-key 列不存在、同 Application 可插入多个 Service、Project code 唯一约束仍生效。
2. Service usecase/HTTP 测试验证多 Service 正向创建、重复 Project code 冲突、基础更新仅改 Version，以及缺失字段不再出现在 JSON/Proto 响应。
3. 部署测试验证两个 Service 使用不同 code 时 Compose project、目录、container name、network alias 和 router 名不同；端口或挂载冲突不作为创建 guard。
4. Gateway、Route、Initialization 与 MCP 测试验证不再读取、写入或输出 default service / instance key，运行时目标以 Service ID 获取。
5. 前端测试与文本检索验证创建默认 code 预填且可编辑，实例表单和 `xxx / default` 格式的 Service 显示被移除；无关 default 文案保持。
6. Environment 测试与契约检查验证 `state` 列、请求字段和响应字段均不存在，目标保存与 Probe/初始化判断保持使用目标修订及 Probe 结果。
7. 运行生成、lint、format、typecheck、Go/Web 测试与 `git diff --check`；Implementation 完成后等待用户明确进入 Verification，再创建验收文档。

## Blockers

无。Service code 的前端预填与服务端 Project 内唯一检查已明确；不需要新增编码预检 API。

## Assumptions

- Service code 创建后仍保持不可编辑，现有 Service code 不重命名；用户在创建表单中解决与已存在 code 的冲突。
- Gateway Application 维持一个受管 Service 的产品约束；普通 Application 可以拥有多条 Service。
- 任务不修改当前工作区已有的三份 `000033_seed_pipeline.up.sql` 变更。
- `Environment.state` 指模型、数据库与环境 HTTP/Proto 契约中的遗留字段；不涉及 Probe 的 `last_probe_status` 或服务/组件等其他领域中的 `status`、`state` 字段。

## Risks

- 移除 instance-key 列及 Proto 字段是破坏性变更，所有 Web、HTTP、MCP 客户端与生成物必须同期更新。
- service code 驱动的 container/network/router 命名会使已部署容器在首次重新部署时按新名称重建；保留 code 可使目录、Compose project 与路由主机保持稳定。
- 多 Service 不再有应用级运行 guard，Docker 资源冲突会在部署时暴露；这是已接受的产品边界。
- 删除 Environment `state` 后，尚未升级的独立 HTTP/Proto 调用方不能继续传递或读取该字段；本项目 Web 与 MCP 会同期更新。

## Rollback

代码与 schema 迁移以本次整体提交回滚，不提供 instance-key fallback、旧 HTTP 参数或双协议路径。已执行迁移的数据库按项目既有迁移回滚能力处理；不以运行时兼容逻辑回退。

## User review notes

- 2026-09-16：用户要求在移除实例概念时同步清理 UI 中的 `xxx / default` 类 Service 展示。
- 2026-09-16：用户明确同一 Application 可以创建多条 Service，Project 内 Service code 不可重复；端口与挂载冲突不作为创建时保障。
- 2026-09-16：用户澄清 `<application-code>-default` 是前端协助生成的初始 Service code，服务端继续检查当前 Project 中尚未使用。
- 2026-09-16：用户要求同步 drop 不再使用的 `Environment.state` 字段；Probe 状态不在删除范围内。
