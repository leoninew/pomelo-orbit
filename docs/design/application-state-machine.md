# 应用状态机设计

## Application 状态

| 状态 | 值 | 含义 |
|------|-----|------|
| 未部署 | `undeployed` | 初始状态，或停止成功后 |
| 部署中 | `deploying` | 部署/重启任务执行中 |
| 部署成功 | `deployed` | 容器正常运行 |
| 部署失败 | `deploy_failed` | 部署/重启/停止失败，或取消 |

### 状态转换

```
                 部署触发                部署成功
  undeployed ──────────► deploying ──────────► deployed
                              │                    │
                              │ 部署失败/取消       │ 停止触发
                              ▼                    ▼
                        deploy_failed         undeployed (停止成功)
                              ▲                    │
                              │                    │ 停止失败
                              └────────────────────┘

  deployed ──重启触发──► deploying ──重启成功──► deployed
```

### 操作前置条件

| 操作 | 条件 |
|------|------|
| 部署 | `enabled` 且 `status != deploying` |
| 停止 | `status == deployed` |
| 重启 | `enabled` 且 `status == deployed` |
| 删除 | `status != deploying` 且 `status != deployed` |

## Deployment 任务状态

部署记录是应用生命周期中的任务单元，每次部署/停止/重启操作都会创建一条 Deployment 记录。

### 状态定义

| 状态 | 值 | 含义 |
|------|-----|------|
| 待运行 | `waiting_to_run` | 已创建，等待后台任务拾取执行 |
| 运行中 | `running` | 后台任务正在执行 Docker 操作 |
| 运行完成 | `ran_to_completion` | 执行成功 |
| 取消 | `canceled` | 用户主动取消 |
| 失败 | `faulted` | 执行过程中抛出异常 |

### 任务生命周期

```
API 创建 Deployment
      │
      ▼
 waiting_to_run ──────────────────────────────────► canceled (用户取消)
      │
      │ 后台任务拾取
      ▼
   running ──────────────────────────────────────► canceled (用户取消)
      │
      ├── 成功 ──► ran_to_completion
      │
      └── 异常 ──► faulted
```

- `waiting_to_run` 和 `running` 都可以被用户取消
- 任务结束状态（`ran_to_completion` / `faulted` / `canceled`）为终态，不可再变更

### 任务结果对 Application 的影响

| Deployment 结束状态 | Application 状态变化 |
|---------------------|---------------------|
| `ran_to_completion` | → `deployed`（部署/重启）或 `undeployed`（停止） |
| `faulted` | → `deploy_failed` |
| `canceled` | → `deploy_failed` |

## 操作执行模式

| 操作 | 模式 | 说明 |
|------|------|------|
| 部署 | 异步 | 返回 deployment_id，后台执行 |
| 重启 | 异步 | 返回 deployment_id，后台执行 |
| 停止 | 同步 | 等待完成后返回 |
