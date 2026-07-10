# CD 部署详情状态轮询修复验证
最后修改时间: 2026-07-10 13:07:57

Review status: Accepted

## Requirement alignment

按 `docs/requirement/20260710-cd-deployment-status-polling.md` 核对：部署详情页对所有非终态 deployment 进行 detail 自动刷新；`stop` 不请求容器日志；进入终态后停止自动刷新；卸载时终止刷新会话。

本次用户进一步明确：不支持同一组件实例内 `/cd/deployments/A → /cd/deployments/B` 的详情间跳转。因此最终实现使用 `onMounted(loadDeployment)` 完成单次页面初始化，不使用 route watcher 或路由守卫。

## Spec and plan alignment

不适用。轻量模式仅有 Requirement；实现按当前 API、当前 deployment 状态模型完成，未加入历史兼容层、fallback、别名或新旧逻辑并存。

## Actual diff summary

- `web/src/views/cd/DeploymentDetail.vue`
  - 将 detail 与容器日志刷新统一为单一页面级自动刷新会话。
  - 非终态 deployment 每 2 秒刷新 detail；非 stop 同时刷新 container logs；stop 不发 container-log 请求。
  - 使用 `AbortController` 与 generation 防止停止后的异步响应更新页面。
  - 取消操作直接使用 API 返回 deployment，禁用重复提交，并仅在 `waiting_to_run` / `running` 显示。
  - 对齐当前 i18n 展示，修复空 application ID 的无效链接。
  - 使用 `onMounted` 单次初始化，不监听 deployment route 参数。
- `web/src/api/cd/deployments.ts`
  - detail/container-log 请求支持传递 Axios request config，以供当前刷新会话中止请求。
- `web/src/utils/time.ts`
  - `delayAsync` 支持 `AbortSignal`。
- `web/src/utils/time.test.ts`
  - 覆盖普通延迟、已中止和等待期间中止。

## Expected vs actual changed files

预期修改的部署详情页、请求 API 和时间工具均已修改；新增的时间工具测试覆盖刷新等待取消。未修改后端、数据库、队列、路由定义、API 路径或生成协议。

## Acceptance checklist

- [x] `waiting_to_run` / `running` 的 stop deployment 自动刷新 detail 直到终态。
- [x] stop deployment 不请求 container logs，并展示不适用状态。
- [x] deploy/restart 非终态时刷新 detail 与 container logs。
- [x] `ran_to_completion`、`faulted`、`canceled` 后结束自动刷新。
- [x] 组件卸载时取消等待与当前 HTTP 请求。
- [x] 取消请求使用返回结果，避免取消成功后的额外 detail GET。
- [x] 不实现同组件 deployment ID 路由切换处理，符合本次明确范围。

## Test results

| Command | Result |
| --- | --- |
| `yarn --cwd web lint:fix` | 通过；0 error，5 条项目既有 import warning。 |
| `yarn --cwd web typecheck` | 通过。 |
| `yarn --cwd web test` | 通过：4 files，17 tests。 |
| `git diff --check` | 通过。 |

## Scope deviation

与最初审视计划相比，未实现 deployment ID route watcher、路由切换状态重置以及对应竞态测试。这是用户明确确认不存在 `/cd/deployments/A → /cd/deployments/B` 同组件跳转后的有意范围收缩，而非遗漏。

## Risks and incomplete items

未启动开发服务器或执行浏览器网络验收，遵循项目不主动启动开发服务器的约束。建议由用户在现有运行环境中确认：运行中的 deploy/restart 页面从 `loading` 转为实时日志，stop 页面只刷新 detail。

## Conclusion

前端静态检查、类型检查、完整 Vitest 套件和 diff 空白检查均通过；实现与已接受需求及后续范围收缩一致。