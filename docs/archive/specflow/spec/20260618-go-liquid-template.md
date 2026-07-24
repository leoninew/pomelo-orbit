# backend-go Liquid 模板实现迁移规格
最后修改时间: 2026-06-18 13:34:44

Review status: Accepted

## Requirement basis

基于 `docs/requirement/20260618-go-liquid-template.md`：

- 流程模式为 strict / 严格模式。
- Go 端模板实现选型为 `github.com/osteele/liquid`。
- 目标是将 `backend-go` 当前 Jinja 兼容模板实现替换为 Liquid 实现。
- `.jinja` 相关内容采用就地修改，不做历史迁移，不做历史兼容。
- 不保留 Jinja/Liquid 双轨兼容逻辑。
- 当前应该没有缺失变量场景；若发现缺失变量，不应静默容错。
- 现有 Jinja 风格模板、filter、默认值写法和示例都需要改为 Liquid 方式重新组织。
- 不修改 Python `backend`，不修改 frontend，不改变 CI/CD task 合同或 worker 分层结构。

## Overview

本次设计以 `internal/templatex` 作为 Go 后端统一模板入口，替换其内部实现：

```text
调用方保持使用 templatex.Render(input, values)
             ↓
internal/templatex 使用 github.com/osteele/liquid 渲染
             ↓
CI / CD / migration 获得 Liquid 渲染结果
```

这样可以把模板引擎替换限制在单一基础设施入口，同时让现有调用点继续表达“渲染模板字符串”这一业务意图。`templatex.Render` 不是兼容层；它是项目内部模板渲染入口，内部不再做 Jinja 语法转换、不再注册 Jinja filter、不再依赖 `pongo2`。

需要同步修改所有 Go 端存量模板文本和测试预期，使它们成为 Liquid 语法，而不是让 Liquid 引擎兼容旧 Jinja 表达。

## External library facts

基于 `https://github.com/osteele/liquid` 文档查证：

- Go import/module 为 `github.com/osteele/liquid`。
- 创建引擎使用 `liquid.NewEngine()`。
- 字符串模板渲染可使用：

  ```go
  engine := liquid.NewEngine()
  out, err := engine.ParseAndRenderString(template, bindings)
  ```

- bindings 使用 `map[string]any`，支持类似 `{{ page.title }}` 的属性访问。
- 支持 Liquid 风格 filter 管道，例如 `downcase`、`split`、`first`、`append` 等。
- 条件语句使用 Liquid 语义，`elsif` 而不是 Jinja 的 `elif`。
- 只有 `false` 和 `nil` 在条件中为 falsey。
- 文档未确认严格 undefined variable 模式；warn/lax error modes 未实现。
- `Engine.LaxFilters()` 会让未知 filter 静默透传，不符合本项目要求，本次不使用。
- 模板自身默认不能访问文件系统或网络，但 untrusted template 仍可能造成死循环或内存消耗；本次不新增用户任意模板执行能力，只替换现有渲染路径。

## Design decisions

### 1. 保留 `templatex.Render` 作为内部渲染入口

保留函数签名：

```go
func Render(input string, values map[string]any) (string, error)
```

设计理由：

- 当前 CI、CD、migration 都已经依赖这个入口，保留入口可以避免把模板引擎依赖散落到业务层。
- 入口名称是通用模板渲染，不携带 Jinja 语义；保留它不构成历史兼容。
- 内部实现必须删除 Jinja 兼容逻辑：
  - 删除 `convertJinjaSyntax`；
  - 删除 `jinjaDefaultFilter`；
  - 删除 `pongo2.RegisterFilter("d"/"default")`；
  - 删除 `pongo2` 依赖。

### 2. `templatex.Render` 使用 Liquid 引擎直接渲染

目标实现形态：

```go
engine := liquid.NewEngine()
out, err := engine.ParseAndRenderString(input, values)
```

错误处理：

- parse/render 错误统一包装为当前调用方可理解的错误信息，例如：
  - `template syntax error: ...`
  - `template render failed: ...`
- 不调用 `LaxFilters()`。
- 不注册旧 Jinja filter。
- 不将未知变量、未知 filter 或语法错误吞掉。

### 3. 缺失变量策略：显式检测，不依赖 Liquid 默认行为

由于 `osteele/liquid` 文档未确认 strict undefined variable 模式，而项目要求不能静默容错，本次继续在 `templatex` 层做缺失变量检测，但检测逻辑应面向 Liquid 语法，而不是 Jinja 兼容。

设计原则：

- 保留“渲染前发现明显缺失变量即报错”的行为。
- 检测对象聚焦当前项目模板实际使用的变量表达：
  - `{{ name }}`；
  - `{{ object.field }}`；
  - `{{ object.field | filter }}`。
- 不把 Liquid control-flow 中声明的循环变量误判为外部缺失变量。
- 不为旧 Jinja filter 做特殊兼容。
- 如果后续发现 Liquid 复杂表达超出现有检测能力，应在 Plan / Implementation 阶段优先补充测试并显式失败，而不是静默放行。

### 4. `.jinja` 就地替换为 Liquid 语义

根据用户决策：就地修改，不迁移，不历史兼容。

目标状态：

- 生产代码不再以 `.jinja` 判断是否需要渲染。
- 应用配置模板文件名改用 Liquid 语义，优先设计为 `.liquid`：
  - `docker-compose.yml.liquid` 渲染后写出为 `docker-compose.yml`；
  - `data/traefik.yml.liquid` 渲染后写出为 `data/traefik.yml`。
- `applicationComposeFile` / `ensureApplicationComposeFile` 识别目标 compose 文件时，应识别：
  - `docker-compose.yml`；
  - `docker-compose.yml.liquid`。
- 不保留 `docker-compose.yml.jinja` 的运行时兼容分支。
- migration 初始化数据中的路径和模板内容需要就地改为 Liquid 语义；这会修改 migration SQL，需要在 Plan 阶段明确风险，因为项目规范通常禁止修改已执行迁移文件。用户已经明确“不迁移、不历史兼容”，但实施前仍需在 Plan 中把该风险列为显式执行项。

### 5. Liquid 语法重组规则

当前 Go 端模板中主要 Jinja 风格需要改写为 Liquid：

- 条件分支：

  ```liquid
  {% if cert.letsencrypt.enabled %}
  ...
  {% elsif cert.letsencrypt.challenge == "dns" %}
  ...
  {% endif %}
  ```

  将 Jinja `elif` 改为 Liquid `elsif`。

- 默认值：

  当前 Jinja 风格：

  ```jinja
  {{ working_dir | default('') }}
  {{ repository_dockerfile | default('Dockerfile') }}
  ```

  目标 Liquid 风格：

  ```liquid
  {{ working_dir | default: '' }}
  {{ repository_dockerfile | default: 'Dockerfile' }}
  ```

- 变量访问：

  ```liquid
  {{ app.code }}
  {{ config.domain_suffix }}
  {{ repository_code }}
  ```

  简单变量和 dotted path 可以保持 Liquid 写法。

- Jinja alias filter `d(...)` 不保留；如出现必须改为 Liquid 标准写法。

### 6. CI 渲染路径

影响文件：

- `backend-go/internal/service/ci/execution.go`
- `backend-go/internal/service/ci/template_test.go`
- migration 初始化数据中的 `build_stage.script` 与 `build_stage.artifacts`

设计：

- `resolveStages` 继续通过 `templatex.Render` 渲染：
  - `StageDefinition.Script`；
  - `ArtifactConfig.Path`；
  - `ArtifactConfig.Name`。
- 模板内容改为 Liquid 语法。
- 缺失变量仍导致 stage resolution failed。
- 不改变 pipeline run 状态流转、不改变 artifact 插入逻辑、不改变 runner 行为。

### 7. CD 渲染路径

影响文件：

- `backend-go/internal/service/cd/application_extra.go`
- `backend-go/internal/service/cd/deployment_execution.go`
- `backend-go/internal/service/cd/service.go`
- `backend-go/internal/service/cd/compose_test.go`
- migration 初始化数据中的 `application_config_file.path/content`

设计：

- `renderApplicationTemplate` 继续作为 CD 应用模板上下文构造入口，内部调用 `templatex.Render`。
- 模板上下文保持：
  - `app.code`
  - `app.physical_dir`
  - `app.physical_app_dir`
  - `config.domain_suffix`
  - `cert.letsencrypt.*`
- preview 和实际部署都必须使用同一 `renderApplicationTemplate` 入口。
- `.liquid` 文件写出时移除 `.liquid` 后缀。
- 不再识别 `.jinja` 后缀。
- `docker-compose.yml` 仍表示无需模板渲染的静态 compose 文件。

### 8. migration 渲染路径

影响文件：

- `backend-go/internal/db/migrator.go`
- `backend-go/internal/db/migrator_test.go`
- `backend-go/internal/migrations/sqlite/v0.1.4__business_data.sql`
- `backend-go/internal/migrations/mysql/v0.1.4__business_data.sql`

设计：

- migrator 继续通过 `templatex.Render` 渲染 migration 文件内容。
- migration 中的模板语法改为 Liquid。
- migration 中应用配置模板路径从 `.jinja` 改为 `.liquid`。
- `default(...)` 改为 `default: ...`。
- `elif` 改为 `elsif`。
- mysql/sqlite 两份 business data 必须保持语义一致。

## Affected components

### 代码

- `backend-go/go.mod`
  - 移除 `github.com/flosch/pongo2/v6`。
  - 新增 `github.com/osteele/liquid`。
- `backend-go/go.sum`
  - 随依赖调整更新。
- `backend-go/internal/templatex/templatex.go`
  - 替换实现为 Liquid。
  - 删除 Jinja 转换和 Jinja filter。
  - 保留或重写缺失变量检测。
- `backend-go/internal/templatex/templatex_test.go`
  - 从 Jinja compatibility 测试改为 Liquid render 测试。
- `backend-go/internal/service/ci/template_test.go`
  - 验证 CI stage Liquid 变量与 artifact 渲染。
- `backend-go/internal/service/cd/compose_test.go`
  - 验证 CD Liquid render、缺失变量、preview/deploy 共享渲染语义。
- `backend-go/internal/service/cd/service.go`
  - compose 文件识别从 `.jinja` 调整为 `.liquid`。
- `backend-go/internal/service/cd/application_extra.go`
  - preview/compose service detection 从 `.jinja` 调整为 `.liquid`。
- `backend-go/internal/service/cd/deployment_execution.go`
  - deploy 写文件从 `.jinja` 后缀处理调整为 `.liquid`。
- `backend-go/internal/db/migrator_test.go`
  - 测试名称从 Go/Jinja 兼容语义调整为 Liquid 语义。

### 初始化数据

- `backend-go/internal/migrations/sqlite/v0.1.4__business_data.sql`
- `backend-go/internal/migrations/mysql/v0.1.4__business_data.sql`

这些文件包含：

- `data/traefik.yml.jinja`
- `docker-compose.yml.jinja`
- `{% elif ... %}`
- `{{ working_dir | default('') }}`
- `{{ repository_dockerfile | default('Dockerfile') }}`

目标是就地改为 Liquid 语义。

## Interfaces

### `templatex.Render`

保持接口：

```go
func Render(input string, values map[string]any) (string, error)
```

错误约定：

- 输入模板语法错误：返回 error。
- 渲染执行失败：返回 error。
- 显式检测到缺失变量：返回 error。
- 不静默吞掉未知 filter。

### CD 文件后缀处理

目标函数行为：

```text
path: docker-compose.yml         -> 不渲染，写 docker-compose.yml
path: docker-compose.yml.liquid  -> 渲染，写 docker-compose.yml
path: data/traefik.yml.liquid    -> 渲染，写 data/traefik.yml
```

不支持：

```text
path: *.jinja
```

### CI/CD task 合同

不变：

- `ci.pipeline_run.execute`
- `cd.application.deploy`
- `cd.application.restart`
- `pipeline_run_id`
- `variables`
- `application_id`
- `deployment_id`

## Technical questions

当前没有阻塞 Spec 的用户决策问题，但 Plan 阶段需要落到可执行细节：

- 是否需要为 Liquid 缺失变量检测增加更完整的 parser-based 检查，还是保留正则检测并限定在当前项目实际模板语法范围内。
- `osteele/liquid` 的 `default` filter 对空字符串的行为需要在实现阶段用测试锁定；如果行为不符合预期，应优先调整模板数据，避免注册兼容 filter。
- migration SQL 已执行文件就地修改会触发 checksum 风险；这是用户明确要求“不迁移、不历史兼容”带来的执行风险，Plan 必须单列。

## Risks

- 修改已执行 migration SQL 可能导致已有数据库 migration checksum mismatch。用户已明确不迁移、不兼容，因此这是接受的设计方向，但实现和验证时必须显式报告。
- 如果 Liquid `default` filter 对空字符串和 nil 的行为与当前 Jinja filter 不完全一致，CI 初始化 stage 的默认 working dir/dockerfile 行为可能变化，需要测试锁定。
- 如果缺失变量检测过窄，Liquid 可能把缺失变量渲染为空；这会违反项目原则。
- 如果缺失变量检测过宽，可能误判 Liquid loop 局部变量；需要用测试覆盖 `for` 场景。
- `.jinja` 改为 `.liquid` 后，已有数据库中的旧 path 不会自动更新；这是“不迁移、不历史兼容”的直接后果。
- CD preview 和 deploy 后缀处理必须同步，否则可能出现 preview 成功但 deploy 不渲染，或 deploy 成功但 preview 找不到 compose。

## Alternatives

### A. 直接在业务层使用 liquid.Engine

不采用。

原因：会让 CI、CD、migration 分别依赖模板引擎，未来变更和错误处理分散。

### B. 保留 Jinja/Liquid 双轨兼容

不采用。

原因：用户明确要求不迁移、不历史兼容；项目规范也禁止兼容层和别名。

### C. 继续使用 `.jinja` 文件名但内容改 Liquid

不采用。

原因：会形成长期认知债，且与“使用 Liquid 方式重新组织”的目标冲突。

### D. 新增 migration 做历史数据转换

不采用。

原因：用户明确要求不迁移、不历史兼容。

## User review notes

- 用户要求进入 Spec / 规格阶段，Requirement 已标记为 Accepted。
