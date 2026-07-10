# Verification: internal 后两级目录结构对齐与架构收敛

最后修改时间: 2026-07-10 14:27:06

## Verification target

本次验证完成 light / 轻量模式下的以下已实施 issue：

1. **Issue 1**：`internal/common/civariable` 迁出 `common`；
2. **Issue 2**：`internal/application/<domain>` 重整为 domain-first 的职责子目录。
3. **Issue 3**：`internal/repository` 的持久化 port 与 SQLC/SQLX implementation 对齐。

对应 requirement：`docs/requirement/20260709-internal-directory-structure-alignment.md`（Review status: Accepted）。

## Requirement alignment

### Issue 1

- CI runtime/template variable 规则最终位于 `internal/application/ci/rule/civariable`，不再位于 `common`。
- application CI usecase 与 workflow CI activity 均直接依赖最终位置。
- 未保留旧路径、wrapper、alias 或兼容转发。

### Issue 2

| Requirement item | Actual result |
| --- | --- |
| application domain 的 usecase 与 DTO 分离 | auth、user、role、project、settings、CI、CD 均已落位到各自 `usecase/` 和 `dto/`。 |
| application-owned capability port 显式归位 | settings、CI、CD 使用各自 `port/`。 |
| CI 业务规则迁出通用层 | CI variable 位于 `ci/rule/civariable`。 |
| CI/CD 的当前 runner 归入职责目录 | 位于各自 `runner/`，仅代表 Issue 2 的目录归位。 |
| 外层调用点迁移 | HTTP handlers、HTTP server、bootstrap、workflow activities 和测试已使用新路径。 |
| 清理旧路径 | 审计未发现旧 flat application import；目标 domain 根目录无遗留 Go 源文件。 |
| DTO 不保留类型别名 | CD `TraefikRouterResp` 已改为独立 application DTO，并在 CD usecase 进行 model 到 DTO 映射。 |

此次验证未改变 API route、request/response contract、后台 task payload key、错误文案、数据库迁移或前端行为。

### Issue 3

| Requirement item | Actual result |
| --- | --- |
| repository root 表达持久化 port | 新增 user、role、project、CI、CD 与 CI/CD execution store；接口只依赖 context、model、基础类型和 repository 的分页值类型。 |
| application / workflow 不重复持久化 interface | 已移除 usecase 与 workflow activity 中的 persistence interface，统一使用 repository root port。 |
| impl 目录表达真实技术 | user、role、project、task 保持 `impl/sqlc`；直接 `sqlx` 的 CI/CD adapter 已迁至 `impl/sqlx`。 |
| bootstrap 负责选择与注入实现 | 移除 `impl/sqlc.Store` 与 `NewRepositoryStore`；HTTP/worker composition root 直接以 DB connection 构造 adapter。 |
| 不保留迁移兼容层 | 旧 port 定义、`impl/sqlc/{ci,cd}` 路径和 Store 包装均已删除；未保留 alias、wrapper 或转发。 |

Repository adapter 增加编译期 interface assertions。API route、请求/响应契约、task payload、migration 与前端均未改动；既有数据库错误 wrapping 和 `sql.ErrNoRows` 消费语义保持不变。

## Spec and plan alignment

不适用。当前任务使用 light / 轻量模式，按 requirement 的 Issue 2 实施结果进行验证。

## Actual diff summary

主体实现由提交 `b26db926` 完成：

```text
refactor(application): restructure domain subpackages into usecase/dto/port/runner
95 files changed, 7689 insertions(+), 6955 deletions(-)
```

本次收尾补充：

- 将 CD `TraefikRouterResp` 从 `model.TraefikRouter` 类型别名改为独立 application DTO；
- 在 `ListTraefikRoutes` 中完成 model 到 application DTO 的转换；
- 增加 integration assertion，确认返回 DTO 不与来源 model slice 共享元素；
- 更新 requirement 与 verification，使 Issue 2 的状态与已提交代码和验证结果一致。

## Expected vs actual changed files

| Expected | Actual |
| --- | --- |
| application domain-first 职责拆分 | 已完成：`usecase/`、`dto/`、`port/`、`runner/`、`rule/` 已按 domain 落位。 |
| 外层 import 与测试随迁移更新 | 已完成。 |
| 删除旧 flat 路径且不保留兼容层 | 已完成；本次全仓 import 与文件树审计未发现旧路径。 |
| DTO 与 model 分离 | 已完成；补齐 CD Traefik router DTO 的真实转换，未保留 type alias。 |
| 保持现有协议与业务行为 | 已完成；全量后端检查通过。 |

## Acceptance checklist

- [x] `common` 不再承载 CI variable 业务规则。
- [x] auth、user、role、project、settings、CI、CD usecase 已迁入各自 `usecase/`。
- [x] application DTO 已迁入各 domain 的 `dto/`。
- [x] settings、CI、CD application-owned capability interfaces 已归入 `port/`。
- [x] CI variable rule 已归入 `ci/rule/civariable`。
- [x] CI/CD 当前 runner 已归入各自 `runner/`。
- [x] CD Traefik router response 为独立 application DTO，而非 model 类型别名。
- [x] HTTP server、bootstrap、HTTP handlers、workflow activities 与测试已更新到新路径。
- [x] 旧 flat application import 与根目录 Go 源文件已清理；未保留 wrapper、alias、兼容转发或新旧 import 并存。
- [x] 既有 integration/unit/e2e 回归及后端质量检查通过。
- [x] 未修改 API route、request/response contract、task payload、数据库迁移或前端行为。

## Test results

| Command | Result |
| --- | --- |
| `go fmt ./cmd/... ./internal/...` | 通过。 |
| `./bin/golangci-lint fmt ./cmd/... ./internal/...` | 通过。 |
| `./bin/golangci-lint run ./cmd/... ./internal/...` | 通过，`0 issues.` |
| `go vet ./cmd/... ./internal/...` | 通过。 |
| `go test ./cmd/... ./internal/...` | 通过。 |
| `git diff --check` | 通过。 |
| 旧 flat application import / 根目录 Go 文件审计 | 通过，未发现遗留。 |

重点通过的相关包包括：

```text
internal/application/{auth,user,role,project,settings,ci,cd}/usecase
internal/api/http
internal/bootstrap
internal/workflow/activity/{ci,cd}
internal/test/e2e
```

## Issue 3 actual diff summary

- 新增 `internal/repository/{user,role,project,ci,cd}.go`，集中表达领域持久化和 execution store port。
- application auth/user/role/project/CI/CD usecase 与 workflow CI/CD activity 改为依赖 repository root port，删除本地重复 interface。
- CI/CD repository implementation 从错误的 `impl/sqlc` 迁入 `impl/sqlx`；保留真正使用 SQLC 的 user、role、project、task adapter。
- 删除 `repository/impl/sqlc/store.go` 与 bootstrap `NewRepositoryStore`；bootstrap 直接使用已建立的 `*sqlx.DB` 构造 adapter。
- 更新 bootstrap、e2e、CI/CD application integration tests 的 concrete adapter import；新增 adapter compile-time conformance assertions。

## Issue 3 acceptance checklist

- [x] repository root 不再仅有分页工具，已承载 user、role、project、CI、CD 及 execution persistence port。
- [x] application usecase 与 workflow activity 不再定义重复 persistence interface。
- [x] application-owned external capability port 继续位于 application，未误迁入 repository。
- [x] `impl/sqlc` 只保留实际 SQLC adapter；CI/CD 直接 SQLX adapter 已迁至 `impl/sqlx`。
- [x] DB/driver holder 不再伪装为 repository implementation；bootstrap 直接装配 adapter。
- [x] production application/workflow 不 import `repository/impl/...`。
- [x] 旧 interface、旧 CI/CD implementation path、Store wrapper 和旧 bootstrap factory 已删除，无兼容层。
- [x] repository 方法签名、数据库错误 wrapping、task contract 与 worker behavior 未改变。
- [x] API route、request/response contract、task payload、migration、前端行为未改变。

## Scope deviation

无产品范围扩张。Issue 2 收尾补齐了一个 DTO 边界；Issue 3 仅重整 persistence port、adapter 分类与 bootstrap composition，没有改变 repository 方法行为、runner、queue、HTTP adapter 或 SQLC 生成结构。

## Risks and incomplete items

以下事项不属于 Issue 2，继续保持未完成：

- [ ] Issue 4：`internal/queue` 与 `internal/worker` 关系重整。
- [ ] Issue 5：Traefik、Turnstile external adapter 分类。
- [ ] Issue 6：HTTP adapter 的 router/routes/binding/mapper/validator 结构。
- [ ] Issue 7：SQLC generated code 的结构结论与实现。
- [ ] Issue 8：workflow execution/runtime/definition 等职责重整。
- [ ] C1：将 concrete Docker/Shell process runner 收敛到 infrastructure execution adapter、由 bootstrap 注入；同时处理 CD status/log 的 direct `os/exec` 路径和 CI runner port 中的 workspace infrastructure type。
- [ ] C2：将 `sql.ErrNoRows` 从 application 用例边界收敛为稳定错误语义。
- [ ] C3：避免 application task port 直接暴露 queue task implementation contract。

未启动、停止或重启开发服务器。

## Conclusion

**Issue 2 与 Issue 3 均已完成并验证通过。** Issue 2 完成 application domain-first 职责子目录整理与 CD Traefik router DTO 边界；Issue 3 将持久化 contract 收敛至 repository root、按 SQLC/SQLX 实际技术归类 adapter、移除 DB/driver Store 包装并统一 bootstrap 装配。全量后端质量检查及相关回归测试通过。

严格的 process runner/infrastructure 收敛、SQL sentinel error 和 queue task contract 问题仍保留为 C1-C3 独立后续任务；HTTP adapter 边界和 SQLC 生成结构分别仍属于 Issue 6、Issue 7。
