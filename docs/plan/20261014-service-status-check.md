# Plan / 计划：服务状态检查
最后修改时间: 2026-10-14 11:12:46

## Review status / 审查状态

Accepted

## Intent basis / 意图依据

依据 [服务状态检查意图](../intent/20261014-service-status-check.md) 的 A01–A06。用户已要求记录后开始实现；本次进行 Implementation 及必要开发检查，整体 Verification 按后续授权推进。

## Technical decisions / 技术设计与决策

- 在既有 deployment usecase 内复用 Project 鉴权、Environment target resolver、local/SSH Runtime。一次在环境根目录执行 `docker ps --all --format '{{json .}}'`，从 JSON 的 Labels 读取 project/oneoff，读取 State/Status；模板不含内层引号，适用于 Windows PowerShell 5 的原生命令参数传递，不依赖 Service 目录或 Compose 文件。
- Service 全量读取使用 `DirectoryServices`，按 code 精确匹配容器。无容器或全部退出/创建为 stopped；全部运行且无 unhealthy 为 running；混合、异常、unhealthy 为 faulted。沿用现有三个 Service 状态，新建 Service 本身也是 stopped。
- 服务端保存短期观测快照，用随机 token 绑定操作者、Project、目标修订与各 Service 的数据库基线。每位操作者/Project 保留最新一轮；过期清理，不建立通用快照框架，不信任客户端提交的观测状态。
- 单数路由 `/api/service/status-check/preview` 返回记录和观测状态；`/api/service/status-check/confirm` 每次提交 token 与一个 service_id。确认不再查询 Docker。
- 预览绕开 HTTP 写事务，Docker I/O 设置超时。失败保留 Runtime 返回的命令输出（最多 8 KiB）及原始错误，交由 HTTP 错误日志记录。确认使用短请求事务和单条条件 SQL，匹配目标 ID/修订、Service code/版本/目录及其修订、原状态、当前部署指针，并拒绝当前活动任务。直接比较运行绑定字段，避免依赖各数据库不同的时间文本表示。零行报告冲突，前端继续后续条目。
- 状态检查只更新 Service status/updated_at，不修改 Deployment、目录、版本和组件声明。
- 页面业务弹窗放在 `web/src/views/service/components/`，复用 AppDialog、AppDialogActions、AppBadge、AppLoadingState、AppTooltip 等。表格沿用路由同步弹窗的固定列宽和布局，记录/容器状态复用服务列表的 AppBadge 与 appStatusTone，处理结果标签、失败原因在可聚焦标签的 Info 提示中展示。入口放在创建服务按钮前。加载预览后逐条确认，处理期间防止关闭，整轮结束通知列表刷新。按用户反馈移除副标题、检查时间、进度和一致性总结。

## Implementation steps / 实现步骤

1. 实现批量容器观测、快照及逐条确认，增加条件 SQL 和 Repository 方法。
2. 定义 proto 请求/响应、HTTP handler/route 与预览事务例外，使用既有工具生成代码。
3. 增加页面业务弹窗、API 方法、工具条入口与双语文案。
4. 更新产品/运行架构活文档，补充主流程和真实写入竞争的最小后端验证。
5. 运行固定开发检查，记录实际变更、结果与偏差。

## Files to change / 需修改文件

- `internal/application/deployment/{usecase,dto}`：批量观测与确认。
- `internal/repository/service.go`、`internal/repository/impl/sqlc/service/`、`sql/query/service/service.sql` 和 sqlc 生成物。
- `proto/orbit/v1/service/service.proto`、Go/TS 生成物、HTTP deployment handler、service routes、bootstrap 事务例外。
- `web/src/views/service/ServicePage.vue`、页面 components、`web/src/api/service/service.ts`、语言文件。
- `docs/product/cd-model.md`、`docs/architecture/cd-runtime.md` 与本任务记录。

## Tasks and coordination / 子任务与协调

不适用。单任务由主会话串行实现。

## Verification plan / 验证计划

- 最小后端验证：外部停止/启动的聚合、精确归属、一次查询及查询失败不写入、服务/目标变化与活动任务的条件写入。
- 前端：`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`。
- 后端：`task check`、`go test ./cmd/... ./internal/...`。
- 不主动启停开发服务器或容器，不扩展 UI、极端边界或性能测试。

## Risks / blockers / assumptions / 风险 / 阻塞项 / 假设

- 状态是检查时刻的快照，外部操作发生后仍需再次检查。
- 快照有期限；服务端重启不保留，界面通过重新检查恢复。
- 活动部署条目不能更新，失败反馈不影响其他条目。

## Rollback / recovery / 回滚 / 恢复

代码回退即可，无 schema 迁移。检查仅修改状态，下一次部署或重新检查会产生新的有效状态。

## User review notes / 用户审查记录

- 2026-10-14：依据「记录本任务 再开始」与此前实现范围授权，将意图与计划记为 Accepted，开始 Implementation。

## Implementation progress / 实现进展

- 2026-10-14：完成批量预览、服务端快照、条件 SQL 逐条确认、HTTP/proto、页面弹窗、工具条入口与双语文案；无迁移、无通用批处理组件。
- 2026-10-14：本地 PowerShell 5 复现内层引号丢失导致 `function "com" not defined`，改为标准 JSON 模板。从正在运行的本地 API 对用户报错的 Windows SSH Project 执行预览，成功返回三条受管服务，未执行容器启停。
- 2026-10-14：状态检查后端主流程、Docker 错误诊断、精确 project/oneoff 归属、活动任务和旧观测条件写入检查通过。完成实现和必要开发检查；整体 Verification 尚未进入。

## Development check results / 开发检查结果

- `yarn --cwd web lint:fix`、`yarn --cwd web typecheck`：最终表格调整后均通过；修改的前端文件 Prettier 检查通过。
- `task check`：通过，Go lint 为 0 issues；其后仅调整前端表格和过程文档，前端固定检查已重新通过。
- `go test ./cmd/... ./internal/...`：最终通过。首轮并行检查时既有 `TestCancellationTimeoutDoesNotBlockNewOperation` 超时，重新运行完整命令通过，未修改该测试或扩大本任务范围。该不稳定用例后来在独立的[后端治理](../verification/20261014-backend-test-architecture.md)中迁为虚拟时间端口测试，复跑通过未被视为修复依据。
- `git diff --check`：通过。
- 实际环境：Windows PowerShell 5 原模板失败可复现；标准 `{{json .}}` 模板通过。用户报错的 Windows SSH Project 经本地运行中的 HTTP API 预览成功返回三条受管服务。
- 未增加或运行 UI 测试套件；未启停开发服务器或容器。不存在未完成的实现项；结果反映查询时刻，外部后续操作仍需重新检查。
- 后续工具条顺序和状态标签调整后，前端固定检查、`task check`（0 issues）与完整 Go 测试再次通过。

## Follow-up / 用户后续反馈

- 状态检查入口移到创建服务前，记录/容器状态恢复为服务列表使用的 AppBadge 和 appStatusTone；保留路由同步的固定列宽与失败原因提示。
- 用户提供路由同步失败日志：全量预览读取 Gateway 工作区中属于其他 Project 的 casdoor 发布记录，原始归属错误未经分类而变成 500/internal_error。按用户要求修复：发现没有当前业务 Route 的发布 ID 时，范围排除有效的其他 Project 记录并记录日志，不读取其运行配置；当前业务 Route 的归属冲突仍返回 409，携带稳定错误码和 Web 中文提示，错误原因保留实际/预期归属。
- 最小回归检查证明本 Project 发布记录保留、其他 Project 文件不被读取或写入、当前 Route 冲突可识别且日志诊断与 UI 摘要分离。用户报错的 Project 经运行中的 HTTP API 重新预览成功返回四条路由。未执行路由发布或修改实际发布记录。
