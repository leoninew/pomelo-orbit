# backend-go 认证查询 SQLITE_BUSY 修复
最后修改时间: 2026-06-18 17:42:06

Review status: Accepted

## Background

流水线运行过程中，前端轮询 `/api/ci/run/{run_id}` 曾稳定返回 200；当 CI worker 在执行 stage、更新 `pipeline_run` / `stage_run` / `task` 状态时，请求偶发返回：

```json
{"detail":"Invalid token"}
```

日志显示真实原因不是 token 过期或签名错误，而是认证阶段查询当前用户时 SQLite 返回锁等待错误：

```text
load current user failed user_id=01KKX2YNPF6VJ9N7QYCWG61KVK error="load user ...: database is locked (5) (SQLITE_BUSY)"
```

当前 `authz.CurrentUser` 将 `UserById` 的所有错误都包装为 401 `Invalid token`。前端收到 401 后会清除 token 并跳转登录页，导致用户在流水线运行/完成时被错误踢出。

## Goal

1. 认证层正确区分认证失败与后端依赖失败：
   - token 格式错误、签名错误、过期：继续返回 401。
   - token 用户不存在或用户禁用：继续返回 401。
   - 数据库查询失败、SQLite busy、其他非用户不存在错误：不得返回 401，应返回 500 或 503。
2. 避免 `SQLITE_BUSY` 被前端当成登录过期，从而清除 token。
3. 改善 backend-go SQLite 初始化，降低 CI worker 写库与 HTTP 高频轮询读库撞锁概率。
4. 保持对外安全性：不向客户端暴露 token 签名细节或用户存在性细节。
5. 增加测试覆盖，防止回归。

## Non-goal

1. 不实现 refresh token 或会话续期。
2. 不修改前端统一 401 处理逻辑；后端不应错误返回 401 是本次根因。
3. 不替换 SQLite 为 MySQL，也不重构 task worker 并发模型。
4. 不改变 CI run、stage run、task 的业务状态机。
5. 不把数据库错误伪装为认证错误。

## User scenarios

1. **流水线运行中前端轮询详情**
   - CI worker 正在写入 stage/run 状态。
   - 如果认证查询用户遇到短暂 SQLite lock，应等待 busy timeout。
   - 如果最终仍失败，应返回 500/503，不应返回 401。
   - 前端不应因为该错误清除 token。

2. **token 真无效**
   - token 缺失、格式错误、签名错误或过期时仍返回 401。
   - 前端继续按现有逻辑跳登录。

3. **用户不存在或禁用**
   - token 对应用户不存在或状态不是 `enabled` 时仍返回 401。
   - 不向客户端泄露用户存在性细节。

4. **正常请求**
   - 当前用户查询成功且状态 enabled 时，现有接口行为不变。

## Acceptance

1. `authz.CurrentUser` 对 `UserById` 的非 `sql.ErrNoRows` 错误不再返回 401 `Invalid token`。
2. `sql.ErrNoRows` 或用户状态非 `enabled` 仍返回 401。
3. token verify 失败仍返回 401。
4. 数据库查询当前用户失败时记录 error 日志，并返回非 401 状态码，建议使用 503 `Authentication service unavailable` 或 500。
5. SQLite 初始化设置 busy timeout，避免短暂锁冲突立即失败。
6. SQLite 初始化启用 WAL，提升读写并发能力。
7. SQLite 初始化启用 foreign_keys，保持约束行为明确。
8. 增加 authz 测试覆盖：
   - store 返回普通数据库错误时不返回 401。
   - store 返回 `sql.ErrNoRows` 时返回 401。
   - 用户 disabled 时返回 401。
   - token 无效时返回 401。
9. 增加或调整 db 测试覆盖 SQLite PRAGMA 设置。
10. backend-go 相关测试和全量测试通过。

## Open questions

暂无需要用户确认的未决事项。按当前日志证据，根因明确：SQLite busy 被 authz 错误归类为 Invalid token。

## Decisions

1. 本次修复后端根因，不修改前端 401 处理。
2. 对外仍不泄露 token 校验细节；但服务端日志要区分数据库失败。
3. 数据库依赖失败使用非 401，避免触发前端清 token。
4. SQLite busy 处理放在 db 初始化层，不在业务查询处到处重试。

## Risk

1. WAL 在极少数网络文件系统或特殊 SQLite 环境中可能受限；本地开发和常规部署应可用。
2. busy timeout 只能缓解短暂锁冲突，不能掩盖长期写锁或多进程错误使用 SQLite 的问题。
3. 如果前端将所有非 2xx 都展示为错误，503 仍会在 UI 上提示失败，但不会清登录态。

## User review notes

- 用户指出：“流水线挂了就401了，这科学吗？”
- 日志确认问题发生在 `load current user failed ... SQLITE_BUSY` 后立刻返回 `Invalid token`。
- 用户已通过 `/specflow light 开始修复` 要求进入轻量修复流程；Requirement 标记为 Accepted 并进入 Implementation。
