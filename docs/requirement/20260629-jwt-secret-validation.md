# JWT Secret 校验修复需求
最后修改时间: 2026-06-29 11:27:17

Review status: Accepted

## Background

当前启动配置校验将 `jwt.secret_key` 按 Fernet key 校验，要求 URL-safe Base64 解码后正好 32 bytes。实际 JWT 签名使用 HS256 HMAC，签名逻辑直接使用 secret 字符串字节，不要求 Fernet 格式。

用户使用 `openssl rand -base64 32` 生成标准 Base64 secret 时，可能包含 `+`、`/` 等非 URL-safe 字符，当前校验会报错：`jwt.secret_key must be a valid Fernet key: illegal base64 data ...`。

## Goal

- 修复 `jwt.secret_key` 配置校验，使其符合 JWT HMAC secret 的实际使用方式。
- 删除 Fernet/base64 decode 校验逻辑。
- 要求 `len(secretKey) >= 32`，避免明显弱密钥。
- 更新配置相关测试用例，覆盖标准 Base64 secret 可用和短 secret 被拒绝。

## Non-goal

- 不修改 JWT 签名/验签算法。
- 不引入 Fernet key 与 JWT secret 的兼容层或双逻辑。
- 不修改已执行迁移文件。
- 不调整前端代码。

## User scenarios

- 运维或开发者使用 `openssl rand -base64 32` 生成 `jwt.secret_key` 后，服务配置加载应通过。
- 当 `jwt.secret_key` 为空或长度不足 32 字符时，配置加载应失败并给出明确错误。

## Acceptance

- `internal/config` 中 JWT secret 校验不再尝试 base64/Fernet decode。
- `jwt.secret_key` 为空时报 required 错误。
- `len(strings.TrimSpace(jwt.secret_key)) < 32` 时报长度不足错误。
- 标准 Base64 格式、长度大于等于 32 的 secret 可通过配置加载。
- 相关 Go 测试更新并通过。

## Open questions

暂无需要用户确认的未决事项；用户已明确指定校验规则为 `len(secretKey) >= 32`。

## Decisions

- 采用轻量模式 / light 处理。
- Requirement 直接标记为 `Accepted`，因为用户明确要求“开始修复、直接实现”。
- JWT secret 校验只按修剪后的字符串长度判断，不解析或规范化 Base64 内容。

## Risk

- `len(secretKey)` 是按字节长度统计；当前配置 secret 预期为 ASCII/Base64/随机字符串，符合本次需求。
- 只校验长度不能完整衡量熵，但能阻止明显过短 secret；更复杂熵校验不在本次范围内。
