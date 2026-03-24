## 迁移脚本移动

从 data 目录分离迁移脚本，避免挂载 /app/data 目录时迁移脚本不可达。

### 项目根目录结构

```bash
backend/
├── data/
│   ├── apps/          # 应用部署目录
│   ├── traefik/       # Traefik 数据目录
│   │   ├── dynamic/   # 动态路由配置
│   │   └── certs/     # 证书文件
│   └── db/            # 数据库
└── migrations/        # 迁移脚本
    ├── v0.3.0__schema.sql
    └── v0.3.1__init_script.sql
```

### 容器内目录结构

```bash
/app/
├── data/              # 挂载点（宿主机 data/ 目录）
│   ├── apps/
│   ├── traefik/
│   └── db/
├── migrations/        # 镜像中的迁移脚本（不挂载）
├── src/
├── static/
└── .venv/
```

### Dockerfile 修改

```dockerfile
# Copy database migrations (移到 /app/migrations/)
COPY backend/migrations/ ./migrations/
```

### 代码修改

```python
# 迁移脚本路径
migrations_dir = Path("/app/migrations")  # 不再从 /app/data 读取
```

## 目录挂载设计

### DooD 场景限制

Pomelo Orbit 以容器运行时，不能使用相对路径：

1. **部署 pomelo-orbit 自身**：
   - `./data:/app/data` 会被解析为 `/app/data/apps/pomelo-orbit/data:/app/data`
   - 宿主机上不存在 `/app/data/...` 路径
   - ✗ 挂载失败

2. **部署 traefik**：
   - `./data/traefik.yml` 会被解析为 `/app/data/apps/traefik/data/traefik.yml`
   - 宿主机上不存在 `/app/data/...` 路径
   - ✗ 挂载失败或创建目录

### 模板变量设计

**全局模板变量**：`{{POMELO_ORBIT_DATA__ROOT}}` 代表 pomelo-orbit 的数据根目录，所有应用的 .env 都可以引用。

**使用示例**：
```ini
SOME_PATH={{POMELO_ORBIT_DATA__ROOT}}/somewhere
APP_DATA={{POMELO_ORBIT_DATA__ROOT}}/apps/my-app
CERT_DIR={{POMELO_ORBIT_DATA__ROOT}}/traefik/certs
```

**替换规则**：部署时自动替换为宿主机绝对路径。

## Pomelo Orbit 配置

### config.defaults.yaml

```yaml
data:
  root: data
  apps: data/apps
  traefik:
    dynamic_route_dir: "data/traefik/dynamic"   # 相对于项目根目录
    cert_dir: "data/traefik/certs"              # 相对于项目根目录
    container_name: "traefik"
```

### .env.windows

```ini
# Pomelo Orbit 数据目录（宿主机绝对路径）
POMELO_ORBIT_DATA__ROOT={{POMELO_ORBIT_DATA__ROOT}}

# JWT 密钥（同时用于凭据加密，必须使用 Fernet 格式）
# 生成方法: python -c "from cryptography.fernet import Fernet; print(Fernet.generate_key().decode())"
POMELO_ORBIT_JWT__SECRET_KEY=your-fernet-key-here
```

### docker-compose.yml

```yaml
services:
  pomelo-orbit:
    image: pomelo-orbit:latest
    container_name: pomelo-orbit
    restart: unless-stopped
    ports:
      - "8000:80"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ${POMELO_ORBIT_DATA__ROOT}:/app/data
    environment:
      - POMELO_ORBIT_DATA__ROOT=${POMELO_ORBIT_DATA__ROOT}  # 传递给容器
      - POMELO_ORBIT_JWT__SECRET_KEY=${POMELO_ORBIT_JWT__SECRET_KEY}
```

## Traefik 配置

### 目录结构

```
data/apps/traefik/          # Traefik 应用目录
├── docker-compose.yml
├── .env.windows
└── data/
    └── traefik.yml         # 静态配置（部署时生成）

data/traefik/               # Traefik 数据目录
├── dynamic/                # 动态路由（Pomelo Orbit 生成）
│   └── routes.yml
└── certs/                  # 证书（Pomelo Orbit 生成）
    ├── example.com.crt
    └── example.com.key
```

### 静态配置（data/traefik.yml）

```yaml
providers:
  file:
    directory: /etc/traefik/dynamic
    watch: true
```

### .env.windows

```ini
# Traefik 数据目录（宿主机绝对路径）
POMELO_ORBIT_DATA__TRAEFIK={{POMELO_ORBIT_DATA__ROOT}}/traefik
POMELO_ORBIT_DATA__TRAEFIK_WORKDIR={{POMELO_ORBIT_DATA__ROOT}}/apps/traefik

# Traefik Environment Variables
TRAEFIK_LOG_LEVEL=INFO
TRAEFIK_DASHBOARD=true
TRAEFIK_DASHBOARD_INSECURE=true
```

### docker-compose.yml

```yaml
services:
  traefik:
    image: traefik:3
    container_name: traefik
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
      - "8080:8080"
    volumes:
      - ${POMELO_ORBIT_DATA__TRAEFIK_WORKDIR}/data/traefik.yml:/etc/traefik/traefik.yml:ro
      - ${POMELO_ORBIT_DATA__TRAEFIK}/dynamic:/etc/traefik/dynamic:ro
      - ${POMELO_ORBIT_DATA__TRAEFIK}/certs:/etc/traefik/certs:ro
    environment:
      - TRAEFIK_LOG_LEVEL=${TRAEFIK_LOG_LEVEL}
      - TRAEFIK_DASHBOARD=${TRAEFIK_DASHBOARD}
      - TRAEFIK_DASHBOARD_INSECURE=${TRAEFIK_DASHBOARD_INSECURE}
```

## 文件生成职责

| 文件/目录 | 生成者 | 生成时机 | 说明 |
|----------|--------|---------|------|
| `data/apps/traefik/docker-compose.yml` | 部署流程 | 部署 Traefik 时 | 从数据库读取 |
| `data/apps/traefik/.env.windows` | 部署流程 | 部署 Traefik 时 | 模板替换后写入 |
| `data/apps/traefik/data/traefik.yml` | 部署流程 | 部署 Traefik 时 | 静态配置 |
| `data/traefik/dynamic/routes.yml` | Pomelo Orbit | 路由变更时 | 按需创建目录 |
| `data/traefik/certs/*.crt` | Pomelo Orbit | 证书上传时 | 按需创建目录 |

**关键设计**：
- 静态配置由部署流程生成（一次性）
- 动态配置由 Pomelo Orbit 运行时生成（频繁变更）
- Pomelo Orbit 在下发路由和证书时自动创建 `data/traefik/` 目录，不依赖 Traefik 是否已部署

**为什么有两个环境变量**：
- `POMELO_ORBIT_DATA__TRAEFIK_WORKDIR` - Traefik 应用工作目录，存放 docker-compose.yml 和静态配置文件（data/traefik.yml）
- `POMELO_ORBIT_DATA__TRAEFIK` - Traefik 数据目录，存放动态路由和证书，由 Pomelo Orbit 运行时生成

## 实现逻辑

### 模板变量替换

```python
class ApplicationManager:
    def _get_pomelo_orbit_data_root(self) -> str:
        """获取 pomelo-orbit 数据根目录的宿主机路径"""
        # 优先从环境变量读取（容器模式）
        data_root = os.getenv("POMELO_ORBIT_DATA__ROOT")

        if not data_root:
            # 本地开发模式：从 application_dir 推导
            # application_dir = .../backend/data/apps/traefik
            # data_root = .../backend/data
            data_root = str(self.application_dir.parent.parent)

        # 统一使用正斜杠（Docker 兼容）
        return data_root.replace("\\", "/")

    def _replace_template_variables(self, content: str) -> str:
        """替换模板变量

        支持的全局模板变量：
        - {{POMELO_ORBIT_DATA__ROOT}} → pomelo-orbit 数据根目录
        """
        data_root = self._get_pomelo_orbit_data_root()
        return content.replace("{{POMELO_ORBIT_DATA__ROOT}}", data_root)

    async def deploy(self, config_files: list, ...):
        for config_file in config_files:
            content = config_file.content

            # 如果是 .env 文件，替换模板变量
            if config_file.path.startswith('.env.'):
                content = self._replace_template_variables(content)

            self.write_file(config_file.path, content, ...)
```

### 路由和证书下发

```python
class TraefikManager:
    def __init__(self):
        config = get_config()
        self.dynamic_route_dir = Path(config.data.traefik.dynamic_route_dir)
        self.cert_dir = Path(config.data.traefik.cert_dir)
        self.container_name = config.data.traefik.container_name

    def sync_routes(self, routes: list[Route]):
        """同步路由配置到 Traefik

        自动创建目录，不依赖 Traefik 是否部署
        """
        traefik_config = self._generate_traefik_config(routes)

        route_file = self.dynamic_route_dir / "routes.yml"
        route_file.parent.mkdir(parents=True, exist_ok=True)  # 按需创建
        route_file.write_text(traefik_config)

        logger.info(f"Routes synced to {route_file}")

    def deploy_certificate(self, domain: str, cert_pem: str, key_pem: str):
        """下发证书到 Traefik

        自动创建目录，不依赖 Traefik 是否部署
        """
        cert_file = self.cert_dir / f"{domain}.crt"
        key_file = self.cert_dir / f"{domain}.key"

        cert_file.parent.mkdir(parents=True, exist_ok=True)  # 按需创建
        cert_file.write_text(cert_pem)
        key_file.write_text(key_pem)

        logger.info(f"Certificate deployed: {cert_file}, {key_file}")
```

## 各场景下的路径解析

| 场景 | POMELO_ORBIT_DATA__ROOT | POMELO_ORBIT_DATA__TRAEFIK | POMELO_ORBIT_DATA__TRAEFIK_WORKDIR |
|------|------------------------|---------------------------|-----------------------------------|
| Windows 本地 | `D:/Works/.../backend/data` | `D:/Works/.../backend/data/traefik` | `D:/Works/.../backend/data/apps/traefik` |
| Linux 本地 | `/opt/pomelo-orbit/data` | `/opt/pomelo-orbit/data/traefik` | `/opt/pomelo-orbit/data/apps/traefik` |
| Windows WSL2 容器 | `/d/Works/.../backend/data` | `/d/Works/.../backend/data/traefik` | `/d/Works/.../backend/data/apps/traefik` |
| Linux 容器 | `/opt/pomelo-orbit/data` | `/opt/pomelo-orbit/data/traefik` | `/opt/pomelo-orbit/data/apps/traefik` |

## 数据流向

### 流程1：部署 Traefik

```
1. 用户点击"部署 Traefik"
   ↓
2. 从数据库读取配置文件
   - docker-compose.yml
   - .env.windows（模板）
   - data/traefik.yml（静态配置）
   ↓
3. 替换 .env.windows 中的模板变量
   {{POMELO_ORBIT_DATA__ROOT}} → /d/Works/.../backend/data
   ↓
4. 写入到 data/apps/traefik/
   - docker-compose.yml
   - .env.windows（已替换）
   - data/traefik.yml
   ↓
5. 执行 init.sh（创建 data/ 目录）
   ↓
6. docker compose up
   ↓
7. Traefik 容器启动
   - 读取 /etc/traefik/traefik.yml（静态配置）
   - 监听 /etc/traefik/dynamic/（动态配置目录）
```

### 流程2：同步路由配置

```
1. 用户在 Web UI 创建/修改路由
   ↓
2. 保存到数据库 route 表
   ↓
3. Pomelo Orbit 路由同步服务
   - 读取 route 表
   - 生成 Traefik 动态配置（YAML）
   ↓
4. 写入到 data/traefik/dynamic/routes.yml
   - 自动创建目录（如果不存在）
   ↓
5. Traefik 自动检测文件变化（watch: true）
   ↓
6. Traefik 重新加载路由配置（无需重启）
```

### 流程3：下发证书

```
1. 用户上传证书（Web UI）
   ↓
2. 保存到数据库 route 表
   - route.cert_pem
   - route.cert_key
   ↓
3. Pomelo Orbit 证书同步服务
   - 读取 route 表
   - 提取证书内容
   ↓
4. 写入到 data/traefik/certs/
   - {domain}.crt
   - {domain}.key
   - 自动创建目录（如果不存在）
   ↓
5. 更新 data/traefik/dynamic/routes.yml
   - 添加 TLS 配置引用证书文件
   ↓
6. Traefik 自动加载证书（watch: true）
```

## 设计优势

1. **解耦**：Pomelo Orbit 和 Traefik 完全独立，可以任意顺序部署
2. **灵活**：支持先配置路由后部署 Traefik，或先部署后配置
3. **简单**：按需创建目录，不需要额外的初始化逻辑
4. **适配性**：自动适配本地开发和容器部署环境
5. **可维护**：配置集中在 config.defaults.yaml，路径不硬编码

## 实施过程中发现的问题

### 1. Traefik 证书路径不匹配

**问题**：动态配置中引用 `/certs/` 但实际挂载到 `/etc/traefik/certs/`

**错误日志**：
```
ERR Unable to parse certificate /certs/pomelo-orbit.pem
error="unable to generate TLS certificate: tls: failed to find any PEM data in certificate input"
```

**原因**：
- 证书目录挂载：`data/traefik/certs:/etc/traefik/certs`
- 配置引用路径：`/certs/pomelo-orbit.pem`（错误）

**修复**：
```python
# domain/route_service.py
config["tls"] = {
    "certificates": [{
        "certFile": f"/etc/traefik/certs/{route.name}.pem",  # 修复路径
        "keyFile": f"/etc/traefik/certs/{route.name}-key.pem",
    }]
}
```

### 2. 配置键名不匹配

**问题**：代码使用 `settings.traefik.*` 但配置定义在 `settings.data.traefik.*`

**修复**：
```python
# interfaces/api/route.py (修复前)
config_path = Path(settings.traefik.dynamic_config_dir)  # ❌

# interfaces/api/route.py (修复后)
config_path = get_project_root() / settings.data.traefik.dynamic_route_dir  # ✅
```

### 3. 应用目录路径硬编码

**问题**：`ApplicationService._get_application_dir()` 硬编码 `data/applications`

**修复**：
```python
# application/application_service.py
def _get_application_dir(self, application_code: str) -> Path:
    if self.base_dir:
        return self.base_dir / application_code
    # 从配置读取应用目录
    settings = get_settings()
    apps_dir: Path = get_project_root() / settings.data.apps
    return apps_dir / application_code
```

### 4. Dockerfile 路径错误

**问题**：
- 第61行：`COPY backend/data/migrations/` (旧路径)
- 第70行：`RUN mkdir -p /app/data/applications` (错误目录名)

**修复**：
```dockerfile
# Copy database migrations
COPY backend/migrations/ ./migrations/

# Create data directories
RUN mkdir -p /app/data/db /app/logs
```

### 5. 配置文件路径不一致

**问题**：多个配置文件中仍使用旧路径 `data/applications`

**修复文件**：
- `.dockerignore`: `backend/data/applications` → `backend/data/apps`
- `backend/.gitignore`: `data/applications/` → `data/apps/`
- `backend/.env.example`: 移除旧的 Traefik 配置，添加 `POMELO_ORBIT_DATA__ROOT`
- `frontend/src/views/ApplicationDetail.vue`: 删除提示中的路径

### 6. 时间函数混用导致负数耗时

**问题**：`domain/entities.py` 使用 `datetime.now()` 而不是 `utc_now()`

**表现**：
```
Started:  2026-03-16 06:28:35  (本地时间 UTC+8)
Finished: 2026-03-16 05:28:39  (UTC 时间)
Duration: -3595381 ms  (负数！)
```

**修复**：
```python
# domain/entities.py
from pomelo_orbit.infrastructure.time_utils import utc_now

# 将所有 datetime.now() 替换为 utc_now()
self.updated_at = utc_now()
```

## 验证清单

- ✅ `make lint` 通过（ruff + mypy）
- ✅ `make test` 通过（180个单元测试）
- ✅ 配置路径统一从 `config.defaults.yaml` 读取
- ✅ Dockerfile 路径与代码一致
- ✅ Traefik 证书路径正确（`/etc/traefik/certs/`）
- ✅ 时间统一使用 UTC（`utc_now()`）
- ✅ 所有配置文件路径更新（`.dockerignore`, `.gitignore`, `.env.example`）

