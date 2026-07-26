# 部署运行时能力 Web 支持计划
最后修改时间: 2026-07-26 21:47:02

## Review status

Draft

## Plan basis

- Requirement: `docs/requirement/20260726-deployment-runtime-web-capability.md` (`Accepted`)
- Flow mode: 标准模式 / `standard`
- RAGFlow 是新增能力的真实验收样本，不是本计划的业务功能或资源创建目标。

## Overview

本计划将已存在的部署运行时领域能力接入既有 Web 管理页面。它覆盖 `runtime_env` Credential、Component 的秘密环境变量引用和高级运行时字段，以及 Service 当前绑定 Version 的运行时环境变量展示。

Service 的运行时环境变量卡片展示的是当前持久化 Credential 解析出的“下次部署配置”，不是从运行中容器读取的环境快照。Deployment 与 Service 记录继续承担部署结果、状态、错误和日志的展示；MCP `runtime_doctor`、`verify_deployment` 和 HTTP Probe 继续作为 MCP 验收工具，不新增浏览器调用路径。

## Interface design

新增只读 HTTP 接口：`GET /api/service/:service_id/runtime-env`。

响应只描述 Service 当前 `version_id` 的 Component `secret_env_refs` 解析结果，每一项包含：

- `component_name`
- `env_key`
- `credential_id`
- `credential_name`
- `data_key`
- `value`

响应还应返回 `service_id` 与 `version_id`，以便前端确认卡片关联的配置版本。无引用时返回空 items；不存在、无权限、跨 Project Credential、缺失 data key 或解密失败沿用既有 HTTP 错误契约。该接口不读取 Docker、workspace、MCP 或 Deployment 日志。

## Implementation steps

### Step 0: 并行修改协调

1. 实施前重新检查工作区。当前已有其他工作涉及 Application handler、Application routes、Version usecase、前端 Application/Version/Service 页面及导航。
2. 对重叠文件先读取实际未提交修改并与拥有者协调；不重置、回退或覆盖他人的工作。
3. 本任务只新增或扩展与 Requirement 直接相关的运行时配置展示能力，不接管 Environment 移除、MCP、RAGFlow 初始化器或 Gateway lifecycle 的实现。

### Step 1: Service 运行时环境读模型

1. 扩展 `proto/orbit/v1/service/service.proto`，定义 Service runtime env 响应及条目消息；按项目既有生成方式更新 Go protobuf 与 TypeScript 生成文件。
2. 在 Service usecase 增加授权后的查询：读取 Service、确认 Project 成员关系、加载当前 Version Components 和 `secret_env_refs`，解析同 Project 的 `runtime_env` Credential JSON 值，组装稳定排序的结果。
3. Service usecase 的构造依赖增加读取 Version Component、Credential 与解密所需的已有受管依赖；复用既有 Credential/Version 校验规则，不复制 Docker 或 MCP 逻辑。
4. 增加 handler、route、response mapper 与 `web/src/api/service/service.ts` 方法。接口只由 Service ID 定位，不能接收任意 Credential ID、Component、Docker target 或路径。
5. 由 Service 详情的当前 `version_id` 解析当前持久化配置。Credential 更新且尚未重新部署时，响应可立即反映新值；前端负责标明这是下次部署将使用的配置。

### Step 2: Credential 的 `runtime_env` 用户体验

1. 将 `runtime_env` 加入 Credential 创建和导入的类型选项；保留其他既有 Credential 类型与行为。
2. 为 `runtime_env` 实现键值行编辑器，校验非空、合法环境变量键、重复键与非空值，并在提交时序列化为后端要求的 JSON object。
3. 在 Credential 详情按键值表展示 `runtime_env` data，并保留查看、编辑、整体替换与导出能力；非 `runtime_env` Credential 保持现有文本体验。
4. 更新中英文 i18n 文案、空态、错误提示和表单提示，避免向用户暴露 JSON 编码细节。

### Step 3: Version Component 高级运行时配置

1. 在 `VersionDetail` 添加 Component 高级配置入口和摘要，使已存在的 `command_json`、`args_json`、`healthcheck_json`、`resources_json`、`restart_policy`、`tmpfs_json` 与 `ulimits_json` 对已发布和未发布 Version 都可审阅。
2. 新增或扩展专用 Dialog，以结构化控件编辑上述字段；复用现有 Component 基础、环境变量、端口和挂载 Dialog，不引入任意 Compose YAML 或自由字段透传。
3. 添加秘密环境变量引用编辑区：从当前 Project 的 `runtime_env` Credentials 中选择 Credential 和 data key，填写目标 `env_key`；将结果写入现有 `secret_env_refs` 请求字段。
4. 仅允许编辑 `unpublished` Version，继续使用后端对 Project 归属、data key 存在性、重复 env key、运行时字段结构与白名单的校验。发布后只展示配置与引用。
5. 保留既有 Component payload 的未知/未修改字段，避免从任一 Dialog 保存时静默丢失已有 mounts、env、ports、exposes 或高级运行时配置。

### Step 4: Service 详情运行时环境变量卡片

1. 在 `ServiceDetail.vue` 的基本信息与容器状态之间增加“运行时环境变量”卡片，加载 Step 1 接口。
2. 卡片以稳定表格展示 Component、目标环境变量、Credential 名称、data key 与当前值；Credential 名称链接到详情页，空列表、加载和错误分别有明确状态。
3. 在卡片中展示“当前持久化配置，供下一次部署使用”的说明。Credential 更新后未重新部署时，不把卡片值描述为容器当前进程环境。
4. Service 部署、停止、刷新和日志流程保持现状；卡片刷新只重新读取业务 HTTP 数据，不执行 Docker、MCP 或额外验证操作。

### Step 5: 前端整合与回归

1. 补齐 Service、Credential、Version 页面在窄屏和长值场景下的布局，确保环境变量键和值可换行、表格可横向查看且操作按钮不溢出。
2. 维持既有 Application、Version、Service、Gateway 与 Deployment 路由，不添加 RAGFlow 专用路由、fixture 导入入口或批量资源创建。
3. 确认应用/版本预览、发布、部署以及 Service 从当前 Version 发起部署仍走现有 Lifecycle API；Deployment 表单不增加 `runtime_env` 值输入。

## Files to change

### Backend and contract

- `proto/orbit/v1/service/service.proto`
- 对应生成的 `internal/gen/proto/orbit/v1/service/service.pb.go`
- 对应生成的 `web/src/gen/proto/orbit/v1/service/service.ts`
- `internal/application/service/usecase/service.go`，或同目录新增运行时环境查询文件
- `internal/application/service/dto/service.go`
- `internal/bootstrap/app.go` 与相关依赖装配文件
- `internal/api/http/handler/service/service.go`
- `internal/api/http/handler/service/service_mapper.go`
- `internal/api/http/routes/service.go`
- Service usecase / handler 的新增或扩展测试

### Web

- `web/src/api/service/service.ts`
- `web/src/views/service/ServiceDetail.vue`
- `web/src/views/credential/CredentialPage.vue`
- `web/src/views/credential/CredentialDetail.vue`
- `web/src/views/application/VersionDetail.vue`
- `web/src/views/application/components/` 下新增或扩展的高级运行时配置与秘密引用 Dialog
- `web/src/i18n/locales/en-US.ts`
- `web/src/i18n/locales/zh-CN.ts`
- 相关前端工具和测试文件

## Verification plan

1. Go usecase / handler 测试：有权限用户能读取 Service 当前 Version 的引用和值；无引用返回空；无权限、缺失 Credential、跨 Project Credential、缺失 data key 与无效加密数据按既有错误契约失败。
2. Go integration 测试：Credential 更新后的 Service runtime env 查询返回新值；Service 仍绑定相同 Version，随后部署使用同一引用，无须新增 Deployment 参数。
3. 前端单元测试：`runtime_env` 键值编辑的解析、校验和序列化；高级 Component 字段及 `secret_env_refs` 的 payload 保留；Service 卡片的加载、空态、错误态、值展示和“下次部署配置”提示。
4. 前端类型检查、lint、format 与测试使用 `web/package.json` 的 `typecheck`、`lint`、`format`、`test` 入口。
5. 后端运行相关 package 的 `go test`，并按项目 Proto 生成入口重新生成后检查生成文件一致性。
6. 本机人工回归：创建 `runtime_env` Credential，绑定到未发布 Version，发布并从 Service 部署；确认 Service 卡片展示值，轮换值后确认卡片更新并在重新部署后复用，无需再次输入。

## Blockers

无功能前置阻塞。实施前须协调当前工作区中 Application、Version、Service、Route 与导航文件的并行修改归属。

## Assumptions

1. 用户已明确允许 Web 查看、编辑和导出已持久化的 `runtime_env` 值。
2. Service 卡片展示当前持久化配置，而不是从运行中容器读取的环境快照。
3. `runtime_env` 值继续持久化在 Credential；Version、Service 和 Deployment 不复制这些值。
4. MCP runtime diagnostics 和 RAGFlow 初始化器保持现有职责，不增加 Web 驱动路径。

## Risks

1. Credential 轮换与实际容器环境存在部署前窗口；卡片文案若不明确，会误导用户把新 Credential 值当作运行中进程的当前值。
2. Version Component 的高级字段采用 JSON 结构，前端编辑器必须在局部保存时保留其他字段，避免误删已配置的运行时约束。
3. 新 Service 查询会读取并返回 Credential value；用户已授权该行为，但实现不得将值写入 Deployment、Version、浏览器控制台或服务端日志。
4. Proto 契约与其 Go/TypeScript 生成文件不同步会导致前后端接口编译或运行失败。
5. 并行开发可能同时触及 Version、Service、路由和 i18n 文件，合并时必须保留双方行为。

## Rollback

1. 本计划不迁移或重写已有 Credential、Version、Service、Deployment 数据；新增 Service 读接口和 UI 可独立回退。
2. 发生前端问题时，移除高级配置入口和 Service 卡片不会改变已有 Version 或 Credential 的持久化值和部署行为。
3. 发生接口问题时，禁用新增 Service runtime env endpoint 不影响既有 Service 查询、部署、停止和日志接口。

## User review notes

用户于 2026-07-26 要求从已接受的 Requirement 进入 Plan / 计划阶段。本 Plan 不授权开始产品代码实现；等待用户接受后进入 Implementation / 实现。
