# API 请求响应 Proto 契约化验证
最后修改时间: 2026-07-06 12:23:11

## Review status

Accepted

## Requirement alignment

- [x] 为 API 请求体和响应体引入 proto 定义与生成链路。
  - 已新增 `orbit/api/v1/*.proto`，按领域拆分请求体和响应体 message。
  - 已新增 `buf.yaml`、`buf.gen.yaml`、`web/buf.gen.yaml`。
  - 已新增 `Taskfile.yml` 中的 `proto` 任务，统一通过 `./bin/buf` 生成。
- [x] 后端 `Req` / `Resp` 类型迁移为 generated Go 类型或 handler 局部别名。
  - 已新增 `internal/transport/http/dto/proto/orbit/api/v1/*.pb.go`。
  - handler 层 DTO 文件改为引用 `apiv1` generated 类型。
- [x] 前端请求/响应类型迁移为 generated TypeScript 类型。
  - 已新增 `web/src/gen/proto/orbit/api/v1/*.ts`。
  - 前端 API 与页面直接从 `@/gen/proto/orbit/api/v1/<domain>` import generated types，不通过 `web/src/types` re-export。
- [x] 保持 HTTP JSON transport、路由、状态码、权限、业务校验和响应包装语义。
  - proto 只定义 body message；未引入 RPC runtime。
  - `transportresponse` 对 proto message 使用 `protojson`，保持 snake_case JSON 字段名。
- [x] 工具链可复现。
  - `task install` 安装 `buf`、`protoc-gen-go`、`sqlc` 到项目 `bin` 目录。
  - `task proto` 从 `bin` 目录工具发起生成，不使用 `go run`。

## Spec alignment

- [x] Proto schema 路径符合最终约定：`orbit/api/v1/*.proto`。
- [x] 领域文件名不使用 `ci_` / `cd_` 前缀。
- [x] `go_package` 保持 HTTP transport boundary：`backend/internal/transport/http/dto/proto/orbit/api/v1;apiv1`。
- [x] TypeScript 生成目录保持 `web/src/gen/proto`。
- [x] `ts-proto` 配置保持 `onlyTypes=true`、`snakeToCamel=false`，不生成 client/runtime 方法。
- [x] `web/src/types` 不 re-export generated API DTO。
- [x] presence 语义已覆盖关键 update 场景：`RepositoryUpdateReq.variable_overrides` 使用 optional wrapper message，区分 omitted 与显式空列表。

## Plan alignment

- [x] 已盘点并替换后端 handler DTO。
- [x] 已引入 proto 工具链配置。
- [x] 已定义 auth/user/role/project/settings/task/common、CI、CD 各领域 proto 文件。
- [x] 已生成并接入后端 Go DTO。
- [x] 已生成并接入前端 TypeScript DTO。
- [x] 已运行计划中的格式化和检查命令。

## Actual diff summary

主要变更范围：

- 工具链：
  - `Taskfile.yml`
  - `buf.yaml`
  - `buf.gen.yaml`
  - `web/buf.gen.yaml`
  - `tools.go`
- Proto source：
  - `orbit/api/v1/*.proto`
- Go generated DTO：
  - `internal/transport/http/dto/proto/orbit/api/v1/*.pb.go`
- Go HTTP transport：
  - `internal/transport/http/response/response.go`
  - `internal/transport/http/handler/**/dto.go`
  - `internal/transport/http/handler/**/*.go`
  - `internal/transport/http/*_routes_test.go`
- Frontend generated DTO and usages：
  - `web/src/gen/proto/orbit/api/v1/*.ts`
  - `web/src/api/**`
  - `web/src/views/**`
  - `web/src/stores/**`
  - `web/src/types/index.ts`

Proto-related diff statistics observed during verification:

```text
Taskfile.yml                                       |  10 +
buf.gen.yaml                                       |   7 +
buf.yaml                                           |   9 +
internal/transport/http/dto/proto/orbit/api/v1/*   | generated Go DTO files
orbit/api/v1/*.proto                               | source proto files
web/buf.gen.yaml                                   |  13 +
```

Full working tree includes large generated-code additions and broad DTO usage replacements, expected for this contract migration.

## Expected vs actual changed files

| Expected | Actual | Result |
| --- | --- | --- |
| `buf.yaml` / `buf.gen.yaml` / `web/buf.gen.yaml` | Added/updated | OK |
| `orbit/api/v1/*.proto` | Added by domain, no `ci_` / `cd_` prefixes | OK |
| Go generated DTO under HTTP transport boundary | Added under `internal/transport/http/dto/proto/orbit/api/v1` | OK |
| Handler DTO aliases / construction sites | Updated under `internal/transport/http/handler/**` | OK |
| HTTP JSON helper | Updated in `internal/transport/http/response/response.go` for protojson | OK |
| Frontend generated TS DTO | Added under `web/src/gen/proto/orbit/api/v1` | OK |
| Frontend API/page type imports | Updated to direct generated imports | OK |
| Git write operations | None executed | OK |
| Migration files | Not modified | OK |

## Acceptance checklist

- [x] 仓库根目录具备 proto 生成工具链配置。
- [x] TaskFile 提供 `install` 和 `proto` 命令。
- [x] `task install` 将 proto 工具链准备到 `bin` 目录。
- [x] `task proto` 从 `./bin/buf` 发起生成。
- [x] 本次最终生成流程不使用 `go run`。
- [x] proto 只定义 request body / response body message。
- [x] 未定义 route、method、query/path/header、错误响应或 RPC service。
- [x] Go handler 使用 generated DTO 或局部别名。
- [x] Frontend API/page 直接 import generated DTO，不使用 re-export。
- [x] 现有 JSON 字段名保持 snake_case。
- [x] Go fmt/vet/test 通过。
- [x] Frontend lint/typecheck 通过。

## Command results

### Taskfile registration

```text
task --list
```

结果：通过，任务列表包含：

```text
install
proto
sqlc
check
test
build
run
clean
```

### Toolchain install

```text
task install
```

结果：通过。执行内容包括：

```text
yarn --cwd web install
mkdir -p bin
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
go install github.com/bufbuild/buf/cmd/buf@v1.59.0
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
go mod download
go mod tidy
```

验证后确认：`go.mod` / `go.sum` 未因 `buf` 工具安装产生最终 diff；`buf` 不作为业务 module 依赖记录在 `tools.go` 中。

### Proto generation

```text
task proto
```

结果：通过。

```text
task: [proto] ./bin/buf generate
task: [proto] ./bin/buf generate . --template web/buf.gen.yaml --output web
```

### Proto service/RPC scope check

```text
Grep pattern: ^\s*(service|rpc)\s+
Path: orbit/api/v1
```

结果：无匹配。proto 中未定义 service 或 rpc。

### Frontend checks

```text
yarn --cwd web lint:fix
yarn --cwd web typecheck
```

结果：通过。

最后一次输出：

```text
$ eslint . --fix --cache
Done in 1.29s.
$ vue-tsc --noEmit
Done in 7.03s.
```

### Go checks

```text
go fmt ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

结果：通过。

最后一次 `go test` 关键输出：

```text
ok  backend/internal/transport/http  17.085s
ok  backend/internal/transport/http/response  (cached)
ok  backend/internal/worker/handler/cd  (cached)
ok  backend/internal/worker/handler/ci  (cached)
```

## Missed or expanded scope

- 未进入 gRPC/Connect/OpenAPI；符合 non-goal。
- 未定义 HTTP route、query/path/header/cookie/auth/error envelope；符合 non-goal。
- 未修改已执行迁移文件；符合项目约束。
- 由于 generated protobuf Go message 包含 `protoimpl.MessageState`，实现中额外修复了 handler/test 中按值复制 proto message 导致的 `go vet copylocks` 问题；这是接入 generated Go DTO 后的必要收敛，不属于功能扩张。
- Taskfile 工具链改动后，曾短暂尝试把 `buf` 记录到 `tools.go`，会导致大量 CLI 依赖进入 `go.mod`；最终已移除 `buf` import，并确认 `go.mod` / `go.sum` 无工具污染 diff。

## Risks

- 本次改动面大，包含大量 generated code 和前后端 DTO 引用替换，review 时应重点关注 proto source、handler DTO 构造和关键 update 语义，而不是逐行审阅生成产物。
- 当前 ts-proto 仍使用 Buf remote plugin；如果 CI 或离线环境无法访问 remote plugin，后续可再引入本地 TS plugin 安装策略。
- `tools.go` 保留 `protoc-gen-go` 记录，`buf` 由 Taskfile 固定版本安装到 `bin`，避免 CLI 大依赖污染业务 module；后续如需要统一工具版本锁定，可考虑专门的 tools module，而不是合入主业务 module。

## Incomplete items

暂无。所有计划内检查均已通过。

## Conclusion

验证通过。实现符合 Requirement / Spec / Plan：API body contract 已由 `orbit/api/v1/*.proto` 定义并生成 Go/TypeScript DTO；后端和前端已迁移到 generated DTO；Taskfile 已提供 `install` / `proto` 工具链入口，并从 `bin` 目录工具发起生成；未引入 route/service/RPC 或兼容 re-export 层。
