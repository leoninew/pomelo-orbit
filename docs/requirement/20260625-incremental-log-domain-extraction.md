# 增量日志写入/读取的领域业务抽取

最后修改时间: 2026-06-25 16:45:00

Review status: Accepted

## Background

当前项目（Pomelo Orbit）有两处日志相关功能：

1. **CI 流水线日志**（`/ci/run/xxxx`）：记录每个 Stage 的 Docker 容器执行输出
2. **CD 部署日志**（`/cd/deployments/xxxx`）：记录 `docker compose up -d` 等部署命令的输出

**写入现状**：两者都采用"一次性写入"模式 — 命令/容器执行完毕后，`cmd.CombinedOutput()` 返回全部输出，一次性 `Write` 到日志文件。这意味着前端在命令执行期间看不到任何日志，必须等到结束才能一次性拿到。

**读取现状**：两者都采用"增量读取"模式 — 前端带 `offset` 轮询，后端 `file.Seek(offset)` + `io.ReadAll` 返回增量部分。

**问题**：
1. 日志的读写逻辑散落在 `Service` 结构体的方法中（`PipelineStageLog`、`readDeploymentLog`），CI 和 CD 各自实现，无共享抽象
2. 写入和读取本应是一个关注点的两端，但被耦合在不同的 Service 方法中
3. 当前"一次性写入 + 增量读取"的组合导致用户无法实时查看执行中的日志

## Goal

将 CI/CD 的日志**写入**和**读取**逻辑从现有 Service 中抽取为统一的基础设施组件 `LogStore`，同时将写入模式从"一次性写入"改为"执行中增量追加"，使用户能在命令/容器执行期间通过轮询看到实时日志。

## Non-goal

- 不实现 SSE 流式推送功能（轮询 API 不变，只是写入端变成增量）
- 不改变现有轮询 API 的外部行为（前端 offset 轮询机制不变）
- 不引入远程日志存储（如 Elasticsearch、消息队列）
- 不重构前端代码
- 不实现日志轮转/清理/保留策略

## User scenarios

1. **作为用户**，在流水线/部署执行过程中，打开日志面板能看到实时输出的日志，而不是等到结束才一次性刷出
2. **作为开发者**，希望在新增日志相关功能（如 SSE 推送、日志搜索）时，不需要理解底层文件读写的细节
3. **作为维护者**，希望 CI 和 CD 的日志逻辑复用同一套抽象，减少重复代码

## Acceptance

1. 抽取 `LogStore` 组件，CI 和 CD 两侧的日志读写都通过它完成，不再直接调用 `os.Open`/`os.Create`
2. 写入模式改为"执行中增量追加"：命令/容器输出实时写入日志文件，前端轮询可在执行中获取新内容
3. 读取保持"offset 增量读"语义，接口不变
4. `LogStore` 接口简洁，不引入不必要的抽象层（无 Repository、无 Domain Event）
5. 现有测试通过，现有 API 外部行为无变化

## Open questions

1. **与现有 runner 的关系**：CI 的 `ContainerRunner` 和 CD 的 `CommandRunner` 当前接收 `io.Writer`。抽取后 runner 是否继续接收 `io.Writer`（由 `LogStore` 实现 `io.Writer`），还是改为回调/channel 模式？

## Decisions

- 抽取的同时改变写入模式：从"一次性写入"改为"执行中增量追加"
- `LogStore` 放在基础设施层（`backend-go/internal/infrastructure/logstore/`），不属于 CI 或 CD 领域
- 不预留远程存储扩展点，本地文件即可
- 不实现日志轮转/清理
- 不加文件锁：单 goroutine 写 + 单 goroutine 读，POSIX 保证读写原子性，无需应用层同步

## Risk

- 过度设计风险：日志逻辑可能不值得独立抽取，强行抽象反而增加复杂度
- 并发写入安全：goroutine 边读边写文件，需要确保读端不会读到写了一半的行
- 抽取后接口不稳定：如果接口设计不当，未来改动面会更大
- 性能风险：引入额外抽象层可能影响高频日志写入的性能（当前场景下概率低）
