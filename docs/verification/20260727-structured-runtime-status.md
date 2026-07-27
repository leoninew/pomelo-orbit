# 结构化运行时状态响应验证
最后修改时间: 2026-07-27 11:55:21

Review status: Draft

## Requirement alignment

| 验收项 | 结果 | 证据 |
|---|---|---|
| 接口返回结构化容器列表 | 通过 | `ApplicationStatusResp.containers` 和 `ApplicationContainerStatusResp` 已生成到 Go / TypeScript。 |
| 后端直接处理 Compose health | 通过 | `compose_ps.go` 解码 `ps --format json` 的 JSON array 或 object record stream，并直接映射 `Health`。 |
| 非契约输出不可成功返回 | 通过 | 单元测试拒绝非法文本；解析失败由 `ApplicationStatus` 转为内部错误。 |
| Web 不解析 Compose 字符串 | 通过 | `ServiceDetail.vue` 直接读取 `resp.containers`；`web/src/utils/compose.ts` 已删除。 |

## Spec / Plan alignment

不适用。此任务采用轻量模式 / light，按 Requirement 实施和核对。

## Actual diff summary

- 将状态 API 从单个 `status` 字符串改为 `containers` 列表，生成相应 Go / TypeScript DTO。
- 在 deployment usecase 中将受管 Compose JSON array 或 object record stream 解码为 `RuntimeContainer`，不再使用动态 map、正则推断或前端解析。
- 将 HTTP mapper 和服务详情页改为直接使用结构化容器字段。
- 新增运行时 JSON 解码、HTTP DTO 映射和根 JSON 响应形状测试。
- 实际复现 Docker Compose `v5.3.0` 的单容器输出：它是一个 JSON 对象而非数组；测试已覆盖该 object stream framing。

## Expected vs actual files

预期文件：Application Proto 及生成物、deployment runtime 查询与 DTO、deployment HTTP mapper、服务详情页、前端 Compose 解析工具及相关测试。

实际文件与预期一致：`proto/orbit/v1/application/application.proto`、`internal/gen/proto/**/application.pb.go`、`web/src/gen/proto/**/application.ts`、`internal/application/deployment/{dto,usecase}/**`、`internal/api/http/handler/deployment/**`、`web/src/views/service/ServiceDetail.vue`、`web/src/utils/compose.ts`。

`ServiceDetail.vue` 同时含任务开始前已有的运行时环境标题与表格样式调整；这些样式调整不属于本任务的验收范围。

## Acceptance checklist

- [x] 响应根 JSON 使用 `containers`，不再使用旧的根 `status` 字符串。
- [x] 每项包含 id、name、service、state、status、health、image。
- [x] `Health` 由 Docker Compose JSON 直接提供。
- [x] 支持受控命令的 JSON 数组和 object record stream 输出。
- [x] Web 未保留 Compose `ps` 字符串解析层。

## Test results

| 命令 | 结果 |
|---|---|
| `./bin/buf.exe generate` 与 Web 模板生成 | 通过 |
| `go fmt ./cmd/... ./internal/...` | 通过 |
| `go vet ./cmd/... ./internal/...` | 通过 |
| `go test ./cmd/... ./internal/...` | 通过，含 e2e 包 |
| `yarn --cwd web lint:fix` | 通过 |
| `yarn --cwd web typecheck` | 通过 |
| `yarn --cwd web test` | 通过，9 个文件、45 个测试 |
| `git diff --check` | 通过 |
| `docker compose -p bge-m3-default -f docker-compose.yml ps --format json` | 通过，实际输出为单个 JSON object，`Health=starting`。 |
| `./bin/buf.exe lint` | 未通过：所有既有 Proto package 均不满足版本后缀规则；本次未变更 package 名称。 |

## Scope differences

未扩展业务范围。仅将既有 `ps --format json` 命令的结果从字符串透传改为结构化 API 契约；实际输出确认后，将 framing 从“仅数组”校正为数组或 object record stream。

## Risks

运行环境必须继续支持既有 `docker compose ps --format json` JSON 输出。任何不符合 JSON array / object record stream 受控契约的输出将得到错误响应，而不是部分或错误的健康状态。

## Incomplete items

本任务无未完成代码项。全仓 `buf lint` 的 package 命名基线问题不在本任务范围内。

## Conclusion

Requirement 的验收项均已满足。Docker Compose `v5.3.0` 的实际输出已复现并纳入测试；相关格式化、静态检查、Go 测试与 Web 测试通过。验证记录等待用户审阅；`buf lint` 基线失败已单独记录。
