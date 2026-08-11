# 日志界面自动刷新与滚底对齐
最后修改时间: 2026-08-11 22:24:29

Review status: Accepted

Mode: light

## Background

产品内运行日志界面：

| 页面 | 日志 | 启停刷新所依据 |
|---|---|---|
| `DeploymentDetail` | 操作日志（offset 追加）+ 容器日志（tail 覆盖） | Deployment **WorkStatus** |
| `PipelineRunDetail` | 页级详情轮询 + Stage 日志抽屉（offset 追加） | Run / Stage **WorkStatus**（Stage 日志以 API `is_complete` 表达终态） |
| `ServiceDetail` | 组件容器日志抽屉（tail 覆盖） | **特殊：有容器即可看日志并刷新**（不是 WorkStatus 终态语义） |

基准实现参考：`DeploymentDetail` 的 generation/Abort 轮询 + Monaco `revealLine` 滚底。

Compose 预览、脚本/挂载编辑、部署对话列表不在范围。

## Goal

统一运行日志界面的交互契约：

1. **打开即滚底**：进入页面 / 打开日志抽屉 / 编辑器 mount / 日志内容更新后，一律自动滚到最后一行。
2. **WorkStatus 类**：非结束自动刷新；结束不自动刷新。
3. **Service 类（特殊）**：只要有容器可看日志，打开抽屉即可自动刷新；**不**用 ServiceStatus / WorkStatus 终态去禁刷。
4. **加载完成不得停刷**：首屏/详情/日志「加载完成」不得打断已在进行的自动刷新；停刷仅限合法条件（见 Acceptance）。

并补齐实现缺口：

- 滚底：Monaco 实例缓存 + 内容更新后 `revealLine` + `@mount` 再滚一次。
- 竞态：generation + AbortController；delay/请求绑定 `signal`。
- 优先小 composable；禁止过度抽象。

## Status machine audit（约定对照）

产品约定见 `docs/product/cd-model.md` 与 `internal/common/constant/status.go`。

### WorkStatus（Deployment / PipelineRun / PipelineStageRun）

```text
waiting_to_run → running → ran_to_completion | faulted | canceled
                ↘ canceled（可从 waiting_to_run 直接取消）
```

| 分类 | 状态值 | 自动刷新？ |
|---|---|---|
| 非结束 | `waiting_to_run`, `running` | **是** |
| 结束（终态，不可逆） | `ran_to_completion`, `faulted`, `canceled` | **否** |

前端权威：`isComplete()`（`web/src/utils/status.ts`）。

#### StageRun 日志语义（澄清：一致，不是另起一套）

**结论：StageRun 日志已与 WorkStatus 终态语义一致，不是「状态语义不一致」。**

| 层级 | 判定方式 | 集合 |
|---|---|---|
| 前端工具 | `isComplete(status)` | `ran_to_completion` \| `faulted` \| `canceled` |
| Stage 日志 API | `is_complete` ← 后端 `status.WorkStatusIsComplete(stage.Status)` | **同一集合** |

- Stage 抽屉用 `is_complete` 停刷 = 用后端投影的 stage 终态，与约定一致。
- 与 Run 页级轮询（`isComplete(run.status)`）相互独立，互不替代。
- UI 色调里的 `skipped` 只是缺省展示/色调兜底，**不是**正式 WorkStatus；真正 stage 记录的停刷仍看 status / `is_complete`。

**Stage 抽屉现况缺口主要是：缺自动滚底、`delayAsync` 未绑 abort；不是状态语义错误。**

**WorkStatus 相关现况：**

| 界面 | 现况 | 状态语义 | 滚底 |
|---|---|---|---|
| Deployment 页级刷新 | 非终态启、终态停 | 符合 | 有 |
| Deployment 操作日志 | 随页级增量 | 符合 | 有 |
| Deployment 容器日志 | 终态后拉；运行中 waiting | 符合 | 有 |
| PipelineRun 页级轮询 | run 终态停 | 符合 | n/a |
| Pipeline Stage 日志抽屉 | 至 `is_complete` 停 | **符合** | **缺** |

### Service（特殊，不走 WorkStatus 终态）

Service 使用 `ServiceStatus`（`running` / `stopped` / `faulted`），但这是运行绑定 SoT，**不是**异步操作终态机。

**产品规则（用户确认）：有容器就可以看日志和刷新。**

| 规则 | 说明 |
|---|---|
| 可看日志 | 组件存在容器观测面时允许打开日志（现 UI 按组件入口打开） |
| 自动刷新 | 打开日志抽屉后自动刷新；**不**因 `stopped` / `faulted` 等 ServiceStatus 禁止刷新 |
| 停刷 | 用户暂停、关闭抽屉、离开页面；**不是** ServiceStatus「结束」 |
| 禁止 | 误用 `isComplete` 套在 Service 上 |

**现况：** Service 抽屉已在打开后自动刷新（与「有容器可刷」方向一致）；**缺滚底**，且请求未传 `signal`。

> 实现上「有容器」若需显式判定，优先沿用现有打开路径（用户从组件列表进日志即视为该组件可观测）；本任务不新增运行时探测 API，除非现有响应已有可用字段。

## Non-goal

- 不改日志 API、后端状态机或 tail/offset 语义。
- 不做 WebSocket / SSE。
- 不做「用户上滚后暂停跟随」智能 sticky。
- 不把 Compose 预览、脚本编辑、对话页纳入。
- 不为 Service 硬套 WorkStatus 终态。
- 不重写 Stage 日志的 `is_complete` 协议（已对齐约定）。

## User scenarios

1. 打开**非结束**部署详情：自动刷新操作日志 + 滚底；进入终态后停刷并按现逻辑处理容器日志。
2. 打开**已结束**部署详情：不启自动刷新；加载日志 + 滚底。
3. 打开 PipelineRun（非结束）：页级自动刷新；打开未完成 Stage 日志：轮询 + 滚底；`is_complete` 后停该抽屉轮询。
4. 打开已完成 Stage 日志：一次加载 + 滚底，不轮询。
5. 打开服务组件日志抽屉（有容器可观测）：自动刷新 + 滚底；不因 ServiceStatus 为 stopped/faulted 而拒绝刷新。
6. 首屏或日志首次加载完成时，**不得**因此停止已在进行的自动刷新。

## Acceptance

### 行为契约

- **滚底**：打开页面/抽屉、日志内容每次成功更新、Monaco `@mount` → 滚到最后一行（有内容时）。
- **自动刷新启停**：
  - **WorkStatus**（Deployment / PipelineRun / Stage）：`!isComplete(status)` 或 Stage `!is_complete` 时刷；终态 / `is_complete` 停。
  - **Service**：打开可观测组件日志抽屉即刷；不按 ServiceStatus 终态停刷；用户暂停 / 关抽屉 / 离页停。
- **停刷仅限合法条件**（见上）；**禁止**因「首次加载成功 / loading→done / 页面数据加载完成」而 stop 已在进行的自动刷新。

### 实现缺口

- `ServiceDetail`：补滚底；`getLogs`/`delay` 传 `signal`；**保持**打开抽屉可自动刷新（有容器可刷），不要改成「仅 running」。
- `PipelineRunDetail` Stage 抽屉：补滚底；`delayAsync(..., signal)`；**保持** `is_complete` 停刷（语义已正确）。
- `DeploymentDetail`：核对滚底与加载完成不停刷；终态停刷保持；可选接入共享工具。

### 工程

- `yarn --cwd web lint:fix` 与 `yarn --cwd web typecheck`。
- 无无关页面改动。

## Open questions

- 暂无需要用户确认的未决事项。  
- Stage 状态语义已澄清为一致；Service 特殊规则已按用户说明采纳。

## Decisions

| # | 决策 | 说明 |
|---|---|---|
| D1 | 打开即滚底 | 所有运行日志界面统一强制滚底 |
| D2 | WorkStatus 跟终态 | 非结束刷、结束不刷；`isComplete` / Stage `is_complete` |
| D3 | Stage 语义一致 | `is_complete` ≡ WorkStatus 终态集；缺口是滚底不是状态语义 |
| D4 | Service 特殊 | 有容器可看可刷；不按 ServiceStatus/WorkStatus 终态禁刷 |
| D5 | 加载完成不停刷 | 首包加载成功 ≠ 停刷条件 |
| D6 | 复用 | 优先小 composable；禁止过度抽象 |
| D7 | sticky | 不做用户上滚暂停跟随 |

## Risk

- Monaco `v-if` 反复 mount：mount 与内容更新都要滚底。
- 强制滚底打断上滚阅读（与现部署详情一致，可接受）。
- Service「有容器」若不做运行时探测，依赖入口即视为可观测；与当前 UI 一致。

## Implementation notes（light，无独立 Plan）

建议文件：

- `web/src/views/service/ServiceDetail.vue`（必改：滚底 + signal）
- `web/src/views/pipeline_run/PipelineRunDetail.vue`（必改：滚底 + abort delay）
- `web/src/views/deployment/DeploymentDetail.vue`（核对/可选接入）
- 可选 `web/src/composables/*` 小工具

验证（Verification 阶段）：lint/typecheck + 三页启停与滚底说明。

## User review notes

- 初版：对齐 Service/Pipeline 滚底与刷新实现意见。
- 补充：打开滚底；WorkStatus 非结束刷、结束不刷；加载完成不停刷。
- 澄清：StageRun 日志**不是**状态语义不一致，`is_complete` 已对齐 WorkStatus 终态；缺口是滚底。
- 用户确认：Service 特殊——有容器即可看日志和刷新，不按 Service 结束态禁刷。
- 用户确认进入实现：全站统一 `is_complete = ran_to_completion | faulted | canceled`，禁止各处各自拼条件。
