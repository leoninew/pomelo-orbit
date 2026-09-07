# 流水线变量嵌套 Liquid 值实施计划
最后修改时间: 2026-09-07 22:04:24

Review status: Accepted

Mode: standard

## Basis

- Requirement: [流水线变量嵌套 Liquid 值](../requirement/20260907-pipeline-variable-nested-liquid-values.md)（`Accepted`）
- 当前阶段字段通过 `internal/common/template.Render` 直接使用 Liquid 引擎渲染，但阶段表达式扫描、`default` filter 解析和 Shell 默认值校验仍位于 `pipelinevariable`。
- 当前 Pipeline 变量解析位于 `internal/application/pipeline/rule/pipelinevariable/runtime.go`：先选择来源优先级，再将未经二次求值的字符串放入 `RuntimeVariables`。
- 当前 `UpdatePipeline` 在 `internal/application/pipeline/usecase/pipeline.go` 中仅规范化变量 JSON；变量保存前没有嵌套引用的语法、可见性或循环校验。

本计划先将所有 Liquid 渲染、语法解析及变量引用提取收敛至 `internal/common/template`，在此基础上建立标准的嵌套字符串变量求值能力，再由 Pipeline 变量层薄适配既有来源优先级和作用域；不增加镜像、制品、组件或模板节点参数的领域规则。

## Design

### 标准嵌套值解析器

`internal/common/template` 新增独立 API，用于解析一组命名字符串模板及只读根上下文。该 API 不认识 Pipeline、Repository、Stage、Artifact、Secret、`value/default` 或任何领域来源；它只负责识别简单 `{{ NAME }}` 输出引用、在同一命名模板集合及根上下文中递归替换、返回最终字符串或未知引用/非法表达式/循环依赖错误。

调用者决定哪些候选字符串进入模板集合、哪些值为根上下文，以及解析结果应用到何处。标准 API 不反复调用完整 Liquid 引擎，不具有变量优先级或默认值语义。

### 受限嵌套值语法

变量 `value` 可由普通文本和零个或多个简单 Liquid 输出引用构成：

```liquid
{{ repository_code }}-web
prefix-{{ IMAGE_BASE }}-suffix
```

嵌套值解析器只接受 Pipeline 变量标识符形式的 `{{ NAME }}`。它拒绝 filter、tag、dotted path、条件、循环及不完整标签；`${...}` 不参与解析。阶段字段继续由既有完整 Liquid 渲染器处理，其 `default` filter 行为不变。

变量 `default` 始终是字面量。候选变量没有有效 `value` 时才可将该字面量作为解析链叶子；包含 Liquid 标记的 `default` 在保存时被拒绝。阶段字段中的 occurrence default 不加入嵌套变量依赖图，仍在既有阶段字段渲染时生效。

### Pipeline 薄适配

`pipelinevariable` 不实现词法解析、DFS 或循环检测。它只将现有变量来源选择转换为标准解析器的输入：

1. 按现有 Repository -> Pipeline Stage -> Pipeline 全局优先级选择每个变量的原始 `value`，无 `value` 时仅选择字面量 `default`。
2. 将系统变量和已解析的高层可见值作为标准解析器根上下文；将待解析的有效 Pipeline 字符串作为命名模板集合。
3. 为全局作用域和每个 Stage 分别构造可见输入，保证 Stage 只能覆盖或读取自身作用域，不能读取其他 Stage 私有值。
4. 将标准解析结果写入 `RuntimeVariables`，后续 Stage 字段渲染、Artifact 收集及 Run 快照只读取最终值。

字段 occurrence default 不作为标准解析器输入，仍在既有 Stage 字段渲染中按当前位置处理。

### 保存时校验

变量更新请求在写入 `pipeline.variable_declarations` 前通过 Pipeline 薄适配调用标准解析器：

- 解析器将系统变量名作为已知根节点，不要求在保存时生成其实际运行值。
- Application Pipeline 同时读取其绑定 Repository 的可配置变量名，验证 Pipeline 变量引用的可见性与优先级。
- Template Pipeline 没有绑定 Repository 时，仍可引用已知系统变量及同一 Pipeline 的全局变量；引用未知 Repository 自定义变量将被拒绝。
- 无效请求直接返回 validation error，Pipeline 版本和已存变量保持不变。

Run 创建沿用同一解析器形成最终 `RuntimeVariables`，但只作为防御性不变量检查，不能成为首次暴露配置错误的正常路径。

## Implementation Steps

1. **实现标准嵌套值解析 API**
   - 将阶段 Liquid 表达式的引用提取、`default` filter 字面量解析、Shell 风格默认值拒绝与既有 `Render` 收敛至 `internal/common/template`，使业务层不再解析 Liquid 文本。
   - 在 `internal/common/template` 增加公共 API：接收命名字符串模板集合和只读根上下文，返回解析后的字符串集合。
   - 在标准层完成受限 `{{ NAME }}` 词法解析、依赖图构建、DFS、直接/间接循环链和未知引用错误；保持现有 `Render` 的完整 Liquid 能力不变。
   - 用严格语法拒绝 filters、Liquid tags、dotted path、未闭合标签与非变量输出表达式；`${...}` 不被识别为引用。
   - 为标准 API 建立独立单元测试，覆盖纯文本、单个/多个引用、拼接、外部根值、多级依赖、未知引用、直接/间接循环及全部拒绝边界。

2. **编写 Pipeline 变量薄适配**
   - 在 `internal/application/pipeline/rule/pipelinevariable` 保留现有来源优先级和 Global/Stage 作用域选择，只把选择后的原始字符串与根上下文传给标准 API；删除其中的 Liquid 词法、filter 与表达式解析实现。
   - 无 `value` 时仅将字面量 `default` 作为叶子值；包含嵌套 Liquid 的 default 由适配层在保存校验时拒绝。
   - 将系统变量作为不可被配置覆盖的根值，确保 `repository_code`、`repository_url`、`repository_ref`、`runtime_datetime` 仍遵循当前注入职责。
   - 使用相同适配入口形成 `RuntimeVariables` 与保存校验输入，避免详情、Trigger、Retry 与执行器产生不同语义。

3. **接入 Pipeline 保存校验**
   - 在 `CreatePipeline`、`UpdatePipeline` 及任何会替换 `pipeline.variable_declarations` 的写路径中，在持久化前调用嵌套值校验。
   - Application Pipeline 校验时加载当前阶段定义和绑定 Repository；Template Pipeline 使用无 Repository 的静态可见域。
   - 校验失败时返回现有 validation 错误契约，不更新 Pipeline 版本、不写入 Repository 或 Pipeline 数据。
   - 保持 HTTP/Proto/TypeScript 请求契约不变，前端收到现有错误响应后继续在变量弹窗展示提交错误。

4. **保持运行时与快照收敛**
   - 调整 `ResolveRuntimeVariables` 的全局/Stage 值装配流程，使嵌套值在调用 `validateStageTemplates` 前完全解析。
   - 保持阶段 `script`、Artifact `reference`、`name`、`command` 的既有 `templatex.Render` 入口不变。
   - 确保 `MarshalRuntimeVariableSnapshot` 写入最终结果而非原始嵌套文本，`UnmarshalRuntimeVariableSnapshot` 无需引入二次求值。

5. **测试和活文档校准**
   - 在 `internal/common/template/template_test.go` 覆盖标准嵌套值解析 API，不使用 Pipeline fixture。
   - 在 `runtime_test.go` 覆盖系统变量拼接、全局多级引用、同一 Stage 的覆盖、跨 Stage 不可见、叶子字面量 default、未知引用、直接/间接循环、非嵌套回归和最终快照值。
   - 在 Pipeline use case 测试覆盖保存时拒绝无效变量更新，并断言未调用持久化或未增加版本。
   - 更新 `docs/guides/ci-pipeline-vars-design.md` 与 `docs/guides/ci-pipeline-design.md`，说明变量 `value` 的受限嵌套语法、字面量 default、保存时校验和作用域边界；不修改归档文档。

## Files To Change

| 区域 | 主要路径 |
| --- | --- |
| 标准嵌套值解析 | `internal/common/template/template.go`、`internal/common/template/template_test.go` |
| Pipeline 薄适配 | `internal/application/pipeline/rule/pipelinevariable/runtime.go`、`internal/application/pipeline/rule/pipelinevariable/runtime_test.go` |
| 保存校验 | `internal/application/pipeline/usecase/pipeline.go`、相关 use case 测试 |
| 活文档 | `docs/guides/ci-pipeline-vars-design.md`、`docs/guides/ci-pipeline-design.md` |

不新增数据库迁移，不修改 Proto、生成物或前端 API；不改动历史 Pipeline/Run/Version/Service 数据及不相关工作区改动。

## Verification Plan

1. 运行受限模板解析与 `pipelinevariable` 定向 Go 测试。
2. 运行 Pipeline use case 定向测试，验证保存失败不产生持久化副作用。
3. 运行 `task check` 与 `go test ./cmd/... ./internal/...`。
4. 审查变量保存、Run 创建、Retry 与 Stage 渲染路径，确认最终快照中没有可解析的嵌套 Liquid 标记。
5. 对照 Requirement 验收项检查：语法限制、叶子 default、作用域隔离、循环链错误、保存时失败和现有字段渲染回归。

## Risks And Assumptions

- `pipelinevariable` 的来源优先级和阶段 default 已在近期变更中调整；适配层必须以当前工作区代码为准，不重新引入按变量名压缩 Stage 值的行为，也不得将标准解析算法复制到业务包。
- 保存校验无法取得未来 `runtime_datetime` 的实际时间，但该变量作为已知系统根节点足以验证引用图；最终值仍在 Run 创建时生成。
- 旧数据可能包含此前可保存但按新规则无效的配置。它们不做迁移；运行时防御校验会拒绝执行，并在用户下次保存时要求修正。
- 变量值可能为 Secret。所有解析与错误路径不得记录或回显值。

## Rollback

本任务不涉及迁移或协议变更。若未完成交付，回退新增解析器调用及其测试即可；不得通过增加 `${...}`、任意 Liquid filter 或无限次渲染回退到模糊语义。

## User Review Notes

用户确认：嵌套值只允许简单 `{{ NAME }}` 引用；叶子使用字面量 default；变量值不支持 filter；保存时校验配置。用户进一步要求解析能力先作为标准实现落在通用模板层，再由业务层薄适配接入。
