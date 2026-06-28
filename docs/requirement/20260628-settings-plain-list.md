# Settings 普通配置列表及表单控件调整需求
最后修改时间: 2026-06-28 22:38:24

Review status: Accepted

当前流程: 轻量模式 / light
当前阶段: 验证 / Verification

## Background

`/settings` 页面用于展示和编辑配置项。此前页面和后端存在 secret / mask / password 相关逻辑，容易造成配置项来源、展示方式和实际值返回语义混淆。

用户明确要求：`/settings` 只是普通配置列表。同时，针对 Webhook 编辑表单，用户要求它也是普通表单，不要“留空则不修改”，但签名密钥控件可以增加类似 `CredentialDetail.vue` 的眼睛图标用于显示/隐藏。

## Frontend mask / password usage inventory

此前分析到的前端 mask / password 相关位置如下，用于说明边界：

- `frontend/src/views/Settings.vue`
  - 原先存在 `secretKeys = new Set(['jwt__secret_key'])`。
  - 原先对命中字段使用 `type="password"`。
  - 原先对命中字段展示 `••••••••`。
  - 本需求要求移除这些 `/settings` 专属遮罩逻辑。
- `frontend/src/views/ci/components/WebhookList.vue`
  - Webhook 编辑弹窗中 `openEditModal()` 原先将 `secret` 置为空字符串。
  - Webhook secret 输入框原先使用 `type="password"`。
  - 更新时原先使用 `secret: form.secret || undefined`，空值表示不修改。
  - 本需求要求移除“留空则不修改”，让它按普通表单字段处理；后续采纳的控件调整是默认隐藏真实值，但真实值保留在表单状态中，可用眼睛图标切换显示。
- `frontend/src/views/ci/CredentialDetail.vue`
  - 凭据详情页根据 `isCredentialDataVisible` 在真实值和 `********` 之间切换。
  - 用户确认该处理合适，不需要特殊处理。
- `frontend/src/views/ci/components/TriggerModal.vue`
  - CI 触发弹窗根据 `variable.secret` 在 `password` 和 `text` 输入之间切换。
  - 用户确认该位置无须特殊处理。
- 账号密码相关页面和组件，例如 `Login.vue`、`UserPage.vue`、`UserDetail.vue`、`AppTopBar.vue`
  - 登录、用户管理、修改密码等表单使用 `type="password"`。
  - 用户确认用户及登录相关实现合理，不需要特殊处理。

## Goal

- `/settings` 按普通配置列表展示配置项。
- 配置项是否展示、是否可编辑，应来自 settings 配置列表本身，而不是在前端额外硬编码一份字段集合。
- `/settings` 配置项值按普通值展示和编辑，不在 `/settings` 页面做星号遮罩。
- `/settings` 配置项输入框按普通文本输入处理，不因为字段名或 secret 元数据切换为 password 输入框。
- 如果存在用于标记哪些字段是 secret 的配置，例如 `settings.secret_keys` / `settings__secret_keys`，它本身也按普通配置项处理。
- Webhook 签名密钥编辑表单移除“留空则不修改”行为。
- Webhook 签名密钥编辑表单使用后端返回值初始化表单字段。
- Webhook 签名密钥保存时始终提交表单中的真实值。
- Webhook 签名密钥控件默认隐藏显示，但可通过眼睛图标切换显示/隐藏；该切换只影响视觉展示，不改变表单值。

## Non-goal

- 不在本需求中重新设计权限模型。
- 不在本需求中新增复杂的配置项 schema 系统。
- 不在本需求中实现专门的数组编辑器、多行编辑器或高级表单控件。
- 不移除登录、用户管理、修改密码、凭据详情、CI 触发变量等其它页面自身合理的 password / mask 行为。
- 不调整 `TriggerModal.vue`。
- 不把 Webhook 签名密钥改成星号假值或后端脱敏值。

## User scenarios

- 用户打开 `/settings`，看到的是配置项列表，而不是根据前端硬编码规则隐藏部分值的特殊页面。
- 用户搜索配置项时，普通配置项和 `settings__secret_keys` 一样参与列表展示和搜索。
- 用户编辑 `/settings` 配置项时，输入框是普通文本输入，不自动切换为 password。
- 用户查看 `/settings` 配置项当前值时，页面展示实际返回值，不显示 `••••••••`。
- 用户打开 Webhook 编辑弹窗时，签名密钥字段已有真实表单值，但默认隐藏显示。
- 用户点击 Webhook 签名密钥眼睛图标时，可以在显示和隐藏之间切换。
- 用户保存 Webhook 表单时，提交当前表单中的签名密钥值，不再通过空值表达“不修改”。

## Acceptance

- `/settings` 前端代码中不存在硬编码 secret key 集合，例如 `secretKeys = new Set(...)`。
- `/settings` 前端不根据 `item.secret` 或字段名进行星号遮罩。
- `/settings` 前端不根据 `item.secret` 或字段名把输入框切换为 `password`。
- `/settings` 前端把配置项当普通列表项渲染；实际能看到哪些配置项由接口返回决定。
- `WebhookList.vue` 不再出现“留空则不修改”文案。
- `WebhookList.vue` 编辑表单从返回的 webhook 数据初始化 `secret`。
- `WebhookList.vue` 更新时提交 `secret: form.secret`，不使用 `form.secret || undefined`。
- `WebhookList.vue` 签名密钥控件默认隐藏，眼睛图标可切换显示/隐藏。
- 其它页面已有的 password / mask 行为不因本需求被扩散修改。
- 前端 lint 和 typecheck 通过。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- `/settings` 是普通配置列表，不是 secret 管理页。
- 不在前端硬编码哪些字段是 secret。
- 不在 `/settings` 页面做 mask / password 行为。
- 用户及登录相关 password 实现合理，不需要特殊处理。
- `CredentialDetail.vue` 的显示/隐藏凭据内容处理合适，不需要特殊处理。
- `TriggerModal.vue` 无须特殊处理。
- Webhook 签名密钥是普通表单字段；默认隐藏只是控件视觉状态，不是脱敏、不是真值替换、不代表“留空不修改”。

## Risk

- Webhook secret 需要由后端返回给前端，才能在编辑表单中填入真实值。该 API 行为需要保持与“普通表单”语义一致。
- 如果后续需要恢复“不可回显 secret”的安全策略，应作为新的独立需求处理，不能混入本需求。

## User review notes

- 2026-06-28：用户强调“/settings 只是普通列表”。
- 2026-06-28：用户要求在文档中列举此前分析到的前端 mask 使用位置。
- 2026-06-28：用户确认用户及登录相关实现合理，`CredentialDetail.vue` 处理合适，其它位置无须特殊处理。
- 2026-06-28：用户要求处理 Webhook 表单，移除“留空则不修改”。
- 2026-06-28：用户采纳 Webhook 签名密钥眼睛图标 UX，并要求表单默认不显示。
