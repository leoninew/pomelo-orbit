# Bootstrap Logger 初始化收敛验证
最后修改时间: 2026-07-13 11:36:45

Review status: Accepted

## Requirement alignment

- 新增 `bootstrap.NewLogger(cfg)`，在 bootstrap 层接收完整应用配置并完成 logger 构造。
- `cmd/server` 与 `cmd/migrate` 仅导入 bootstrap/config，不再直接依赖 `internal/infrastructure/logger/logging`。
- 两个入口保留既有 logger 关闭 defer、关闭错误记录、CLI、signal、stdout/stderr 与退出码行为。
- `bootstrap.App` 恢复为接收注入 logger 的应用运行期编排对象，未承担 logger writer 关闭或默认 logger 状态恢复。

## Spec alignment

不适用：light 模式按 Requirement 核对。

## Plan alignment

不适用：light 模式按 Requirement 核对。

## Actual diff summary

- 新增 `internal/bootstrap/logger.go`，承载 logger 初始化实现及其测试。
- `cmd/server/main.go` 与 `cmd/migrate/main.go` 改为调用 `bootstrap.NewLogger(cfg)`。
- `internal/bootstrap/app.go` 与 `internal/bootstrap/app_test.go` 保持既有 App 注入和运行行为，撤回此前将 logger 生命周期并入 App 的实现。
- 删除 `internal/infrastructure/logger/logging` 及其测试，不保留兼容包装或重复实现。
- `docs/requirement/20260713-bootstrap-logger-lifecycle.md` 已按正确边界更新。
- `docs/verification/20260713-bootstrap-logger-lifecycle.md` 记录最终实现与实际检查结果。

## Expected versus actual changed files

| 预期文件 | 实际状态 | 说明 |
| --- | --- | --- |
| `internal/bootstrap/logger.go` | 已新增 | 统一日志构造入口。 |
| `cmd/server/main.go` | 已修改 | 改为依赖 bootstrap 日志构造。 |
| `cmd/migrate/main.go` | 已修改 | 改为依赖 bootstrap 日志构造。 |
| `internal/bootstrap/app.go` | 未修改 | 保持现有 App 注入模型。 |
| `internal/bootstrap/app_test.go` | 未修改 | 保持现有 App 行为测试。 |
| `internal/infrastructure/logger/logging/` | 已删除 | 不保留旧的日志基础设施包或兼容层。 |
| `internal/bootstrap/logger_test.go` | 已新增 | 迁移 logger 构造行为测试。 |
| `docs/requirement/20260713-bootstrap-logger-lifecycle.md` | 已新增 | Requirement 记录。 |
| `docs/verification/20260713-bootstrap-logger-lifecycle.md` | 已新增 | Verification 记录。 |

## Acceptance checklist

- [x] `internal/bootstrap/logger.go` 提供 `NewLogger(config.Config)`。
- [x] logger 初始化实现已迁移至 bootstrap，不保留 infrastructure wrapper 或兼容层。
- [x] cmd 不再直接导入日志基础设施包。
- [x] cmd 保持既有 logger 关闭和顶层错误处理模型。
- [x] `bootstrap.App` 不持有 logger 生命周期资源。
- [x] 未启动开发服务器。

## Test results

以下命令于 2026-07-13 执行并通过：

```text
go fmt ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./internal/bootstrap ./internal/infrastructure/logger/logging ./cmd/server ./cmd/migrate
go test ./cmd/... ./internal/...
git diff --check
```

结果：全部通过。`cmd/server` 和 `cmd/migrate` 当前没有测试文件；其余 `cmd/...` 和 `internal/...` 包测试均成功。

## Scope deviation

无。此前错误地将 logger 生命周期并入 `bootstrap.App` 的实现已撤回，最终范围仅为 `internal/bootstrap/logger.go` 的构造依赖收敛。

## Risks

- `NewLogger` 是薄包装；不得在其中复制或改变基础设施 logger 的校验、格式、输出与轮转行为。
- 本次不改变 `os.Exit` 与 defer 的既有交互，避免将范围扩大为进程退出模型重构。

## Incomplete items

无。

## Conclusion

日志初始化依赖已收敛至 `internal/bootstrap/logger.go`，最终实现符合 Requirement，全部既定 Go 检查通过。