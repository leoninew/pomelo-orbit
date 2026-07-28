# Service 聚合运行时配置重构计划
最后修改时间: 2026-07-27 14:01:00

Review status: Accepted

## Plan basis

- Requirement: `docs/requirement/20260727-service-runtime-config-aggregate.md` (`Accepted`)
- Flow mode: 标准模式 / `standard`
- 现有 `RuntimeConfig` 的语义是 Version `env_json` 中 `${KEY}` 与 `${KEY:-default}` 占位符的 K/V 解析输入。该能力保留，但其来源从 HTTP Deploy 请求和 Credential 引用改为 Service 聚合，并在 Deployment 创建时快照。

## Overview

将 Service 的运行时配置建模为普通明文 K/V 属性，持久化在 `service.runtime_config_json`。Version 继续保存静态 Component 与环境变量规格；其中的占位符定义 Service 配置所需的 key，而不保存 Credential 或秘密引用。

Service 配置可以独立编辑，编辑后不会改变正在运行的容器。创建 Deploy 或 Restart 任务时，命令用例复制完整 Service 配置到既有 `deployment.options_json.runtime_config`；worker 只读取该快照，解析目标 Version 的占位符并渲染 Compose。任务创建后的 Service 配置变更不影响已排队或正在执行的任务。

移除 `runtime_env` Credential 类型、`VersionComponent.secret_env_refs`、组件 `.runtime/*.env` 文件与 Docker Compose `env_file` 注入。Credential 领域保持其他类型的独立用途，不再被持续部署读取或引用。

## Design decisions

### Service runtime config

1. 在 `service` 增加 `runtime_config_json`，语义为 JSON object `map[string]string`。空对象是合法配置；key 不为空，value 可以为空字符串。
2. `model.Service` 持有 `RuntimeConfig map[string]string`；Repository 在所有 Service 查询、插入与更新中编解码它。Service 配置修改使用专用替换操作，不能借由部署状态更新覆盖配置。
3. 配置明文保存、明文返回。它不是 Credential 或秘密，不添加加密、Credential 引用、脱敏字段或秘密专用权限；Service 仍沿用既有项目成员访问校验。
4. GET 响应对 K/V 以 key 升序返回或由 Web 进行稳定排序，避免 map 顺序造成界面和测试不稳定。

### Version matching and warnings

1. 复用 `collectPlaceholders`、`${KEY}`、`${KEY:-default}` 的现有语义。没有 default 的 placeholder 是必需 key；带 default 的 key 可以缺失。
2. 保存 Version 的 Components 或 `env_json` 前，找出当前 `version_id` 指向该 Version 的所有 Service，并逐一校验其配置。缺少必需 key 时拒绝保存，错误列出 Service 和缺失 key。
3. Service 配置中没有被目标 Version 使用的 key 不阻止 Version 修改，也不被删除。部署解析时这些 key 被忽略，worker 在 Deployment 日志写入稳定、可检索的 warning。
4. 发起部署选择不同 Version 时，对该 Service 的现有配置执行同一匹配校验；缺 key 时任务不创建，避免将无效任务放进队列。

### Deployment snapshot

1. HTTP `ApplicationDeployReq` 不再接收 `runtime_config`，改为以 `service_id` 识别被部署的 Service；`version_id` 表示本次目标静态规格。
2. Command usecase 在创建 Deploy 或 Restart Deployment 前读取 Service 的完整 `RuntimeConfig`，深拷贝到 `DeployOptionsJSON.RuntimeConfig`，再持久化到现有 `deployment.options_json`。
3. Worker 不读取最新 Service 配置；它仅解析 `options_json.runtime_config`。对快照缺少必需 key、格式不合法或与任务 Version 不匹配的情况，任务标记 faulted。
4. Deployment 的当前外部详情接口不新增配置回显；快照是任务执行记录而非新的 Service 配置读模型。

### Service lifecycle and Web workflow

1. 增加 Service 创建用例与 `POST /api/service`：指定 Application、Version、instance key 和初始 K/V 配置，创建状态为 `stopped` 的 Service，不触发 Docker 操作。这样含必需 placeholder 的 Service 可在首次部署前完成配置。
2. 增加 `GET` / `PUT /api/service/:service_id/runtime-config`：前者读取当前 Service K/V，后者整体替换 K/V 并按当前 Version 校验；保存不创建 Deployment、不重启容器。
3. Service 详情将现有“运行时环境变量”卡片替换为“运行时配置”：显示并编辑 K/V，标示“保存后需重新部署才生效”，不显示 Credential、data key 或 component secret reference。
4. Service 列表增加创建 Service 的入口。Application/Version/Service/Gateway 的部署对话框改为选择已有 Service；提交时只传 `service_id`、`version_id` 与 `force_recreate`，不再传 K/V。不存在可选 Service 时，引导先创建并保存配置。

## Implementation steps

### Step 1: Migration, model, repository, and generated SQL

1. 按用户明确授权直接修改 SQLite/MySQL 的既有 `000023_application` 与 `000026_service` 建表迁移：新库不再创建 `version_component_secret_env_ref`，`service` 直接包含非空 `runtime_config_json`。同步运行中的 SQLite `orbit` 库时，事务重建 `service`、为既有 Service 回填 `{}`、删除旧引用表与 `credential.type = 'runtime_env'` 遗留行；不从 Credential 回填 Service 配置。
2. 扩展 `model.Service` 与 `ServiceListItem` 所需的配置模型，更新 `sql/query/service/service.sql`、SQLC generated files 和 SQLC Service Repository，使配置在 Service 读写时完整保留。
3. 从 `model.VersionComponent`、Application DTO、HTTP mapper、Proto 及 SQL queries 中移除 `SecretEnvRefs`。删除 Credential Repository 中仅为该关系存在的引用查询、删除限制和相关生成物。
4. 从 Credential usecase 与 Web Credential 页面移除 `runtime_env` 类型、K/V 编辑器、导入/导出分支和相关测试；其余 Credential 类型的行为保持不变。

### Step 2: Service aggregate commands and read model

1. 在 Service DTO/usecase 中定义 Service 创建、运行时配置读取和完整替换输入；校验 application/version/project 关系、instance key、K/V JSON 形状和当前 Version 的 required placeholders。
2. Service 配置替换只写 `runtime_config_json` 与 `updated_at`，不修改 status、Version、last successful Version 或 Deployment。
3. 扩展 Service Proto、handler、route、mapper 与 TypeScript 生成契约，提供创建 Service 和 K/V runtime-config 的 GET/PUT 接口。删除旧 `GET /api/service/:service_id/runtime-env` 接口和其 Credential 字段响应。
4. 所有 Service 读写仍使用现有 Project membership 授权；不引入 Credential 查询、解密依赖或 secret key 配置。

### Step 3: Version validation

1. 将 placeholder 解析和 Service 配置匹配抽为可供 Application 与 Deployment usecase 调用的纯函数，返回缺失 key 与多余 key，避免两处语义漂移。
2. 为 Application Version usecase 注入只读 Service Repository 依赖。创建/更新 Version 时，对已绑定到该 Version 的 Service 执行匹配；缺失 key 返回 validation error，多余 key 使用结构化 logger warning 记录 Service、Version 与排序后的 key。
3. 更新 Version Component 的输入、响应、validation 与 Web runtime dialog，删除 Credential 选择器及 `secret_env_refs` 表单；Version 编辑只维护静态环境变量以及 placeholder 文本。

### Step 4: Deployment and render

1. Command usecase 以 Service ID 定位聚合，验证该 Service 属于目标 Application；以 Service RuntimeConfig 校验目标 Version，深拷贝完整 map 到 Deployment options 后才写 Deployment/派发任务。
2. Restart 任务同样快照 Service RuntimeConfig。移除 HTTP/Digital DTO 中由调用方传入 RuntimeConfig 的路径。
3. Execution usecase 从 options snapshot 获取配置，解析 placeholder 后传给 `RenderComposeDetailed`。移除 Credential Store、secret key、`materializeRuntimeEnvFiles`、`WriteRuntimeEnv`、`.runtime` 路径、`env_file` renderer 字段与 secret-redacting writer 的运行时依赖。
4. 保留 Render 中已有的 environment placeholder 代换。将多余快照 key 的排序 warning 写入该 Deployment 的日志；不把它们注入 Compose。

### Step 5: Web integration

1. 在 Service 页面实现创建 Service 对话框，使用 Application 与 Version 选择、instance key 和可编辑 K/V 表；失败时展示 Version 匹配的缺失 key。
2. 在 Service 详情实现 K/V 配置表和编辑对话框，支持增加、修改、删除、保存、加载、空态和错误态；保存完成后保留“尚未部署”的语义提示。
3. 删除 Credential 页面 `runtime_env` UI 和 Version Component runtime dialog 的 Credential/secret-ref 区域；清理 i18n、工具函数和前端测试。
4. 修改所有部署入口：从已有 Service 选择目标，发送 Service ID，移除 `runtime_config: {}`。部署提交后继续跳转 Deployment 详情，配置更新不会自动触发该操作。
5. MCP 删除 `runtime_env` Credential 工具，新增 Service 创建与配置 K/V 工具；Deploy、Stop、Restart 与状态/日志目标均以 Service ID 定位。RAGFlow desired-state fixture 用 `${KEY}` 表达 Version 占位符，初始化脚本从仓库外的普通 K/V 配置文件创建 Service 后再发起部署。

### Step 6: Regression coverage and documentation

1. 更新 Service、Application/Version、Deployment 和 Credential 的 usecase/repository/handler 测试，覆盖 K/V 持久化、服务隔离、版本匹配、过量 key warning、创建与更新授权、任务快照和 worker 不读取新配置。
2. 更新 Web 单元测试，覆盖 K/V 编辑、Service 创建、详情读写、删除 Credential `runtime_env` 入口，以及部署 payload 不含 K/V。
3. 回写 `docs/product/cd-model.md` 与 `docs/architecture/cd-runtime.md`，将 RuntimeConfig 的 SoT、部署快照和 Version placeholder 匹配写为活文档事实；不修改归档文档。

## Files to change

### Data and backend

- `sql/migration/{sqlite,mysql}/000023_application.{up,down}.sql`
- `sql/migration/{sqlite,mysql}/000026_service.up.sql`
- `sql/query/{service,application,credential}/*.sql` 与 `internal/gen/sqlc/{service,application,credential}/**`
- `internal/model/{service,application}.go`
- `internal/repository/{service,application,credential}.go`
- `internal/repository/impl/sqlc/{service,application,credential}/repository.go`
- `internal/application/service/{dto,usecase}/**`
- `internal/application/application/{dto,usecase}/**`
- `internal/application/deployment/{dto,port,usecase}/**`
- `internal/infrastructure/storage/local/deploymentworkspace/**`
- `internal/bootstrap/app.go` 及受影响的构造注入
- `proto/orbit/v1/{service,application}/{service,application,version}.proto`
- `internal/gen/proto/orbit/v1/**` 与 Web TypeScript 生成契约
- `internal/api/http/{handler,routes}/**`

### Web

- `web/src/api/{application,service}/**`
- `web/src/views/service/{ServicePage,ServiceDetail}.vue`
- `web/src/views/{application/VersionDetail.vue,application/components/VersionComponentRuntimeDialog.vue,credential/CredentialPage.vue}`
- `web/src/utils/runtimeEnv.ts` 及引用它的测试（删除）
- `web/src/i18n/locales/{zh-CN,en-US}.ts`
- 相关 Vue/Vitest 测试
- `mcp/src/pomelo_orbit_mcp/{orbit_client.py,tools/orbit.py}` 与相关 MCP 测试
- `scripts/{ragflow_initialize.py,test_ragflow_initialize.py}`
- `docs/guides/ragflow-split-deployment.fixture.yaml`

### Documentation

- `docs/product/cd-model.md`
- `docs/architecture/cd-runtime.md`
- `docs/requirement/20260727-service-runtime-config-aggregate.md`
- `docs/plan/20260727-service-runtime-config-aggregate.md`

## Verification plan

1. Schema integration：新 SQLite/MySQL 建库后 Service 获得空 K/V 配置；运行中的 SQLite `orbit` 已重建 `service` 并保留 7 条 Service，旧 secret-ref 表与 runtime_env Credential 数据已删除；SQLC 生成文件一致。
2. Service usecase/HTTP：创建、读取、替换 K/V；无权限、错误 Version、缺失必需 placeholder 和错误 instance key 的错误契约；保存配置不创建 Deployment。
3. Version usecase：修改关联 Version 时，缺 key 的 Service 使更新失败；多余 key 被保留并发出稳定 warning；无关联 Service 时正常保存。
4. Deployment usecase：Deploy/Restart 将 Service config 完整快照到 options；随后修改 Service config，worker 仍只用旧快照；多余 key 被忽略并记录到该 Deployment 日志；运行路径未调用 Credential/解密/`.runtime` 文件。
5. Web：Service K/V 创建和编辑、提示文本、部署入口 Service 选择、Version/ Credential 旧入口移除，以及 TypeScript payload 契约。
6. 运行项目约束的 `go fmt ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`、`yarn --cwd web test`、`git diff --check`；按项目生成入口运行 Proto 与 SQLC 生成检查。
7. MCP 执行 `uv run pytest`、`uv run ruff check src tests`、`uv run mypy`；RAGFlow 初始化脚本执行 `python -m unittest scripts/test_ragflow_initialize.py`。

## Blockers

无。现有数据中的 `runtime_env` Credential 与 secret references 会按本计划删除而非迁移，属于已被确认的错误模型清理。

## Assumptions

1. Service K/V 配置绑定 Version 的 `${KEY}` placeholder 名称，不直接成为所有 Component 的无条件 environment 条目。
2. 值允许为空字符串；是否必需由 Version placeholder 的 required/default 语义决定。
3. 配置是普通业务配置，正常 Service 项目访问校验仍然存在，但没有加密、Credential、脱敏或秘密专用管理逻辑。
4. 新 Service 创建为 `stopped`，直到用户显式部署；这使首次部署前的配置成为可能。

## Risks

1. `runtime_config_json` 以明文保存且 Deployment 也保存快照；这是用户确认的语义，后续若配置包含秘密，需要创建不同的秘密配置模型，而不能重新连接 Credential。
2. 删除运行时 Credential 与引用表不可恢复旧配置；用户明确授权该一次性清理，已在运行库中以事务执行并进行外键完整性检查。
3. Service 创建和部署入口从 instance key 转向 Service ID，会改变 Web 工作流与 HTTP 契约；本项目处于活跃开发期，不保留旧字段兼容层。
4. Version 更新需要读取关联 Service 配置；实现必须避免 Application 与 Service usecase 的循环依赖，仅通过 Repository port 和纯校验函数协作。

## Rollback

1. 代码层可回退到本次改造前版本；但已清理的 legacy runtime_env Credential 与 reference 数据不能恢复其具体值。
2. 本任务由用户授权直接修改既有迁移并同步当前运行库；若需恢复旧数据库结构，必须由备份恢复，而不是再次创建兼容链路。

## User review notes

用户于 2026-07-27 要求从已接受 Requirement 进入 Plan / 计划阶段，并确认配置修改的生效时机、Deployment 快照、Version 匹配以及普通 K/V 明文语义。
