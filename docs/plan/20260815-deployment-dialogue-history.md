# 持续部署对话记录持久化计划
最后修改时间: 2026-08-15 20:06:00

Review status: Accepted

流程模式: 标准 / standard

## 需求依据

实施已接受的[持续部署对话记录持久化](../requirement/20260815-deployment-dialogue-history.md)。持续部署对话改为项目级 Conversation 与 Message 的持久化模型，通过 RBAC 与项目成员关系控制访问。只保存最终问答；既有工具调用和中间 SSE 事件继续按当前契约实时传输，但不写入记录，也不在历史回放中展示。

## 实施设计

```text
持续部署对话页
  -> Conversation HTTP API + 当前用户 RBAC
  -> dialogue usecase
       -> 读取 Conversation / Message 历史
       -> LLM + MCP tool-calling loop
       -> 获得最终回答后，原子写入 Conversation（首次）及本轮两条 Message
  -> SQLite / MySQL
```

### 数据与权限

1. 新建 `deployment_dialogue_conversation`：保存项目 ID、自动标题、创建者、创建/更新时间；新建 `deployment_dialogue_message`：保存 conversation ID、`user` / `assistant` 角色、内容和创建时间。删除 Conversation 级联删除 Message。
2. 对话属于项目，不按创建者私有过滤。读取和变更均先验证当前用户是项目成员，再分别要求 `dialogue:read` 或 `dialogue:write`。
3. 新增可执行的数据迁移注册 `dialogue:read`、`dialogue:write`，并赋予系统管理员角色；不修改已有种子迁移。
4. 首轮请求在最终回答产生后，于同一事务创建 Conversation、从首条用户问题生成标题，并写入本轮用户/助手两条 Message。后续轮在同一事务追加两条 Message 并更新 Conversation 更新时间。失败不会产生 Message 或伪造助手回答。

### API 与运行流程

1. 保持现有 `DeploymentDialogueTurnReq.messages`、`/turn/stream` SSE 与工具调用事件，以及既有 Turn 与 `complete` 响应字段。Turn 请求仅新增可选 Conversation ID，客户端在新对话开始时生成该 ID，服务端在最终回答成功后使用它创建 Conversation。
2. 提供单数资源路由：列出项目 Conversation、读取一个 Conversation 及其消息、删除 Conversation。路由均使用当前认证用户，不能由客户端传递用户身份。
3. dialogue usecase 保持现有页面消息历史到 LLM 上下文的输入方式与 MCP tool-calling loop。工具事件继续通过 SSE 传输，但不持久化到 Conversation/Message。
4. 首轮没有 Conversation ID 时，usecase 在最终回答成功后创建记录；已有 Conversation ID 时，usecase 验证其归属项目并继续写入。标题取首条用户问题的规范化截断文本，不调用 LLM 生成。

### 页面行为

1. 对话历史是当前工作区的导航，而不是独立路由、数据表格或业务记录列表。桌面端在消息流左侧提供稳定宽度、可收起的 Conversation 侧栏；移动端将同一导航放入既有共享抽屉。
2. 侧栏顶部仅提供“新对话”图标命令，主体按时间分组展示会话自动标题与相对更新时间。选择标题切换右侧完整消息流；选择项目时清空旧项目的选择和消息。
3. 会话删除不作为每项常驻按钮或操作列展示。用户在当前会话标题的更多操作菜单中选择删除，经共享确认对话框确认后删除；随后选择相邻历史或回到空白新对话状态。
4. 页面继续订阅现有 SSE、显示现有流式状态并禁用重复提交；收到最终 `complete` 事件后更新侧栏和消息。Conversation 持久化不包含工具名称、输入、输出或推理过程。
5. 所有日期显示复用 `web/src/utils/time.ts`，操作反馈复用既有 toast 和共享 UI 组件。

## 实施步骤

1. 增加结构迁移与 RBAC 数据迁移。
   - 在 `sql/migration/{sqlite,mysql}/` 新增未执行的迁移，创建 Conversation、Message、外键、级联删除和按项目/更新时间、按会话/创建时间的查询索引。
   - 在 `sql/migration/data/{sqlite,mysql}/` 新增未执行的数据迁移，创建 `dialogue:read`、`dialogue:write` 并向管理员角色授予权限。
   - 更新 `sql/schema/{sqlite,mysql}.sql`、`sql/query/dialogue/` 和对应 MySQL 查询，执行 sqlc 生成。

2. 建立模型、Repository 和 application usecase。
   - 在 `internal/model/`、`internal/repository/` 与 `internal/repository/impl/sqlc/dialogue/` 增加 Conversation/Message 的模型、读写接口和 sqlc 实现。
   - 扩展 `internal/application/dialogue/`：校验项目成员身份、按 ID 读取会话、列表读取、删除，以及最终回答后的原子持久化。
   - 将 title 生成、角色/内容校验、最终回答后写入与数据库错误处理放在 application 边界；Repository 不依赖 LLM、MCP 或 HTTP。

3. 收敛 HTTP/proto 到最终回答契约。
   - 更新 `proto/orbit/v1/dialogue/dialogue.proto`，增加 Conversation 详情、列表和可选 Conversation ID；不改变既有 Turn、ToolCall 或 SSE event 字段与事件类型。
   - 重新生成 Go 与 TypeScript proto 类型。
   - 更新 `internal/api/http/handler/dialogue/`、`internal/api/http/routes/dialogue.go` 及 bootstrap 装配。读取接口要求 `dialogue:read`；Turn 和删除要求 `dialogue:write`；所有接口同时经项目成员校验。
   - 保留现有流式 Turn HTTP 路由、SSE 编解码和前端 stream client；在最终完成前写入消息，流结束后由前端通过历史读取接口刷新导航。

4. 改造持续部署对话页。
   - 更新 `web/src/api/dialogue/dialogue.ts`、`web/src/stores/deploymentDialogue.ts` 与 `DeploymentDialoguePage.vue`，以 Conversation ID 驱动读取、发送和删除。
   - 在当前页面实现可收起的历史导航侧栏和移动抽屉；新建使用图标命令，删除通过会话上下文菜单和共享确认对话框完成，避免新增 CRUD 列表页。
   - 更新 i18n 文案，并保证工具活动和工具结果不进入持久化历史。

5. 覆盖关键行为并验证。
   - 为 Repository 覆盖创建、按项目列出、读取消息顺序和删除级联。
   - 为 dialogue usecase 覆盖新建首轮、继续会话、失败不写入、最终回答后原子写入、标题生成、越权/非成员拒绝和删除。
   - 为 handler 覆盖 `dialogue:read` / `dialogue:write` 以及项目成员检查；为前端覆盖历史载入、选择、删除和普通最终回答发送。

## 预期文件

| 路径 | 计划改动 |
| --- | --- |
| `sql/migration/{sqlite,mysql}/000030_deployment_dialogue_history.*` | Conversation/Message 结构迁移及回滚。 |
| `sql/migration/data/{sqlite,mysql}/000030_deployment_dialogue_permission.*` | 新增对话读写权限与管理员授权。 |
| `sql/schema/**`、`sql/query/{dialogue,mysql/dialogue}/`、`internal/gen/sqlc/**` | schema、查询与 sqlc 生成物。 |
| `internal/model/dialogue.go`、`internal/repository/dialogue.go`、`internal/repository/impl/sqlc/dialogue/` | 持久化模型与存储实现。 |
| `internal/application/dialogue/**` | 会话查询、RBAC/成员校验、最终回答后写入。 |
| `proto/orbit/v1/dialogue/dialogue.proto`、`internal/gen/proto/**`、`web/src/gen/proto/**` | 对话历史读取契约与可选 Conversation ID 生成物，不改变既有 SSE 字段。 |
| `internal/api/http/handler/dialogue/**`、`internal/api/http/routes/dialogue.go`、`internal/bootstrap/{http,stores}.go` | HTTP 路由、权限检查与依赖装配。 |
| `web/src/api/dialogue/dialogue.ts`、`web/src/stores/deploymentDialogue.ts`、`web/src/views/deployment/DeploymentDialoguePage.vue`、`web/src/i18n/locales/**` | 对话记录浏览、普通请求和删除交互。 |

## 验证计划

1. 运行 `./bin/sqlc generate`、`./bin/buf generate` 和 `./bin/buf generate . --template web/buf.gen.yaml --output web`，确认生成物与手写契约一致。
2. 运行相关 Go 单元/集成测试，随后执行 `go fmt ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...` 与 `go test ./cmd/... ./internal/...`。
3. 执行 `yarn --cwd web lint:fix` 与 `yarn --cwd web typecheck`，并运行受影响的前端测试。
4. 手工核对：无 `dialogue:read` / `dialogue:write`、无项目成员关系、删除后读取、模型失败，以及跨项目切换均符合预期。

## 风险与回滚

1. 新权限若未分配给自定义角色，将按设计拒绝对话访问；管理员迁移仅覆盖内置管理员角色。
2. 回答后才持久化使写入故障可导致页面已得到回答但历史未保存；API 必须将该故障返回为本轮失败，不把回答视为持久化成功。
3. 结构迁移回滚仅删除新增对话表与新增权限数据；不影响部署、MCP 或既有对话页面之外的数据。

## User review notes

1. 用户要求以 SpecFlow 先完成过程文档，未接受前不进入代码实现。
2. 用户确认 Conversation + Message 的多轮记录模型、RBAC 控制、自动标题和删除能力。
3. 用户要求省略思考过程，并且仅在大模型给出最终回答后更新记录。
4. 用户要求历史作为对话工作区导航呈现，不能采用普通业务列表的展示和删除方式。
