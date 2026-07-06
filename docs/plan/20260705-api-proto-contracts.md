# API 请求响应 Proto 契约化计划
最后修改时间: 2026-07-06 00:00:00

## Review status

Accepted

## Basis

- Requirement: `docs/requirement/20260705-api-proto-contracts.md` (`Accepted`)
- Spec: `docs/spec/20260705-api-proto-contracts.md` (`Accepted`)
- 用户已要求“直接实现”，因此进入 Implementation / 实现。

## Implementation steps

1. **盘点 DTO 与引用**
   - 梳理 `internal/transport/http` 下所有 API 请求体/响应体 DTO。
   - 梳理 `web/src/types` 与 `web/src/api` 的手写类型对应关系。
   - 识别特殊字段：optional/pointer、`any`、`json.RawMessage`、自定义 `UnmarshalJSON`、列表/分页响应。

2. **引入 proto 工具链**
   - 添加 `buf.yaml`、根目录 `buf.gen.yaml`、`web/buf.gen.yaml`。
   - 添加生成脚本，尽量与现有 Go/前端工具链一致。
   - 确保 Go 生成目录和 TypeScript 生成目录清晰隔离。

3. **定义 proto schema**
   - 按业务域拆分 auth/user/role/project/ci/cd/task/settings/common 等 proto 文件。
   - 只定义请求体和响应体 message。
   - 对列表/分页响应定义具体 message，不定义错误响应。
   - 使用 wrapper/Value 等表达可选字段和动态 JSON。

4. **生成并接入后端类型**
   - 运行 proto 生成。
   - 将 `internal/transport/http/**/dto.go` 改为 generated Go 类型别名或删除手写 struct。
   - 调整 handler 构造、字段访问和 JSON decode 中的编译错误。
   - 保留必要的自定义 JSON presence 处理，例如 webhook branch filter 的显式置空。

5. **生成并接入前端类型**
   - 运行 TypeScript proto 生成。
   - 删除 `web/src/types/**` 中 API DTO 的重复 interface，不通过 re-export 做兼容出口。
   - 修正 `web/src/api/**` 与页面组件中的类型引用，使 API Req/Resp 直接从生成文件 import。

6. **格式化与检查**
   - 执行 `go fmt ./cmd/... ./internal/...`。
   - 执行 `go vet ./cmd/... ./internal/...`。
   - 执行 `go test ./cmd/... ./internal/...`。
   - 如果修改前端，执行 `yarn --cwd web lint:fix` 和 `yarn --cwd web typecheck`。

## Files to change

预计新增或修改：

- `buf.yaml`
- `buf.gen.yaml`
- `web/buf.gen.yaml`
- `orbit/api/v1/*.proto`
- `internal/transport/http/dto/proto/**`（生成产物）
- `internal/transport/http/**/dto.go`
- `internal/transport/http/**/*routes*.go` / handler 文件中 DTO 构造处
- `web/src/gen/proto/**`（生成产物）
- `web/src/types/**`
- `web/src/api/**`
- `go.mod` / `go.sum`（如需 protobuf runtime）
- `web/package.json` / lockfile（如需本地生成依赖或脚本）

## Verification plan

- 检查生成工具链可重复执行。
- Go 格式化、vet、测试全量通过或如实记录失败原因。
- 前端 lint fix 与 typecheck 通过或如实记录失败原因。
- 对照 diff 确认 proto 只包含请求体和响应体，没有新增 route/service 定义。

## Blockers

暂无必须由用户决定的阻塞项。若环境缺少生成工具且无法联网获取插件，将记录为实现风险并采用可复现的替代方式。

## Assumptions

- 可以参考 TermBridge 的 buf 配置和目录结构，但不复制与 Pomelo Orbit 无关的业务 proto。
- 当前 API 外部 JSON 字段名和路由保持不变。
- 生成产物允许入库，以便项目在未安装生成器时仍可编译和类型检查。

## Risks

- 工作面较大，编译修正可能牵涉多个 handler 和前端页面。
- proto optional/wrapper 与 JSON 编解码存在边界，需要通过测试覆盖关键 update 请求。
- 自动生成代码可能触发 lint 规则，需要确保生成目录被合理处理或生成代码本身通过检查。

## Rollback

- 回滚新增 proto 工具链配置、proto schema、生成产物与 DTO 替换改动。
- 因未执行 git 写操作，回滚由用户或后续授权的 git 操作完成。

## User review notes

- 用户已要求直接实现；本计划标记为 `Accepted`。
