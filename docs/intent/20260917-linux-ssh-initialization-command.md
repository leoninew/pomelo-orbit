# Linux SSH 初始化命令与受管密钥流程
最后修改时间: 2026-09-17 21:48:53

Review status: Accepted

Mode: standard

## Background

当前 Linux SSH Environment 在保存 target 或执行 Probe 时可能隐式生成受管密钥；随后 Linux bootstrap 再用一次性密码或私钥把该公钥写入远端。这个顺序要求目标已经可被 Orbit 连接，不能覆盖全新的 Linux 服务器，也让用户无法从页面看出部署身份何时创建、远端何时接受该身份。

Windows 已提供“生成初始化命令 -> 用户在目标端执行 -> 检查连接/Probe”的交互。Linux 应采用相同的主流程：Orbit 只在用户明确请求初始化命令时创建 Environment 专用部署密钥，用户通过云控制台、已有 SSH 登录或其他带外途径在目标端执行命令，然后由 Orbit 使用该密钥验证连接与运行时前置条件。

本机执行 `ssh tencent` 使用的是操作者机器的本地 SSH 配置和私钥。Orbit 服务端不应读取该配置，也不能由一次裸 TCP/SSH 握手推断出 Orbit 已拥有可用于部署的认证身份。

## Goal

1. 用户完成并通过当前平台 SSH Environment 表单校验后，Linux 与 Windows 的“初始化命令”显式请求以该表单为准创建或更新当前 Environment target；该请求首次创建并绑定 Environment 受管密钥。保存、编辑、状态读取和 Probe 不得隐式生成、替换或轮换密钥。
2. 初始化命令请求在同一写事务内持久化 target 与 credential binding，使刷新后的页面可从 Environment 读取当前表单。第二次打开时复用有效绑定的同一 Environment 密钥，并展示同一公钥，避免已执行的历史命令因重新生成而失效。
3. Linux 初始化命令可由配置的部署 SSH 用户在目标端执行，幂等创建 `~/.ssh`、修正相应权限并将 Orbit 公钥写入 `authorized_keys`；不删除或覆盖已有授权密钥。
4. Linux 初始化命令不安装或配置 `sshd`、Docker、Docker Compose、Docker 用户组、防火墙或其他系统服务。它不要求 Orbit 在生成命令前已能连接目标。
5. SSH 检查和 Probe 必须使用 Environment 已绑定的受管私钥完成实际公钥认证。未生成初始化命令时，返回明确、可操作的“请先生成并执行初始化命令”诊断；不得退回为裸网络连通性检查。
6. Probe 在 SSH 认证成功后继续验证 host-key pinning、Docker 和 Docker Compose；保留底层失败诊断和结构化错误日志，但不得记录私钥、口令或其他认证秘密。
7. 停止把“上传操作者已有私钥”作为 Linux 默认流程。Orbit 不自动读取浏览器或服务端的 `~/.ssh/config`、agent 或私钥；后续若保留显式导入能力，必须作为独立的高级能力重新评审。

## Non-goal

- 不为新 Linux 主机安装或启动 OpenSSH Server；用户通过云控制台等带外通道执行初始化命令。
- 不自动发现、读取或持久化操作者的个人 SSH 私钥。
- 不将 SSH 密码认证、交互式 shell、PTY、端口转发引入部署或 Probe。
- 不变更 local Environment、Docker runtime、Gateway 或 Deployment snapshot 的领域边界。
- 不修改已执行的数据库迁移文件。

## User scenarios

1. 用户完整填写并通过校验的 Linux SSH target 后点击“初始化命令”。Orbit 用提交的表单保存当前 Environment，创建并绑定该 Environment 的密钥对并展示命令，无需此时已有从 Orbit 到目标的 SSH 连通性。刷新页面后仍显示该 target。用户在云控制台以部署用户执行命令后，点击检查连接或 Probe；Orbit 用已保存私钥认证成功并记录 host key 与 Docker 前置条件。
2. 用户已有可用的个人 SSH 登录，例如本机 `ssh tencent`。用户使用该登录或其他带外方式执行 Orbit 展示的命令。Orbit 不取得其个人私钥；Probe 成功只证明 Orbit 的受管密钥已被远端接受，不关心公钥是通过命令还是等价的人工操作加入。
3. 用户修改 Linux 或 Windows target 后再次打开初始化命令。最新完整表单覆盖当前 Environment target 并清除旧 target 的 Probe/host-key 状态；已绑定的有效受管密钥继续复用。页面返回原有受管公钥，不生成新密钥；此前针对同一密钥执行过的命令和后续 Probe 仍匹配。
4. 用户直接执行检查连接但尚未生成初始化命令，或尚未在远端安装公钥。页面展示明确失败原因，日志保留底层握手或认证诊断，且不会因此生成或替换密钥。

## Acceptance

- [ ] 保存、编辑、读取和 Probe SSH Environment 不创建或替换 `environment_credential`。
- [ ] Linux 与 Windows 初始化命令的显式 API 接收并原子保存已校验的当前 SSH target，是创建受管密钥的唯一入口；重复调用复用有效绑定的密钥。
- [ ] Linux 初始化命令仅以幂等方式配置配置用户的 `authorized_keys` 及其必要权限，不需要 Orbit 预先成功认证该 target。
- [ ] 检查连接和 Probe 都使用受管私钥进行 SSH 公钥认证；网络可达但密钥不被接受必须失败。
- [ ] 未初始化时的诊断可区分“尚未生成/执行初始化命令”与底层 SSH、host-key、Docker、Compose 失败。
- [ ] HTTP 响应、页面、日志和数据库以外的边界不泄露 SSH 私钥、私钥口令或一次性认证数据。
- [ ] 初始化向导和 Environment 详情页都提供一致的 Linux/Windows 初始化命令与检查连接路径。
- [ ] Go 和前端测试覆盖首次生成、重复复用、未初始化 Probe、初始化后认证 Probe，以及 Linux 命令的幂等公钥写入语义。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- 采用标准模式 / standard。
- Linux 以初始化命令作为主路径，与 Windows 的“显式准备 -> 用户在目标端执行 -> Orbit 检查”对齐。
- 受管密钥只在用户点击初始化命令时生成；其余操作只能读取已绑定密钥或返回未初始化状态。
- 初始化命令以提交的完整、已校验 SSH 表单覆盖当前 Environment target。刷新后的表单从该持久化 target 回填；有效 credential binding 随 target 保留，不因再次展示命令而轮换。
- Linux 命令在配置的部署用户上下文执行，只处理该用户的公钥授权与权限，不承担操作系统或 Docker 安装配置。
- “导入已有私钥”不进入本任务默认流程。已有个人 SSH 登录可用于执行初始化命令，但不会被 Orbit 自动取得或复用。
- SSH 连通性以 Orbit 受管密钥认证为准；裸网络可达不是成功的部署连接。

## Risk

- 未生成或未执行初始化命令时，SSH Probe 必然失败；这属于受控的未就绪状态，页面诊断必须使用户能够直接恢复。
- Linux 的用户必须通过云控制台、已有登录或其他带外通道执行命令；Orbit 不负责建立这种初始访问通道。
- 从旧的 Linux 自动 bootstrap 流程切换到命令流程会触及 HTTP/proto、前端向导、Environment 页面和现有测试，需要彻底删除旧入口，不能保留双路径兼容逻辑。
- 受管密钥变更会使已有部署快照失效；实现必须只在首次显式初始化或明确的未来轮换操作时改变 credential revision。

## User review notes

- 用户指出全新的 Linux 服务器不能以“Orbit 已能连接”为初始化前提，Linux 应与 Windows 一样先展示初始化命令。
- 用户确认：点击初始化命令才生成并持久化密钥；用户未点击时后台不得生成密钥。
- 用户确认：用户执行命令将 Orbit 公钥加入 `authorized_keys`，随后由 Orbit 检查连接；是否通过该命令或等价人工操作完成不影响成功语义。
- 用户确认：必须先完成 Environment 表单，初始化命令才能判断生成 PowerShell 或 Bash；初始化请求应保存当前表单和密钥，以便刷新后回填并继续复用密钥。
