# backend-go 运行时配置入口收敛验证
最后修改时间: 2026-06-22 11:09:31

Review status: Draft

## Requirement alignment

依据 `docs/requirement/20260622-backend-go-runtime-config.md` 核对，实际实现与需求一致：

- Go 后端本地默认启动行为已收敛到 `backend-go/config.defaults.yaml`。
- `justfile` 和 `scripts/dev.py` 已移除重复注入的 `POMELO_ORBIT_SERVER__HOST` / `POMELO_ORBIT_SERVER__PORT`。
- Viper 的 `POMELO_ORBIT_*` 覆盖机制保留，并修正 `orbit.root` 对应的环境变量绑定。
- 未新增 `--host` / `--port`，未在启动层、HTTP server 层或 worker 层新增 `os.Getenv` / `os.LookupEnv` 旁路。
- 未修改旧 Python backend、`scripts/manage.py` 或 Go 进程 `.env` 加载优先级。

## Spec alignment

不适用。当前采用 SpecFlow 轻量模式 / light，没有单独创建 spec 文档。

## Plan alignment

按轻量模式直接依据 Requirement 实现；实际改动与 Requirement 中的实现边界一致。

## Actual diff summary

- `backend-go/config.defaults.yaml`
  - `server.host` 改为 `127.0.0.1`。
  - `server.port` 改为 `9001`。
- `justfile`
  - `dev-backend` 不再通过命令前缀设置 server host/port 环境变量。
- `scripts/dev.py`
  - 删除 `backend_env` 对 server host/port 的硬编码注入。
  - 后端 API 和 worker 启动不再传入这份重复 env。
- `backend-go/internal/config/config.go`
  - `bindEnv` 中的错误 key `repository.root` 修正为 `orbit.root`。
- `backend-go/internal/config/config_test.go`
  - 默认配置测试断言 `127.0.0.1:9001`。
  - env override 测试覆盖 `POMELO_ORBIT_ORBIT__ROOT`。
- `docs/requirement/20260622-backend-go-runtime-config.md`
  - Requirement 已标记为 `Accepted`，并记录需求阶段检查结果。

## Expected vs actual changed files

符合预期：

- `backend-go/config.defaults.yaml`
- `backend-go/internal/config/config.go`
- `backend-go/internal/config/config_test.go`
- `justfile`
- `scripts/dev.py`
- `docs/requirement/20260622-backend-go-runtime-config.md`

额外新增：

- `docs/verification/20260622-backend-go-runtime-config.md`，用于记录本验证阶段结果。

未修改：

- 旧 Python backend。
- `scripts/manage.py`。
- Go HTTP server、app、worker 启动路径。
- 前端代码。

## Acceptance checklist

- [x] `backend-go/config.defaults.yaml` 明确提供本地开发默认监听地址，`server.host` 与前端代理目标一致。
- [x] `justfile` 不再设置 `POMELO_ORBIT_SERVER__HOST` / `POMELO_ORBIT_SERVER__PORT`。
- [x] `scripts/dev.py` 不再为后端 API / worker 硬编码注入 server host/port 环境变量。
- [x] Go 后端启动监听地址仍通过 `config.Load` -> `config.Config.Server` -> HTTP server 地址派生。
- [x] `backend-go/internal/config/config.go` 绑定 `orbit.root`，不再绑定不存在的 `repository.root`。
- [x] Viper env override 测试覆盖 `POMELO_ORBIT_SERVER__HOST`、`POMELO_ORBIT_SERVER__PORT` 和 `POMELO_ORBIT_ORBIT__ROOT`。
- [x] 静态搜索已区分运行时配置、测试和文档；未误改旧后端或部署管理脚本。
- [x] `cd backend-go && go test ./...` 通过。

## Commands and results

```bash
go -C "backend-go" fmt ./...
```

结果：通过，无输出。

```bash
go -C "backend-go" test ./...
```

结果：通过。

```bash
go -C "backend-go" vet ./...
```

结果：通过，无输出。

```bash
git diff --check
```

结果：通过，无输出。

```bash
python -m py_compile "scripts/dev.py"
```

结果：通过，无输出。

```text
搜索: POMELO_ORBIT_SERVER__HOST|POMELO_ORBIT_SERVER__PORT|POMELO_ORBIT_REPOSITORY__ROOT|repository\.root
```

结果：

- `POMELO_ORBIT_SERVER__HOST` / `POMELO_ORBIT_SERVER__PORT` 只剩配置测试和 Requirement 文档中的预期引用。
- `repository.root` 只剩 Requirement 文档中的问题说明。
- 未在 `justfile`、`scripts/dev.py` 或 Go 配置代码中发现旧旁路。

```text
搜索: os\.Getenv|os\.LookupEnv|POMELO_ORBIT_[A-Z0-9_]*，范围 backend-go/**/*.go
```

结果：

- `POMELO_ORBIT_*` 运行时绑定仍集中在 `backend-go/internal/config/config.go`。
- `backend-go/internal/service/settings/service.go` 保留 `POMELO_ORBIT_` 前缀常量，用于配置管理读写 `.env`。
- `backend-go/internal/config/config_test.go` 保留 `t.Setenv` 测试覆盖。
- `backend-go/internal/app/mysql_e2e_test.go` 的 `BACKEND_GO_E2E_CONFIG` 仍仅用于 e2e 测试配置文件选择。

## Missed or expanded scope

- 未运行 `just dev-backend` / `just dev`，因为项目约束要求开发服务器由用户管理。
- 未运行前端 `yarn lint --fix && yarn typecheck`，因为本次没有修改前端代码。
- 未将 Go 进程改为直接读取 `.env`，符合 Requirement 的 non-goal。
- 未重构 `scripts/manage.py`，符合 Requirement 的 non-goal。

## Risks

- 默认 `server.host` 从 `localhost` 改为 `127.0.0.1`，如果某个本地环境依赖 IPv6 localhost 解析，行为可能变化；当前前端代理明确使用 `127.0.0.1:9001`。
- 曾误用 `POMELO_ORBIT_REPOSITORY__ROOT` 的外部环境需要改为 `POMELO_ORBIT_ORBIT__ROOT`；本项目按 Requirement 不保留兼容别名。

## Incomplete items

- 无实现未完成项。
- 如需验证真实开发启动行为，需要用户手动运行 `just dev-backend` 或 `just dev`。

## Conclusion

验证通过。实现满足轻量 Requirement 的目标和验收标准，未发现范围外代码改动或测试失败。