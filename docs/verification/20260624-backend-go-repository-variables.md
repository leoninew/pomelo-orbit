# backend-go 仓库变量配置补齐验证
最后修改时间: 2026-06-24 19:15:15

Review status: Accepted

## Requirement alignment / 需求对齐

依据 `docs/requirement/20260624-backend-go-repository-variables.md` 验证，本次实现与需求对齐：

- 已对照 Python `backend` 的 `VariableResolver.get_repository_variables` 与 repository API 行为确认语义：仓库内置变量不是独立落表，而是根据 `Repository` 实体动态生成并追加自定义变量。
- `backend-go` 仓库详情现在生成仓库相关内置变量声明：
  - `repository_id`
  - `repository_name`
  - `repository_code`
  - `repository_url`
  - `repository_ref`
- 新建仓库响应已覆盖上述变量声明，因此“新建仓库后拥有变量配置”的用户可见行为已补齐。
- 自定义变量仍使用现有 `variable_overrides` 存储，没有新增表结构或修改迁移，符合不扩大 schema 范围的约束。
- 内置变量会从 `variable_overrides` 输入中过滤，避免用户自定义配置覆盖系统内置变量。

## Spec alignment / 规格对齐

不适用。当前为轻量模式 / light，未创建单独 spec 文档；按 requirement / 需求核对。

## Plan alignment / 计划对齐

不适用。当前为轻量模式 / light，未创建单独 plan 文档；按 requirement / 需求直接实现并验证。

## Actual diff summary / 实际 diff 摘要

### `backend-go/internal/service/ci/repository.go`

- `CreateRepository` 与 `UpdateRepository` 保存 `variable_overrides` 前调用 `sanitizeRepositoryVariables`。
- `repositoryDetail` 改为基于完整 `model.Repository` 生成变量声明。
- 新增仓库变量生成逻辑：
  - `repositoryVariables`
  - `repositoryCustomVariables`
  - `repositoryBuiltinVariableDeclarations`
  - `repositoryBuiltinVariableDeclaration`
  - `sanitizeRepositoryVariables`
- 仓库详情变量声明由“仅返回自定义变量”改为“内置变量 + 自定义变量”。

### `backend-go/internal/service/ci/repository_test.go`

- 更新 `TestRepositoryVariablesCompatibility`：
  - 覆盖空自定义变量时返回 5 个仓库内置变量。
  - 覆盖 `repository_id`、`repository_name`、`repository_ref` 的字段值、source 和 editable 行为。
  - 覆盖自定义变量追加为 `repository_custom`。
  - 覆盖输入中的内置变量不会作为自定义变量返回。
  - 保留 malformed JSON 的错误测试。

### `backend-go/internal/transport/http/repository_routes_test.go`

- 更新仓库创建路由测试：
  - 新建仓库响应期望 `variable_declarations` 包含 5 个内置变量 + 1 个自定义变量。
  - 断言 `repository_id` 与 `repository_name` 的默认值来自新建仓库实体。
  - 断言自定义变量 source 为 `repository_custom`。

### `docs/requirement/20260624-backend-go-repository-variables.md`

- 新增轻量模式需求文档并标记 `Review status: Accepted`。

### `docs/verification/20260624-backend-go-repository-variables.md`

- 新增本验证记录。

## Expected vs actual changed files / 预期与实际改动对比

预期改动：

- backend-go 仓库 service 层变量声明生成逻辑。
- backend-go 相关单元/HTTP 路由测试。
- SpecFlow requirement 与 verification 文档。

实际改动：

- `backend-go/internal/service/ci/repository.go`
- `backend-go/internal/service/ci/repository_test.go`
- `backend-go/internal/transport/http/repository_routes_test.go`
- `docs/requirement/20260624-backend-go-repository-variables.md`
- `docs/verification/20260624-backend-go-repository-variables.md`

结论：实际改动与预期范围一致。未修改 frontend、数据库迁移、运行时执行链路或 git 历史。

## Acceptance checklist / 验收清单

- [x] 对照 Python `backend` 找到新建仓库变量配置字段集合和语义。
- [x] 在 `backend-go` 仓库创建后的详情响应中补齐对应变量声明。
- [x] 新建仓库后至少生成 `repository_id`、`repository_name`；实际生成完整仓库内置变量集合。
- [x] 变量配置逻辑位于 service 层，HTTP handler 只负责请求/响应映射。
- [x] 未为业务字段添加无依据默认值；未使用 `getattr` 或防御性兼容写法。
- [x] 添加/更新 backend-go 测试，覆盖新建仓库变量声明生成。
- [x] 运行 backend-go 格式化与测试检查。

## Command results / 命令结果

### 格式化

```powershell
gofmt -w "backend-go/internal/service/ci/repository.go" "backend-go/internal/service/ci/repository_test.go" "backend-go/internal/transport/http/repository_routes_test.go"
```

结果：通过，无输出。

### 相关测试

第一次在仓库根目录运行：

```powershell
go test ./internal/service/ci ./internal/transport/http
```

结果：失败，原因是仓库根目录不是 Go module：

```text
go: cannot find main module, but found .git/config in D:\SourceCodes\mywork\pomelo-orbit
	to create a module there, run:
	go mod init
```

随后在 `backend-go` module 内运行：

```powershell
go -C "backend-go" test ./internal/service/ci ./internal/transport/http
```

结果：通过。

```text
ok  	backend/internal/service/ci	0.227s
ok  	backend/internal/transport/http	16.844s
```

### 完整 backend-go 测试

```powershell
go -C "backend-go" test ./...
```

结果：通过。

```text
ok  	backend/internal/service/ci	(cached)
ok  	backend/internal/transport/http	17.564s
ok  	backend/internal/worker/handler/ci	(cached)
```

其余无测试文件或 cached 包未发现失败。

## Missed or expanded scope / 范围偏差

- 未新增数据库表或迁移。对照 Python backend 后确认内置变量配置不是独立持久化配置，而是从仓库实体动态生成；因此没有进行 schema 扩展。
- 未新增历史仓库回填逻辑。由于内置变量由详情响应动态生成，历史仓库也会在读取详情时获得内置变量声明；自定义变量仍沿用已有 `variable_overrides`。
- 未修改 frontend。本次 API 响应字段结构仍是既有 `variable_declarations`，只补齐内容。

## Risks / 风险

- `git status --short` 在验证阶段显示本次改动已经位于 index 侧（`M  / A  `），但本会话未执行 `git add`。该状态应由用户确认是否符合预期；本次未执行任何 git 写操作。
- `repository_ref` 在仓库详情中使用 `default_branch` 作为默认值，运行时仍由触发分支覆盖注入；这与 Python backend 的“详情展示默认分支、运行时注入触发 ref”语义一致。
- 当前测试覆盖 service 与 HTTP 路由层，未启动真实服务做浏览器级手工验证。

## Incomplete items / 未完成项

- 无实现未完成项。
- 若用户希望验证真实页面展示，需要另行进入手工运行/浏览器验证流程；本次未启动开发服务器，符合项目指令。

## Conclusion / 结论

验证通过。backend-go 新建仓库后的详情响应已补齐 Python backend 语义下的仓库变量配置，包含 `repository_id`、`repository_name` 等内置变量，并保留自定义变量追加与内置变量保护逻辑。完整 backend-go 测试通过，未发现范围外代码改动。
