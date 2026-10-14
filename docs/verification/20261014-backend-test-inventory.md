# 后端测试分类快照（2026-10-14）

依据第十三轮最终 Windows 默认构建的 `go test -json -count=1 ./cmd/... ./internal/... ./sql`，按顶层测试计数，子测试不重复相加。818 通过、13 跳过，共 831 个顶层入口。与其他平台、后续变更或 unit tag 的范围不能混用。

分类按实际执行边界复核，包括夹具和辅助函数。临时目录仅用于生成路径时不算文件集成；`httptest.NewRecorder` 与内存 MCP 连接计入内存协议，真实 listener/WebSocket/SSH/SFTP 计入 socket。混合测试按真实资源计一次。静态 SQL 文件检查属于静态检查，非数据库集成。

**当前真实本地资源集成共 206 项：SQLite 115、文件 57、socket 24、子进程 10。** 另有 13 项显式外部/环境场景，当前跳过。611 项规则、映射、端口替身、内存协议与静态检查不统称纯单元；其中 357 项属于日常核心入口。`TestStoreProcessHelper` 是多进程测试的子进程入口，默认空返回，单独计为 1 个 helper，不冒充独立行为验证。

| 范围 | 内存规则/协议/静态 | SQLite | 文件 | Socket | 子进程 | 外部跳过 | Helper | 总数 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Application | 305 | 50 | 4 | 0 | 0 | 2 | 0 | 361 |
| Auth primitives | 6 | 0 | 0 | 0 | 0 | 0 | 0 | 6 |
| Bootstrap | 6 | 2 | 7 | 2 | 0 | 0 | 0 | 17 |
| Common | 29 | 0 | 4 | 0 | 1 | 1 | 1 | 36 |
| Config | 8 | 0 | 29 | 0 | 0 | 0 | 0 | 37 |
| Database adapter / Tx | 7 | 24 | 0 | 0 | 0 | 0 | 0 | 31 |
| External entry / SQLite smoke | 0 | 1 | 0 | 0 | 0 | 6 | 0 | 7 |
| HTTP | 87 | 3 | 6 | 10 | 0 | 0 | 0 | 106 |
| Local storage | 12 | 0 | 2 | 0 | 2 | 0 | 0 | 16 |
| MCP API | 29 | 1 | 0 | 0 | 0 | 0 | 0 | 30 |
| MCP client | 1 | 0 | 0 | 0 | 0 | 0 | 0 | 1 |
| Model | 7 | 0 | 0 | 0 | 0 | 0 | 0 | 7 |
| Queue | 16 | 0 | 0 | 0 | 0 | 0 | 0 | 16 |
| runner/docker | 16 | 0 | 0 | 0 | 0 | 0 | 0 | 16 |
| runner/local | 2 | 0 | 5 | 0 | 7 | 1 | 0 | 15 |
| runner/pipeline | 3 | 0 | 0 | 0 | 0 | 0 | 0 | 3 |
| runner/ssh | 25 | 0 | 0 | 8 | 0 | 2 | 0 | 35 |
| runner/target | 1 | 0 | 0 | 0 | 0 | 0 | 0 | 1 |
| SQL error mapping | 2 | 0 | 0 | 0 | 0 | 0 | 0 | 2 |
| SQLC Repository | 0 | 34 | 0 | 0 | 0 | 0 | 0 | 34 |
| Traefik adapter | 49 | 0 | 0 | 0 | 0 | 1 | 0 | 50 |
| Turnstile adapter | 0 | 0 | 0 | 4 | 0 | 0 | 0 | 4 |
| 总计 | 611 | 115 | 57 | 24 | 10 | 13 | 1 | 831 |

## 保留依据与统计边界

- SQLite：Application 50、Repository 34、DB/Tx 24、HTTP/MCP 4、Bootstrap 2、SQLite 启动 smoke 1。验证真实查询、约束、事务、迁移与组装，不能由模拟 SQL 或业务替身替代。Settings 的 4 项文件测试没有重复计入 Application SQL。
- 文件：Config 29、Settings 4、日志装配 7、HTTP 静态文件 6、dotenv 存储 4、local workspace 5、执行日志/删除目录 2。配置解析与覆盖、原子修改/恢复、真实路径隔离是保留原因；纯校验已迁出 7 项文件场景。
- Socket：HTTP 终端 9、SSE 1、Turnstile 4、SSH/SFTP 8、Bootstrap server/worker 2；包含真实网络建立失败和取消边界。HTTP 票据未登录验证仅使用 recorder，不计入 socket。
- 子进程：本地 terminal/log stream 7、Git 2、dotenv 跨进程写 1。验证进程树释放、PTY 尺寸/尾输出、真实 Git 仓库行为和文件锁；没有增加性能或压力测试。
- 内存协议：Gin recorder、内存 MCP SDK、SQL mock 与适配器返回事实可以验证协议契约，但不是实际数据库/网络/进程集成的替代。MCP API 的 1 项真实回滚已归入 SQLite，其余 29 项留在内存协议列。
- 外部 13 项保持原启用条件：应用/适配器远端 Pipeline 各 2、Windows SSH 4、MySQL/PostgreSQL 2、Traefik 重启 1、DooD socket 1、挂载 dotenv 1。

此表是一次审计快照，不新增测试门槛或维护第二套运行清单；后续数量变化以代码和最新日志为准。必要性和失败修复见[治理记录](./20261014-backend-test-architecture.md)。
