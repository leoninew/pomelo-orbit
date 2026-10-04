# 服务容器日志与运行目录记录解耦验收
最后修改时间: 2026-10-04 18:15:15

Flow mode: light
Review status: Accepted

## Basis and scope / 依据与范围

- 依据 [Accepted Intent](../intent/20261004-service-container-log-directory-dependency.md)、本次实际 diff、自动化检查、Pomelo PW 浏览器结果和用户人工验收。
- 用户在实现及验证结果汇报后明确确认“已经测试没有问题”，要求补记 light 流程并完成 Verification。
- 记录前相对 HEAD 的范围为 13 个文件、122 行新增、60 行删除；本轮只新增同名 Intent 与 Verification 两份文档。
- 当前实现已由用户暂存。本轮核对工作区与暂存内容，没有执行 Git 暂存、提交、推送或操作开发服务器。

## Intent alignment / 意图对齐

根因定位与目标一致：日志容器查询已能找到目标容器，但新服务的运行目录记录为空，原读取逻辑因此持续等待。请求中的 `ready/waiting` 是连接与等待事件，不代表返回了日志正文。用户给出的主机目录与默认目录计算结果一致。

实现保留既有 Compose 标签定位，改为在 Environment 根目录执行显式 project name 的 Compose 日志命令。读取不再依赖 Service 状态、运行目录记录或 Compose 文件；没有自动填充目录、改变库存状态或执行部署。首次范围、续读、来源更新、授权及取消仍按原契约运行。

## Spec alignment / 规格对齐

不适用：light 模式不创建 Spec，按 Intent 核对。

## Plan alignment / 计划对齐

不适用：light 模式不创建 Plan，按 Intent 验收标准核对。

## Actual diff summary / 实际差异摘要

- 应用用例：Compose 命令删除配置文件参数，调用 `StreamAtEnvironmentRoot`；删除日志读取对运行目录绑定及目录变化的处理。
- Runtime：流式端口及 target dispatcher 收敛到 Environment 根目录；local 删除 Compose 文件检查，SSH 删除 SFTP 配置文件检查。
- 测试：新增 stopped 服务运行目录为空时的 local/SSH 输出与取消回归；新增 local/SSH adapter 无 Compose 配置的读取测试，适配已有测试替身。
- 活文档：更新 CD 产品模型与运行时，区分日志容器观测和依赖运行目录的部署生命周期操作。
- 过程文档：补记本 Intent 与 Verification，状态均为 `Accepted`。

## Expected vs actual changed files / 预期与实际文件对比

| 预期范围 | 实际文件 | 核对结论 |
| --- | --- | --- |
| 日志编排与目录依赖清理 | `internal/application/deployment/usecase/container_log_stream.go`、`service_location.go` | 符合；删除不再使用的日志目录变化错误，其他运行目录解析保留 |
| 流式端口和目标执行 | `internal/application/deployment/port/port.go`、`internal/infrastructure/runner/target/runtime.go`、`local/log_stream.go`、`ssh/log_stream.go` | 符合；替换旧端口，无新旧逻辑并存 |
| 正向与既有行为回归 | `container_log_stream_test.go`、`workspace_fake_test.go`、`internal/infrastructure/external/traefik/route_test.go`、local/SSH `log_stream_test.go` | 符合；Traefik 文件仅适配 Runtime 测试替身，没有扩展路由行为 |
| 当前产品与架构依据 | `docs/product/cd-model.md`、`docs/architecture/cd-runtime.md` | 符合；同步日志读取与运行目录边界 |
| light 流程记录 | 同名 Intent 与本 Verification | 符合；记录已完成工作与人工验收 |

## Acceptance checklist / 验收清单

| 验收项 | 结果与证据 |
| --- | --- |
| stopped 且运行目录为空仍读取目标容器 | 通过：`TestStoppedServiceContainerLogs` 的 local/SSH 子测试收到内容和游标；取消后 follower 关闭 |
| 命令不加载 Compose 文件 | 通过：用例断言完整命令为 `compose -p demo-default logs --follow --timestamps --no-color --tail 200 web` |
| local 无 Compose 配置可执行流式读取 | 通过：adapter 在无配置、首次创建的 Environment 根目录执行测试子进程并取得输出 |
| SSH 不依赖 SFTP 或 Compose 配置 | 通过：SSH adapter 测试取得命令输出，SFTP 客户端数为 0 |
| 无容器等待、出现后读取、完成后持续跟随、重建续读 | 通过：既有 `TestDeploymentContainerLogsWaitThenFollowRebuildAfterDeploymentCompletes` 回归 |
| 成员撤销、目标/凭据修订变化、订阅取消 | 通过：既有 `TestContainerLogSubscriptionStopsWhenMembershipIsRevoked`、`TestContainerLogSubscriptionRejectsChangedTarget` 与 follower 关闭断言 |
| 实际服务首次打开有日志上屏 | 通过：Pomelo PW 观察到 `ready/chunk`，采集前 98 个日志块共 83,842 字节；Vue 日志状态为 `streaming`，编辑器有可见内容 |
| 关闭重开恢复内容并继续读取 | 通过：重新请求 HTTP 200，URL 携带 cursor，抽屉日志仍可见；保留关闭重开截图 |
| 固定检查与用户人工验收 | 通过：`task check`、规范化 TMPDIR 后的完整 Go 测试通过；用户确认测试没有问题 |

## Test / command results / 测试与命令结果

项目为 Go 后端与 `web` Vue/TypeScript 前端，使用现有 Taskfile、Go 测试入口及用户指定 Pomelo PW。下列结果收集自本次修复过程；本轮仅补记文档，没有在代码未变化时重复运行。

| 命令或检查 | 结果 |
| --- | --- |
| `task check` | 通过：前端 typecheck、lint、Prettier，以及 Go lint 配置、格式与静态检查；Go lint 为 0 issues |
| 部署用例与 local/SSH/target 的针对性 Go 回归 | 通过：stopped/空运行目录、adapter 无配置、等待/重建、成员与目标复核测试；target 包参与编译但该筛选没有命中其测试 |
| `go test ./cmd/... ./internal/...` | 初次失败：既有 `repositorysource.TestSourceValidatesRepositoryAndResolvesRevision` 将 macOS `/var/...` 临时路径与规范化后的 `/private/var/...` 直接比较；失败与日志变更无关 |
| `env TMPDIR=/private/var/folders/25/5tttv2h95rv93c769y5x460w0000gn/T go test ./cmd/... ./internal/...` | 通过：使用相同临时目录的物理路径重跑完整入口，没有修改该测试或项目配置 |
| `pomelo-pw --version` | 安装版本为 0.19.0 |
| `pomelo-pw validate /tmp/pomelo-orbit-logs.yaml` | 通过：修复后流程共 25 步 |
| `pomelo-pw run /tmp/pomelo-orbit-logs.yaml --headless --json` | 通过：报告 `status: passed`，25 步完成、无错误；验证真实日志内容与重开恢复 |
| `git diff HEAD --check` | 通过：实现与本轮过程文档核对无空白错误 |
| 用户人工验证 | 用户明确反馈“已经测试没有问题” |

针对性 Go 实际命令：

```sh
go test ./internal/application/deployment/usecase ./internal/infrastructure/runner/local ./internal/infrastructure/runner/ssh ./internal/infrastructure/runner/target -run 'TestStoppedServiceContainerLogs|TestStreamAtEnvironmentRootWithoutComposeConfiguration|TestDeploymentContainerLogsWaitThenFollowRebuildAfterDeploymentCompletes|TestContainerLogSubscription' -count=1
```

Pomelo PW 在现有 `http://localhost:9020` 上检查服务详情的 `k12-delivery-os` 组件。修复前响应仅有零字节的 ready/waiting，Vue 缓存长度与版本均为 0；修复后收到日志正文，缓存长度为 81,689 字符、状态为 streaming。采集的事件数限制为 100，因此 98 个 chunk 是观测样本，不是服务器日志总量。

本机临时证据：`/tmp/pomelo-orbit-logs.yaml`、`/tmp/pomelo-orbit-logs/first-open.png`、`/tmp/pomelo-orbit-logs/reopened.png`。未将登录凭证或会话状态写入仓库。

## Missed or expanded scope / 范围偏差

没有未落实的目标或额外业务扩展。共享日志用例同时被 Service、Gateway、Route 与部署容器入口使用，统一删除目录依赖；部署记录的目标快照与 stop 操作限制继续保留。测试替身适配属于 Runtime 接口收敛的必要变化。

前端实现、数据库与迁移、目录绑定、库存状态和部署/重启/停止执行均未修改。原有目录确认与日志统一任务的过程文档保持原记录，当前行为由本 Intent 和已更新的活文档说明。

## Risks / 风险

- 匹配依赖 Compose project 与 Component 标签；目标 Docker 权限与日志保留能力仍是读取前提。
- 两秒复核、follow、SSE、初始 tail 和续读重叠策略未增加频率或缓存。移除文件检查与 SFTP 的收益仅作代码级评估，未做性能压测。
- 本次真实浏览器验收针对用户提供的服务与当前目标环境，local/SSH adapter 另有自动化覆盖；未进行全部操作系统、Docker/Compose 版本或生产代理组合的专项认证。

## Incomplete items / 未完成事项

本次修复验收范围内无未完成事项。性能压测与全部平台组合认证不是本次交付条件，不据此推断它们已经通过。

## Conclusion / 结论

修复与 Intent 对齐，固定检查、相关回归、真实 Pomelo PW 流程和用户人工验收通过。问题确认为日志读取对未绑定运行目录记录的依赖，修复后无需保存该记录即可查看目标上同一 Compose project 的实际输出。

用户已明确要求完成验证，Review status 标记为 `Accepted`。本阶段只补充流程文档，未改变用户暂存状态或创建提交。
