# 登录 CSRF Fernet 校验恢复验证
最后修改时间: 2026-08-08 14:38:02

Review status: Accepted

流程模式: 标准 / standard

## Requirement alignment

对照 [需求](../requirement/20260808-csrf-fernet-restore.md)：

| 验收项 | 结果 |
| --- | --- |
| Fernet 签发 CSRF，密钥 `jwt.secret_key`，明文 `csrf` | 通过 — `csrf.Issue` + bootstrap 注入 |
| Fernet 内置 timestamp + TTL 3 分钟 | 通过 — `DecryptStringWithTTL` + `csrf.TokenTTL` |
| 类型 / 过期 / 篡改 / 空 token 拒绝 | 通过 — 单元与集成测试 |
| Login 先验 CSRF | 通过 — `service.Login` 在凭据校验前 `csrf.Verify` |
| 登录页未 ready 禁用 Login | 通过 — `canSubmit` / `disabled` |
| 3 分钟后不可登录并提示刷新 | 通过 — `CSRF_SESSION_MS` + `sessionExpired` UI |
| 无静默续签 | 通过 — 已删除失败后 `getCsrfToken` 续签 |
| 硬编码 `"csrf"` 调用方改掉 | 通过 — 仅保留「应失败」用例 |
| 无新旧并存 / 随机 hex 占位 | 通过 |

用户确认：**验证无问题**。

## Spec alignment

标准模式无独立 spec 文档；按 requirement + plan 核对，无额外规格偏差。

## Plan alignment

对照 [计划](../plan/20260808-csrf-fernet-restore.md)：步骤 1–6 均已落地。

- Fernet TTL API、auth/csrf、Service 接线、前端门闩与 3 分钟 UI、测试修正、质量门均完成。
- 未引入 JWT CSRF、Cookie 双提交或静默续签。

## Actual diff summary

已暂存范围（CSRF 相关，15 文件）：

- `internal/common/crypto`：`DecryptStringWithTTL`
- `internal/auth/csrf`：Fernet Issue/Verify
- `internal/application/auth/usecase`：密钥注入 + 登录校验
- `internal/bootstrap/http.go`：注入 `cfg.Jwt.SecretKey`
- 测试：auth 集成、authz、e2e 真实 CSRF
- `web/src/views/auth/Login.vue` + 中英文 login 文案
- SpecFlow requirement/plan 文档

## Expected vs actual changed files

| 预期（plan） | 实际 | 备注 |
| --- | --- | --- |
| fernet.go / fernet_test.go | 是 | |
| auth/csrf | 是 | 含 csrf_test.go |
| auth usecase + 集成测 | 是 | |
| bootstrap/http.go | 是 | |
| authz_test / e2e | 是 | |
| Login.vue + i18n | 是 | i18n 仅 CSRF 键进暂存 |
| Proto | 否 | 符合 plan「不改」 |
| Credential 路径 | 否 | DecryptString 无 TTL 行为保留 |

## Acceptance checklist

- [x] Fernet CSRF 签发与 3 分钟 TTL 校验
- [x] 登录强制有效 CSRF
- [x] 登录页初始化门闩与 3 分钟会话失效 UI
- [x] 无静默续签
- [x] 测试不再依赖任意字符串 `"csrf"` 登录成功
- [x] 质量门通过

## Test results

```
go fmt ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./internal/common/crypto/ ./internal/auth/csrf/ \
  ./internal/application/auth/... ./internal/api/http/security/ \
  ./internal/bootstrap/ ./internal/test/e2e/ -count=1
# all ok (e2e skipped without BACKEND_GO_E2E_CONFIG)

yarn --cwd web lint:fix   # ok
yarn --cwd web typecheck     # ok
```

手工：用户确认验证无问题。

## Missed or expanded scope

- 无范围外实现混入 CSRF 暂存区。
- 工作区仍有其它未暂存 feature（service-code、pipeline stage 等），与本验证无关。

## Risks

- 运行环境 `jwt.secret_key` 必须为合法 Fernet key，否则签发失败。
- 3 分钟窗口对慢填表单用户不友好，已为确认产品行为。
- MySQL e2e 依赖外部配置，本地未强制执行；逻辑由集成测覆盖。

## Incomplete items

无。

## Conclusion

**通过。** 登录 CSRF Fernet 校验恢复满足需求与计划；自动化检查通过，用户确认无问题，可交付提交。
