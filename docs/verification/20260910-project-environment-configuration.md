# Project / Environment 配置归属、初始化与 MCP 验证
最后修改时间: 2026-09-15 12:44:39

Review status: Accepted

Mode: strict

## Requirement alignment

本次实现覆盖已接受需求中的 Project 初始化边界、Environment/Gateway 配置归属、MCP connection-local Project scope、空库 seed 收敛，以及未初始化 Project 的 Web Wizard 路由。新增的 Windows SSH 辅助流程补齐了需求中的实际部署前置：用户可在目标 Windows 主机的管理员 PowerShell 中执行当前表单生成的命令，自动安装/启动 OpenSSH、放行端口，并在 SSH 公钥认证尚未建立时完成部署公钥、工作目录和 Docker 前置检查。

环境保存失败后的重试也已纳入需求闭环。受管 `deployment-ssh` 凭据按 Project 和名称复用，避免部分失败留下的凭据在下一次保存时触发唯一约束；非受管同名凭据仍保持冲突语义。

## Spec alignment

`ProjectInitialization` 仍是唯一的 Environment/Probe/Gateway 初始化 application boundary。部署公钥读取接口只返回公钥，并在 Environment 尚未保存时按当前 Project 创建或复用受管凭据。Windows 命令不绕过页面的 SSH 测试和 Environment Probe，远端命令完成后仍须由用户显式测试和探测。

Web Wizard 与 Environment detail 复用同一个命令生成器和项目 Dialog。命令使用当前表单或已保存 Environment 的 Host、Port、SSH 用户和 Windows workspace，不读取上一个 Project 的数据，也不把初始化来源配置当作运行时 fallback。

2026-09-15 交互意图已实现并修正：Wizard 第一步的主操作为“检查连接”“生成初始化命令”“下一步”。Windows SSH 的命令生成只依赖当前表单，连接失败时仍可用，以便先在目标机完成 OpenSSH 初始化；SSH 目标须先检查连接成功后才开放“下一步”。第二步“准备环境”展示目标连接/认证、Docker 引擎、Docker Compose（Windows SSH 额外包括 WSL2 与 Docker Desktop）待检查清单，点击“检查就绪”后逐项推进，Probe 失败后仍保留“检查就绪”文案。Local 目标隐藏不适用的连接与命令动作，目标字段变更会使既有连接结果失效。

## Plan alignment

计划中的配置迁移、seed 收敛、ProjectInitialization boundary、HTTP/proto 生成、Gateway/MCP 收敛、Web readiness guard 和活文档同步均已反映在当前工作区。Environment/Gateway migrations 已重组为 `000039`（最终 Environment schema）、`000040`（GatewayConfig 废弃列清理）和 `000041`（develop(38) 遗留 Gateway 关系迁移）。

计划后续追加的 Windows SSH 命令、保存前公钥读取和受管凭据重试复用，已同步补入 requirement、spec、plan 和本验证文档。

## Actual diff summary

- 配置、数据库 schema/migration、SQLC/proto generated files 和活文档收敛 Gateway/Environment 的唯一配置来源。
- 新增 Project Initialization HTTP API、application service、Web store、router guard 和三步 Wizard。
- 删除独立 Gateway 初始化入口及 `traefik_component_name` 契约，MCP 改为 connection-local Project scope。
- 新增 `WindowsSshInitializationDialog` 和 `windowsSshCommand`，在 Wizard 与 Environment detail 提供 Windows 初始化命令；命令执行 authorized keys、workspace、WSL2/Docker Desktop/Linux containers/Compose/daemon 检查。
- `CreateDeploymentSSHCredential` 复用同 Project 的受管 `deployment-ssh`，并增加保存重试覆盖测试。
- 活文档补充 Windows SSH 操作边界、保存前生成公钥和错误重试语义。

## Expected vs actual changed files

预期变更区域为配置、Project/Environment/Gateway/MCP application、HTTP/proto、SQL migration、Web Wizard、测试和活文档。实际 diff 覆盖这些区域，并额外包含 Windows 命令共享组件、生成器测试、Environment detail 入口和部署公钥接口测试；这些扩展均直接服务于 Windows SSH 在公钥认证前无法连接的实际场景。

上一版记录中的“151 个已修改路径和 37 个未跟踪路径”是提交前快照，不再代表当前仓库状态。相关实现已纳入当前 feature 分支的提交历史；本次文档同步只修正文档表述，不重新定义代码提交边界。

## Acceptance criteria checklist

- [x] 初始化来源配置、Environment、GatewayConfig、Version/Component 和 Project 生命周期按需求收敛。
- [x] 空库保留 identity Project/membership seed，删除默认部署资源 seed；新建 Project 不自动创建 Environment。
- [x] 未初始化 Project 进入 Wizard；初始化完成后 Gateway 保持待部署状态并进入网关详情页。
- [x] Wizard 的 Environment、Probe、Gateway 三步使用派生状态，Probe 失败保留诊断并允许重新检查；Environment 的遗留启用/禁用字段不参与页面或运行时语义。
- [x] Wizard 按两步按钮门控运行：第一步 Windows SSH 命令按当前表单即可生成，不要求连接成功；连接检查成功后才可进入下一步。第二步 Probe/就绪检查成功后才可进入 Gateway。命令使用当前 Host、Port、SSH 用户和 workspace，支持复制，并负责安装/启动 OpenSSH、放行端口，再检查部署公钥、工作目录和 Docker 前置。
- [x] Environment detail 对已保存 Windows SSH target 展示同一初始化命令入口；Linux 继续使用现有自动连接/初始化路径。
- [x] 保存失败后重试复用受管 `deployment-ssh`；非受管同名凭据返回冲突。
- [x] MCP 使用显式 Project selection 和 connection-local scope，不为运行时工具继续传递 `project_id`，也不提供 Gateway 初始化工具；Environment readiness 仅依据最新 Probe，不读取遗留启用/禁用状态。
- [x] `traefik_component_name` 从 GatewayConfig、schema、proto、HTTP/MCP 和 Web 表单移除。

## Test results

- `yarn --cwd web lint:fix`：通过。
- `yarn --cwd web typecheck`：通过。
- `yarn --cwd web test`：通过，25 个测试文件、114 个测试。
- `task check`：前端 typecheck/lint 通过；format 仍被三个本轮未触及的既有文件阻断（`RuntimeContainerLogsDrawer.test.ts`、`RuntimeContainerLogsDrawer.vue`、`windowsSshCommand.test.ts`）。本轮 Go `golangci-lint` config/fmt/run 均通过，报告 0 issues。
- `go test ./cmd/... ./internal/...`：通过。
- 定向 Go 测试：ProjectInitialization、Environment、Credential 和 HTTP handler 测试通过。
- `git diff --check`：通过；未针对无关文件做换行格式化。

## Missed or expanded scope

相对最初计划，Windows SSH helper 是根据实际使用反馈追加的范围：命令生成不再依赖 Environment 已保存或 SSH 公钥认证成功，并在命令中加入分阶段 Docker/Compose 检查。该扩展已写入 requirement/spec/plan，不属于未记录的隐式变更。

上述交互门控已纳入前端实现；后续可补充覆盖按钮可用性与字段变更失效的专用组件测试。

本次按无兼容基线重组 `000039`–`000042`：`000039` 直接建立最终 Environment schema，`000040` 独立删除 `gateway_config.traefik_component_name`，`000041` 仅将 develop(38) 遗留的完整 Traefik bundle 纳入 Environment 关系；移除无效的空库 Environment seed/no-op 与重复 target-type migration。未新增多 Project 共用宿主的冲突防御，也未恢复任何全局 deployment workspace fallback。

## Risks and incomplete items

- Windows 命令需要用户在目标 Windows 主机的管理员 PowerShell 执行；命令负责安装/启动 OpenSSH Server、放行配置端口并通过 SSH 建立初始会话。Docker Desktop、WSL2 和账户权限仍需目标主机满足。
- 生成命令前可能创建受管 `deployment-ssh` 凭据。该记录用于后续 Environment 保存并按 Project/name 复用，不应被当作临时数据删除。
- 本记录不再维护提交前的暂存状态；Git 提交边界和当前工作树状态以仓库实际历史与 `git status` 为准。

## Conclusion

实现与已接受 Requirement、Spec 和 Plan 对齐，Windows SSH 初始化和重复凭据重试两个追加场景已补入过程文档并完成验证。代码测试、Go lint 和本轮前端检查通过；完整 `task check` 仍受三个未触及文件的既有 Prettier 告警影响，后续重点是实机初始化验证，Git 暂存状态不在本验证文档中维护。
