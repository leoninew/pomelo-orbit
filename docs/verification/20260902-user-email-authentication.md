# 用户邮箱认证与关联约束调整验证
最后修改时间: 2026-09-02 16:50:24

Review status: Accepted

## Requirement alignment

- 登录已改为 `email + password`；用户名仍作为用户资料字段保存与展示。
- 用户邮箱为必填、格式校验并小写规范化的字段，数据库保留唯一索引。
- 用户引用物理外键已从三种驱动的 DDL 中移除；用户删除由业务层删除角色和项目成员关系，登录与对话历史保留。
- 三种 identity seed 的管理员凭据均为 `admin@lvh.me` / `admin@lvh.me`；本地 MySQL 已完成离线更新。
- 用户删除直接复用 HTTP 写请求事务切面；用户域不存在额外 `TransactionRunner`，密码直接按 6–36 校验。

## Spec alignment

不适用。该任务采用标准模式 / standard，未创建独立 Spec / 规格文档。

## Plan alignment

- DDL、SQLC schema、Proto、后端、前端和生成代码均已更新。
- 本地 MySQL 已更新管理员记录、`user.email` 非空约束和四个引用用户的外键。
- 计划的生成、全量测试、前端检查、数据库查询和 API 登录验证均已执行。

## Actual diff summary

- 数据层：三个驱动的用户、认证、项目成员、部署对话和 identity seed migration；SQLC schema/query 同步更新。
- 契约与后端：认证请求、用户 DTO/Usecase、用户和项目 Repository、HTTP mapper 与生成 Proto/SQLC 代码同步改为邮箱认证和逻辑关联。
- 前端：登录页、用户创建/详情/编辑页面及中英文文案改用邮箱字段，并限制密码输入为 6–36。
- 文档：补建本 feature 的 Requirement、Plan 和 Verification 过程文档。
- 机械修正：`web/src/utils/login-redirect.test.ts` 仅经 Prettier 格式化，无断言或行为变化。

## Expected vs actual changed files

预期范围与实际代码改动一致，涵盖 `sql/`、`proto/`、`internal/`、`web/` 和本 feature 过程文档。

工作区另有 `.codex/config.toml` 及既有 MCP/CSRF 过程文档改动，不属于本 feature 验收范围，未作为本次功能结果采信。

## Acceptance checklist

- [x] API、Proto 和前端使用邮箱登录/编辑字段。
- [x] `user.email` 在本地 MySQL 为 `NOT NULL` 且为唯一键。
- [x] 本地 MySQL 不存在任何引用 `user` 的外键。
- [x] 管理员记录为 `admin@lvh.me`，运行中 API 使用该凭据登录成功。
- [x] 用户删除在既有 HTTP 写请求事务中执行，用户域无 `TransactionRunner` / `validPassword` 残留，密码长度为 6–36。
- [x] SQLC、Proto、前端和后端检查全部通过。

## Test results

| Command or check | Result |
| --- | --- |
| `task sqlc` | Passed |
| `task proto` | Passed |
| `yarn --cwd web lint:fix` | Passed |
| `yarn --cwd web typecheck` | Passed |
| `task check` | Passed (`golangci-lint` 0 issues, frontend typecheck/lint/format passed) |
| `go test ./cmd/... ./internal/...` | Passed |
| `git diff --check` | Passed |
| Local MySQL schema/data query | Passed: admin email set, email non-null unique, no user foreign keys |
| `POST /api/auth/login` against `127.0.0.1:9021` | Passed: HTTP 200, bearer token returned |
| Final 6–36 password boundary checks | Passed: 36 accepted and 37 rejected in user usecase test |

## Missed or expanded scope

- 在用户明确要求完成 SpecFlow 验收后，补建了 Requirement 和 Plan 文档以形成完整 standard 流程记录。
- 不含在线迁移或旧用户名登录兼容层，符合已确认的破坏式收敛范围。

## Risks

- 历史 migration 已按明确授权原地修改；其他已初始化数据库不会自动获得这些结构和 seed 变更，仍需离线处理。
- 本次数据库验收仅覆盖本地开发 MySQL。

## Incomplete items

无。

## Conclusion

本 feature 的实现与已接受需求、计划一致，验收通过。
