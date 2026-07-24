# Settings 普通配置列表及表单控件调整验证
最后修改时间: 2026-06-28 22:38:24

Review status: Accepted

当前流程: 轻量模式 / light
当前阶段: 验证 / Verification

## Requirement alignment

按 `docs/requirement/20260628-settings-plain-list.md` 及后续表单控件调整指令核对：

- `/settings` 前端已移除硬编码 `secretKeys` 集合。
- `/settings` 前端已移除基于字段名或 `item.secret` 的星号遮罩。
- `/settings` 前端输入框固定为普通 `text`，不再按 secret 切换为 `password`。
- Webhook 编辑表单已按普通表单处理：后端返回 `secret` 字段，前端编辑时填充 `wh.secret`，保存时提交 `secret: form.secret`。
- Webhook 签名密钥控件增加眼睛图标用于视觉显示/隐藏；该切换只影响 input type，不改变表单真实值。
- Webhook 签名密钥在打开创建/编辑弹窗时默认隐藏，点击眼睛后显示。

## Spec alignment

不适用。当前使用轻量模式 / light，未创建独立 Spec 文档。

## Plan alignment

不适用。当前使用轻量模式 / light，未创建独立 Plan 文档；按 requirement / 需求和用户后续明确指令核对。

## Actual diff summary

与本次验证直接相关的改动：

- `frontend/src/views/Settings.vue`
  - 移除 `secretKeys` 硬编码集合。
  - 移除 `••••••••` 展示。
  - 移除 secret 字段输入框 `password` 切换和空值保持逻辑。
- `frontend/src/views/ci/components/WebhookList.vue`
  - 编辑表单从 `wh.secret` 初始化签名密钥。
  - 创建和编辑都要求填写 `secret`。
  - 更新时始终提交 `secret: form.secret`。
  - 移除“留空则不修改”。
  - 增加 `Eye` / `EyeOff` 图标切换显示状态。
  - 默认 `isSecretVisible.value = false`，即表单打开时默认隐藏。
- `frontend/src/types/ci/webhook.ts`
  - `RepositoryWebhook` 增加 `secret: string`。
- `backend-go/internal/transport/http/handler/ci/repository.go`
  - `RepositoryWebhookResp` 增加 `secret` 字段。
  - webhook response 返回当前 `EncryptedSecret` 值作为普通表单字段值。
- `docs/requirement/20260628-settings-plain-list.md`
  - 记录 `/settings` 普通列表需求、前端 mask/password 位置分析、Webhook 表单控件调整和用户确认的边界。

## Expected vs actual changed files

预期范围内：

- `frontend/src/views/Settings.vue`
- `frontend/src/views/ci/components/WebhookList.vue`
- `frontend/src/types/ci/webhook.ts`
- `backend-go/internal/transport/http/handler/ci/repository.go`
- `docs/requirement/20260628-settings-plain-list.md`
- `docs/verification/20260628-settings-plain-list.md`

当前工作区还存在本验证未完全覆盖的其它已暂存改动：

- `backend-go/config.defaults.yaml`
- `backend-go/configs/config.example.yaml`
- `backend-go/configs/config.yaml`
- `backend-go/internal/config/config.go`
- `backend-go/internal/config/config_test.go`
- `backend-go/internal/service/ci/execution.go`
- `backend-go/internal/service/ci/runtime_variables.go`
- `backend-go/internal/service/ci/runtime_variables_test.go`
- `backend-go/internal/service/settings/service.go`
- `frontend/src/components/ApplicationCreateWizard.vue`
- `frontend/src/i18n/locales/en-US.ts`
- `frontend/src/i18n/locales/zh-CN.ts`
- `frontend/src/types/cd/settings.ts`
- `frontend/src/views/cd/ApplicationCreatePage.vue`

这些文件属于同一会话中的其它变更或早前变更，不全部属于本次 `/settings` 普通列表与 Webhook 表单控件调整验证的核心范围。最终提交前建议用户按实际交付边界自行审视是否拆分。

## Acceptance criteria checklist

- [x] `/settings` 前端代码中不存在 `secretKeys`。
- [x] `/settings` 前端不根据 `item.secret` 或字段名进行星号遮罩。
- [x] `/settings` 前端不根据 `item.secret` 或字段名把输入框切换为 `password`。
- [x] `/settings` 前端把配置项当普通列表项渲染。
- [x] Webhook 编辑弹窗不再“留空则不修改”。
- [x] Webhook 编辑弹窗使用后端返回的 `secret` 初始化表单。
- [x] Webhook 保存时始终提交 `secret: form.secret`。
- [x] Webhook 签名密钥输入框有眼睛图标，可切换显示/隐藏。
- [x] Webhook 表单打开时默认隐藏签名密钥，但真实值仍保存在表单状态中。
- [x] 前端 lint 和 typecheck 通过。

## Test results

已运行：

```text
yarn --cwd frontend lint --fix
结果：通过
```

```text
yarn --cwd frontend typecheck
结果：通过
```

```text
go test -C backend-go ./internal/transport/http ./internal/service/ci
结果：通过
```

```text
git diff --cached --check
结果：通过，无 whitespace error
```

补充检查：

```text
frontend/src/views/Settings.vue 中搜索 secretKeys|item.secret|type="password"|••••
结果：无匹配
```

```text
frontend/src/views/ci/components/WebhookList.vue 中确认：
- 存在 :type="isSecretVisible ? 'text' : 'password'"
- 存在 secret: wh.secret
- 存在默认 isSecretVisible.value = false
- 无“留空则不修改”/emptyKeepUnchanged/form.secret || undefined
```

## Missed or expanded scope

- 本次验证包含 `/settings` 普通列表与 Webhook 签名密钥表单控件调整两部分。Webhook 部分来自用户在需求记录后追加的明确指令。
- 工作区存在若干不属于本验证核心范围的已暂存改动，已在“Expected vs actual changed files”列出。
- 未执行完整 `go test -C backend-go ./...`，本轮只执行了与 HTTP handler / CI service 相关的 Go 测试。
- 未运行全量前端测试，仅运行了项目要求的 lint 和 typecheck。

## Risks

- `backend-go/internal/transport/http/handler/ci/repository.go` 当前将 webhook `secret` 返回给前端，以满足普通表单编辑真实值的需求。该 API 行为会扩大响应字段，需要用户确认这是期望的产品语义。
- Webhook secret 当前以字段名 `secret` 暴露给前端，符合表单语义；但如果后续恢复“不可回显 secret”的安全策略，需要另起需求重新设计。
- 当前 staged diff 混有其它变更，提交前建议按功能边界拆分或至少再次人工审视。

## Incomplete items

- 未对浏览器中的实际点击交互做手动验证。
- 未对 Webhook 创建/编辑接口做端到端 UI 验证。
- 未整理或拆分工作区中与本验证范围无关的其它 staged 改动。

## Conclusion

验证通过：`/settings` 已按普通配置列表处理；Webhook 签名密钥表单已按用户后续要求改为普通表单值，默认隐藏但可通过眼睛图标显示，且不再保留“留空则不修改”行为。

交付前主要注意事项是当前工作区包含其它范围的改动，建议用户在提交前按交付边界审视。
