# CD 部署详情展示执行命令与容器日志规格
最后修改时间: 2026-06-30 12:18:00

Review status: Accepted

## Requirement basis / 需求依据

依据 `docs/requirement/20260630-cd-deployment-detail-container-logs.md`：

- 部署详情页“基本信息”需要新增“执行命令”。
- 执行命令必须作为完整命令字符串持久化，并通过部署详情 API 返回，前端不临时拼接。
- 部署、强制重建部署、重启类操作的日志区域以容器日志为主，优先展示本次操作开始后的容器日志。
- 最近 N 行容器日志只作为无法按时间范围获取时的回退，并且需要明确提示可能包含历史输出。
- 停止操作详情页不展示实时容器日志，需要明确说明停止操作不展示实时容器日志。
- 容器日志默认开启自动刷新，并提供自动刷新 / 暂停刷新切换。
- `?from=application` 只作为返回来源，不作为操作方式、执行命令或日志策略的数据来源。

## Overview / 总体方案

本变更把部署详情页从“展示 deployment 执行日志”调整为“展示一次应用操作的关键信息和容器运行输出”。核心设计如下：

1. 在 `deployment` 表新增 `command_text` 字段，用于持久化完整执行命令字符串。
2. 创建 deployment 记录时写入本次操作预期执行的完整 Docker Compose 命令：
   - 普通部署：`docker compose -f docker-compose.yml up -d --remove-orphans --pull <policy>`
   - 强制重建部署：上述命令追加 `--force-recreate`
   - 重启：`docker compose -f docker-compose.yml restart`
   - 普通停止：`docker compose -f docker-compose.yml down`
   - 删除 volumes 停止：上述命令追加 `-v`
3. 后端 Deployment API response 增加 `command_text`。
4. 部署详情页“基本信息”增加“执行命令”，直接展示 API 返回的 `command_text`。
5. 部署详情页日志区域改为“容器日志”：
   - deploy / restart：读取应用容器日志，优先按 deployment `started_at` 起始时间过滤。
   - stop：不启动容器日志读取，显示停止操作不展示实时容器日志的说明。
6. 保留现有 deployment 操作日志写入能力用于后台排查，但它不再作为详情页默认主日志内容。

## Design decisions / 设计决策

### 1. 持久化完整执行命令字符串

采用新增 `deployment.command_text` 字段，而不是前端根据 `operation_type` 拼接命令。

原因：

- 执行命令是本次操作事实的一部分，详情页、刷新、分享链接和列表入口都需要稳定可查。
- 前端拼接容易和后端实际执行参数漂移。
- 强制重建、删除 volumes、image pull policy 等选项会影响命令字符串，持久化后能直接审计。

字段语义：

```text
command_text TEXT / LONGTEXT NOT NULL DEFAULT ''
```

新创建的 deployment 必须写入完整命令。历史 deployment 若字段为空，前端显示 `未记录`，不再尝试根据旧数据猜测。

### 2. 命令生成与实际执行参数同源

后端需要抽出命令构造函数，避免创建 deployment 时写入的 `command_text` 与执行层 `runner.Run` 参数分叉。

建议使用结构化结果：

```go
type composeCommand struct {
    Name string
    Args []string
}

func (c composeCommand) String() string
```

部署、重启、停止分别通过同一构造函数生成：

- 创建 deployment 时：调用 `.String()` 写入 `command_text`。
- 执行时：使用同一个 `Name` / `Args` 调用 `runner.Run`。

### 3. 容器日志接口复用应用上下文，但按 deployment 语义读取

新增部署维度的容器日志接口，避免前端直接调用应用日志接口后丢失“本次操作”语义：

```text
GET /api/cd/deployment/{deployment_id}/container-logs?tail=200
```

响应：

```json
{
  "logs": "...",
  "source": "since",
  "is_realtime_supported": true
}
```

字段含义：

- `logs`：容器日志文本。
- `source`：`since` 表示按本次 operation `started_at` 过滤；`tail` 表示回退到最近 N 行。
- `is_realtime_supported`：当前接口是否适合继续轮询刷新。

接口行为：

- 对 `deploy` 和 `restart`：调用 `docker compose -f docker-compose.yml logs --since <deployment.started_at RFC3339>`。
- 如果按 `--since` 读取失败，允许回退到 `docker compose -f docker-compose.yml logs --tail <N>`，并在 response 中标记 `source: "tail"`。
- 对 `stop`：返回空日志和明确状态，或前端不调用该接口；建议前端基于 `operation_type === 'stop'` 不调用接口，以减少无意义 Docker 调用。

### 4. 容器日志展示采用轮询而非无限 SSE

当前 deployment log 支持 SSE / offset 读取，但容器日志来自即时 `docker compose logs` 查询，不是一个已有 offset log 文件。为避免引入复杂流式协议，本需求第一版使用近实时轮询：

- 详情页加载后立即拉取一次容器日志。
- 详情页加载到运行中的 deployment 时默认开启自动刷新，按钮文案和状态参考 `/ci/run/{id}`：自动刷新中显示旋转图标和“自动刷新”，暂停后显示“暂停刷新”。
- 详情页加载到已完成、失败、取消等终态 deployment 时，只立即拉取一次容器日志，不默认启动自动刷新。
- 用户点击自动刷新按钮可暂停轮询；再次点击恢复轮询并立即刷新一次。
- 终态 deployment 的“不默认自动刷新”仅约束初始行为；用户后续点击自动刷新时，应允许持续轮询，不因 deployment 已进入终态而立即停止。
- 容器日志区域不保留独立的原“刷新”按钮；错误态“重试”直接触发一次容器日志拉取。
- 自动刷新期间每 2 秒刷新一次。
- 为避免重复追加和日志错位，前端每次使用接口返回的完整容器日志替换编辑器内容，而不是按 offset 追加。
- 页面卸载时停止轮询。

### 5. 停止操作不展示实时容器日志

停止类详情页日志区域显示说明，例如：

```text
停止操作不展示实时容器日志。请在应用详情页查看可用的历史日志。
```

标题不使用“部署日志”，避免误导。

### 6. 操作日志降级为后台排查信息

`ExecuteApplicationDeploy`、`ExecuteApplicationRestart`、`ExecuteApplicationStop` 仍可写入 deployment log 文件，用于内部排查命令执行错误。但 UI 默认不展示该日志，也不把它称为“部署日志”。后续如需要，可另开需求添加“执行日志”折叠面板或 debug 入口。

## Affected components / 影响组件

### 数据库 / migrations

需要新增迁移文件，不能修改已执行迁移：

- `internal/migrations/sqlite/v0.1.6__deployment_command_text.sql`
- `internal/migrations/mysql/v0.1.6__deployment_command_text.sql`

字段建议：

```sql
ALTER TABLE deployment ADD COLUMN command_text TEXT NOT NULL DEFAULT '';
```

MySQL 使用 `TEXT` 或 `LONGTEXT` 均可；命令字符串较短，`TEXT` 足够。

### 后端 model / repository

- `internal/repository/model/cd.go`
  - `Deployment` 增加 `CommandText string `db:"command_text"``。
- `internal/repository/cd/repository.go`
  - `CreateDeployment` insert 增加 `command_text`。
  - `ListDeployments` / `Deployment` select 增加 `command_text`。

### 后端 service

- `internal/service/cd/service.go`
  - `newApplicationDeployment` 或调用方支持传入 command text。
  - `DeployApplication` 根据 `ApplicationDeployInput.ForceRecreate` 和 app `ImagePullPolicy` 写入 command text。
- `internal/service/cd/application_extra.go`
  - `StopApplication` 根据 `removeVolumes` 写入 command text。
  - `RestartApplication` 写入 restart command text。
  - 新增或扩展 deployment container logs service 方法。
- `internal/service/cd/deployment_execution.go`
  - 抽取 deploy/restart/stop 命令构造函数并复用到实际执行。

### 后端 HTTP handler

- `internal/transport/http/handler/cd/handler.go`
  - `DeploymentResp` 增加 `command_text`。
  - `DeploymentResponse` 返回 `CommandText`。
  - 新增 route：`GET /api/cd/deployment/{deployment_id}/container-logs`。
  - handler 验证用户权限后调用 service 读取容器日志。

### 前端 API / types

- `web/src/types/cd/deployment.ts`
  - `Deployment` 或 `DeploymentDetail` 增加 `command_text`。
- `web/src/api/cd/deployments.ts`
  - 新增 `getContainerLogs(id, params)`。

### 前端页面

- `web/src/views/cd/DeploymentDetail.vue`
  - 基本信息中新增“执行命令”。
  - 日志标题改为“容器日志”。
  - deploy / restart 拉取 container logs。
  - stop 显示“不展示实时容器日志”的说明，不调用实时日志接口。
  - 当接口返回 `source: "tail"` 时显示提示：当前展示最近日志，可能包含本次操作前的历史输出。
  - 日志区域提供自动刷新 / 暂停刷新切换，默认开启自动刷新。
  - 移除原独立“刷新”按钮；错误态重试按钮调用容器日志拉取。

### 测试

- 后端 service 测试：验证不同操作写入正确 `command_text`。
- 后端 execution 测试：验证命令构造函数与执行参数一致。
- HTTP route 测试：验证 deployment detail 返回 `command_text`，container logs 接口权限和响应结构。
- 前端 typecheck 覆盖类型变更；如项目已有前端单测模式，可补充页面逻辑测试。

## Interfaces / 接口设计

### Deployment detail response

增加字段：

```json
{
  "id": "...",
  "operation_type": "deploy",
  "trigger_type": "manual",
  "command_text": "docker compose -f docker-compose.yml up -d --remove-orphans --pull missing --force-recreate"
}
```

### Container logs response

```http
GET /api/cd/deployment/{deployment_id}/container-logs?tail=200
```

成功响应：

```json
{
  "logs": "...",
  "source": "since",
  "is_realtime_supported": true
}
```

当回退为最近 N 行：

```json
{
  "logs": "...",
  "source": "tail",
  "is_realtime_supported": true
}
```

当停止操作不适用时，建议前端不调用接口；如果后端收到 stop deployment 的请求，可返回 400 validation：

```json
{
  "detail": "container logs are not available for stop deployments"
}
```

## Technical questions / 技术问题

暂无必须由用户确认的问题。Plan 阶段需要在代码层确认：

- MySQL `TEXT NOT NULL DEFAULT ''` 是否符合当前迁移执行器和 MySQL 版本限制；如果不支持 TEXT 默认值，应使用 `VARCHAR(1024) NOT NULL DEFAULT ''` 或 nullable `TEXT`。
- Docker Compose `logs --since <RFC3339>` 在目标运行环境中的兼容性；如果不稳定，按本 spec 回退到 `--tail` 并标记 source。
- 现有 `runApplicationCommand` 是否适合复用到 deployment container logs，或需要注入 runner 以便测试。

## Risks / 风险

- 新增迁移会影响现有数据库，需要确保 SQLite / MySQL 两套迁移一致。
- 持久化命令字符串必须和实际执行参数同源，否则会形成错误审计信息。
- 容器日志轮询如果每次取完整日志，日志较大时可能导致页面和后端压力，需要设置合理 tail 和刷新间隔。
- `docker compose logs --since` 可能仍返回多服务混合输出；这符合当前应用级 compose 视角，但页面应避免暗示是单容器日志。

## Alternatives / 备选方案

### 1. 前端根据 operation_type 临时拼接执行命令

拒绝。该方案不能保证与后端实际执行参数一致，且无法可靠表达强制重建、删除 volumes 等执行选项。

### 2. 只展示最近 N 行容器日志

不作为首选。最近 N 行可能包含历史输出，容易误导用户；仅作为 `--since` 不可用时的回退。

### 3. 继续展示 deployment 操作日志

不符合本次需求。用户明确认为操作日志意义不大，详情页应展示容器日志；操作日志保留为后台排查信息即可。

### 4. 为容器日志实现 SSE 流式接口

暂不采用。容器日志来自命令查询，不是当前 logstore offset 文件；第一版采用近实时轮询更简单，也便于控制输出量。

## User review notes / 用户审查记录

- 用户确认执行命令来源采用“持久化完整命令字符串”。
- 用户确认停止操作日志策略为“不展示实时容器日志”。
- 用户询问并确认容器日志应优先使用“本次操作开始后”的时间范围语义，最近 N 行只作为回退。