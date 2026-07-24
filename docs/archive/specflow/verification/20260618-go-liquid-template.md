# backend-go Liquid 模板实现迁移验证
最后修改时间: 2026-06-18 14:17:29

Review status: Draft

## Requirement alignment

对照 `docs/requirement/20260618-go-liquid-template.md`：

- 已使用 strict / 严格模式推进，并已完成 Requirement、Spec、Plan、Implementation 后进入 Verification。
- Go 端模板依赖已切换为 `github.com/osteele/liquid`。
- `backend-go` 模板渲染入口已从 Pongo2/Jinja 兼容实现替换为 Liquid 实现。
- CI stage script、CI artifact path/name、CD config render、CD compose preview、CD deploy render、migration 初始化渲染均仍通过 Go 端模板入口覆盖。
- `.jinja` 路径和运行时识别已改为 `.liquid`，不保留 `.jinja` 兼容分支。
- 未修改 Python `backend` 和 frontend。
- 未改变 CI/CD task 类型、payload key、状态流转或 worker 分层结构。
- 相关 Go 测试和检查已运行并通过。

## Spec alignment

对照 `docs/spec/20260618-go-liquid-template.md`：

- `templatex.Render(input, values)` 作为项目内部统一模板渲染入口被保留。
- `templatex` 内部改为使用 `liquid.NewEngine()` 和 `ParseAndRenderString`。
- 已删除 Pongo2 依赖、Jinja 语法转换、Jinja `default`/`d` filter 注册和兼容函数。
- 未调用 `LaxFilters()`；未知 filter 由测试覆盖为错误。
- 缺失变量仍由 `templatex` 渲染前显式检测，覆盖简单变量、dotted path、condition 和 loop collection。
- Liquid loop 局部变量不会被误判为外部缺失变量。
- CD `.liquid` 文件在部署写出时移除 `.liquid` 后缀。
- migration 初始化数据中的 `.jinja` 路径已就地改为 `.liquid`，`elif` 已改为 Liquid `elsif`。

### Spec deviation / 用户后续裁定

Spec 中曾把 Jinja 默认值写法规划为 Liquid `default: ...`。实现阶段用户明确裁定“不还原写法”，因此最终实现保留显式变量引用：

```liquid
cd {{ working_dir }}
docker build -t {{ repository_code }}:{{ runtime_datetime }} -f {{ repository_dockerfile }} .
```

该点与 Plan 中“默认值由变量声明 / 触发变量负责，不通过模板 filter 隐式兜底”的实现策略一致。底层 Liquid `default:` 行为仍通过 renderer / CD / migration 测试覆盖，以证明引擎支持该语义，但初始化业务模板不依赖该 fallback。

## Plan alignment

对照 `docs/plan/20260618-go-liquid-template.md`：

- 依赖替换完成：`github.com/osteele/liquid` 已加入 Go module，`github.com/flosch/pongo2/v6` 已从当前源码依赖中移除。
- `internal/templatex` 已重写为 Liquid 实现，并保留缺失变量检测。
- CI template variable extraction 已从 Jinja `default(...)` / `d(...)` 改为 Liquid `default: ...`。
- CI 初始化 stage 模板按计划使用显式变量引用。
- CD compose 识别、preview 渲染和 deploy 写文件路径已统一从 `.jinja` 改为 `.liquid`。
- migration 渲染测试已改为 Liquid 语义。
- 旧 Jinja 痕迹已通过结构搜索确认清理。

## Actual diff summary

当前验证基于暂存区变更，涉及 17 个文件：

- `backend-go/go.mod`
- `backend-go/go.sum`
- `backend-go/internal/db/migrator_test.go`
- `backend-go/internal/migrations/mysql/v0.1.4__business_data.sql`
- `backend-go/internal/migrations/sqlite/v0.1.4__business_data.sql`
- `backend-go/internal/service/cd/application_extra.go`
- `backend-go/internal/service/cd/compose_test.go`
- `backend-go/internal/service/cd/deployment_execution.go`
- `backend-go/internal/service/cd/deployment_execution_test.go`
- `backend-go/internal/service/cd/service.go`
- `backend-go/internal/service/ci/template.go`
- `backend-go/internal/service/ci/template_test.go`
- `backend-go/internal/templatex/templatex.go`
- `backend-go/internal/templatex/templatex_test.go`
- `docs/plan/20260618-go-liquid-template.md`
- `docs/requirement/20260618-go-liquid-template.md`
- `docs/spec/20260618-go-liquid-template.md`

变更规模：`17 files changed, 1008 insertions(+), 132 deletions(-)`。

主要改动：

- 模板引擎依赖从 Pongo2/Jinja 兼容实现改为 `github.com/osteele/liquid`。
- `templatex` 使用 Liquid 直接渲染，并显式检查缺失变量。
- CI template variable parser 改为识别 Liquid `default: ...`。
- CD config file、compose preview、deployment write path 改为 `.liquid` 后缀语义。
- 初始化 SQL 中应用配置路径从 `.jinja` 改为 `.liquid`，Liquid 条件语法使用 `elsif`。
- 测试覆盖 Liquid 变量、filter、default、缺失变量、未知 filter、CD deploy 写出、migration render。

## Expected vs actual changed files

| 计划文件 | 实际状态 | 说明 |
| --- | --- | --- |
| `backend-go/go.mod` | 已修改 | 新增 Liquid 依赖并由 Go 工具更新 module 信息。 |
| `backend-go/go.sum` | 已修改 | 随 Go 依赖解析更新。 |
| `backend-go/internal/templatex/templatex.go` | 已修改 | Pongo2/Jinja 兼容实现替换为 Liquid。 |
| `backend-go/internal/templatex/templatex_test.go` | 已修改 | 改为 Liquid 行为测试。 |
| `backend-go/internal/service/ci/template.go` | 已修改 | 变量提取正则改为 Liquid default 写法。 |
| `backend-go/internal/service/ci/template_test.go` | 已修改 | CI stage 渲染和 Liquid default 提取测试。 |
| `backend-go/internal/service/cd/service.go` | 已修改 | compose 文件识别改为 `.liquid`。 |
| `backend-go/internal/service/cd/application_extra.go` | 已修改 | preview/render compose 改为 `.liquid`。 |
| `backend-go/internal/service/cd/deployment_execution.go` | 已修改 | deploy 写文件改为 `.liquid` 后缀处理。 |
| `backend-go/internal/service/cd/compose_test.go` | 已修改 | CD Liquid default / compose 测试。 |
| `backend-go/internal/service/cd/deployment_execution_test.go` | 已修改 | 新增 `.liquid` deploy 写出测试。 |
| `backend-go/internal/db/migrator_test.go` | 已修改 | migration Liquid render/default 测试。 |
| `backend-go/internal/migrations/sqlite/v0.1.4__business_data.sql` | 已修改 | `.jinja` -> `.liquid`，`elif` -> `elsif`，CI stage 显式变量。 |
| `backend-go/internal/migrations/mysql/v0.1.4__business_data.sql` | 已修改 | 与 SQLite 语义一致。 |
| `docs/requirement/20260618-go-liquid-template.md` | 已新增 | strict Requirement 文档。 |
| `docs/spec/20260618-go-liquid-template.md` | 已新增 | strict Spec 文档。 |
| `docs/plan/20260618-go-liquid-template.md` | 已新增 | strict Plan 文档。 |
| `docs/verification/20260618-go-liquid-template.md` | 已新增 | 本验证文档。 |

未发现计划外修改 Python `backend` 或 frontend。

## Acceptance checklist

- [x] 使用 strict / 严格模式推进到 Verification。
- [x] Go 模板依赖选型为 `github.com/osteele/liquid`。
- [x] `backend-go` 生产代码不再依赖 Pongo2/Jinja 兼容渲染实现。
- [x] CI stage script 和 artifact 渲染仍通过统一模板入口。
- [x] CD preview 和 deploy 使用 `.liquid` 后缀语义。
- [x] migration 初始化模板路径和条件语法改为 Liquid。
- [x] 不保留 `.jinja` runtime 兼容分支。
- [x] 未修改 frontend。
- [x] 未修改 Python `backend`。
- [x] Go format/test/vet 已通过。
- [x] 结构搜索未发现旧 Jinja/Pongo2 关键残留。

## Test results

### Format

```powershell
go -C "backend-go" fmt ./...
```

结果：通过，无输出。

### Focused tests

```powershell
go -C "backend-go" test ./internal/templatex ./internal/service/ci ./internal/service/cd ./internal/db
```

结果：通过。

```text
ok  backend/internal/templatex      (cached)
ok  backend/internal/service/ci     (cached)
ok  backend/internal/service/cd     (cached)
ok  backend/internal/db             (cached)
```

### Full tests

```powershell
go -C "backend-go" test ./...
```

结果：通过。

### Vet

```powershell
go -C "backend-go" vet ./...
```

结果：通过，无输出。

### Structure search

搜索范围：`backend-go/**/*.{go,sql,mod,sum}`。

搜索目标：

```text
pongo2
flosch
convertJinjaSyntax
jinjaDefaultFilter
.jinja
| d(
{% elif
default(
```

结果：未发现匹配。

## Missed or expanded scope

- 未运行 frontend 检查，因为本次没有 frontend 变更。
- 未运行 Python backend 检查，因为本次需求明确不修改 Python `backend`。
- 未执行开发服务器启动、停止或重启。
- 未执行任何 git 写操作，例如 commit、push、merge、reset、stash、checkout。

## Risks

- 已修改已执行 migration SQL：这会导致已有数据库中对应 migration checksum mismatch。该风险来自用户已确认的“就地修改、不迁移、不历史兼容”决策，需要交付时再次提示。
- `.jinja` runtime 支持已移除；旧数据库中已有 `.jinja` path 或 Jinja 内容不会被 Go 端继续兼容。
- CI 初始化 stage 当前依赖变量声明 / 触发变量提供 `working_dir` 和 `repository_dockerfile`，不再通过模板 `default` filter 做 fallback；这是用户在验证前明确裁定“不还原写法”后的实现状态。
- `go get` / `go mod tidy` 带来了 Go module directive 和若干 indirect dependency 更新；这是依赖引入过程的工具链结果，需由维护者确认是否接受。

## Incomplete items

暂无代码层未完成项。

Review status 仍为 `Draft`，等待用户确认本 Verification 文档。

## Conclusion

Verification 通过。当前实现满足已确认的 Liquid 迁移目标，并按用户最新裁定保留 CI 初始化 stage 中的显式变量写法，不恢复模板 default fallback。
