# 用户邮箱认证与关联约束调整计划
最后修改时间: 2026-09-02 16:50:24

Review status: Accepted

## Implementation steps

1. 调整三种驱动的用户 DDL、SQLC schema 和 identity seed：邮箱非空、保留唯一索引、移除引用用户的物理外键。
2. 更新用户、认证、项目成员的 DTO、Repository、Usecase 和 Proto，使邮箱成为登录标识并在业务层处理用户删除关联。
3. 更新登录页和用户管理页面的邮箱输入、校验及文案。
4. 重新生成 Proto 和 SQLC 代码。
5. 离线更新本地 MySQL：设置管理员邮箱和密码哈希、修改邮箱非空约束、移除四个用户引用外键。
6. 复用 HTTP 写请求事务切面，不在用户域引入额外事务抽象；密码长度统一限制为 6–36。

## Files to change

- `sql/migration/{mysql,postgres,sqlite}/`、`sql/schema/`、`sql/query/`
- `proto/orbit/v1/`、`internal/gen/`、`web/src/gen/`
- `internal/application/{auth,user,project}/`、Repository、HTTP handler 与 bootstrap
- `web/src/views/auth/Login.vue`、用户管理视图和国际化文案
- 本 feature 的 requirement、plan、verification 文档

## Verification plan

- 复核实际 diff 与需求范围，排除用户个人 `.codex/config.toml` 改动。
- 运行 `task sqlc`、`task proto`、`go test ./cmd/... ./internal/...`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`、`task check` 和 `git diff --check`。
- 查询本地 MySQL 管理员、邮箱列与引用用户的外键，并以运行中的 API 验证管理员登录。

## Assumptions and risks

- 验收对象是本地开发 MySQL 库；其他已初始化数据库不在本次自动更新范围。
- 用户删除仅通过 HTTP API 暴露，因此由请求事务切面保证跨 Repository 删除的原子性。

## Rollback

本次为用户明确要求的破坏式设计收敛，不提供运行时兼容回退；在提交前可通过 Git 还原工作区改动，本地数据库需按离线变更前的备份恢复。
