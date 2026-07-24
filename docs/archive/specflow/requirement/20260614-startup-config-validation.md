# 启动阶段配置校验

Review status: Accepted

## Goal

- 后端启动阶段即校验关键配置是否就绪，尤其是 `jwt.secret_key` 是否存在且符合 Fernet key 格式。
- 当开发者从 `.env.example` 复制 `.env` 但未替换占位密钥时，服务应 fail fast，而不是等到登录流程才报错。

## Non-goal

- 不自动生成或覆盖 `jwt.secret_key`，密钥由开发者自行生成并写入 `.env`。
- 不引入兼容旧密钥格式或默认密钥。

## Acceptance

- 启动生命周期开始时会校验 `POMELO_ORBIT_JWT__SECRET_KEY`。
- 缺失或格式非法时，后端启动失败，并复用现有清晰错误信息。
- 合法 Fernet key 不影响正常启动和现有安全服务使用。

## Risk

- 已存在非法本地 `.env` 的开发环境会在启动时更早失败，需要先按 `.env.example` 中命令生成并替换密钥。
