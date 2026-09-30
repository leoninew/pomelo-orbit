# Environment SSH 宿主机终端
最后修改时间: 2026-09-30 12:02:59

Review status: Accepted

Mode: standard

## Background

当前 Environment 页面提供 SSH 目标配置、初始化命令、Probe 状态与诊断。一个 Project 的 SSH Environment 保存一个 `username`，并绑定一份受管 SSH 凭据；Probe、部署和 CI 均使用这个环境配置的远端登录用户。Orbit 已有首次 Probe 固定的 host key 指纹，以及 Linux/Windows SSH 运行时，但运行时只执行平台预定的非交互命令，不开放 PTY 或交互式 shell。用户希望从 Environment 页面直接与目标宿主机交互。

本任务的“终端”指 Environment 已配置的 SSH 用户在目标宿主机上的交互式 shell。当前登录的 Orbit 用户只用于页面请求身份和 Project 成员校验，不会替换远端 SSH 用户。终端不进入应用容器、Docker Desktop 内部发行版或 Orbit 服务进程所在容器。

## Goal

1. 在当前 Project 的 SSH Environment 页面提供可打开、关闭的交互式宿主机终端，支持现有 Linux OpenSSH 与 Windows native OpenSSH 目标。
2. 终端沿用该 Environment 保存的 host、port、`username`、受管私钥和已固定的 host key；与该环境的 Probe、部署及 CI 使用同一个远端身份。浏览器不接收私钥，也不能改写连接目标、SSH 用户或认证方式。
3. 支持键盘输入、连续输出、终端尺寸变化、退出状态与连接失败反馈；关闭抽屉仅隐藏终端，保留实例、缓冲区和 SSH 会话，重新打开继续同一会话。离开环境页、切换 Project 或主动断开时释放远端会话。
4. 沿用现有 Environment 的授权边界：只允许已登录的当前 Project 成员连接已完成当前修订 Probe 的 SSH Environment，并记录发起会话的 Orbit 用户。Environment 的 host、port、SSH 用户或凭据在会话期间变化时，不允许继续使用旧目标会话。
5. 对连接数、空闲时间和会话最长时间设置边界；记录操作者、Project、Environment、连接时间和结束原因，不把交互输入、终端输出或认证秘密写入常规 HTTP 日志。

## Non-goal

- 本期不为 `local` Environment 提供宿主机终端；它没有 SSH 身份，连接宿主机需要独立设计。
- 不增加每个 Orbit 用户各自的远端 SSH 账号或密钥，也不允许在打开终端时临时选择其他 SSH 用户。
- 不提供容器内 shell、SSH 端口转发、文件上传/下载、密码登录或操作者个人 SSH 密钥。
- 不改变部署、流水线、Probe 的固定命令执行方式，也不将交互式 shell 暴露给这些流程或 MCP。
- 不记录可回放的完整终端会话内容；终端可能显示或输入敏感数据。

## User scenarios

1. 已登录的 Project 成员打开已就绪的 Linux SSH Environment，在页面中以该 Environment 保存的 `username` 进入目标宿主机 shell，执行命令并实时看到输出，调整窗口后远端终端同步变化。
2. 已登录的 Project 成员打开 Windows SSH Environment，以该 Environment 保存的 Windows SSH 用户在目标 Windows 主机的 OpenSSH shell 中交互；不会进入 WSL2 或应用容器。
3. SSH 断线、主机指纹不匹配、Probe 已过期或权限不足时，页面给出明确状态，不自动切换目标或改走本地执行。
4. 用户离开环境页、切换 Project、主动断开，或会话达到限制时，终端会话结束并释放连接。
5. 用户临时关闭终端抽屉后继续查看环境信息，再次打开时保留原 shell 状态和终端输出；抽屉隐藏期间仍接收输出，空闲和最长时长限制继续生效。

## Acceptance

- [ ] 页面仅在 SSH Environment 上提供终端入口；成功连接后可以双向交互，支持终端尺寸变化、断开和退出反馈。
- [ ] 关闭抽屉不销毁终端或断开 SSH，重新打开不重新签发票据或创建会话；隐藏期间的输出保留，连接中的会话也继续建立。已断开的会话仅由用户手动重连。
- [ ] Linux 与 Windows 会话均在配置的 SSH 目标宿主机上以 Environment 保存的 `username` 运行，复用受管密钥并严格校验已固定的 host key；当前 Orbit 用户不能在终端入口覆盖该身份。
- [ ] 未通过当前修订 Probe、未登录或非 Project 成员的请求在建立 SSH 会话前被拒绝。
- [ ] 目标或凭据变更后的旧会话被终止，新连接只能使用新修订且重新通过 Probe 的目标。
- [ ] 断线、页面关闭、Project 切换、空闲或时长上限到达后，后端关闭 SSH 会话；不会留下长期占用的连接。
- [ ] 浏览器响应和常规 HTTP 日志不包含 SSH 私钥、终端输入或完整终端输出；会话元信息有可追踪记录。
- [ ] 现有 Probe、部署及 CI 的非交互 SSH 行为不受影响。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- 采用标准模式 / standard；本意图是新功能，不沿用旧 SSH 初始化任务的过程文档。
- “终端”是完整交互式宿主机 shell，不是预设诊断命令的输出面板。
- SSH 终端只绑定当前 Project 已保存且就绪的 Environment，使用其中唯一配置的远端 SSH 用户，不允许浏览器自行指定 host、port、用户或密钥。
- 终端的 Orbit 用户授权沿用现有 Environment 的 Project 成员校验，不另设 `admin` 专属或独立终端权限；远端 SSH 用户仍是 Environment 的 `username`。
- 当前对话聚焦远程 SSH Environment；`local` 宿主机终端不在本期范围。

## Risk

- SSH 用户可操作 Docker 时，其宿主机权限很高；沿用 Project 成员授权意味着所有当前成员都能以同一 SSH 用户打开 shell。服务端必须校验成员身份并保留 Orbit 操作者的会话元信息。
- 浏览器双向连接需要独立的认证和生命周期处理。现有 Bearer 认证依赖 HTTP Authorization header，不能直接套用浏览器 WebSocket 构造器。
- Windows OpenSSH 的 shell 与 PTY 行为需在实际 Windows 目标上验证；不能仅凭 Linux 行为推断一致。
- 终端可能接触敏感信息，应避免请求/响应正文、会话流和认证票据进入通用日志。

## User review notes

- 用户希望在 Environment 页面添加类似终端的元素，与环境直接交互。
- 用户明确纠正：终端应连接环境所指向的宿主机，不能打开 Orbit 容器。
- 用户要求新建分支，并指定使用 SpecFlow 标准模式。
- 用户指出远程 SSH Environment 在创建时已经配置了 SSH 用户；终端应沿用该用户，而不是引入或选择另一远端账号。
- 用户要求进入计划阶段，视为接受当前意图。
- 用户自行测试后明确要求关闭抽屉不立即销毁组件或会话；关闭仅隐藏，重新打开继续同一会话。
- 用户要求工作完成后进入验收阶段。
