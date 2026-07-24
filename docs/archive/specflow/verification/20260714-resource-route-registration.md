# HTTP 资源路由注册拆分验证
最后修改时间: 2026-07-14 14:38:34

Review status: Accepted

## Requirement alignment

已对照 `docs/requirement/20260714-resource-route-registration.md` 验证：CI/CD URL 命名空间保留给前端 API 使用，但后端路由注册不再以 CI/CD 人为聚合。

- `internal/api/http/routes/ci.go` 与 `cd.go` 已移除。
- `routes.go` 保留 Gin engine、中间件、健康检查、资源注册顺序和 fallback，并直接调用资源级 `Router` registrar。
- 每个资源路由文件自行从 `Router` 的已注入依赖创建 HTTP handler，并绑定该资源 endpoint。
- `bootstrap` 仍负责 concrete repository、infrastructure adapter 与 application service 的装配；`application` 未获得 Gin 依赖。

## Spec alignment

不适用：light / 轻量模式未创建 Spec。

## Plan alignment

不适用：light / 轻量模式直接按 Requirement 实施。

## Actual diff summary

- 删除按前端 URL 命名空间聚合的 `ci.go` 与 `cd.go`。
- 新增或拆分 CI 资源路由文件：`repository.go`、`template.go`、`build_stage.go`、`pipeline_run.go`、`snapshot.go`、`artifact.go`、`credential.go`。
- 新增或拆分 CD 资源路由文件：`application.go`、`application_extra.go`、`deployment.go`、`route.go`、`traefik_route.go`。
- 更新 `routes.go`，将 CI/CD 聚合调用替换为资源级注册调用，保留原有资源和 fallback 的全局先后顺序。

## Expected vs actual changed files

| 预期 | 实际 | 结论 |
|---|---|---|
| 删除 `ci.go`、`cd.go` | `ci.go` 已删除；`cd.go` 的内容迁移为 `application_extra.go` | 符合：旧 CD 聚合文件不再存在，历史检测为 rename 不影响架构边界 |
| 资源文件分别注册 endpoint | 12 个资源路由文件独立创建 handler 并注册 endpoints | 符合 |
| `routes.go` 负责 HTTP adapter 装配 | `routes.go` 直接调用资源 registrar | 符合 |
| 不改 bootstrap/application | 未修改这两个层，且无 Gin import | 符合 |
| 记录本次任务 | 新增 requirement 与本 verification 文档 | 符合 |

## Acceptance checklist

- [x] `ci.go` 与旧的 CD 聚合职责已移除；不存在 CI/CD 聚合注册文件。
- [x] 路由包中不存在 `registerCI`、`registerCD`、`registerCIRoutes` 或 `registerCDRoutes`。
- [x] 每个资源 registrar 为 `Router` 方法，并只绑定其资源 endpoints。
- [x] 现有 CI/CD endpoint 的 method、path、handler method 与注册顺序保持不变。
- [x] `internal/bootstrap` 和 `internal/application` 没有 Gin import。
- [x] 后端格式化、vet 与全量测试通过。

## Test results

```text
go fmt ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

结果：全部通过；`internal/api/http/routes` 可正常构建，完整 Go 测试套件通过。

附加静态核对：

- `git diff --check` 通过。
- `internal/api/http/routes` 中搜索 `register(CI|CD)` 无结果。
- `internal/api/http/routes/{ci,cd}.go` 无匹配文件。
- `internal/bootstrap` 与 `internal/application` 中搜索 Gin import 均无结果。

## Missed or expanded scope

- 范围内新增了 Requirement 与 Verification 过程文档。
- 未修改 handler、application service、repository、infrastructure、bootstrap、前端 API 调用或 API URL。

## Risks

- 这是结构性移动，后续新增 endpoint 时需继续遵守“资源文件拥有 handler 创建与 endpoint 声明”的边界，避免重新引入按前端 URL namespace 聚合的注册文件。
- 工作区中 `.claude/worktrees/` 为本次任务外的未跟踪目录，未纳入交付范围。

## Incomplete items

无。

## Conclusion

交付完成。路由注册已经按 HTTP 入站适配器中的资源边界组织，CI/CD 仅保留为 API URL 命名空间；依赖方向和 HTTP 契约保持不变。
