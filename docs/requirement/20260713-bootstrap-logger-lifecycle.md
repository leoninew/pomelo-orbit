# Bootstrap Logger 初始化收敛需求
最后修改时间: 2026-07-13 11:24:00

Review status: Accepted

## Background

日志初始化实现位于 `internal/infrastructure/logger/logging`，但 `cmd/server` 与 `cmd/migrate` 直接导入它并初始化 logger。与数据库、repository 等其他依赖构造路径相比，日志初始化不在 `internal/bootstrap`，导致其构造职责和应用组装边界不一致。

## Goal

将日志初始化实现迁移到 `internal/bootstrap/logger.go`，由 `bootstrap.NewLogger(config.Config)` 统一完成 logger 构造。服务与迁移命令仅依赖 bootstrap 来获取 logger，并继续在进程入口管理 logger 的关闭时机。

## Non-goal

- 不把 logger writer 的关闭、全局 `slog` 默认值恢复或其他资源生命周期状态并入 `bootstrap.App`。
- 不改变 `bootstrap.App` 现有的 `New(cfg, logger)` 注入接口、值语义或运行方法。
- 不改变现有 `slog`、`lumberjack`、stdout/file 双写、日志级别和轮转行为。
- 不改变命令入口的 CLI 参数、signal 处理、stdout/stderr、退出码和 migration status 输出。
- 不启动、停止或重启开发服务器。

## User scenarios

- 作为维护者，我需要从 bootstrap 层清晰看到日志基础设施的构造入口，与 `OpenDatabase` 等 bootstrap 装配函数保持一致。
- 作为命令入口维护者，我需要只依赖 `internal/bootstrap`，不直接依赖 logger 基础设施包。
- 作为运维者，我需要保持既有的日志文件关闭、错误记录和进程退出行为。

## Acceptance

- 新增 `internal/bootstrap/logger.go`，提供 `NewLogger(cfg config.Config) (*slog.Logger, func() error, error)`。
- `NewLogger` 在 bootstrap 内部实现既有的配置校验、目录创建、stdout/file 双写、轮转 writer、日志级别和默认 logger 设置行为。
- `cmd/server` 与 `cmd/migrate` 通过 `bootstrap.NewLogger(cfg)` 初始化 logger，不再直接导入 `internal/infrastructure/logger/logging`。
- logger 关闭仍由两个 cmd 入口的现有 defer 管理；关闭失败继续写入已初始化 logger。
- `bootstrap.App` 继续只保存注入的 `cfg` 与 `logger`，保持 `New(cfg, logger) App` 和既有运行行为。
- Go 格式化、vet、相关测试、完整 cmd/internal 测试和 `git diff --check` 均通过。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- `internal/bootstrap/logger.go` 是日志初始化和应用组合的一部分，承载 handler、writer、配置校验和默认 logger 设置。
- `cmd` 继续是进程资源关闭的边界；本次只迁移日志初始化实现，不改变 logger 的生命周期模型。
- 删除不再使用的 `internal/infrastructure/logger/logging` 包及其测试，logger 行为测试迁移至 `internal/bootstrap/logger_test.go`。

## Risk

- 迁移时必须保持 logger 的配置校验、日志格式与 writer 行为，避免改变现有运行结果。
- cmd 保留 `os.Exit` 失败路径的既有语义，本次不扩大范围重构进程退出模型。