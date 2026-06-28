# 配置加载最佳实践

本文记录当前 Go 后端的配置文件布局和加载规则。旧 Python 后端仍使用 Dynaconf，但新改动优先按本规则更新 `backend-go`。

## 优先级链

未指定环境时：

```text
backend-go/configs/config.yaml < backend-go/.env < OS env
```

指定环境时，环境名只通过 OS env 提供：

```bash
POMELO_ORBIT_APP__ENV=develop
```

加载顺序为：

```text
backend-go/configs/config.yaml < backend-go/configs/config.<env>.yaml < backend-go/.env.<env> < OS env
```

`backend-go/configs/config.example.yaml` 和 `backend-go/.env.example` 只用于给人看字段、注释和复制模板，不参与运行时加载。

## 配置文件职责

| 文件/来源 | 用途 |
|------|------|
| `backend-go/configs/config.yaml` | 共享基础配置，提交到版本库 |
| `backend-go/configs/config.<env>.yaml` | 指定环境的结构化覆盖，不提交本地私有文件 |
| `backend-go/.env` | 未指定环境时的本地敏感覆盖，不提交 |
| `backend-go/.env.<env>` | 指定环境时的敏感覆盖，不提交 |
| OS env | Docker/systemd/Kubernetes/CI 注入，优先级最高 |

不要依赖 `.env` 里的 `POMELO_ORBIT_APP__ENV` 决定环境；程序必须先读取 OS env，才能知道加载 `.env` 还是 `.env.<env>`。

## 环境变量命名规则

前缀 `POMELO_ORBIT_`，嵌套 key 用双下划线 `__` 分隔：

```bash
# 对应 jwt.secret_key
POMELO_ORBIT_JWT__SECRET_KEY=xxx

# 对应 app.debug
POMELO_ORBIT_APP__DEBUG=true

# 对应 database.sqlite.path
POMELO_ORBIT_DATABASE__SQLITE__PATH=/data/db/app.db
```

## 新增配置项的步骤

1. 在 `backend-go/internal/config/config.go` 的 `Config` 结构体中添加字段。
2. 在 `bindEnv()` 的白名单中加入完整 key，允许 OS env 覆盖。
3. 在 `backend-go/configs/config.yaml` 写入非敏感默认值；敏感项只保留空字符串占位或通过 env 提供。
4. 如果需要给人看示例，同步更新 `backend-go/configs/config.example.yaml` 和 `backend-go/.env.example`。
5. 在 `backend-go/internal/config/config_test.go` 补充 YAML 加载、env 文件覆盖、OS env 优先和非法值校验测试。

## 测试中的注意事项

配置测试直接在临时目录写入 `configs/config.yaml`、`configs/config.<env>.yaml`、`.env` 或 `.env.<env>`，再调用 `config.Load()`。

需要测试指定环境时，用 `t.Setenv("POMELO_ORBIT_APP__ENV", "develop")` 显式设置环境名。

## 修改后的验证步骤

```bash
go -C backend-go test ./internal/config
go -C backend-go test ./internal/app
git diff --check
```
