# 流水线变量嵌套 Liquid 值
最后修改时间: 2026-09-07 22:04:24

Review status: Accepted

Mode: standard

## Background

流水线阶段文本已经使用 Liquid 渲染，但变量的 `value` 与 `default` 目前始终被视为字面量。例如变量值为 `{{ repository_code }}-web` 时，阶段模板首次渲染后仍会保留该字符串，不会获得 `repository_code` 的实际值。

这使可复用的阶段模板无法通过通用变量组合运行时上下文与配置值。此前的阶段默认值规范限定 `default` filter 的参数为字面量；本需求仅扩展变量配置值的求值能力，不改变阶段字段本身的 Liquid 语法，也不引入领域对象或镜像命名等业务规则。

## Goal

1. 支持 Pipeline 变量的 `value` 使用嵌套 Liquid 变量引用，例如 `{{ repository_code }}-web`。
2. 提供独立于领域的嵌套引用解析能力，并在生成 Run 运行时变量前接入它，使阶段脚本及 Artifact 字段只接受最终值。
3. 保持现有全局与阶段作用域、来源优先级、Secret 标记和 Run 快照职责。
4. 在保存 Pipeline 变量时对未知引用、循环引用和无法渲染的变量值返回清晰错误，且不写入无效配置。

## Non-goal

- 不支持 `${NAME}`、`${NAME:-value}` 或其他 Shell 风格插值；嵌套值只使用 Liquid `{{ NAME }}` 语法。
- 不支持变量 `value` 内的 Liquid filter、tag、条件或表达式；变量 `default` 保持字面量，不参与嵌套解析。
- 不增加镜像名、组件名、制品、仓库或其他领域专用变量、默认值或自动命名规则。
- 不改变阶段 `script`、Artifact `reference`、`name`、`command` 的字段语法与渲染入口。
- 不新增 Pipeline 节点参数、Template Pipeline 阶段变量持久化、数据库表或迁移。
- 不修改既有 Pipeline、Run、Version 或 Service 数据，也不尝试修复历史制品引用。

## User scenarios

### 基于系统上下文构造变量值

Application Pipeline 为某个阶段配置：

```liquid
image_repository = {{ repository_code }}-web
```

阶段文本使用：

```liquid
{{ image_repository }}:{{ runtime_datetime }}
```

创建 Run 时，`image_repository` 先解析为实际仓库编码加后缀；阶段文本及最终 Run 快照中不保留嵌套 Liquid 标记。

### 多级变量引用

Pipeline 配置允许一个普通变量引用另一个普通变量，并解析为最终字符串。例如 `IMAGE={{ IMAGE_BASE }}-web`，`IMAGE_BASE={{ repository_code }}`。引用只在当前可见的全局作用域和当前阶段作用域内查找，阶段变量不得读取其他阶段的私有值。

解析链的最深处可使用该变量原有的字面量 `default` 作为回退值；例如 `IMAGE_BASE` 未配置 `value` 时，使用其 `default`。`default` 中不能再包含 Liquid 引用，且变量值中不支持 `default` filter。

### 错误配置

当变量引用不存在、出现直接或间接循环，或值中包含不支持的 Liquid 表达式时，保存 Pipeline 配置失败且不写入数据。错误至少包含相关变量名；循环错误还应说明依赖链。

## Acceptance

- [ ] 变量 `value` 可包含 `{{ NAME }}` 形式的 Liquid 引用，并在 Run 创建前解析为最终值。
- [ ] 变量 `default` 保持字面量，只能作为解析链叶子变量缺少 `value` 时的回退；不支持其内嵌 Liquid 引用。
- [ ] 嵌套值可以引用现有可见的系统、Repository、Pipeline 全局和当前 Stage 变量，且遵循现有来源优先级与 Stage 作用域隔离。
- [ ] 最终传给阶段字段渲染、Artifact 收集和 Run 变量快照的值不再含有可解析的嵌套 Liquid 标记。
- [ ] 支持多级引用；直接和间接循环均返回包含变量依赖链的 validation error。
- [ ] 保存时校验嵌套值语法、可见变量引用和直接/间接循环；校验失败时不写入 Pipeline 变量配置。
- [ ] 未知变量、未满足的必填引用和不支持的 Liquid 表达式均失败，不静默保留模板文本。
- [ ] `${NAME}`、`${NAME:-value}` 继续作为普通文本，不纳入变量值插值语义。
- [ ] 全局变量不可读取其他 Stage 私有值；Stage 变量不可读取其他 Stage 私有值。
- [ ] 现有非嵌套变量、Liquid `default` 字面量回退、变量优先级和阶段字段渲染行为保持不变。
- [ ] 测试覆盖系统变量引用、多级引用、全局/Stage 作用域、未知引用、直接循环、间接循环及非嵌套回归。

## Decisions

1. 嵌套值采用现有 Liquid 的简单输出引用 `{{ NAME }}`，不新增 Shell 插值、filter 或第二套表达式语言。
2. Liquid 渲染、阶段字段引用提取、`default` 字面量校验、嵌套语法解析、引用图构建、递归字符串求值与循环检测统一属于 `internal/common/template` 的标准能力，不依赖 Pipeline、Artifact、Version 或其他领域模型；业务层只能调用该能力，不自行解释 Liquid 文本。
3. Pipeline 变量层只负责依据既有来源优先级和全局/Stage 作用域构造输入、调用标准解析器并接收最终值；业务执行器、制品和 Version 逻辑不参与该规则。
4. 变量引用遵循可见性边界：全局解析只见全局值，Stage 解析可见全局值及本 Stage 值。
5. 解析链的叶子变量可使用现有的字面量 `default` 回退；中间 `value` 仅承担引用/拼接，不支持 `default` filter 或嵌套 `default`。
6. Pipeline 变量保存是配置正确性的主校验点：必须在持久化前检查语法、可见性和循环。Run 创建保留防御性校验，防止历史数据或绕过 API 的无效配置进入执行。
7. 相比“反复渲染直到不变”，标准解析器必须显式跟踪依赖并检测循环，避免无限循环或不确定的部分替换。
8. 本需求取代 `20260906-pipeline-stage-variable-defaults` 中“变量配置值不进行嵌套变量展开或二次模板解析”的限制；其余阶段字段默认值和作用域决策保持有效。

## Open questions

暂无需要用户确认的未决事项。

## Risk

- 变量来源与作用域优先级已存在于运行时解析中；若在多个位置分别做二次渲染，可能导致详情、Trigger、Retry 与执行器得到不同值。实现必须集中在单一解析入口。
- Secret 值参与解析时不能被错误消息、日志或 Run 详情泄露；保存时错误应只报告变量名和依赖关系，不回显值。
- 此需求不包含 Template Pipeline 节点级参数持久化；它能让已配置的值组合系统上下文，但不会自动为不同复用节点推导不同变量值。

## User review notes

用户要求采用标准模式，并明确限定为嵌套 `{{ xx }}` 语法的通用支持，不引入业务化规则。用户确认：只允许解析链叶子通过字面量 `default` 回退，变量值不支持 filter；嵌套引用在保存 Pipeline 配置时校验。用户进一步明确：先构建标准解析实现，再通过薄适配接入业务，不在业务层耦合解析逻辑。
