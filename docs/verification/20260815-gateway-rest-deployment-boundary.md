# Gateway REST Deployment Boundary 验证
最后修改时间: 2026-08-15 17:10:56

Review status: Accepted

## 需求对齐

按 `docs/requirement/20260815-gateway-rest-deployment-boundary.md` 核对。该任务采用轻量模式 / light；无独立 Spec 或 Plan，按已接受的 Requirement 验证。

- 普通 Service 的 Deploy 与 Restart 已不再包含 `EnsureGatewayRunning` 运行态预检；全仓 `internal/`、`proto/`、`web/` 无此符号引用。
- `join_traefik_network` 已进入 HTTP、MCP、Proto、Web API、部署选项和有效计划哈希；省略时为 `true`。
- 标准应用取消加入共享网络时，Compose 渲染会跳过 `traefik` 网络、别名、Gateway labels 与 Gateway 配置读取；Gateway Application 始终加入其运行网络。
- Gateway 部署后仍通过既有 `publishGatewayRoutes` 调用发布 Route 快照；Route REST 路径的运行态检查没有纳入普通部署。

## 实际差异

预期范围为部署/预览请求边界、有效计划和 Compose 渲染、Gateway 部署回调、HTTP/MCP/Proto/Web 调用端与针对性测试。暂存区实际包含 26 个文件、328 行新增和 48 行删除，均属于该范围：

| 范围 | 实际文件 |
| --- | --- |
| 过程文档 | `docs/requirement/20260815-gateway-rest-deployment-boundary.md` |
| 部署领域 | `internal/application/deployment/{dto,usecase}`、`internal/model/effective_service_plan.go` |
| Gateway 回归测试 | `internal/application/gateway/usecase/deployment_test.go` |
| 传输层与协议 | HTTP handler、MCP delivery、`proto/orbit/v1/{application/version,service/service}.proto` 及生成文件 |
| Web 调用端 | Service API、Version Preview、Service Detail、Service Page |

`web/src/views/service/ServiceDetail.vue` 的暂存部分只包含共享网络选项；其中“faulted Service 可删除”的工作区改动仍未暂存。其他未暂存和未跟踪文件也未纳入本次验证范围。

## 验收清单

- [x] `DeployService`、`RestartApplication` 不再引用 Gateway 运行态预检。
- [x] 标准 Service 部署和两类 Compose Preview 暴露可选的 `join_traefik_network`，默认为 `true`，并写入部署选项与有效计划哈希。
- [x] 关闭选项时，标准应用不渲染或读取共享 Traefik/Gateway 投影；新增 Compose 与 Gateway 查询回归测试覆盖该分支。
- [x] Gateway Application 强制加入运行网络，部署完成后仍走既有 Route 快照发布路径。
- [x] 不再保留只为普通部署服务的 `EnsureGatewayRunning` 引用。
- [x] 已持久化的网络选项参与该次部署和重启的有效计划重建。当前部署验证的运行期比较只核对服务存在、运行状态和镜像，不比较网络，因此不会将隔离部署误报为 Compose drift。

## 命令结果

| 命令 | 结果 |
| --- | --- |
| `go fmt ./cmd/... ./internal/...` | 通过 |
| `go vet ./cmd/... ./internal/...` | 通过 |
| `go test ./cmd/... ./internal/...` | 通过 |
| `go test -count=1 ./internal/application/deployment/usecase ./internal/application/gateway/usecase` | 通过 |
| `yarn --cwd web lint:fix` | 通过 |
| `yarn --cwd web typecheck` | 通过 |
| `yarn --cwd web test` | 11 个测试文件、65 个用例全部通过 |
| `git diff --cached --check` | 通过 |

## 范围偏差与风险

未执行真实 Docker Compose、Traefik REST 或浏览器交互流程，运行期网络存在性、Gateway 停止时入口不可达以及 Route 快照发布仅完成代码和单元测试层验证。

部署验证展示的 `preview_compose` 仍以默认网络选项生成，而不是读取该次部署的 `OptionsJSON`。这不影响当前比较结论，因为 `compareRuntimeCompose` 不比较网络；但若后续将网络纳入 drift 比较或把该字段作为精确部署证据，应先从部署选项恢复 `JoinTraefikNetwork`。建议补充 `false` 选项的端到端或验证回归测试。

## 结论

代码质量检查和现有测试全部通过，需求验收项满足，可交付。部署验证证据未恢复原始网络选项属于后续增强风险，不阻塞当前变更。
