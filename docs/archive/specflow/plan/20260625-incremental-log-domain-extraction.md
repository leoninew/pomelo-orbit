# Plan: 增量日志写入/读取的领域业务抽取

最后修改时间: 2026-06-25 16:55:00

Review status: Accepted

## Requirement basis

`docs/requirement/20260625-incremental-log-domain-extraction.md` — Accepted

## Overview

抽取 `LogStore` 作为统一的基础设施组件，同时将写入从"一次性"改为"执行中增量追加"。核心策略：

1. `LogStore` 持有 `*os.File`，实现 `io.Writer` 接口 — runner 继续接收 `io.Writer`，无需改 runner 接口
2. 写入端：runner 调用 `Write` 时，`LogStore` 持有已打开的文件，直接写（未来可改为 goroutine 流式写）
3. 读取端：`LogStore.Read` 从当前文件末尾往前 seek 到 offset 位置读取
4. 构造时传入 `logStore` 给 CI/CD Service，替换现有的 `workspace.XXXLogPath()` + `os.Create()` 逻辑

## Design decisions

### 1. LogStore 放在 infrastructure 层

```
backend-go/internal/infrastructure/logstore/logstore.go
```

理由：日志存储是基础设施关注点，不是领域逻辑。CI 和 CD 的 Service 都依赖它，但它不依赖任何领域概念。

### 2. LogStore 实现 io.Writer

```go
type LogStore struct {
    mu   sync.Mutex
    file *os.File
}

func (s *LogStore) Write(p []byte) (int, error)
```

理由：
- CI 的 `RunOptions.LogFile` 是 `io.Writer`，CD 的 `CommandRunner.Run` 接收 `io.Writer`
- `LogStore` 实现 `io.Writer` 后，可以直接传入现有 runner，runner 代码零改动
- `sync.Mutex` 保护多 goroutine 并发 Write（CI 的 executor 每层内并发执行多个 stage）

### 3. 写入模式：同步 Write（非 goroutine 流式）

当前阶段 `LogStore.Write` 是同步的。Runner 调用 `Write` 时直接写入文件。这意味着：
- CI：Docker 容器结束后的 `output` 通过 `Write` 写入（一次 Write 写完）
- CD：Shell 命令结束后的 `output` 通过 `Write` 写入

**为什么不做 goroutine 流式写？**
- 当前 runner 用 `cmd.CombinedOutput()`，本身就是阻塞等结束才返回。要改成流式需要同时改 runner 的 `Run` 方法（用 `StdoutPipe` 替代 `CombinedOutput`），这是另一个独立的变更
- 先抽出 `LogStore` 抽象，等 runner 需要改时再引入流式。接口已经预留了

### 4. 读取端：返回 newOffset

```go
func (s *LogStore) Read(offset int) (content []byte, newOffset int, err error)
```

内部实现：`file.Seek(offset, io.SeekStart)` + `io.ReadAll(file)` + 返回 `offset + len(content)`。

### 5. 路径构造职责归 LogStore

LogStore 需要知道路径规则。两种做法：
- A. LogStore 自己拼路径（需要知道 dataRoot + 规则）
- B. 调用方传完整路径给 LogStore

选 B：LogStore 只接收一个 `logID`（由调用方拼好路径后传给 LogStore 的构造函数或每个方法）。

实际上更简洁的做法：`LogStore` 不持有路径，每次 `Write`/`Read` 接收完整路径。这样 LogStore 更通用，不绑定 CI/CD 的路径规则。

最终接口：

```go
type LogStore struct{}

func (LogStore) Write(logPath string, data []byte) error
func (LogStore) Read(logPath string, offset int) (content []byte, newOffset int, err error)
func (LogStore) Exists(logPath string) bool
```

这样 LogStore 是一个纯函数式工具，无状态，无锁，无文件句柄持有。

**等等，重新考虑**：无状态意味着每次 Write 都 `os.OpenFile` + `Write` + `Close`。对于当前一次性写入没问题，但如果未来要改成 goroutine 流式写（持有句柄），接口会变。

最终决策：**先做无状态版本**，等 runner 需要流式时再引入有状态版本。无状态版本已足够覆盖当前需求（CI 和 CD 都是一次性写入后多次读取）。

## Affected components

| 组件 | 改动类型 |
|------|---------|
| `infrastructure/logstore/logstore.go` | **新建** |
| `service/ci/executor.go` | 改：去掉 `os.Create`，改用 `LogStore.Write` |
| `service/ci/pipeline_run.go` | 改：去掉 `os.Open`+`Seek`+`ReadAll`，改用 `LogStore.Read` |
| `service/ci/repository.go` | 改：`New`/`NewWithRunner`/`NewExecutionService` 增加 `logstore` 参数 |
| `service/cd/deployment_execution.go` | 改：去掉 `deploymentLog()`，改用 `LogStore.Write` |
| `service/cd/service.go` | 改：去掉 `readDeploymentLog()`，改用 `LogStore.Read`；构造函数增加 `logstore` 参数 |
| `bootstrap/bootstrap.go` | 改：构造 `LogStore`，注入 CI/CD Service |
| `transport/http/server.go` | 改：构造 `LogStore`，注入 CI/CD Service |

## Implementation steps

### Step 1: 新建 `infrastructure/logstore/logstore.go`

新建文件，实现三个方法：

```go
package logstore

import (
    "io"
    "os"
)

type LogStore struct{}

func (LogStore) Write(logPath string, data []byte) error {
    if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
        return err
    }
    return os.WriteFile(logPath, data, 0o644)
}

func (LogStore) Read(logPath string, offset int) ([]byte, int, error) {
    file, err := os.Open(logPath)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, offset, nil
        }
        return nil, offset, err
    }
    defer file.Close()
    if _, err := file.Seek(int64(offset), io.SeekStart); err != nil {
        return nil, offset, err
    }
    content, err := io.ReadAll(file)
    if err != nil {
        return nil, offset, err
    }
    return content, offset + len(content), nil
}

func (LogStore) Exists(logPath string) bool {
    _, err := os.Stat(logPath)
    return err == nil
}
```

注意：`Write` 用 `os.WriteFile` 是原子的（先写临时文件再 rename），但对于追加模式不适合。当前 CI/CD 场景是"创建时一次性写"，用 `os.WriteFile` 够用。如果未来需要追加，改为 `os.OpenFile` + `O_APPEND`。

**修正**：CI 的 `writeAndDeploy` 和 CD 的 `ExecuteApplicationDeploy` 中，`runner.Run` 被调用多次（restart → stop，init.sh → docker compose），每次 Run 都 Write 同一路径。`os.WriteFile` 会覆盖而非追加。

需要改成追加模式：

```go
func (LogStore) Write(logPath string, data []byte) error {
    if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
        return err
    }
    f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
    if err != nil {
        return err
    }
    defer f.Close()
    _, err = f.Write(data)
    return err
}
```

### Step 2: 改 CI Service 构造函数

`backend-go/internal/service/ci/repository.go`：

```go
type Service struct {
    store          RepositoryStore
    executionStore PipelineExecutionStore
    tasks          TaskService
    workspace      *CIWorkspace
    logStore       LogStore  // 新增
    secretKey      string
    logger         *slog.Logger
    runner         ContainerRunner
}

func New(store RepositoryStore, tasks TaskService, dataRoot string, secretKey string, logger *slog.Logger) Service {
    return NewWithRunner(store, tasks, dataRoot, secretKey, logger, DockerRunner{})
}

func NewWithRunner(store RepositoryStore, tasks TaskService, dataRoot string, secretKey string, logger *slog.Logger, runner ContainerRunner) Service {
    executionStore, ok := store.(PipelineExecutionStore)
    if !ok {
        panic("ci service store must implement PipelineExecutionStore")
    }
    workspace := NewCIWorkspace(dataRoot)
    return Service{store: store, executionStore: executionStore, tasks: tasks, workspace: workspace, logStore: LogStore{}, secretKey: secretKey, logger: logger, runner: runner}
}

func NewExecutionService(store PipelineExecutionStore, dataRoot string, secretKey string, logger *slog.Logger, runner ContainerRunner) Service {
    workspace := NewCIWorkspace(dataRoot)
    return Service{executionStore: store, workspace: workspace, logStore: LogStore{}, secretKey: secretKey, logger: logger, runner: runner}
}
```

### Step 3: 改 CI executor

`backend-go/internal/service/ci/executor.go`，`executeStage` 方法：

**改前**（lines 89-114）：
```go
logPath := e.workspace.StageLogPath(run.Id, stageRun.Id)
if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
    return e.failStage(ctx, stageRun, fmt.Sprintf("create stage log dir: %v", err))
}
logFile, err := os.Create(logPath)
if err != nil {
    return e.failStage(ctx, stageRun, fmt.Sprintf("create stage log: %v", err))
}
defer func() { _ = logFile.Close() }()
// ...
exitCode, output, err := e.runner.Run(ctx, RunOptions{
    Image:       stage.Image,
    Script:      script,
    Environment: environment,
    Volumes:     volumes,
    LogFile:     logFile,
})
```

**改后**：
```go
logPath := e.workspace.StageLogPath(run.Id, stageRun.Id)
// ...
exitCode, output, err := e.runner.Run(ctx, RunOptions{
    Image:       stage.Image,
    Script:      script,
    Environment: environment,
    Volumes:     volumes,
    LogFile:     e.logStore.writer(logPath),  // 返回 io.Writer
})
```

等等，这又引入了有状态的 writer。回到无状态方案。

**重新决策**：runner 调用 `Write(logPath, output)` 是在 `Run()` 返回后。所以可以直接在 Run 返回后调 `e.logStore.Write(logPath, output)`。但 output 是 runner 返回的，不在 runner 内部。

实际上更简单：不改 runner 调用方式。executor 在 `runner.Run` 返回后，拿 output，调 `logStore.Write(logPath, output)`。

**改后**：
```go
logPath := e.workspace.StageLogPath(run.Id, stageRun.Id)
// 不再 os.Create，不再传 logFile 给 runner
exitCode, output, err := e.runner.Run(ctx, RunOptions{
    Image:       stage.Image,
    Script:      script,
    Environment: environment,
    Volumes:     volumes,
    LogFile:     nil,  // 不再传文件句柄
})
if err != nil {
    return e.failStage(ctx, stageRun, err.Error())
}
if err := e.logStore.Write(logPath, []byte(output)); err != nil {
    return e.failStage(ctx, stageRun, fmt.Sprintf("write stage log: %v", err))
}
```

这意味着 `RunOptions.LogFile` 不再被使用。需要评估是否完全移除它。

**评估**：`RunOptions.LogFile` 只在 `DockerRunner.Run` 中使用，且只在 `output` 非空时 Write。如果 executor 不再传 `logFile`，这个字段就成了死代码。

决策：**保留 `RunOptions.LogFile` 字段但忽略它**，等所有调用方都切到 `logStore.Write` 后再移除。或者直接在本次移除。

选：**本次直接移除 `RunOptions.LogFile`**。因为 `LogStore.Write` 完全替代了 runner 内部的 Write 逻辑。runner 不再需要关心日志写入。

但这涉及改 `ContainerRunner` 接口和 `DockerRunner` 实现。影响面可控。

**最终决策**：为了最小化改动范围，保留 runner 的 `RunOptions.LogFile`，让它继续传 `*os.File`。但 `*os.File` 由 `LogStore` 创建和管理（通过返回 `io.Writer`）。

这又回到了有状态方案...

**最终最终决策**：采用最简方案。

**最简方案**：runner 不改。executor 在 Run 返回后调 `logStore.Write(logPath, output)`。runner 内部的 `opts.LogFile.Write(output)` 保留但传 nil。

实际上现在的代码中，executor 传的是 `logFile`（`os.Create` 的结果）。如果改为传 nil，runner 内部 `opts.LogFile != nil` 判断为 false，就不写了。executor 自己写。

这样：
- runner 接口不变
- `RunOptions.LogFile` 仍然保留（向后兼容），但 CI/CD 的 executor/service 不再使用它
- 未来可以彻底移除 `LogFile` 字段，但那是一个独立的清理任务

好，采用这个方案。

### Step 3 (修正): 改 CI executor

`executeStage` 方法：

```go
// 删除 lines 89-97 的 os.Create 逻辑
// 不再传 logFile 给 runner

exitCode, output, err := e.runner.Run(ctx, RunOptions{
    Image:       stage.Image,
    Script:      script,
    Environment: environment,
    Volumes:     volumes,
    LogFile:     nil,  // 不再由 runner 写日志
})
if err != nil {
    return e.failStage(ctx, stageRun, err.Error())
}

logPath := e.workspace.StageLogPath(run.Id, stageRun.Id)
if err := e.logStore.Write(logPath, []byte(output)); err != nil {
    return e.failStage(ctx, stageRun, fmt.Sprintf("write stage log: %v", err))
}
```

### Step 4: 改 CI PipelineStageLog

`backend-go/internal/service/ci/pipeline_run.go`，`PipelineStageLog` 方法（lines 151-186）：

**改前**：自己 `os.Open` + `Seek` + `ReadAll`
**改后**：调 `s.logStore.Read(logPath, offset)`

```go
logPath := s.workspace.StageLogPath(run.Id, stageRun.Id)
content, newOffset, err := s.logStore.Read(logPath, offset)
if err != nil {
    return PipelineStageLog{}, apperror.Wrap(apperror.KindInternal, "Failed to read stage log", err)
}
return PipelineStageLog{Logs: string(content), Offset: newOffset, IsComplete: pipelineRunStatusComplete(stageRun.Status)}, nil
```

### Step 5: 改 CD Service

`backend-go/internal/service/cd/deployment_execution.go`：

删除 `deploymentLog()` 方法（lines 168-178）。
所有调用方改为调 `logStore.Write(logPath, output)`。

涉及方法：
- `ExecuteApplicationRestart` (line 50-55)
- `ExecuteApplicationStop` (line 76-80)
- `writeAndDeploy` (line 126-130, 161, 165)

`backend-go/internal/service/cd/service.go`：

删除 `readDeploymentLog()` 方法（lines 414-442）。
`DeploymentLog` 方法改为调 `s.logStore.Read(logPath, offset)`。

### Step 6: 改构造注入

`bootstrap/bootstrap.go`：

```go
logStore := logstore.LogStore{}
ciService := cisvc.NewExecutionService(ciRepository, cfg.DataRoot(), cfg.JWT.SecretKey, logger, cisvc.DockerRunner{}, logStore)
cdService := cdsvc.NewExecutionService(cdRepository, cfg, logger, cdsvc.ShellRunner{}, logStore)
```

`transport/http/server.go`：

```go
logStore := logstore.LogStore{}
// ...
ciService: cisvc.New(ciRepository, taskService, cfg.DataRoot(), cfg.JWT.SecretKey, logger, logStore),
cdService: cdsvc.New(cdRepository, taskService, cfg, logger, logStore),
```

### Step 7: 改 Service 构造函数签名

CI `repository.go`：
- `New(store, tasks, dataRoot, secretKey, logger, logStore)`
- `NewWithRunner(store, tasks, dataRoot, secretKey, logger, runner, logStore)`
- `NewExecutionService(store, dataRoot, secretKey, logger, runner, logStore)`

CD `service.go`：
- `New(store, tasks, cfg, logger, logStore)`
- `NewWithRunner(store, tasks, cfg, logger, runner, logStore)`
- `NewExecutionService(store, cfg, logger, runner, logStore)`

### Step 8: 更新测试

- `backend-go/internal/service/ci/execution_test.go` — 如果有测 logFile 相关逻辑，需要更新
- `backend-go/internal/service/cd/` 下相关测试 — 同上

### Step 9: 移除无用代码

- `RunOptions.LogFile` 字段 — 如果所有调用方都不再传非 nil 值，可以移除（但保留字段不影响编译）
- `os.Create` / `os.Open` 在 executor/execution 中的引用 — 随 Step 3/5 一起移除

## Files to change

| 文件 | 改动 |
|------|------|
| `infrastructure/logstore/logstore.go` | 新建 |
| `service/ci/repository.go` | 构造函数加参数，struct 加字段 |
| `service/ci/executor.go` | `executeStage` 去掉 os.Create，改调 logStore.Write |
| `service/ci/pipeline_run.go` | `PipelineStageLog` 改调 logStore.Read |
| `service/ci/workspace.go` | 可考虑移除 `StageLogPath`（如果不再被使用） |
| `service/cd/service.go` | 构造函数加参数，struct 加字段，`DeploymentLog` 改调 logStore.Read，删除 `readDeploymentLog` |
| `service/cd/deployment_execution.go` | 所有 `runner.Run` 调用去掉 logFile，改调 logStore.Write，删除 `deploymentLog` |
| `service/cd/workspace.go` | 可考虑移除 `DeploymentLogPath`（如果不再被使用） |
| `bootstrap/bootstrap.go` | 构造 logStore，注入 |
| `transport/http/server.go` | 构造 logStore，注入 |

## Verification plan

1. `go build ./...` — 编译通过
2. `go test ./infrastructure/logstore/...` — LogStore 单元测试
3. `go test ./internal/service/ci/...` — CI service 测试通过
4. `go test ./internal/service/cd/...` — CD service 测试通过
5. 手动：触发一次流水线，观察 `/api/ci/run/{id}/stages/{id}/log` 返回正确的分页日志
6. 手动：触发一次部署，观察 `/api/cd/deployment/{id}/logs` 返回正确的日志

## Blockers

无。所有涉及的代码路径已明确，无外部依赖。

## Assumptions

- `LogStore` 无状态（每次 Write 都 OpenFile + Close），不持有文件句柄
- `RunOptions.LogFile` 字段保留但传 nil，未来单独清理
- 路径构造仍由 `CIWorkspace` 和 `CDWorkspace` 负责，LogStore 只接收完整路径

## Risk

- 构造函数签名变更影响所有调用方（bootstrap + http server），但这是机械式改动
- `os.OpenFile` + `O_APPEND` 在极端并发下可能丢数据（两个 goroutine 同时 append），但当前场景下每个 stage/run 有独立的 logPath，不会并发写同一文件
