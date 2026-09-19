# Project 级 CD 接管数据传输计划
最后修改时间: 2026-09-17 17:27:47

Review status: Accepted

Mode: standard

## Intent basis

本计划落实已接受的 [Project 级 CD 接管数据传输 Intent](../intent/20260917-project-cd-handover-transfer.md)。能力以 Orbit 的 Project CD 配置聚合为边界，不使用现有 Python `database-transfer` 的 Service / Environment JSONL 脚本。当前任务保留该脚本及其入口。新能力不执行远程部署或路由发布，也不提供通用数据库导入能力。

## Design boundary

实现遵从当前 `application -> repository + model -> repository/impl` 架构，而不新增 Handler/Application 事务脚本：

| 层 | 职责 |
| --- | --- |
| HTTP / Web | 从认证上下文取得当前用户；下载 JSON 包；上传文件并选择新建或覆盖模式；显示结果。不得直接访问 CD 表。 |
| Application | Project Handover 检查源 / 目标 Project 成员资格，解析和生成接管 JSON，选择模式，做私钥持久化表示转换，并按依赖顺序调用领域服务。它维护包内源 ID 到目标 ID 的映射，但不得访问仓储、排列表删除或逐行写入。 |
| Domain | Project、Environment、Application、Gateway、Service、Route 分别提供其完整业务定义的读取、校验、创建、更新与控制面移除能力。Gateway 定义拥有其 Application、Version、runtime Service、dashboard Route、Config 与 Environment binding；不能将其降解为独立 Config 注册。Pipeline 拥有对 Application / Version 引用的查询。缺失能力须先作为完整资源生命周期的一部分实现，不能为 Handover 逐项添加一次性入口，也不能新建跨 Project 的“大聚合仓储”或“替换全部 CD 配置”命令。 |
| Repository implementation | 沿用各领域现有 SQLC repository；新增查询只进入所属领域的 `sql/query/<domain>/`，新增写入只操作该领域自身的表。 |

现有 POST 请求 UoW 已通过 `internal/infrastructure/database/tx/request.go` 将同一 `context` 的 SQLC 仓储绑定到同一个事务。导入 HTTP 路由不加入现有事务豁免列表，因此“创建/覆盖 Project + 成员关系 + 清理 + 恢复”天然处于一次请求事务；任何错误返回后由 middleware 回滚。

### 跨领域删除边界

本轮同步审计并纠正既有删除链路。聚合内部的 `Application -> Version -> VersionComponent`、`Service -> ServiceComponent/overlay`、`GatewayConfig -> profile binding`、`Pipeline -> stage/reference` 可以由聚合拥有者删除。跨领域引用则不得由数据库级联或仓储顺带删除：`Application -> Service`、`VersionComponent -> ServiceComponent`、`Application -> GatewayConfig`、`Version -> Gateway profile binding` 都必须改为拒绝父对象删除。

Gateway 是完整业务闭包：它依次移除 dashboard Route、runtime Service、Environment binding、GatewayConfig，再调用 Application 的正常删除能力移除自己的静态 Application。Application 仓储不再提供 `DeleteGatewayApplication` 或删除 GatewayConfig 的专用接口。旧的“把既有 Application 注册为 Gateway”半配置入口与其拆分删除入口一并移除，避免形成无法按完整生命周期管理的 Gateway。

该规则同样应用于非 CD 领域：Project 的废弃前检查分别由 Repository / Application 领域提供只读资源事实；User 删除前由 Project 应用服务删除 Project-owned membership；Role 删除由 User 领域报告是否仍有 user-role assignment，并在有引用时拒绝。SQLite / PostgreSQL 的 `user_role -> role` 从 `ON DELETE CASCADE` 迁为 restrict，MySQL 现行 schema 不存在该外键；Role 的权限关联仍属于 Role 聚合内部。

已发布的 SQLite / PostgreSQL schema 以新前向 migration 移除上述跨领域 `ON DELETE CASCADE`；MySQL 当前 schema 不存在这些外键，迁移保持为空操作。数据库约束在此只负责阻止错误删除，不用于表达导入顺序或实现导入事务脚本。

## Transfer package

接管文件采用 UTF-8 单 JSON 文档，建议 media type 为 `application/vnd.pomelo-orbit.project-handover+json`，文件名为 `<project-code>.orbit-project-handover.json`。它是 Application 层的业务包而不是领域行转储，首版固定为：

```json
{
  "format": "pomelo-orbit/project-handover",
  "version": 1,
  "project": { "name": "", "code": "", "is_active": true },
  "environment": {},
  "environment_credential": {},
  "applications": [],
  "gateway": {},
  "services": [],
  "routes": []
}
```

- `project.id` 和所有直接 `project_id` 均不在文件中；模式 A 生成目标 ID，模式 B 使用 UI 选择的目标 ID。
- `Application`、`Version`、`VersionComponent`、`Service`、`ServiceComponent`、Route、Environment、Environment credential 等源记录 ID 只保留在包内充当引用键。导入时既有领域命令生成目标记录 ID，Handover Application 维护映射后再传递 Gateway binding、Route Service binding 等目标引用；源记录 ID 不写入目标数据库。
- Application 按 Application 嵌套所有 Version 及 VersionComponent 子表；Service 嵌套 Service Env、Service Component 和所有 overlay 子表。Gateway 是自包含的 Gateway 定义，含其 Application / Version、runtime Service、dashboard Route、Config、profile bindings 与 Environment binding；这些记录不会在 `applications`、`services`、`routes` 中重复出现。
- SSH credential 在包中使用 `encrypted_private_key`，不携带明文 `private_key` 或数据库持久化密文。导出时以源实例 Fernet key 加密；导入时使用提交的 `decryption_key`，为空则使用目标实例 system key，解密后由 Environment 领域以目标实例 key 加密持久化。其他密钥、PEM 和 token 直接使用业务值。
- JSON decoder 必须验证 `format` 与仅支持的 `version`，拒绝未知或缺失的必要字段、重复 ID、跨聚合引用、循环 Version lineage、SSH credential / Environment revision 不匹配、Gateway / Route / Service 指向聚合外记录等不完整包。编码只产生上述当前版本格式，不提供 JSONL 或旧版本兼容。
- 导入 Application 在交给仓储前保留 Service status；Route `enabled` 继续按现有规则规范化为 `false`。

## Domain composition and replacement

导出与恢复的 CD 配置聚合精确包括：

1. Project 配置字段、Environment、Environment credential。
2. 全部 Project Application、完整 Version lineage、VersionComponent 及其 env / endpoint / mount / dependency / healthcheck / resource / tmpfs / ulimit / device 子表。
3. Gateway Application、GatewayConfig、`gateway_acme_profile_version` bindings、Gateway Service 和 dashboard Route。
4. 全部其他 Service、`service_env`、Service Component 与 env / mount / resource / endpoint overlay 子表。
5. 全部 Project Route，包含 managed target binding、HTTP/TCP 配置、HTTPS / ACME 字段、手工证书和私钥。

Handover Application 以一个请求 UoW 协调下列完整领域操作。它只传递已解析的资源定义和已建立的目标 ID 映射，不访问 SQL、表级删除接口或组装跨域 SQL：

1. Project 领域创建模式 A 的 Project 与当前用户成员关系，或更新模式 B 的 Project 业务字段。
2. Pipeline 领域在模式 B 判断既有 Pipeline、snapshot 与 run binding 是否引用待替换的 Application / Version；有引用时返回冲突，后续领域操作不执行。
3. 模式 B 的现有数据由其所属领域完整移除：普通 Route、普通 Service、Gateway（连同其拥有的记录）、普通 Application，随后由 Environment 领域替换自身和 credential。所有控制面移除均不触发 Docker、SSH、workspace、Traefik 或 Probe；接管用例不提供“清空 Project CD 数据”的调用。
4. 再按依赖顺序调用领域创建 / 更新行为：Project 元数据、Environment、普通 Application / Version、普通 Service、普通 Route、Gateway。每个行为只接收并持久化自身完整业务定义；源记录 ID 仅通过映射转换为新对象的关联值。
5. Deployment 领域清理目标 Project 的本地 Deployment 历史；它不属于导入包，且其输入会指向被替换的 CD 配置。

源 Deployment、Pipeline、Repository、Artifact、Dialogue 和其他控制面记录不进入包。模式 B 不删除 Pipeline / Repository 等非 CD 数据；Pipeline 领域发现其引用即拒绝替换，避免遗留失效引用。

## HTTP and UI contract

- 增加 `GET /api/project/:project_id/handover`。Handler 以当前用户检查源 Project 成员资格，Application 返回 JSON，Handler 设置 attachment response 和建议文件名。
- 增加 `POST /api/project/handover` 用于模式 A，及 `POST /api/project/:project_id/handover` 用于模式 B。请求为 `multipart/form-data`，含 `file`、`mode` 和可选 `decryption_key`；模式 B 的目标 Project ID 只从路径参数读取。Handler 读取受限文件内容并将当前认证用户、模式、目标 ID、解密 Key 和 JSON bytes 交给 Application；成功以现有 `ProjectResp` 返回目标 Project。
- 不为文件包新建 proto schema。它不是普通 JSON API DTO，而是可下载、可上传且有独立版本字段的业务文档；导入结果继续复用 Project protobuf response。
- `internal/bootstrap/http.go`、`routes.Dependencies` 和 `routes.Router` 注入并注册新的 Project Handover Application service / handler。新 POST 保持在默认 request UoW 内。
- `web/src/views/project/ProjectDetail.vue` 增加导出命令；`ProjectPage.vue` 增加导入命令和对话框。对话框使用模式选择控件，模式 B 显示当前用户 Project 列表作为目标选择，文件输入仅接受 JSON 包。成功后刷新 Project store、选择返回的 Project，并导航到其详情；错误在对话框内显示。
- `web/src/api/project/project.ts` 增加下载与 multipart 上传调用；补充中英文 i18n 文案。下载保留服务端 content-disposition 文件名，不在浏览器解析或改写 JSON 内容。

## Implementation steps

1. 先收敛既有领域服务的完整资源定义和生命周期，而不是沿导入链路逐项加接口：Environment 读取和保存 target / credential 且不 Probe；Application 读取和创建完整 Application / Version definition 并保留 lineage；Gateway 读取、创建与移除自己的完整业务闭包；Service 读取、创建与移除完整 runtime definition；Route 读取、创建与移除完整 ingress definition。普通交互式创建/删除改为复用这些领域行为并补充其运行时编排。同步审计所有跨领域删除和 schema cascade，改为由拥有完整删除流程的领域显式协调，SQL 仍放入所属 `sql/query/<domain>/`，只操作所属领域表。
2. 新增 `internal/application/project_handover/{dto,usecase}`。DTO 定义 Handover JSON package v1；usecase 组合领域定义生成包，严格编解码后按依赖顺序调用这些领域能力。它检查 Project / member 权限，调用 `common/crypto` 以源实例 key 加密包内 SSH 私钥，并以提交或 system key 解密后交由目标 Environment 持久化，维护源 ID 到目标 ID 的映射，保留 Service status、规范化 Route enabled，并依赖 request UoW 保证整体回滚。
3. 不新增 `ImportProject...`、`RestoreProject...`、`ReplaceProject...` 或按 Project 表集合读写的领域仓储 / SQL。Application、Gateway、Service、Route、Environment 各只收到并处理自身完整的业务对象；Pipeline 的引用检查与 Deployment 历史清理分别留在所属领域。
4. 新增 Project Handover HTTP handler 和路由，完成 multipart 解析、attachment 写出、现有认证器集成、服务注入和标准错误输出。为下载和上传端点建立 handler 测试，确保入口不调用 Docker、SSH、Traefik、Probe 或 workspace port。
5. 扩展 Web Project API、Project list / detail 页面和 i18n。导出使用下载响应，导入对话框本地校验 file、mode 和模式 B 的目标 Project；提交期间禁用重复操作，成功后按返回 Project 刷新状态并跳转。
6. 保持 `scripts/src/pomelo_orbit_cli/commands/database_transfer.py`、其 CLI 注册、测试、dbtalk 依赖和现有脚本文档不变。Project 接管文档只描述新的 Orbit API / UI，不把 Python JSONL 脚本作为其实现或兼容路径。

## Files to change

- 更新 `internal/repository/{project,environment,application,gateway,service,route,deployment,pipeline}.go` 及对应 SQLC implementation / query / generated code / 测试
- 新增 `internal/application/project_handover/dto/**`、`internal/application/project_handover/usecase/**` 及测试
- 更新 `internal/bootstrap/stores.go`、`internal/bootstrap/http.go`
- 新增 `internal/api/http/handler/project_handover/**`，更新 `internal/api/http/routes/{routes.go,project.go}`
- 更新 `web/src/api/project/project.ts`、`web/src/views/project/{ProjectPage.vue,ProjectDetail.vue}`、`web/src/i18n/locales/{zh-CN.ts,en-US.ts}`，以及相关前端测试
- 本轮不修改 `scripts/` 下的 `database-transfer` 实现、测试、CLI、依赖或文档；不修改 `docs/guides/service-data-transfer.md` 与 `docs/INDEX.md`

## Verification plan

1. 各领域 SQLite 集成测试先分别验证新增的保存、删除和解绑行为：它们只影响自身数据，复用现有模型校验，并且不触发 Probe、Docker、SSH、Traefik 或 workspace 行为。
2. Application 集成测试验证模式 A 创建新 Project 和当前用户成员、模式 B 通过领域操作替换目标 CD 数据但保留原成员、源 / 目标相同 Project ID 无影响、源记录 ID 不写入目标、SSH 私钥包加密、system key 与自定义解密 Key、Service status 原值保留、Route 强制 disabled。
3. 失败路径测试在包不完整、引用失效、format/version 不支持、模式 B 非成员、目标 Project 不存在、目标存在 CI 外部引用和仓储写入失败时，断言整个请求事务回滚，模式 A 不遗留 Project / member，模式 B 不遗留部分清理。
4. HTTP handler 测试导出权限 / attachment、multipart 模式字段校验、导入成功 response 与标准错误映射；通过 fake ports 或调用断言证明导入没有触发运行时副作用。
5. Web 测试覆盖导入模式切换、模式 B Project 选择、缺失文件本地校验、API 错误显示，以及成功后的 Project store 刷新和导航；检查导出动作调用正确 API。
6. 运行 `task sqlc`，`yarn --cwd web lint:fix`，`yarn --cwd web typecheck`，`task check`，`go test ./cmd/... ./internal/...`；本轮不因接管功能改动 Python scripts。
7. 对照活文档和最终 diff，确认新导入代码没有 Python transfer / dbtalk / JSONL 运行时依赖，现有脚本入口保持不变，未改动已执行 migration，且导入代码没有 Docker、SSH、workspace、Traefik、certificate 或 Probe 调用。

## Mode B protection

模式 B 不迁移 CI 数据，也不删除 Pipeline / Repository 等非 CD 数据。Pipeline 领域只要发现已有数据引用待替换的 Application / Version，就在任何 CD 清理前返回冲突。这样保持 Project CD 接管范围，又不会默默留下失效引用。

## Assumptions

- 当前 Orbit 的 SQLC 查询可用同一套 MySQL 语法生成并运行于受支持的 SQLite / MySQL / PostgreSQL 数据库；新增查询沿用这一模式。
- Project code 是 Project / Environment 的现有业务约束。模式 A 或模式 B 更新时若包内 code 与另一个 Project 冲突，导入失败并整体回滚，不重命名或推断替代 code。
- 导出包允许包含当前业务数据中的私钥和 token；文件权限、签名和跨实例身份校验不在本功能范围。
- 目标 Project 成员资格是覆盖导入的充分业务授权，与现有 Project API 一致；不新增一套独立 transfer role。

## Risks

- 聚合结构涉及多张逻辑 / 物理关联表。若 schema 新增了 Project CD 成员而 Snapshot 未同步，导入不会自动携带它；单一 Snapshot / repository 测试将作为变更检查点。
- 包的版本只支持当前 v1。后续 schema 演进必须显式新增 decoder / migration 策略，不能按未知字段或隐式默认值猜测。
- 目标 `local` Environment 会按目标 Orbit 进程宿主解释 workspace / Docker target；这是用户要求的原样导入语义，不做预检或转化。
- 覆盖不影响现存容器和 Traefik provider 状态。运维方必须保证源控制面已退出，且在明确接管前不让两个 Orbit 同时管理同一 deployment target。

## Rollback

本轮不为接管 API / UI 增加 JSONL 或 Python CLI compatibility path，但保持既有脚本入口不变。尚未发布时回退本轮 Go/Web 变更；已完成的导入由已存在的数据库备份与明确的后续业务操作恢复，不在导入 API 内加入补偿脚本。

## User review notes

- 用户明确要求此功能是应用业务能力而非脚本；导出、导入均应有 API 和 UI。
- 用户明确要求 Application 层拥有导入导出用例，领域层拥有查询和写入；不接受在 Application / Handler 中实现表级事务脚本。
- 用户明确要求不继续沿用 dbtalk JSONL，因为本功能不再是通用数据库传输。
- 用户确认模式 A 为新 Project，模式 B 为当前用户可访问的已有 Project；模式 A 将当前用户加入成员关系。
- 用户要求当前任务先保留 `database_transfer.py`，不删除其 CLI、依赖、测试或文档。
- 用户要求不以导入为名保留跨领域级联删除后门；此原则同样适用于其他领域，需借本次变更纠正历史实现。
