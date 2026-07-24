# CD 部署失败时展示完整错误原因
最后修改时间: 2026-07-21 10:18:30

Review status: Accepted

## Background

部署失败后，详情页 `/cd/deployments/{id}` 的「错误信息」只显示类似：

```text
run "docker" "compose" "-f" "docker-compose.yml" "up" "-d" "--remove-orphans" "--pull" "missing": exit status 1
```

实际失败原因（例如 Docker daemon 未运行、镜像拉取失败）已经写入部署操作日志文件 `data/cd/<app>/deployments/<id>.log`，但没有进入 `deployment.error_message`，详情页也看不到。

根因：

1. **后端**：`ShellRunner.Run` 将 stdout/stderr 写入操作日志文件，但返回错误只包装 `exec.ExitError`（`exit status 1`）。`CompleteDeployment` 用 `err.Error()` 写入 `error_message`，因此丢失真实 stderr。
2. **前端**：详情页按既有需求默认展示容器日志（`/container-logs`），不展示操作日志（`/logs`）。compose 在容器启动前失败时，容器日志为空或无用，用户无法在页面看到真实失败原因。

相关代码：

- `internal/infrastructure/runner/cd/runner.go` — 命令执行与错误返回
- `internal/application/cd/usecase/deployment_execution.go` — 失败时写入 `error_message`
- `web/src/views/cd/DeploymentDetail.vue` — 错误信息展示与日志区数据源

## Goal

- 部署 / 重启 / 停止执行失败时，`error_message` 必须包含可诊断的失败原因，而不仅是 exit status。
- 用户在部署详情页（及列表页错误列）无需翻磁盘日志，即可看到与操作日志中一致的关键失败信息。
- 部署失败（`faulted`）时，详情页应能看到**操作日志**中的 compose 真实输出；成功/运行中的主日志区仍可保持容器日志策略。
- 错误信息在详情页完整展示（支持换行、不截断关键内容）。

## Non-goal

- 不重做部署详情页整体信息架构或视觉风格。
- 不改变 Docker Compose 实际执行行为、镜像拉取策略、部署状态机。
- 不回填历史已失败 deployment 的 `error_message`（仅影响新失败记录；历史失败可依赖操作日志展示）。
- 不实现独立的日志归档、搜索或下载系统。
- 不主动启动 / 停止 / 重启开发服务器。
- 不修改 CI 流水线错误信息链路（除非复用同一 runner 时自然受益）。

## User scenarios

1. 用户部署应用时 Docker daemon 未运行，进入失败详情页后，在「错误信息」中能看到类似 `failed to connect to the docker API` / `dockerDesktopLinuxEngine` 的原因，而不是只有 `exit status 1`。
2. 用户在部署列表页扫到失败记录时，错误列能看到可识别的失败摘要（至少包含关键 stderr 或可读失败原因）。
3. 用户打开失败的部署详情页时，日志区能看到本次 compose 操作日志（含 Command、stderr、Command failed），即使容器从未成功启动。
4. 用户打开成功或运行中的部署详情页时，主日志区仍以容器日志为主，不因本次修复回退到始终展示操作日志。
5. 用户刷新失败详情页或通过分享链接进入时，错误原因与操作日志仍可从 API 读到，不依赖本地临时状态。

## Acceptance

- 命令执行失败写入 `error_message` 时，内容必须包含命令失败的真实输出（stderr/stdout 中的失败文本），不能只剩 `exit status N`。
- 若输出较长，`error_message` 至少保留尾部关键片段，且足以定位常见失败（daemon 未启动、镜像拉取失败、compose 配置错误等）；完整过程仍以操作日志文件为准。
- 部署详情页 `faulted` 状态：
  - 「错误信息」完整展示（允许换行，不以单行 truncate 隐藏正文）。
  - 日志区展示或可切换到本次**操作日志**（现有 `GET /api/cd/deployment/:id/logs`），使用户能看到 compose 完整输出。
- 部署 / 重启类在非失败主路径下，默认日志策略仍以容器日志为主，兼容既有「容器日志」需求。
- 部署列表页错误列仍展示 `error_message`；内容应比原先的 exit status 更有诊断价值。
- 相关 Go 测试覆盖 runner 或部署失败路径：失败时 error 文本包含命令输出，而不只是 exit status。
- 前端 lint/typecheck 与 Go fmt/vet/test 按项目约束通过。

## Open questions

1. **失败时日志区策略（需确认）**
   - A（推荐）：`faulted` 时主日志区自动切换为操作日志；成功/运行中仍用容器日志。
   - B：主日志区始终容器日志，另增加「操作日志」Tab/折叠区，失败时默认展开操作日志。
   - C：仅增强 `error_message`，详情页日志区不改。

2. **`error_message` 长度上限**
   - 推荐：截取尾部约 2–4KB（或最后 N 行），避免 SQLite 文本膨胀；完整内容仍在操作日志文件。
   - 是否有项目既有字段长度约束需要对齐？

3. **历史失败记录**
   - 确认：不回填历史 `error_message`；失败详情通过操作日志 API 展示即可。

## Decisions

- 使用轻量模式 / light：Requirement → Implementation → Verification。
- 后端必须修复错误捕获：不能只依赖 `exec.ExitError` 的 exit status 作为 `error_message`。
- 操作日志文件继续作为完整过程真相源；`error_message` 提供可检索/可列表展示的诊断摘要。
- 不改变成功路径下「容器日志为主」的既有产品决策；本次补齐失败路径可见性。
- 开放问题 1 若用户未另行指定，实现时默认采用 **A：faulted 自动切操作日志**。

## Risk

- 将大段命令输出写入 `error_message` 可能导致列表页展示过长；需要截断策略与 UI 换行/截断配合。
- 失败时切换操作日志，需避免与运行中容器日志自动刷新逻辑互相干扰。
- 若仅修后端不改前端，用户在 daemon 未启动等场景下仍可能在日志区看到空容器日志，诊断体验不完整。
- 并发/多次重试写入同一日志文件时，截取「最近输出」需明确以本次命令缓冲或文件尾部为准，避免拼错上下文。
