# RAGFlow 拆分应用部署验证
最后修改时间: 2026-07-26 20:22:38

## Review status

Accepted

## Verification basis

- Requirement: `docs/requirement/20260726-ragflow-split-deployment.md` (`Accepted`)
- Spec: `docs/spec/20260726-ragflow-split-deployment.md` (`Accepted`)
- Plan: `docs/plan/20260726-ragflow-split-deployment.md` (`Accepted`)
- Flow mode: 严格模式 / `strict`
- Scope: Task 2 的 fixture、Runbook、Python 初始化器，以及 Gateway MCP、受管 `traefik` 预检、固定 HTTP Probe 与 Task 1 运行时契约的集成核对。

## Requirement and spec alignment

1. fixture 确认定义且仅定义 RAGFlow、MySQL、Redis、MinIO、Elasticsearch 五个 `standard` Application；依赖服务没有 Expose，RAGFlow 使用 local/public 两个互斥变体。
2. 初始化器具备离线 `plan`、受管 MCP `check`、显式确认的 `apply`、部署等待与 `verify`；运行目录只记录 fixture / plan 摘要、非秘密参数、步骤状态和 resource ID。
3. Component 编码与 Task 1 的 `restart_policy`、`tmpfs_json`、`ulimits_json`、`secret_env_refs` 对齐；`tmpfs_json` 在 Compose 中渲染为 Docker Compose 支持的短语法字符串列表。
4. 未发布或部署的草稿运行可在 fixture 修订后自动重基，并经 MCP 更新同一 journal 已记录的 Version；已发布或已部署的运行仍不能静默修改。
5. 初始化器以已部署 Gateway 的受管 target 证明 bridge `traefik` 网络；RAGFlow Version 使用 fixture 候选 healthcheck，最终就绪由部署后的固定 HTTP Probe 判断。

## Actual diff summary

| 预期文件 | 实际结果 |
| --- | --- |
| `docs/guides/ragflow-split-deployment.fixture.yaml` | 已提供五个 Application 的非秘密 desired state。 |
| `docs/guides/ragflow-split-deployment.md` | 已提供 Gateway 参数、秘密边界、受管网络预检、恢复和 Probe gate 说明。 |
| `scripts/ragflow_initialize.py` | 已实现 stdio MCP 编排、原子 journal、Gateway-aware capability gate、冲突检测、依赖顺序和 RAGFlow Probe 验证。 |
| `scripts/test_ragflow_initialize.py` | 已覆盖 fixture、无秘密 journal、Credential ID 编码、缺失能力、恢复资源 ID、Gateway 网络证明与 RAGFlow Probe 顺序。 |
| `mcp/**` | 已增加 Gateway HTTP 映射、Gateway-aware doctor 与受限 HTTP Probe，并由 unit tests 锁定。 |
| `internal/application/deployment/usecase/runtime_fields.go` | 将 structured `tmpfs_json` 渲染为 Docker Compose 可执行的短语法，修复真实 Elasticsearch 部署发现的类型错误。 |

## Acceptance checklist

| 验收项 | 结果 | 证据 |
| --- | --- | --- |
| 五个 Application 的映射、镜像、alias、挂载、Credential ref 和运行时字段可审阅且无秘密 | 通过 | fixture 校验及初始化器 14 个离线测试。 |
| 初始化器可重复计划并持久化无秘密运行状态 | 通过 | 当前 run 保持 30 个操作，fixture digest `54547139...f06961`，plan digest `cced130ef...034d5`。 |
| 缺少 Task 1 契约、Gateway 参数或网络证明时不写入资源 | 通过 | 初始化器 capability tests 记录 `blocked`，journal resources 均为空。 |
| Gateway MCP、受管 `traefik` 预检与固定 HTTP Probe 可编译、可测试 | 通过 | Gateway 类型、缺失网络、固定 argv 与失败脱敏由 MCP unit tests 覆盖。 |
| 受管 MCP 证明外部 `traefik` 网络存在 | 通过 | Gateway `6jwmevj7osezfltaezznjfxava` 配置为 `http://localhost:8080`；`runtime_doctor` 返回 `healthy=true` 和 bridge `traefik`。 |
| 钉死 RAGFlow 镜像的真实 HTTP Probe | 通过 | run journal 的 `verify:ragflow` 记录 `consistent` 和 `ragflow-cpu:80/` `reachable`。 |
| 实际 Credential / Application / Version / Deployment 创建与端到端 HTTP 验收 | 通过 | 五个 Deployment 均为 `consistent`；`http://127.0.0.1:9380/` 返回 HTTP 200。 |

## Test results

| 命令 | 结果 |
| --- | --- |
| `make -C mcp check` | 通过：Ruff、format、mypy、pytest；26 passed，1 个 Docker 集成测试按默认标记排除。 |
| `uv --directory mcp run python -m py_compile ../scripts/ragflow_initialize.py ../scripts/test_ragflow_initialize.py` | 通过。 |
| `uv --directory mcp run mypy --strict ../scripts/ragflow_initialize.py ../scripts/test_ragflow_initialize.py` | 通过。 |
| `uv --directory mcp run ruff check ../scripts/ragflow_initialize.py ../scripts/test_ragflow_initialize.py` | 通过。 |
| `uv --directory mcp run ruff format --check ../scripts/ragflow_initialize.py ../scripts/test_ragflow_initialize.py` | 通过。 |
| `uv --directory mcp run python ../scripts/test_ragflow_initialize.py` | 通过：14 tests。 |
| `go test ./internal/application/deployment/usecase` | 通过。 |
| `python scripts/test_ragflow_initialize.py` | 通过：14 tests。 |
| `python -m pytest mcp/tests/test_workspace.py mcp/tests/test_runtime_tools.py -q` | 通过：5 tests。 |
| `python -m ruff check scripts/ragflow_initialize.py scripts/test_ragflow_initialize.py` | 通过。 |

## Missing or expanded scope

1. 真实预检于 2026-07-26 通过：Gateway `6jwmevj7osezfltaezznjfxava` 使用 `http://localhost:8080`，其受管 target 已报告 bridge `traefik` 网络 healthy。首次实际 Application import 暴露了初始化器将秘密引用错误编码为 `secret_env_refs_json` 的问题；已更正为 API 的结构化 `secret_env_refs` 字段。
2. Preview 发现平台拒绝带内容的 `.sql` logical mount，MySQL 初始化改为官方 `MYSQL_DATABASE=rag_flow`。Elasticsearch 首次部署发现 `tmpfs` map 不是 Docker Compose 可接受的类型；渲染已修复为短语法，并以新的 Deployment `lugd4ptgc5sj557zm2dcjfnvui` 通过实际验证。
3. 当前 live run 参数为 `instance_key=default`、local HTTP `9380`；秘密文件位于仓库外，journal 不含其值。备份位置和维护窗口仍是运行维护事项，未由本次自动化配置。

## Risks

1. 放宽 Gateway target / `external_networks` 检查或使用未受管 Docker CLI 检查网络，会破坏本任务约定的 MCP 受管边界。
2. 候选 RAGFlow healthcheck 可能不反映实际可用性；固定受管 Probe 失败时不得将仅容器 running 误判为服务就绪。
3. 本机 Docker 高权限用户可读取容器环境和 workspace；本任务的脱敏和 `runtime_env` 引用不改变该信任边界。

## Conclusion

Task 2 已完成真实闭环：Gateway 以 `http://localhost:8080` 预检 bridge `traefik` 网络，四个依赖和 RAGFlow 均经 MCP 创建、发布、部署并验证为 `consistent`，RAGFlow 的受管 HTTP Probe 与宿主机 local Expose 均成功。运行记录位于 `data/ragflow/runs/20260726T115903Z`，不包含实际 Credential value。
