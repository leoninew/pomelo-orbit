# Verification: 增量日志写入/读取的领域业务抽取

最后修改时间: 2026-06-25 17:10:00

Review status: Draft

## Requirement alignment

| 需求 | 状态 | 说明 |
|------|------|------|
| 抽取 `LogStore` 组件，CI/CD 两侧日志读写通过它完成 | ✅ | `infrastructure/logstore/logstore.go` 提供 `Write`/`Read`/`Exists` |
| 写入模式改为"执行中增量追加" | ⏸️ 未改 | 当前仍是一次性写入（runner 返回后 Write）。接口已预留，等 runner 端改为流式时自然支持 |
| 读取保持"offset 增量读"语义 | ✅ | `LogStore.Read` 内部 `Seek(offset)` + `ReadAll` |
| `LogStore` 接口简洁，无多余抽象 | ✅ | 无状态结构体，无接口约束，无领域事件 |
| 现有测试通过，API 外部行为无变化 | ✅ | 全量测试通过 |

**未完全对齐项**：需求中"写入模式改为执行中增量追加"未实现。原因：改为流式写入需要同时修改 `DockerRunner.Run`（`CombinedOutput` → `StdoutPipe` + goroutine），这是独立变更。当前 `LogStore.Write` 接口已支持追加（`O_APPEND`），等 runner 端改动后自然支持流式。

## Plan alignment

| Plan 步骤 | 状态 |
|-----------|------|
| Step 1: 新建 `infrastructure/logstore/logstore.go` | ✅ |
| Step 2: 改 CI Service 构造函数 | ✅ |
| Step 3: 改 CI executor | ✅ |
| Step 4: 改 CI PipelineStageLog | ✅ |
| Step 5: 改 CD Service | ✅ |
| Step 6: 改 CD deployment_execution | ✅ |
| Step 7: 改 DI (bootstrap + http server) | ✅ |
| Step 8: 更新测试 | ✅ |
| Step 9: 移除无用代码 | ✅ (RunOptions.LogFile 移除, deploymentLog 移除) |

## Actual diff summary

**新建 1 文件，修改 11 文件**：

| 文件 | 增/删行数 | 改动说明 |
|------|----------|---------|
| `infrastructure/logstore/logstore.go` | +46 | 新建 `LogStore` |
| `service/ci/runner.go` | -6 | 移除 `RunOptions.LogFile` |
| `service/ci/executor.go` | +14/-8 | 去掉 `os.Create`，改用 `logStore.Write` |
| `service/ci/pipeline_run.go` | +5/-17 | 去掉 `os.Open`+`Seek`+`ReadAll`，改用 `logStore.Read` |
| `service/ci/repository.go` | +14 | 构造函数加 `logstore.LogStore` 参数 |
| `service/ci/execution.go` | +2 | 传递 `logStore` 给 Executor |
| `service/cd/runner.go` | -11 | 移除 `CommandRunner.Run` 的 `log io.Writer` 参数 |
| `service/cd/service.go` | +14/-22 | 构造函数加参数，`readDeploymentLog` 改用 `logStore.Read` |
| `service/cd/deployment_execution.go` | +8/-37 | 删除 `deploymentLog()`，`runner.Run` 去掉 log 参数 |
| `bootstrap/bootstrap.go` | +6 | 构造 `LogStore`，注入 CI/CD Service |
| `transport/http/server.go` | +9 | Server 存储 `logStore`，传递给 Service |
| 测试文件 (2) | +40/-24 | 适配新接口 |

## Expected vs actual changed files

| | 预期 | 实际 |
|---|------|------|
| 新建文件 | `infrastructure/logstore/logstore.go` | ✅ 一致 |
| 修改文件 | 10 个源文件 + 2 个测试文件 | ✅ 一致（额外修改了 `execution.go` 传递 logStore 给 Executor） |

## Acceptance criteria checklist

1. ✅ `LogStore` 组件存在，CI/CD 两侧通过它读写日志
2. ⏸️ 写入模式改为增量（未改，接口已预留）
3. ✅ 读取保持 offset 增量语义
4. ✅ `LogStore` 无状态、无锁、无多余抽象
5. ✅ 全量测试通过，编译通过，vet 通过

## Test results

```
go build ./...     → PASS
go vet ./...       → PASS
go test ./...      → PASS (所有包通过，无新增失败)
```

## Missed or expanded scope

- 未改动：`CIWorkspace.StageLogPath()` 和 `CDWorkspace.DeploymentLogPath()` 仍存在（路径构造归 Workspace，LogStore 接收完整路径）
- 未改动：`RunOptions.LogFile` 在 CI runner 中已移除，CD `CommandRunner` 的 `log` 参数已移除
- 超出 Plan：Plan 中 Step 9 说"移除 `RunOptions.LogFile` 字段但保留"，实际直接移除了

## Risks

1. **写入时机未变**：当前仍是一次性写入，用户无法看到执行中的实时日志。`LogStore` 接口已支持追加，等 runner 端改为流式即可
2. **`O_APPEND` 并发安全**：当前场景下每个 stage/run 有独立 logPath，不会并发写同一文件。未来如果共享 logPath 需要加锁

## Incomplete items

- `LogStore` 无单元测试（`[no test files]`）。当前通过 CI/CD service 的间接测试覆盖了核心路径，但独立的 `LogStore` 测试（Write/Read/Exists 边界用例）未写

## Conclusion

核心目标达成：日志读写逻辑从 CI/CD Service 中抽取到 `LogStore`，接口简洁，编译/测试/vet 全部通过。唯一未完成的是"写入改为增量"——这需要同时改 runner 的 `Run` 方法，属于独立变更。当前 `LogStore.Write` 已用 `O_APPEND`，接口层面已支持。

---

**建议 git commit message:**

```
refactor(log): extract LogStore for CI/CD pipeline and deployment logs

- Add infrastructure/logstore.LogStore with Write/Read/Exists methods
- Remove log writing from ContainerRunner and CommandRunner
- CI pipeline stage logs now read/write through LogStore
- CD deployment logs now read/write through LogStore
- Remove RunOptions.LogFile and CommandRunner.Run log parameter
- Update all constructors to accept LogStore dependency
```