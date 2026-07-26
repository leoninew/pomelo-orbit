# 枚举标签保留原始值验证

最后修改时间: 2026-07-26 09:43:49

Review status: Draft

## Requirement alignment

与 [Requirement](../requirement/20260726-enum-label-raw-values.md) 一致。接口字符串枚举在展示与选择器中直接使用原始值；字段名、按钮、提示和布尔型业务状态继续使用现有本地化文案。

## Spec alignment

不适用。该任务采用轻量模式 / light，未创建 Spec。

## Plan alignment

不适用。该任务采用轻量模式 / light，按 Requirement 直接实施。

## Actual diff summary

- 应用、版本、部署、服务、网关、流水线、制品、变量、凭据、用户及项目成员页面的枚举展示改为原始值。
- 应用类型、镜像拉取策略、状态、认证来源、制品类型、变量来源等选择器直接使用原始值数组。
- 枚举和值选择使用 `RawValueSelect`、`RawValueCombobox`，直接将原始值传给控件；通用选择控件仍只接收确有自定义标签、描述或禁用态的对象选项。
- 移除接口枚举的伪默认值与标签回退，不以其他枚举值掩盖缺失或异常值。
- 枚举展示统一使用带边框的 `AppBadge`；状态继续使用现有状态色，类型、策略、模式、协议与认证来源使用 `pill` 标签。
- 删除枚举 i18n 词条、`te(...)` 枚举回退路径及未再使用的 `credentialTypeLabels` 常量。

## Expected vs actual changed files

| 范围 | 预期 | 实际 | 结论 |
| --- | --- | --- | --- |
| `web/src/views/**` | 更新枚举展示与选择器 | 应用、部署、服务、网关、流水线、凭据、用户等 22 个视图/组件已更新 | 一致 |
| `web/src/components/{RawValueSelect,RawValueCombobox,ApplicationFormFields}.vue` | 枚举选项直接使用原始值数组 | 已新增原始值控件；通用选择控件保留对象选项对自定义文案与禁用态的支持 | 一致 |
| `web/src/views/settings/Settings.vue` | 消除同值布尔选择项对象映射 | 已改为原始值数组，界面行为不变 | 一致 |
| `web/src/i18n/locales/*.ts` | 清理废弃枚举翻译 | 中英文词条已清理 | 一致 |
| `web/src/constants/credential.ts` | 移除未使用类型标签表 | 文件已删除 | 一致 |
| `web/src/router/index.test.ts` | 只验证现有业务路由 | 已移除对不存在 legacy path 的负向断言 | 一致 |
| `docs/requirement/20260726-enum-label-raw-values.md` | 记录并接受需求 | 已新增，状态为 `Accepted` | 一致 |
| `docs/verification/20260726-enum-label-raw-values.md` | 记录验证结果 | 已新增 | 一致 |

## Acceptance criteria

- [x] 目标页面不再通过 i18n 映射接口枚举标签。
- [x] 枚举选择器直接使用原始值数组，显示与提交相同的原始值。
- [x] 枚举实现没有标签对象规范化或伪默认值回退。
- [x] 枚举展示使用一致的带边框标签样式。
- [x] 废弃 i18n 映射和凭据标签常量已删除，无残留调用方。
- [x] 前端测试、类型检查、lint 和 diff 检查通过。

## Test results

| 命令 | 结果 |
| --- | --- |
| `npm run test` | 通过，7 个测试文件、38 个用例 |
| `npm run typecheck` | 通过 |
| `npm run lint` | 通过 |
| `git diff --cached --check` | 通过 |
| `rg -n "\\bte\\(" web` | 无匹配 |
| 同值 `value` / `label` 对象映射扫描 | 无匹配 |

路由回归测试只覆盖当前存在的业务路由。按需求移除了对不存在 legacy path 的负向断言，不为任意缺失功能或路径建立专门测试。

## Scope deviation

无。处理范围从应用、版本和部署页扩展至其余前端页面，是用户明确要求继续检查与处理其他枚举后的结果；路由测试仅移除不应维护的不存在路径断言。

## Risks

- 原始枚举值可读性低于自然语言标签，这是需求规定的行为。
- 后端新增枚举值会直接展示给用户，后端值命名需要保持适合作为用户可见文本。
- 未运行浏览器端到端测试；本次验证覆盖前端单元测试、类型与静态检查。

## Incomplete items

无。

## Conclusion

实现与 Requirement 对齐，验证通过。暂存区仅包含枚举任务的前端改动及其 Requirement/Verification 文档；MCP requirement 未被暂存。
