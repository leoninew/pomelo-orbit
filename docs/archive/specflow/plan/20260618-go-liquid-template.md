# backend-go Liquid 模板实现迁移计划
最后修改时间: 2026-06-18 13:50:25

Review status: Accepted

## Requirement / Spec basis

基于：

- `docs/requirement/20260618-go-liquid-template.md`，Review status: Accepted
- `docs/spec/20260618-go-liquid-template.md`，Review status: Accepted

已确认决策：

- 继续使用 Liquid，不切换 Jet。
- Go 端依赖选型为 `github.com/osteele/liquid`。
- `.jinja` 相关内容就地修改为 Liquid 语义。
- 不做历史迁移，不做历史兼容，不保留 Jinja/Liquid 双轨。
- 不修改 Python `backend` 和 frontend。
- 不改变 CI/CD task 类型、payload key、状态流转或 worker 分层结构。

## Implementation steps

### 1. 替换 Go 模板依赖

改动文件：

- `backend-go/go.mod`
- `backend-go/go.sum`

步骤：

1. 在 `backend-go` 中引入 `github.com/osteele/liquid`。
2. 删除 `github.com/flosch/pongo2/v6` 依赖。
3. 通过 `go mod tidy` 或等价 Go 依赖整理命令更新 `go.mod` / `go.sum`。

注意：

- 不手写 `go.sum`。
- 不引入 Shopify 仓库。
- 不引入 Jet 或其他模板引擎。

### 2. 重写 `internal/templatex` 为 Liquid 实现

改动文件：

- `backend-go/internal/templatex/templatex.go`
- `backend-go/internal/templatex/templatex_test.go`

步骤：

1. 保留项目内部接口：

   ```go
   func Render(input string, values map[string]any) (string, error)
   ```

2. 内部使用：

   ```go
   engine := liquid.NewEngine()
   out, err := engine.ParseAndRenderString(input, values)
   ```

3. 删除所有 Jinja 兼容实现：
   - `convertJinjaSyntax`；
   - `jinjaDefaultFilter`；
   - `pongo2.RegisterFilter("d"/"default")`；
   - `pongo2.SetAutoescape(false)`；
   - `pongo2` import。

4. 不调用 `LaxFilters()`。
5. 保留并重写缺失变量检测，使其面向当前项目 Liquid 表达式：
   - 支持 `{{ name }}`；
   - 支持 `{{ object.field }}`；
   - 支持带 filter 的 `{{ name | default: 'x' }}`；
   - 避免误判 `{% for item in items %}{{ item }}{% endfor %}` 中的 loop variable。
6. 实现阶段查证发现 `github.com/osteele/liquid` 未在文档中声明 `default` filter，但实际库支持 `default: 'value'`；仅测试覆盖该行为，不把业务模板依赖在 fallback 上。CI 初始化模板改为使用显式变量，避免把默认值逻辑藏在模板引擎 filter 中。
6. 更新测试：
   - 简单变量；
   - dotted path；
   - number / bool 渲染；
   - `if / else / endif`；
   - `elsif`；
   - `for`；
   - Liquid `default: 'value'`；
   - 缺失变量报错；
   - 语法错误报错；
   - 不再测试 Jinja `d(...)` alias。

### 3. 更新 CI 模板测试和初始化 stage 模板

改动文件：

- `backend-go/internal/service/ci/template_test.go`
- `backend-go/internal/migrations/sqlite/v0.1.4__business_data.sql`
- `backend-go/internal/migrations/mysql/v0.1.4__business_data.sql`

步骤：

1. 保持 `service/ci/execution.go` 通过 `templatex.Render` 渲染：
   - `StageDefinition.Script`；
   - artifact `Path`；
   - artifact `Name`。
2. 更新 CI template tests 为 Liquid 语义。
3. 修改初始化 `build_stage` 中的 Jinja default 写法为显式 Liquid 变量引用：

   ```liquid
   {{ working_dir }}
   {{ repository_dockerfile }}
   ```

   默认值由变量声明 / 触发变量负责，不通过模板 filter 隐式兜底。

4. 保持 CI 变量名不变：
   - `repository_url`
   - `repository_ref`
   - `working_dir`
   - `repository_code`
   - `repository_dockerfile`
   - `runtime_datetime`
5. 保持 task payload、stage 执行、artifact 插入逻辑不变。

### 4. 更新 CD 模板后缀与渲染路径

改动文件：

- `backend-go/internal/service/cd/service.go`
- `backend-go/internal/service/cd/application_extra.go`
- `backend-go/internal/service/cd/deployment_execution.go`
- `backend-go/internal/service/cd/compose_test.go`
- `backend-go/internal/migrations/sqlite/v0.1.4__business_data.sql`
- `backend-go/internal/migrations/mysql/v0.1.4__business_data.sql`

步骤：

1. 将 compose 文件识别从：

   ```text
   docker-compose.yml
   docker-compose.yml.jinja
   ```

   改为：

   ```text
   docker-compose.yml
   docker-compose.yml.liquid
   ```

2. 将部署写文件逻辑从 `.jinja` 后缀处理改为 `.liquid` 后缀处理：

   ```text
   data/traefik.yml.liquid -> data/traefik.yml
   docker-compose.yml.liquid -> docker-compose.yml
   ```

3. 移除运行时 `.jinja` 识别逻辑，不保留兼容分支。
4. 保持 `renderApplicationTemplate` 上下文不变：
   - `app.code`
   - `app.physical_dir`
   - `app.physical_app_dir`
   - `config.domain_suffix`
   - `cert.letsencrypt.*`
5. 修改 CD 初始化配置文件 path：
   - `data/traefik.yml.jinja` -> `data/traefik.yml.liquid`
   - `docker-compose.yml.jinja` -> `docker-compose.yml.liquid`
6. 修改 CD 初始化模板语法：
   - `{% elif ... %}` -> `{% elsif ... %}`
   - 保留 Liquid 支持的变量访问 `{{ app.code }}` / `{{ config.domain_suffix }}`。
7. 更新 CD compose tests：
   - Liquid 渲染成功；
   - 缺失变量报错；
   - `.liquid` compose preview；
   - `.liquid` deploy 写出目标文件；
   - route label 注入仍可工作。

### 5. 更新 migration 渲染测试

改动文件：

- `backend-go/internal/db/migrator_test.go`

步骤：

1. 将测试名称从 Go/Jinja 兼容语义改为 Liquid 语义。
2. 保持 migration 内容渲染仍通过 `templatex.Render`。
3. 增加或更新 Liquid default / missing variable 测试。

### 6. 全局清理旧 Jinja 痕迹

只读搜索目标：

- `pongo2`
- `convertJinjaSyntax`
- `jinjaDefaultFilter`
- `.jinja`
- `default(`
- `| d(`
- `{% elif`

步骤：

1. 搜索 `backend-go` 下旧 Jinja 引用。
2. 保证生产代码、测试、初始化 SQL 中不再出现 Jinja 兼容实现。
3. 如果日志文件或非源码生成文件出现旧字符串，不纳入产品改动范围，但最终汇报需说明。

## Files to change

预计修改：

- `backend-go/go.mod`
- `backend-go/go.sum`
- `backend-go/internal/templatex/templatex.go`
- `backend-go/internal/templatex/templatex_test.go`
- `backend-go/internal/service/ci/template_test.go`
- `backend-go/internal/service/cd/service.go`
- `backend-go/internal/service/cd/application_extra.go`
- `backend-go/internal/service/cd/deployment_execution.go`
- `backend-go/internal/service/cd/compose_test.go`
- `backend-go/internal/db/migrator_test.go`
- `backend-go/internal/migrations/sqlite/v0.1.4__business_data.sql`
- `backend-go/internal/migrations/mysql/v0.1.4__business_data.sql`

可能无需修改但需检查：

- `backend-go/internal/service/ci/execution.go`
- `backend-go/internal/db/migrator.go`

不修改：

- Python `backend/`
- `frontend/`
- CI/CD task status constants
- worker handler task payload 合同
- Git 状态，不执行 git 写操作

## Verification plan

### 局部测试

在 `backend-go` 下执行：

```powershell
go test ./internal/templatex
go test ./internal/service/ci
go test ./internal/service/cd
go test ./internal/db
```

重点验证：

- Liquid 基础变量渲染；
- dotted path 渲染；
- `if / else / elsif / endif`；
- `for`；
- `default: ...`；
- 缺失变量报错；
- CI stage script/artifact 渲染；
- CD preview 与 deploy 使用 `.liquid`；
- migration 内容渲染。

### 全量 Go 检查

在 `backend-go` 下执行：

```powershell
go fmt ./...
go test ./...
go vet ./...
```

### 结构搜索

确认旧实现消失：

```text
pongo2
convertJinjaSyntax
jinjaDefaultFilter
| d(
{% elif
.jinja
```

说明：如果 `backend-go/logs/*.log` 这类生成日志中仍有旧字符串，不作为源码问题；最终汇报需明确排除范围。

## Blockers

当前没有需要用户再次决策的阻塞项。

## Assumptions

- 用户已经确认继续 Liquid，不切换 Jet。
- 用户已经确认就地修改，不迁移，不历史兼容。
- 当前不需要支持旧数据库中已经存在的 `.jinja` path 或 Jinja 内容。
- 当前 Go 端模板实际用法可以通过项目内 `templatex.Render` 统一覆盖。
- `github.com/osteele/liquid` 的 `default` filter 行为仅用于底层 renderer 测试锁定；业务初始化模板不依赖 fallback filter，而使用显式变量。

## Risks

- 修改已执行 migration SQL 会导致已有数据库 migration checksum mismatch。这是用户明确“不迁移、不历史兼容、就地修改”带来的风险，实施后需要在结果中再次提示。
- 删除 `.jinja` runtime 支持后，旧数据库中已有 `.jinja` 配置文件不会再被 Go 端识别为模板。
- Liquid 缺失变量默认行为可能较宽松；必须依赖 `templatex` 显式检测和测试来防止静默空值。
- Liquid `default` 对空字符串 / nil 的行为需要测试锁定，不能凭经验假设；业务模板不应依赖该 fallback 行为。
- migration SQL 中同时存在 SQL 字符串转义和 Liquid 引号，修改时要避免破坏 SQL 字符串。

## Rollback

如果实现阶段发现 Liquid 行为无法满足需求：

1. 不回退为 Jinja 兼容层。
2. 停止实现并回到 Spec 阶段，重新确认模板引擎或语义方案。
3. 已修改文件通过普通文件编辑恢复到实现前内容，不执行 `git reset`、`git checkout` 等 git 写操作，除非用户明确授权。

## User review notes

- 用户询问是否改 Jet 更容易；已评估后建议继续 Liquid。
- 用户确认：继续 Liquid，进入 Plan。
