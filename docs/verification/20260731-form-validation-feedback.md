# 表单校验反馈统一验证
最后修改时间: 2026-07-31 10:14:49

Review status: Draft

流程模式: 轻量 / light

## 需求对齐

- [需求](../requirement/20260731-form-validation-feedback.md) 的审查状态为 `Accepted`；轻量模式没有单独的规格或计划文档，按需求逐项核对。
- 业务表单的本地校验已改为字段级状态：失败字段使用 `app-input-error` 红框，并在控件正下方渲染 `app-field-error`；公共样式为 12px、正常字重、红色文本。
- `SelectControl`、`RawValueSelect` 和既有的 `ComboboxSelect` 都支持 `invalid` 红框，动态行的错误与对应行索引绑定。
- `AppDialogActions` 的确认按钮固定为 `type="button"`，不再接受或渲染跨表单 `form` 关联；业务模态窗改由自身确认事件处理。
- 应用、版本/组件、服务、网关、流水线、项目、仓库、凭据、角色、路由与用户等管理表单的逻辑必填字段已显示红色 `*`；可选字段不再显示“可选 / optional”文案。登录和密码类非业务表单未纳入此规则。
- 不对旧数据实施前端兼容或静默适配。保留配置值非法、无法归属单一字段的异步保存失败仍使用现有页面级错误或 Toast。

## 实际改动与范围

| 区域 | 结果 |
| --- | --- |
| 公共控件与样式 | `AppDialogActions` 取消跨层 form submit；选择控件支持 invalid 状态；`app-field-error` 统一为 12px 红色字段错误样式。 |
| 业务表单 | 34 个前端源文件改为控件旁字段错误、输入/选择时清错、必要的 `aria-invalid`，并补齐必填星号。 |
| 组件与服务配置 | 组件基础信息、健康检查、端口、环境变量、依赖、tmpfs、ulimit、挂载、受控文件，以及服务基础信息、运行时配置和暴露配置均获得字段级校验。 |
| 文案 | 中英文业务 UI 不再以“可选 / optional”向用户描述可选字段；无星号即代表可选。 |

预期范围为 `web/src/components/`、`web/src/style.css`、`web/src/i18n/locales/` 及受影响业务视图。实际范围与之匹配；`web/src/gen/proto/orbit/v1/application/version.ts` 及后端、MCP、SQL、RAGFlow 文档/脚本改动属于并行工作，不纳入本需求，也不会被本轮暂存。

## 验收清单

- [x] 已识别的本地业务校验在控件下方呈现 12px 红色文本，并有红色错误边框。
- [x] 原生输入、原生选择、`SelectControl`、`RawValueSelect` 和 `ComboboxSelect` 均可表达无效状态。
- [x] 本地必填/格式校验不再仅以 Toast 或底部通用 tip 反馈；网络、加载和无法归属单一字段的异步失败仍保持页面级反馈。
- [x] 未发现 `confirm-type`、`confirm-form`、`confirm-disabled` 或 Vue 模板 `form=` 的跨表单确认机制。
- [x] 未发现面向业务 UI 的“可选 / optional”显示文案；保留的 `optionalText` 等代码标识符和 i18n key 不会渲染该文案。
- [x] 业务表单的逻辑必填标签以红色 `*` 标注；登录与密码类表单未强制调整。
- [x] 未引入旧数据兼容或静默适配分支。

## 验证结果

| 命令或检查 | 结果 |
| --- | --- |
| `task check` | 通过：`vue-tsc --noEmit`、ESLint、Prettier、Go format 与 `golangci-lint` 均完成，无诊断；前端格式化输出为 unchanged，Go lint 报告 `0 issues`。 |
| `git diff --check` | 通过，无空白错误。 |
| 静态搜索 | 未发现跨表单确认属性；未发现会向业务 UI 显示“可选 / optional”的文案。 |
| 开发服务器与运行态交互 | 未执行，遵从用户自行测试且不操作开发服务器的约束。 |

## 范围偏差、风险与未完成项

- 本轮未运行 `task test` 或 Docker build；当前请求指定先运行 `task check`，且运行态交互由用户自行验证。
- 由于不启动或访问开发服务器，无法在浏览器中复核模态窗层级、焦点和逐字段消错动画；类型检查和静态审视覆盖了模板、props 与提交路径。
- 当前工作树包含 Version device、MCP、SQL 和 RAGFlow 相关的并行改动。暂存时必须只选择本需求的前端表单文件及本需求文档。

## 结论

实现与需求一致，`task check` 和静态审视均通过。验证记录保持 `Draft`，等待用户完成运行态交互验收后确认。
