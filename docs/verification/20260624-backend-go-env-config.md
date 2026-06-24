# backend-go 环境配置统一验证
最后修改时间: 2026-06-24 17:20:32

Review status: Draft

## Requirement alignment

对照 `docs/requirement/20260624-backend-go-env-config.md` 核对：

- backend-go 启动配置已按 `config.defaults.yaml < config.local.yaml < .env < OS env` 实现。
- `.env` 使用 `github.com/joho/godotenv` 的 `Load` 语义，不覆盖已有 OS env。
- `.env` 路径绑定当前工作目录：开发从 `backend-go` 目录启动时为 `backend-go/.env`；镜像 `WORKDIR /app` 时为 `/app/.env`。
- `.env` 缺失时允许继续加载默认配置、本地 YAML 和 OS env。
- Settings 服务改为使用 `Config.EnvFilePath` 读写启动配置入口一致的 `.env`。
- Settings 服务未实现全局热更新，配置描述中说明需要重启 backend-go 后生效。
- `backend-go/.env.example` 已新增，覆盖 JWT、数据库、Traefik、Turnstile、证书和 worker 示例。
- `scripts/manage.py` 已改为传递应用级 `POMELO_ORBIT_...` 变量，并排除脚本变量 `POMELO_ORBIT_IMAGE`。
- `scripts/.env.example` 已补充远端部署应用配置示例和 Turnstile 生产配置提示。

结论：核心需求已覆盖。

## Spec alignment

对照 `docs/spec/20260624-backend-go-env-config.md` 核对：

- `.env` 路径规则：实现为当前工作目录下 `.env`，未额外查找仓库根目录，符合规格。
- 配置优先级：先加载 `.env` 到进程环境，再由 Viper 读取 defaults/local 并绑定 env，符合规格。
- dotenv 实现：使用 `github.com/joho/godotenv`，未使用 `Overload`，符合规格。
- `Config.EnvFilePath`：已添加运行时字段并在 `Load()` 返回前设置，符合规格。
- Settings 热更新策略：保持写文件和重启后生效语义，符合规格。
- 部署脚本 `.env` 传输：筛选 `POMELO_ORBIT_` 前缀并排除 `POMELO_ORBIT_IMAGE`，符合规格。
- API contract：Settings API 未改变响应结构，只通过 description 表达重启生效语义，符合规格。

结论：实现与规格一致。

## Plan alignment

对照 `docs/plan/20260624-backend-go-env-config.md` 核对：

- 已引入 dotenv 依赖并更新 `backend-go/go.mod` / `backend-go/go.sum`。
- 已调整 `backend-go/internal/config/config.go`：加载 `.env`、记录 `EnvFilePath`、保留 Viper env binding。
- 已扩展 `backend-go/internal/config/config_test.go`：覆盖 `.env` 缺失、`.env` 覆盖 YAML、OS env 优先、`.env` 覆盖 `config.local.yaml`、Turnstile key 覆盖、无效 `.env` 报错等场景。
- 已调整 Settings 服务和路由测试，验证写入 `EnvFilePath` 指向的 `.env`。
- 已新增 `backend-go/.env.example`。
- 已更新 `scripts/manage.py` 和 `scripts/.env.example`。
- 计划中的可选 README/部署文档更新未执行；当前已有 `.env.example` 与过程文档说明配置优先级和路径，未额外扩展文档以避免超范围。

结论：必改项已完成，可选文档更新未执行且可接受。

## Actual diff summary

当前 diff 涉及：

- `backend-go/.env.example`：新增 backend-go `.env` 模板。
- `backend-go/go.mod`、`backend-go/go.sum`：新增 `github.com/joho/godotenv` 依赖。
- `backend-go/internal/config/config.go`：加载当前工作目录 `.env`，记录 `EnvFilePath`，保持配置校验。
- `backend-go/internal/config/config_test.go`：新增 dotenv 和优先级测试。
- `backend-go/internal/service/settings/service.go`：Settings `.env` 路径改为 `Config.EnvFilePath`，配置描述标注重启生效。
- `backend-go/internal/transport/http/settings_routes_test.go`：验证 Settings 写入 `EnvFilePath` 指向文件。
- `scripts/manage.py`：部署 `.env` 生成改为传递应用级 `POMELO_ORBIT_...`，排除 `POMELO_ORBIT_IMAGE`。
- `scripts/.env.example`：补充应用配置示例和 Turnstile 生产配置。
- `docs/requirement/20260624-backend-go-env-config.md`、`docs/spec/20260624-backend-go-env-config.md`、`docs/plan/20260624-backend-go-env-config.md`：过程文档。
- `docs/verification/20260624-backend-go-env-config.md`：本验证文档。

另外，工作区存在 `frontend/src/views/Settings.vue` 的未暂存改动：注释掉 Settings 页面 description 单元格。该文件不在本任务计划范围内，且本次未按前端检查流程验证它。

## Expected vs actual changed files

| 文件 | 预期 | 实际 | 说明 |
| --- | --- | --- | --- |
| `backend-go/internal/config/config.go` | 是 | 是 | 实现 `.env` 加载和 `EnvFilePath` |
| `backend-go/internal/config/config_test.go` | 是 | 是 | 增加配置优先级和 dotenv 测试 |
| `backend-go/internal/service/settings/service.go` | 是 | 是 | Settings 统一 `.env` 路径和重启提示 |
| `backend-go/internal/transport/http/settings_routes_test.go` | 是 | 是 | 验证 Settings 写入目标路径 |
| `backend-go/go.mod` | 是 | 是 | 新增 dotenv 依赖 |
| `backend-go/go.sum` | 是 | 是 | 新增依赖校验 |
| `backend-go/.env.example` | 是 | 是 | 新增模板 |
| `scripts/manage.py` | 是 | 是 | 部署 `.env` 筛选逻辑 |
| `scripts/.env.example` | 是 | 是 | 补充应用配置示例 |
| `docs/requirement/...` | 是 | 是 | 过程文档 |
| `docs/spec/...` | 是 | 是 | 过程文档 |
| `docs/plan/...` | 是 | 是 | 过程文档 |
| `docs/verification/...` | 是 | 是 | 本验证文档 |
| `frontend/src/views/Settings.vue` | 否 | 是 | 范围外未暂存改动，未纳入验证结论 |

## Acceptance criteria checklist

- [x] backend-go 启动配置加载顺序明确为 `config.defaults.yaml < config.local.yaml < .env < OS env`。
- [x] `.env` 加载使用不覆盖已有环境变量的语义。
- [x] `.env` 缺失时配置加载可继续。
- [x] 开发阶段 `.env` 位置为当前 `backend-go` 工作目录下 `.env`。
- [x] 镜像阶段 `.env` 位置为 `<WORKDIR>/.env`，当前 Dockerfile 工作目录对应 `/app/.env`。
- [x] Turnstile 相关 `POMELO_ORBIT_TURNSTILE__...` 可通过 `.env` 覆盖配置。
- [x] OS env 与 `.env` 同名变量同时存在时，OS env 优先。
- [x] `config.local.yaml` 与 `.env` 同时配置同一项时，`.env` 优先。
- [x] 新增 `backend-go/.env.example`，并说明 Turnstile test keys 不适用于生产防护。
- [x] Settings 服务读写的 `.env` 路径与启动时加载的 `.env` 路径保持一致。
- [x] Settings 服务未实现热更新，并通过 description 明确需要重启 backend-go 生效。
- [x] 部署脚本不再遗漏新增应用级 `POMELO_ORBIT_...` 配置；Turnstile 变量可进入远端 `.env`。
- [x] 增加配置加载测试覆盖 `.env`、OS env 优先级、`.env` 缺失、Turnstile 覆盖和 local config 优先级等场景。

## Test results

功能级验证已运行并通过：

```powershell
go -C backend-go test ./internal/config -run "TestLoadConfig(EnvFileOverridesConfig|EnvFileOverridesLocalConfig|OSEnvOverridesEnvFile|RejectsInvalidEnvFile)$" -v
```

验证结论：

- `.env` 能覆盖默认 YAML 配置。
- `.env` 能覆盖 `config.local.yaml`。
- OS env 能覆盖 `.env`，确认 `.env` 不覆盖真实系统环境变量。
- `.env` 格式错误会使配置加载失败，而不是静默忽略。

功能级验证已运行并通过：

```powershell
go -C backend-go test ./internal/transport/http -run "TestSettingsConfigRoutes$" -v
```

验证结论：

- `GET /api/settings/config` 能读取配置项。
- `PUT /api/settings/config` 写入的是 `Config.EnvFilePath` 指向的 `.env`。
- `DELETE /api/settings/config` 能从同一个 `.env` 删除覆盖项。

功能级验证已运行并通过：

```powershell
# 临时 smoke test：复制 config.defaults.yaml 到临时工作目录，写入 .env，调用 config.Load()
go -C backend-go test ./internal/config -run TestEnvSmokeExternal -v
```

验证结论：

- `config.Load()` 实际从当前工作目录 `.env` 读取 Turnstile 配置。
- `Config.EnvFilePath` 指向该工作目录 `.env`。
- 同名 OS env 的 server port 优先于 `.env`。

功能级验证已运行并通过：

```powershell
# monkey patch copy_to_remote 后直接调用 Deployer._deploy_env()
python -
```

验证结论：

- 远端目标路径为 `/opt/pomelo-orbit/.env`。
- 写入内容只包含应用级 `POMELO_ORBIT_...`。
- `POMELO_ORBIT_IMAGE` 被排除。
- `SSH_HOST`、`REMOTE_DEPLOY_DIR` 等脚本变量不会写入远端应用 `.env`。
- 输出 key 按字典序稳定排序。

完整回归也已运行并通过：

```powershell
go -C backend-go fmt ./... && go -C backend-go test ./internal/config ./internal/service/settings ./internal/transport/http && go -C backend-go test ./...
python -m py_compile scripts/manage.py
git diff --check HEAD
```

## Missed or expanded scope

- 未执行前端 `yarn lint --fix && yarn typecheck`，因为本任务实现范围不应包含前端；但当前工作区确实存在 `frontend/src/views/Settings.vue` 的范围外改动，建议单独处理或单独验证。
- 未启动应用做手工 UI 验证；本次验证基于配置加载单元测试、路由测试和脚本语法检查。
- 未更新 README 或额外部署文档；计划中该项为可选，当前 `.env.example` 和过程文档已覆盖必要说明。

## Risks

- Settings 服务写 `.env` 后仍需要重启 backend-go 才能让已初始化组件采用新配置；这符合规格，但产品使用时需要用户理解该语义。
- `scripts/manage.py` 现在会传递所有应用级 `POMELO_ORBIT_...` 变量，拼错的应用变量也会进入远端 `.env`；这比静默丢弃更可诊断，但部署前仍需审查 `scripts/.env`。
- 当前工作区存在范围外 `frontend/src/views/Settings.vue` 改动，建议不要与本次后端配置变更混在同一个提交中，除非用户确认它属于同一交付边界。
- `scripts/manage.py` 在 Git diff 统计中显示较大，可能受换行符差异影响；内容级变更集中在 env vars 读取保存和 `_deploy_env()` 生成逻辑。

## Incomplete items

- 无后端配置功能未完成项。
- 范围外前端改动未验证。

## Conclusion

Verification 通过。backend-go `.env` 配置加载、Settings `.env` 路径统一、部署脚本应用配置传递和相关测试均符合已接受的 Requirement / Spec / Plan。

交付前建议先决定如何处理范围外的 `frontend/src/views/Settings.vue` 改动：单独提交、撤回，或明确纳入另一个变更。