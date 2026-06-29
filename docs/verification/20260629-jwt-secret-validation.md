# JWT Secret 校验修复验证
最后修改时间: 2026-06-29 11:39:03

Review status: Accepted

## Requirement alignment

按 `docs/requirement/20260629-jwt-secret-validation.md` 核对：

- 修复 `jwt.secret_key` 配置校验，使其符合 JWT HMAC secret 的实际使用方式：已完成。
- 删除 Fernet/base64 decode 校验逻辑：已完成。
- 要求 `len(secretKey) >= 32`：已完成，实际以 `strings.TrimSpace(secretKey)` 后的字符串长度校验。
- 更新配置相关测试用例，覆盖标准 Base64 secret 可用和短 secret 被拒绝：已完成。

## Spec alignment

不适用。当前变更采用轻量模式 / light，未创建独立 spec 文档；按 requirement / 需求核对。

## Plan alignment

不适用。当前变更采用轻量模式 / light，未创建独立 plan 文档；按 requirement / 需求核对。

## Actual diff summary

- `internal/config/config.go`
  - 删除 `encoding/base64` import。
  - 删除 `fernetKeySize` 常量。
  - 新增 `jwtSecretKeyMinLength = 32`。
  - 将 `Validate()` 中的 JWT secret 校验从 `validateFernetKey` 切换为 `validateJWTSecretKey`。
  - `validateJWTSecretKey` 只校验非空和长度不少于 32 字符，不再执行 Fernet/base64 decode。
- `internal/config/config_test.go`
  - 将配置测试中的 Fernet 语义命名改为 JWT secret 语义。
  - 使用包含 `/` 和 `+` 的标准 Base64 JWT secret 覆盖环境变量覆盖场景。
  - 将无效 Fernet 测试改为短 secret 测试。
  - 新增标准 Base64 JWT secret 可通过配置加载的测试。
- `docs/requirement/20260629-jwt-secret-validation.md`
  - 记录轻量模式需求、验收标准、决策和风险。

## Expected vs actual changed files

| 文件 | 预期 | 实际 | 结论 |
| --- | --- | --- | --- |
| `internal/config/config.go` | 修改 JWT secret 校验逻辑 | 已修改 | 符合 |
| `internal/config/config_test.go` | 更新配置测试 | 已修改 | 符合 |
| `docs/requirement/20260629-jwt-secret-validation.md` | 轻量需求文档 | 已新增 | 符合 |
| 前端文件 | 不修改 | 未修改 | 符合 |
| 迁移文件 | 不修改 | 未修改 | 符合 |

## Acceptance checklist

- [x] `internal/config` 中 JWT secret 校验不再尝试 base64/Fernet decode。
- [x] `jwt.secret_key` 为空时报 required 错误。
- [x] `len(strings.TrimSpace(jwt.secret_key)) < 32` 时报长度不足错误。
- [x] 标准 Base64 格式、长度大于等于 32 的 secret 可通过配置加载。
- [x] 相关 Go 测试更新并通过。

## Command results

```powershell
go fmt ./cmd/... ./internal/...
```

结果：通过，无输出。

```powershell
go vet ./cmd/... ./internal/...
```

结果：通过，无输出。

```powershell
go test ./cmd/... ./internal/...
```

结果：通过。关键包结果包括：

- `backend/internal/config`：ok
- `backend/internal/transport/http`：ok / cached
- 其余 `./cmd/... ./internal/...` 包均通过或无测试文件。

## Missed or expanded scope

- 未发现遗漏的验收项。
- 未扩大到 JWT 签名/验签算法修改。
- 未修改前端代码。
- 未修改迁移文件。

## Risks

- 当前规则只检查 secret 字符串长度，不验证随机性或熵；这是本次需求明确指定的范围。
- `len(secretKey)` 使用 Go 字符串字节长度；对本项目预期的 ASCII/Base64 secret 适用。

## Incomplete items

无。

## Conclusion

验证通过。实现与轻量需求一致，相关 Go 格式化、静态检查和测试均通过。
