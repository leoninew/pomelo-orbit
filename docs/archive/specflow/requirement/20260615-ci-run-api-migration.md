# backend-go CI Run API 迁移需求

Review status: Accepted

## Background

`backend-go` 是 Python `backend` 的 Go 版本实现和改良。当前 Backend API Migration TODO 中 `/api/ci/run` 相关接口仍有未迁移项，用户要求按端点逐个检查和补齐，不要一开始读取所有文件。

本需求使用轻量模式 / light，只记录实现前必须明确的目标、边界、验收和风险。

## Goal

补齐 `backend-go` 中 Python backend 已存在的 `/api/ci/run` 相关 HTTP API，使前端和调用方可以在 Go 后端中完成流水线运行相关查询和操作。

当前待检查范围以 `docs/todo.md` 中 CI Run 小节为准：

- `GET /api/ci/run`
- `GET /api/ci/run/{run_id}`
- `GET /api/ci/run/{run_id}/artifacts`
- `GET /api/ci/run/{run_id}/stages/{stage_run_id}/log`
- `POST /api/ci/run/{run_id}/cancel`
- `POST /api/ci/run/{run_id}/retry`

实施时逐个端点检查 Python backend 行为和 backend-go 现状，已存在且符合行为的端点只补测试或记录，不重复实现。

## Non-goal

- 不新增 Python backend 不存在的 CI Run HTTP API。
- 不做跨模块的大范围重构。
- 不修改已执行数据库迁移。
- 不修改前端，除非实现过程中发现现有前端调用与 Python contract 明确冲突且必须同步修复。
- 不处理 `/api/ci/template`、`/api/ci/artifact`、`/api/ci/credential`、`/api/ci/snapshot` 等已单独处理过的范围。
- 不执行 git 写操作，除非用户明确授权。

## User scenarios

- 用户打开流水线运行列表，backend-go 能返回分页运行记录。
- 用户打开某次流水线运行详情，backend-go 能返回 run 基本信息、变量快照、stage 运行信息等 Python 已有响应字段。
- 用户查看某个 stage run 日志，backend-go 能返回对应日志内容或 Python 对齐的响应。
- 用户查看某次 run 产出的 artifacts，backend-go 能返回该 run 的制品列表。
- 用户取消或重试 run 时，backend-go 按 Python 已有行为处理或在能力未实现时明确返回一致错误。

## Acceptance

- 逐个检查 `/api/ci/run` 相关 Python endpoint，并在 backend-go 中补齐缺失的同 method/path 路由。
- 响应字段、状态码、错误语义尽量对齐 Python backend；若 backend-go 已有改良约定，优先保持项目内一致并记录原因。
- 对不存在的 run/stage/artifact 等资源返回合理 404。
- 对需要项目权限的 run 操作执行项目成员校验。
- 添加或补齐 backend-go route tests，覆盖成功路径、鉴权、关键错误路径。
- 更新 `docs/todo.md` 中 Backend API Migration TODO 的 CI Run 小节状态和统计。
- 运行 backend-go 相关测试并通过。

## Open questions

暂无必须阻塞实现的问题。取消和重试接口可能依赖 backend-go 当前 worker/task 执行模型；实施时需要先窄范围检查 Python 行为与 Go 现状，再决定是完整实现、复用已有能力，还是按当前能力返回明确错误。

## Decisions

- 采用轻量模式 / light。
- 本轮只进入 Requirement / 需求阶段，不直接写产品代码。
- 按用户要求，Implementation 阶段继续逐端点读取相关文件，不做一次性全量扫描。
- 如果 Python backend 没有某个 CRUD 类 endpoint，不在 backend-go 中凭空新增。

## Risk

- `cancel` / `retry` 可能涉及后台任务状态机，存在与当前 backend-go executor/worker 能力不一致的风险。
- `GET /api/ci/run/{run_id}/stages/{stage_run_id}/log` 可能涉及日志存储字段或运行时日志文件，需以 Python 行为和 Go 现有模型为准。
- 当前工作区已有前序 artifact/credential/snapshot 迁移和 `Makefile` 的未提交改动，实施时需要避免混淆本次 run API diff；`Makefile` 不属于本需求范围。

## User review notes

- 待用户 review。
