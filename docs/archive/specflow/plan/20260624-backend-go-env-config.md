# backend-go 环境配置统一计划
最后修改时间: 2026-06-24 16:38:23

Review status: Accepted

## Requirement / Spec basis

基于已接受的文档：

- `docs/requirement/20260624-backend-go-env-config.md`
- `docs/spec/20260624-backend-go-env-config.md`

目标实施范围：

- 统一 backend-go 配置优先级：`config.defaults.yaml < config.local.yaml < .env < OS env`。
- 开发阶段使用 `backend-go/.env`；镜像阶段使用 `<WORKDIR>/.env`。
- Settings 服务读写启动时使用的同一个 `.env` 文件。
- 不做全局运行时热更新，明确 Settings 修改后重启生效语义。
- 部署脚本传递所有应用级 `POMELO_ORBIT_...` 配置，排除脚本元变量。
- 补齐 `.env.example` 模板与测试。

## Implementation steps

### 1. 引入 dotenv 依赖

- 在 `backend-go` 模块中添加 `github.com/joho/godotenv`。
- 使用常规 Go module 流程更新：
  - `backend-go/go.mod`
  - `backend-go/go.sum`

注意：只使用 `godotenv.Load` 语义，不使用 `Overload`。

### 2. 调整 `config.Load()` 配置加载

修改 `backend-go/internal/config/config.go`：

1. 新增 `Config.EnvFilePath string` 运行时字段，标记为不参与 mapstructure/yaml。
2. 新增 helper，例如：
   - `envFilePath() (string, error)`：返回当前工作目录下 `.env` 的绝对路径。
   - `loadEnvFile(path string) error`：如果 `.env` 不存在则跳过；存在则调用 `godotenv.Load(path)`。
3. 在创建 Viper loader 前加载 `.env`：
   - 先解析当前工作目录 `.env` 绝对路径。
   - 调用不覆盖 OS env 的加载逻辑。
   - `.env` 缺失不报错。
   - `.env` 格式错误返回 `read env file: ...` 类型错误。
4. 保持当前流程：
   - 读取 `config.defaults.yaml`
   - 合并 `config.local.yaml`
   - `Unmarshal`
   - `Validate`
5. 在返回前设置 `cfg.EnvFilePath = resolvedPath`。

### 3. 扩展配置加载测试

修改 `backend-go/internal/config/config_test.go`：

- 增加 `.env` 缺失场景：不影响默认配置加载，`EnvFilePath` 指向当前工作目录 `.env`。
- 增加 `.env` 覆盖 YAML 场景：例如 `.env` 设置 `POMELO_ORBIT_SERVER__PORT`、`POMELO_ORBIT_TURNSTILE__SITE_KEY`。
- 增加 OS env 优先于 `.env` 场景。
- 增加 `config.local.yaml < .env` 场景。
- 增加 `.env` Turnstile 生产 key 覆盖默认 test key 场景。
- 增加 `.env` 格式错误返回错误场景。

测试约束：

- 每个测试使用 `t.TempDir()` 和 `os.Chdir` 隔离工作目录。
- 对会进入进程环境的变量，使用唯一值并通过 `t.Setenv` / 显式 unset 控制，避免污染其他测试。
- 不依赖仓库真实 `.env`。

### 4. 统一 Settings 服务 `.env` 路径

修改 `backend-go/internal/service/settings/service.go`：

1. `envPath()` 改为返回 `s.cfg.EnvFilePath`。
2. 如果 `EnvFilePath` 为空，返回当前工作目录 `.env` 或报明确错误；优先在 config.Load 中保证非空。
3. 配置项 description 中加入重启后生效语义，例如：
   - “Application log level (restart required)”
   - 或统一在 SystemConfig 增加说明字段。为避免 API contract 变化，优先更新 description。
4. 保持 `Config` / `Update` / `Reset` API 不变。
5. 写文件行为继续保持排序稳定。

### 5. 调整 Settings 测试

修改 `backend-go/internal/transport/http/settings_routes_test.go`：

- 测试 server config 中显式设置 `server.appCfg.EnvFilePath = filepath.Join(t.TempDir(), ".env")`。
- 验证 `PUT /api/settings/config` 写入的是 `EnvFilePath` 指向的文件，而不是 `OrbitRoot()/.env`。
- 保留 `GET` / `PUT` / `DELETE` 行为测试。

必要时补充 settings service 单元测试，直接验证 `Config` / `Update` / `Reset` 对 `EnvFilePath` 的读写路径。

### 6. 新增 `backend-go/.env.example`

创建 `backend-go/.env.example`，内容包括：

- JWT secret 必填说明。
- Server host / port 示例。
- SQLite / MySQL 配置示例。
- Traefik 配置示例。
- Turnstile 配置示例：
  - `POMELO_ORBIT_TURNSTILE__ENABLED=true`
  - `POMELO_ORBIT_TURNSTILE__SITE_KEY=...`
  - `POMELO_ORBIT_TURNSTILE__SECRET_KEY=...`
  - `POMELO_ORBIT_TURNSTILE__VERIFY_URL=https://challenges.cloudflare.com/turnstile/v0/siteverify`
- Let's Encrypt 配置示例。
- Worker 配置示例。

必须明确：

- 复制为 `backend-go/.env` 用于开发。
- 镜像部署时放到 `<WORKDIR>/.env`，当前为 `/app/.env`。
- 默认 Turnstile test keys 不提供生产防护，生产必须覆盖真实 key。

### 7. 更新部署脚本

修改 `scripts/manage.py`：

1. `Config.load_env()` 保留读取 `scripts/.env` 的能力。
2. 保存完整 `env_vars`，例如 `self.env_vars = env_vars`。
3. `_deploy_env()` 改为：
   - 校验 `POMELO_ORBIT_JWT__SECRET_KEY` 存在。
   - 从 `self.config.env_vars` 中筛选 key：
     - 以 `POMELO_ORBIT_` 开头。
     - 排除 `POMELO_ORBIT_IMAGE`。
   - 按 key 排序写入远端 `.env`。
   - 不写 SSH、REMOTE_DEPLOY_DIR、GHCR_TOKEN、SOURCE_REPO 等脚本变量。
4. 保持远端目标路径为 `${REMOTE_DEPLOY_DIR}/.env`。

### 8. 更新 `scripts/.env.example`

修改 `scripts/.env.example`：

- 保留 SSH、REMOTE_PORT、REMOTE_DEPLOY_DIR、SOURCE_REPO、GHCR_TOKEN 等脚本配置。
- 增加应用配置区块，说明这些 `POMELO_ORBIT_...` 会被写入远端 `.env`。
- 增加 Turnstile、数据库、证书示例。
- 标注 `POMELO_ORBIT_IMAGE` 是本地部署脚本变量，不写入远端应用 `.env`。

### 9. 可选文档更新

如 README 或部署文档已有配置章节，则补充：

- backend-go 配置优先级。
- 开发 `.env` 路径：`backend-go/.env`。
- 镜像 `.env` 路径：`<WORKDIR>/.env`。
- Settings 修改配置后需要重启生效。

如果没有集中配置章节，可暂不新增大段文档，避免超出范围。

## Files to change

必改：

- `backend-go/internal/config/config.go`
- `backend-go/internal/config/config_test.go`
- `backend-go/internal/service/settings/service.go`
- `backend-go/internal/transport/http/settings_routes_test.go`
- `backend-go/go.mod`
- `backend-go/go.sum`
- `backend-go/.env.example`
- `scripts/manage.py`
- `scripts/.env.example`

过程文档：

- `docs/plan/20260624-backend-go-env-config.md`

可能改：

- README 或部署文档，如现有配置章节适合补充。

## Verification plan

后端：

```bash
cd backend-go && go fmt ./...
cd backend-go && go test ./internal/config ./internal/service/settings ./internal/transport/http
cd backend-go && go test ./...
```

部署脚本静态校验：

```bash
python -m py_compile scripts/manage.py
```

如变更文档/脚本不触及前端，不需要运行前端 `yarn lint --fix && yarn typecheck`。

## Blockers

暂无。

## Assumptions

- 开发者按项目约定从 `backend-go` 目录启动 backend-go。
- 镜像运行目录保持 Dockerfile 中的 `WORKDIR /app`。
- Settings 服务本次不承担全局热更新职责，只负责文件编辑和返回 `.env` 覆盖状态。
- `POMELO_ORBIT_IMAGE` 继续作为部署脚本本地变量，而不是 backend-go 应用配置。

## Risks

- `godotenv.Load` 修改进程环境，测试必须隔离，否则可能影响同进程其他测试。
- Settings 不热更新会带来“写入成功但需重启”的用户认知风险，需要在 description 或文档中说明。
- `_deploy_env()` 传递所有 `POMELO_ORBIT_...` 变量会让拼错的应用变量也进入远端 `.env`，但这比静默丢弃更可诊断。
- 当前工作区已有其他未提交改动；实施时必须只修改本任务相关文件，避免混入之前 Docker 迁移 diff。

## Rollback

如实现后发现配置加载有问题，可回滚：

- 移除 dotenv 依赖。
- 移除 `Config.EnvFilePath` 与 `.env` 加载逻辑。
- Settings 服务恢复 `OrbitRoot()/.env`。
- `scripts/manage.py` 恢复原白名单写入逻辑。
- 删除新增 `backend-go/.env.example`。

## User review notes

- 用户要求进入 Plan / 计划阶段。
