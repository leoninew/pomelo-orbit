# Environment SSH 宿主机终端验收
最后修改时间: 2026-09-30 12:02:59

Review status: Draft

Mode: standard

## Basis

- [Intent](../intent/20260930-environment-ssh-terminal.md)：Accepted。
- [Plan](../plan/20260930-environment-ssh-terminal.md)：Accepted。
- 用户已要求工作完成后进入验收。本文记录实际实现与自动化检查，真实主机和浏览器人工验收仍待确认。
- 工作分支：`feat/environment-ssh-terminal`。

## Intent alignment

- 终端连接 Environment 保存的目标宿主机，使用其 SSH 用户、受管私钥和 pinned host key；不进入应用容器或 WSL2，不允许浏览器指定其他账号。
- 以当前 Orbit 用户的 Project 成员身份签发票据；票据绑定 Environment、目标修订与 SSH 凭据修订。建立连接和会话期间均检查授权与目标有效性。
- 仅 SSH Environment 提供入口。当前修订 Probe、凭据和 host key 的就绪性由后端 TargetResolver 校验。
- 按用户测试反馈修正生命周期：关闭抽屉仅隐藏，保留原终端、输出和 SSH 会话；重新打开不创建新连接。页面离开、Project/目标变更、主动断开及服务端限制仍终止连接。

## Spec alignment

标准模式未创建独立 Spec，按 Intent、Plan 和用户后续明确指令核对。

## Plan alignment

| 计划内容 | 实际实现与核验 |
| --- | --- |
| 短时一次性票据和 WebSocket | 单数路径；30 秒有效期；仅保存票据哈希；浏览器以子协议传票据，服务端只协商 `orbit-terminal-v1` |
| 宿主机 SSH PTY | 独立窄端口，复用既有 `openSSH`，申请 `xterm-256color` PTY 并启动默认 shell；未修改 Probe/部署/CI 命令 |
| 双向交互 | 二进制消息传字节，JSON 控制尺寸、就绪、退出和失败；SSH shell 启动后才发送 `ready` |
| 会话边界 | 连接 15 秒、空闲 15 分钟、总时长 2 小时；全局 16 个、每操作者 2 个连接；2 秒周期检查及输入前检查 |
| 前端终端 | `@xterm/xterm 6.0.0`、`@xterm/addon-fit 0.11.0`；后端使用 `github.com/coder/websocket v1.8.15` |
| 抽屉外观 | 复用共享 AppDrawer 和日志抽屉宽度、留白及主题；紧凑标题；连接状态与动作集中在一个图标控件，保留 tooltip 和无障碍状态 |
| 关闭与重开 | 保留 xterm 的 DOM 和 WebSocket，抽屉隐藏后继续接收输出；重开重新挂载、适配尺寸并恢复焦点；显式断开后仅手动重连 |
| 目标变更 | 页面以 Project ID 与 target revision 标识终端组件，目标修订变化清理旧实例；服务端持续复核凭据及成员身份 |
| 日志保护 | 仅两个终端路径停用请求/响应正文采集；原有 HTTP 元信息日志保留；会话只记录身份、关联对象、时长与结束原因 |

## Actual diff summary

- 后端新增终端应用用例、窄端口、SSH 会话适配器和 HTTP/WebSocket handler，并接入现有路由与组合根。
- 前端新增终端抽屉、票据 API 与中英文文案，环境页增加 SSH 入口和目标变化清理；开发代理透传 WebSocket Upgrade。
- 补充票据 proto DTO、生成代码与锁定依赖；同步产品、运行时和部署活文档。
- 增加票据、授权、真实 WebSocket、受控 SSH server、前端交互及生命周期测试。
- 未改数据库模型或已执行迁移，未引入 local/容器终端、额外 SSH 身份或独立终端权限。

## Expected versus actual files

| 范围 | 实际文件 | 对齐情况 |
| --- | --- | --- |
| 终端业务和 SSH | `internal/application/environment/{port,usecase}/terminal*.go`、`internal/infrastructure/runner/ssh/terminal*.go` | 符合计划；进程内票据与配额直接由终端用例管理，未增加存储接口或表 |
| HTTP 和组合根 | `internal/api/http/handler/environment/`、`internal/api/http/routes/`、`internal/bootstrap/http.go` | 符合计划；票据签发不占写事务，WebSocket 长连接在事务外运行 |
| 日志 | `internal/api/http/middleware/logging.go` 及测试 | 限于终端正文保密，无日志子系统重构 |
| DTO 与依赖 | `proto/`、Go/TS 对应生成代码、`go.mod`、`go.sum`、`web/package.json`、`web/yarn.lock` | 符合计划 |
| 环境页及终端 | `web/src/views/environment/EnvironmentPage*`、`EnvironmentTerminalDrawer*`、`web/src/api/project/environment.ts`、locale 文件 | 符合计划及用户后续生命周期要求 |
| 共享抽屉 | `web/src/components/AppDrawer.vue` | 为紧凑标题增加 title/actions slot 和 header class；保留其他抽屉默认行为 |
| WebSocket 代理 | `web/vite.config.ts` | 本地 ticket 返回 200 但连接不就绪的修复；启用 Upgrade，并保留浏览器 Host 供 Origin 检查 |
| 服务退出 | `internal/bootstrap/runtime.go` | HTTP BaseContext 关联服务生命周期，让被 hijack 的 WebSocket 会话观察服务取消 |
| 文档 | Intent、Plan、本文及三个活文档 | 符合计划；关闭抽屉语义已按用户要求更新 |

## Acceptance checklist

- [x] SSH-only 入口、票据子协议、就绪状态、双向输入输出、resize、退出状态与连接超时：前后端测试通过。
- [x] 受管 SSH 用户、私钥和 pinned host key：真实 SSH 协议测试通过，错误主机指纹被拒绝。
- [x] 未登录、非成员、local/未就绪目标、过期与重放票据：应用用例与 HTTP 测试通过。
- [x] 目标/凭据/成员变化：用例校验通过，HTTP 测试确认输入前拒绝和无输入时周期性关闭。
- [x] 关闭/重开复用原连接与终端，隐藏时继续接收输出，连接中关闭后继续建立且不抢焦点：前端测试通过。
- [x] 隐藏会话在 Project/host/username 变化时清理；组件卸载取消尚未返回的票据；环境修订变化销毁原组件：前端测试通过。
- [x] 主动断开后重开不会自动连接；手动重连重新签发票据：前端测试通过。
- [x] 浏览器断线取消正在建立的 SSH 连接；未消费输出时 SSH Close/Wait 能结束；连接配额只释放一次：后端测试通过。
- [x] 终端路径不采集请求/响应正文，不记录 WebSocket 流或认证头；会话日志仅含元信息：日志测试与代码核对通过。
- [x] 现有 Go 测试及日志抽屉回归通过，Probe/部署/CI 非交互路径保持原实现。
- [ ] 真实 Linux/Windows 宿主机 shell、中文输入输出、交互程序与退出表现：仍需人工实机验收。
- [ ] 实际浏览器中关闭与重开保持工作目录、shell 变量和隐藏期间输出，桌面/移动视口的布局、焦点和滚动：仍需人工验收。

空闲与最长时长上限已核对代码，本轮未真实等待 15 分钟或 2 小时；服务退出关联已核对代码并运行既有 runtime 测试，未单独执行真实服务重启测试。

## Test results

| 命令与目录 | 结果 |
| --- | --- |
| 根目录 `yarn --cwd web lint:fix` | 通过 |
| 根目录 `task check` | 通过，含 `yarn --cwd web typecheck`、lint、format、Go 配置、格式和静态检查；Go lint 为 0 issues |
| 根目录 `go test ./cmd/... ./internal/...` | 通过；已有测试结果使用 Go 缓存 |
| 根目录 `yarn --cwd web test src/views/environment/EnvironmentTerminalDrawer.test.ts src/views/environment/EnvironmentPage.test.ts src/components/RuntimeContainerLogsDrawer.test.ts` | 3 个文件、15 项测试通过 |
| 根目录 `git diff HEAD --check` | 通过 |
| 根目录 `task proto` | 实现阶段已通过，本轮未重复生成未变化的契约 |

本轮 `task check` 首次发现新测试数据缺少必需的 `local` 字段，补齐为 `undefined` 后完整重跑通过。格式化命令首次未找到 binary，改用项目 Yarn 的直接 binary 调用后通过。

## Scope changes and risks

- 用户明确修正原计划的“关闭抽屉立即断开”，现以当前环境页为会话持有范围；Intent/Plan 已同步。
- 共享 AppDrawer 的 slot、服务 BaseContext 和 Vite WebSocket 代理属于接入终端所需支持，未扩展为通用抽屉持久化或日志改造。
- 隐藏的终端仍占用连接配额，也仍受授权复核、空闲和总时长限制；刷新浏览器会断开，本文不承诺跨页面或跨刷新恢复会话。
- 当前单节点设计使用进程内票据和配额；多节点会话路由不在本期范围。
- Windows OpenSSH 的实际默认 shell、字符编码与 PTY 表现尚未实机验证。

## Incomplete items

1. 本次浏览器工具发现没有可用浏览器，未完成截图和桌面/移动视觉验收。组件测试使用真实共享抽屉和可控 xterm/WebSocket 替身，不代替真实浏览器渲染检查。
2. 真实宿主机交互与关闭重开场景待用户复测；受控 SSH server 已验证协议、身份、主机指纹、PTY、输入输出、resize 和退出。
3. 生产反向代理的 WebSocket Upgrade/超时尚未实测；实现阶段已确认本地 9020 代理的 Upgrade 请求可以到达后端并立即返回无效票据 401。

## Conclusion

代码实现与自动化验收通过，已修复关闭抽屉导致会话销毁的问题。人工实机与视觉验收尚待确认，本文保持 Draft，不代表用户已接受最终验收。未创建提交。
