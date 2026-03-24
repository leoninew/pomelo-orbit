# 实施任务进度

## 已完成任务

### ✅ Task #2: 移动 migrations 目录位置
- 将 migrations 从 `backend/data/migrations/` 移动到 `backend/migrations/`
- 避免卷挂载 `/app/data` 时覆盖 migrations 目录

### ✅ Task #3: 更新 Dockerfile migrations 路径
- 修改 Dockerfile 第 61 行：`COPY backend/migrations/ ./migrations/`
- 修改 Dockerfile 第 70 行：移除 `/app/data/applications` 目录创建

### ✅ Task #4: 更新代码中 migrations 路径引用
- 修改 `backend/src/pomelo_orbit/main.py`
- migrations_dir 从相对路径改为绝对路径 `Path("/app/migrations")`

### ✅ Task #5: 更新 config.defaults.yaml 目录配置
- 添加 `data.root` 和 `data.apps` 配置
- 更新 `data.traefik.dynamic_route_dir` 从 `data/applications/traefik/data/dynamic` 到 `data/traefik/dynamic`
- 更新 `data.traefik.cert_dir` 从 `data/applications/traefik/data/certs` 到 `data/traefik/certs`

### ✅ Task #6: 实现模板变量替换逻辑
- 在 ApplicationManager 添加 `application_name` 参数
- 实现 `_get_pomelo_orbit_data_root()` 方法（容器模式从环境变量读取，本地模式从路径推导）
- 实现 `_replace_template_variables()` 方法（替换 `{{POMELO_ORBIT_DATA__ROOT}}`）
- 在 `deploy()` 方法中对 .env 文件进行模板替换
- 更新 `application_service.py` 传递 `application_name` 参数

### ✅ Task #7: 更新 TraefikManager 按需创建目录
- 修改 `__init__()` 移除初始化时创建目录
- 修改 `_write_cert_files()` 写入前按需创建 cert_dir
- 修改 `deploy_route()` 写入前按需创建 config_dir

### ✅ Task #8: 更新数据库迁移脚本配置
- 更新 `backend/migrations/v0.3.1__init_script.sql` 中的 init.sh 注释
- 明确只创建应用本地数据目录

### ✅ Task #9: 运行 lint 和 test 验证
- 执行 `make lint` - 通过（ruff + mypy）
- 执行 `make test` - 通过（180个单元测试）

### ✅ Task #10: 修复 ApplicationManager 缺少 self.application_dir
- 在 `__init__()` 中添加 `self.application_dir = application_dir`
- 修复 19 处使用 `self.application_dir` 的地方

### ✅ Task #11: 修复 ApplicationService 硬编码路径
- `_get_application_dir()` 从配置读取 `settings.data.apps` 而不是硬编码 `data/applications`
- 添加类型注解避免 mypy 报错

### ✅ Task #12: 修复 Traefik 配置路径键名不匹配
- 修复 `interfaces/api/route.py` 中的配置读取
- 从 `settings.traefik.*` 改为 `settings.data.traefik.*`
- 简化路径处理逻辑

### ✅ Task #13: 更新配置文件路径
- `.dockerignore`: `backend/data/applications` → `backend/data/apps`
- `backend/.gitignore`: `data/applications/` → `data/apps/`
- `backend/.env.example`: 移除旧的 Traefik 配置，添加 `POMELO_ORBIT_DATA__ROOT`

### ✅ Task #14: 修复前端显示路径
- `frontend/src/views/ApplicationDetail.vue`: 删除提示中的路径从 `data/applications/` 改为 `data/apps/`

### ✅ Task #15: 修复 Traefik 证书路径
- 修复 `domain/route_service.py` 中的证书路径
- 从 `/certs/` 改为 `/etc/traefik/certs/`
- 与容器卷挂载路径一致

### ✅ Task #16: 修复时间函数混用问题
- 修复 `domain/entities.py` 中使用 `datetime.now()` 的问题
- 导入 `utc_now` 并替换所有 `datetime.now()` 调用（6处）
- 避免时区不一致导致负数耗时

## 验证结果

- ✅ `make lint` 通过（ruff + mypy）
- ✅ `make test` 通过（180个单元测试）
- ✅ 配置路径统一从 `config.defaults.yaml` 读取
- ✅ Dockerfile 路径与代码一致
- ✅ Traefik 证书路径正确（`/etc/traefik/certs/`）
- ✅ 时间统一使用 UTC（`utc_now()`）
- ✅ 所有配置文件路径更新

## 关键问题修复

### 1. Traefik 证书路径不匹配
**错误日志**: `ERR Unable to parse certificate /certs/pomelo-orbit.pem`

**原因**: 证书挂载到 `/etc/traefik/certs/` 但配置引用 `/certs/`

**修复**: 修改 `domain/route_service.py` 生成正确的证书路径

### 2. 部署记录负数耗时
**表现**: 耗时显示 `-3595381ms`

**原因**: `started_at` 使用 `datetime.now()` (本地时间)，`finished_at` 使用 `utc_now()` (UTC)，时区不一致

**修复**: 统一使用 `utc_now()`，遵循 CLAUDE.md 中的时间处理规范

