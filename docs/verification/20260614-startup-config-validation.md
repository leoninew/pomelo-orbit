# 启动阶段配置校验验证

Review status: Draft

## What changed

- 新增 `validate_security_config(settings)`，复用原有 Fernet key 校验逻辑。
- `SecurityService` 初始化继续调用同一校验函数，保持既有使用路径不变。
- FastAPI lifespan 启动时先执行安全配置校验，再执行数据库迁移。
- `.env.example` 明确提示 `POMELO_ORBIT_JWT__SECRET_KEY` 启动时会校验，必须自行生成。
- 新增单元测试覆盖缺失密钥和非法 Fernet key 两种失败场景。

## Acceptance

- [x] 启动生命周期开始时会校验 `POMELO_ORBIT_JWT__SECRET_KEY`。
- [x] 缺失或格式非法时，复用现有清晰错误信息并阻止启动继续。
- [x] 不自动生成或覆盖密钥。
- [x] 合法密钥路径继续由现有 `SecurityService` 使用。

## Commands

```bash
cd backend && PYTHONUTF8=1 uv run ruff check --fix src/pomelo_orbit/main.py src/pomelo_orbit/infrastructure/security.py src/pomelo_orbit/infrastructure/__init__.py tests/unit/infrastructure/test_security.py && PYTHONUTF8=1 uv run mypy src/ tests/ && PYTHONUTF8=1 uv run pytest tests/unit/infrastructure/test_security.py
```

结果：通过。`test_security.py` 共 4 个测试通过。

```bash
cd backend && POMELO_ORBIT_JWT__SECRET_KEY=00000000000000000000000000000000000000000000 PYTHONUTF8=1 uv run python -c "from fastapi.testclient import TestClient; from pomelo_orbit.main import create_app; TestClient(create_app()).__enter__()"
```

结果：启动阶段抛出 `AssertionError: jwt.secret_key 不是有效的 Fernet 格式。请参考 .env.example 文件中的说明生成有效的密钥。`，符合预期。

## Remaining risk

- 本地 `backend/.env` 如果仍使用旧占位值或空值，后端会在启动阶段立即失败；需要先按 `.env.example` 中命令生成 Fernet key 并写入。
- 未运行完整后端测试套件；本次运行了相关单元测试、全量 mypy 和目标文件 ruff。
