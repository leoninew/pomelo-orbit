# Environment SSH 宿主机终端实施计划
最后修改时间: 2026-09-30 12:02:59

Review status: Accepted

Mode: standard

## Intent basis

依据 [已接受的意图](../intent/20260930-environment-ssh-terminal.md)：当前 Project 的 SSH Environment 保存唯一的远端 `username` 和受管密钥。环境页终端必须以这个用户连接 Linux/Windows SSH 目标宿主机；Orbit 用户只用于现有的登录与 Project 成员校验以及会话元信息。仅在当前修订 Probe 成功后开放完整交互式 shell；`local` Environment、容器终端及另选远端用户均不在本期范围。

## Implementation steps

### 1. 明确会话契约和认证

- 在单数资源路径下增加 `POST /api/environment/terminal/ticket` 和 `GET /api/environment/terminal`（WebSocket Upgrade），均按 `project_id` 选择当前 Project。前者沿用 Bearer 登录鉴权与 Project 成员校验，调用现有 TargetResolver 确认当前修订 Probe、受管凭据与 pinned host key 均就绪；只对 SSH target 签发短时、一次性票据。
- 票据在单节点进程内保存，绑定 Orbit 用户、Project、Environment ID、target revision 和 SSH credential ID/revision；兑换时原子消费并重新核对身份、成员资格与目标状态。浏览器通过 `Sec-WebSocket-Protocol` 传递票据，服务端只协商固定协议名；票据和 Bearer token 均不进入 URL。
- 定义 WebSocket 帧协议：二进制帧承载终端输入/输出原始字节；文本 JSON 帧承载初始行列数、resize、退出状态及可显示错误。前端流式解码输出，Windows shell 的实际字符编码须单独验证。限制消息大小、行列范围和无效消息行为。Upgrade 前按当前页面源与已配置跨域源校验 `Origin`，不能把现有 CORS middleware 当作 WebSocket Origin 授权。
- 票据响应必须绕过现有通用 HTTP 响应正文日志；WebSocket 帧、票据及认证头不得写入请求日志。票据签发属于无数据库写入的短操作，不占用长请求事务；WebSocket 会话始终在事务外运行。

### 2. SSH PTY 与会话生命周期

- 在环境应用层增加终端用例和窄端口，在现有 SSH runner 中扩展交互会话适配器。复用 TargetResolver 解出的受管私钥，以及现有 SSH dial、严格 host-key callback；不把泛用 shell 能力加到部署或 CI runtime 接口。
- 以 Environment 保存的 `host:port` 和 `username` 打开 SSH session，申请 PTY 后启动目标宿主机的登录 shell。桥接 stdin/stdout、窗口尺寸变更和远端退出；页面关闭或连接断开时取消上下文、关闭 SSH session/client。Windows 仍由 native OpenSSH 在 Windows 宿主机启动 shell，不进入 WSL2 或容器。
- 票据兑换时核对 target revision；会话运行中定期检查当前 Environment revision、凭据 binding 和 Project 成员资格，并在客户端输入前检查是否已失效。浏览器切换 Project、离开环境页、目标修订变化或主动断开时关闭连接；关闭抽屉仅隐藏。后端在目标变更或身份失效后关闭旧会话，不重连到新目标。
- 设置固定且集中管理的票据有效期、每用户与全局连接数、空闲时间和最长会话时间上限。记录 actor ID、Project/Environment ID、远端 SSH 用户、连接/结束时间与原因，不记录终端流内容。服务端关闭、网络中断和远端 shell 退出都释放配额。

### 3. Environment 页面终端

- 在 Environment 页面 SSH 详情中增加终端入口，只对已保存的 SSH target 展示；使用共享 `AppDrawer` 承载终端，并显示环境的 `username@host`、连接状态、断开与重试动作。就绪性仍由服务端裁决，未通过 Probe 时页面给出当前诊断并引导先 Probe。
- 使用 `@xterm/xterm` 与 fit addon 管理显示、键盘输入、粘贴、尺寸变化和焦点。首次打开抽屉后调用票据接口，再建立 WebSocket。关闭抽屉保留终端实例、DOM、缓冲区和连接，包括正在建立的连接；隐藏时停止尺寸调整和焦点操作，继续接收输出。重新打开将原终端 DOM 挂回抽屉，重新适配尺寸并恢复焦点，不重新签发票据。组件卸载、路由离开、当前 Project 或目标修订改变时关闭连接并销毁终端实例。断线后仅手动重连，重新签发票据。
- 以现有 `web/src/config.ts` 的公开 API 地址生成 `ws:`/`wss:` URL；保持前端现有组件、样式、图标和中英文文案约定。终端不提供远端 host、port、用户名或密钥输入。

### 4. 同步契约与活文档

- 为票据 HTTP 响应补充 `proto/orbit/v1/environment/environment.proto` 契约并生成 Go/TypeScript DTO；WebSocket 帧协议集中定义，不改动既有 Environment 请求/响应字段。
- 更新 `docs/product/cd-model.md`、`docs/architecture/cd-runtime.md` 和 `docs/guides/deployment.md` 中“SSH 不支持交互式 shell/任意命令”的现行表述，限定新增能力仅由环境页终端使用；Probe、部署和 CI 仍为非交互固定流程。

## Files to change

- 后端：`internal/application/environment/usecase/`、`internal/application/environment/port/`、`internal/infrastructure/runner/ssh/`、`internal/api/http/handler/environment/`、`internal/api/http/routes/environment.go`、`internal/api/http/routes/routes.go`、`internal/api/http/middleware/logging.go`、`internal/bootstrap/http.go`，以及新增的单节点票据/会话存储适配器。
- 契约与依赖：`proto/orbit/v1/environment/environment.proto`、`internal/gen/proto/`、`web/src/gen/proto/`、`go.mod`、`go.sum`、`web/package.json`、`web/yarn.lock`。
- 前端：`web/src/views/environment/EnvironmentPage.vue`、新增终端组件与 API 模块、`web/src/i18n/locales/zh-CN.ts`、`en-US.ts`，及对应测试。
- 活文档：`docs/product/cd-model.md`、`docs/architecture/cd-runtime.md`、`docs/guides/deployment.md`。不修改已执行迁移文件。

## Verification plan

1. Go 用例/HTTP 测试：本 Project 成员且最新 Probe 成功可拿票据；未登录、非成员、local 或未就绪目标拒绝；票据过期、重放、目标/凭据修订变化和 Origin 不符拒绝；HTTP 日志不包含票据。
2. 使用测试 SSH server 验证受管用户及 host-key pinning、PTY 输入输出、resize、退出、断线关闭与配额回收。对目标变更和成员移除验证活跃会话终止；保持现有 Probe/部署/CI 行为测试。
3. 前端组件测试：只在 SSH 环境显示入口；连接、输入输出、尺寸变化、错误反馈；关闭与重开保留会话和输出，隐藏期间继续连接且不夺取焦点；Project/目标变更和组件卸载清理；主动断开后只手动重连。WebSocket 使用可控替身，不把具体渲染库内部实现当作断言目标。
4. 执行 `task proto`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`、`task check`、`go test ./cmd/... ./internal/...`，并运行相关前端测试。若有可用的真实 Windows SSH 测试目标，执行 Windows 会话验证；否则在 Verification 记录该实机验证缺口。
5. 在 Verification 阶段对照 intent/plan 与实际 diff 核验范围、日志、鉴权及断线行为；按仓库规则不主动启动开发服务器。

## Blockers

无已知代码级阻塞。Windows OpenSSH 的真实 PTY 行为需要可用的 Windows SSH 目标才能完成端到端实机验证。

## Assumptions

- 继续采用当前单节点 API/worker 部署形态，因此短时票据和活跃会话配额可以保存在进程内，不增加数据库表或跨节点会话协议。
- 终端沿用 Project 成员授权，不增加独立权限码；远端身份固定为该 Environment 的 `username`，不按 Orbit 用户分配远端账号。
- 使用成熟的 WebSocket 和浏览器终端库；Linux/Windows 都请求 OpenSSH PTY 和登录 shell，具体 Windows 编码/回显行为由实现测试确认。

## Risks

- 终端授予现有 Project 成员以受管 SSH 用户身份执行任意宿主机命令的能力，需在页面入口与服务端鉴权中保持这一授权语义清晰。
- WebSocket 的 Origin、票据传递与 HTTP 日志处理若遗漏，会导致跨站连接或短时票据泄露；该路径必须有真实 HTTP Upgrade 测试。
- 目标在活跃会话期间变更与权限撤销依赖服务端持续复核；检查间隔及断开行为需要可观测且有上限，不能仅靠前端关闭。
- WebSocket 依赖代理正确透传 Upgrade；现有本地测试可证明 API 行为，部署链路需在验证阶段检查。
- Windows native OpenSSH 的默认 shell、PTY、终端编码可能与 Linux 不同，需按平台验证而不改变为 WSL2 shell。

## Rollback

- 移除终端入口、票据/Upgrade 路由与会话适配器即可回退；该功能不修改远端文件、Environment 存储模型或已执行迁移。
- 回退时已建立的终端连接随服务进程停止而断开；既有 SSH Probe、部署与 CI 路径保持独立。

## User review notes

- 用户要求新建分支 `feat/environment-ssh-terminal` 并使用 SpecFlow 标准模式。
- 用户强调 Environment 已配置远端 SSH 用户，终端应直连目标宿主机并沿用这个用户。
- 用户明确要求开始计划阶段，前一阶段 Intent 已标记为 Accepted。
- 用户明确要求开始实现，并要求新增依赖采用当前主流版本；计划据此标记为 Accepted。
- 用户自行测试后要求关闭抽屉保留组件与会话；据此修正原计划中的关闭即断开行为，保留页面离开、Project/目标变更和主动断开的清理边界。
- 用户要求工作完成后进入验收阶段；实现阶段结束后已进入 Verification。
