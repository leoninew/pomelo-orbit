# Traefik 动态路由与 HTTPS 证书机制

## 1. Traefik 动态路由机制

### 1.1 配置结构

**静态配置** (`traefik.yml`)：
```yaml
entryPoints:
  web:
    address: ":80"   # HTTP 入口
  websecure:
    address: ":443"  # HTTPS 入口

providers:
  file:
    directory: /etc/traefik/dynamic
    watch: true  # 自动监听文件变化
```

**动态配置** (`dynamic/{route_name}.yml`)：
- 每个路由一个独立的 YAML 文件
- Traefik 自动监听文件变化并重载
- 无需重启容器

### 1.2 Docker 挂载映射

```yaml
volumes:
  - ${POMELO_TRAEFIK_DATA_DIR}/data/dynamic:/etc/traefik/dynamic:ro
  - ${POMELO_TRAEFIK_DATA_DIR}/data/certs:/certs:ro
```

**路径映射**：
- 宿主机：`backend/data/applications/traefik/data/dynamic/`
- 容器内：`/etc/traefik/dynamic/`
- 证书目录：`backend/data/applications/traefik/data/certs/` → `/certs/`

### 1.3 路由配置生成

**HTTP 模式**：
```yaml
http:
  routers:
    {route_name}-http-router:
      rule: Host(`domain.com`)
      service: {route_name}-service
      entryPoints: [web]
  services:
    {route_name}-service:
      loadBalancer:
        servers:
        - url: http://target:port
```

**HTTPS 模式**：
```yaml
http:
  routers:
    {route_name}-https-router:
      rule: Host(`domain.com`)
      service: {route_name}-service
      entryPoints: [websecure]
      tls: {}
  services:
    {route_name}-service:
      loadBalancer:
        servers:
        - url: http://target:port
tls:
  certificates:
  - certFile: /certs/{route_name}.pem
    keyFile: /certs/{route_name}-key.pem
```

**代码实现** (`domain/route_service.py:12-54`)：
```python
def generate_route_config(route: Route) -> dict[str, Any]:
    service_name = f"{route.name}-service"
    rule = f"Host(`{route.domain}`)"
    if route.path_prefix != "/":
        rule += f" && PathPrefix(`{route.path_prefix}`)"

    config = {
        "http": {
            "services": {
                service_name: {
                    "loadBalancer": {
                        "servers": [{"url": route.target_url}]
                    }
                }
            }
        }
    }

    if route.https_enabled and route.cert_pem:
        # HTTPS 模式
        config["http"]["routers"] = {
            f"{route.name}-https-router": {
                "rule": rule,
                "service": service_name,
                "entryPoints": ["websecure"],
                "tls": {}
            }
        }
        config["tls"] = {
            "certificates": [{
                "certFile": f"/certs/{route.name}.pem",
                "keyFile": f"/certs/{route.name}-key.pem"
            }]
        }
    else:
        # HTTP 模式
        config["http"]["routers"] = {
            f"{route.name}-http-router": {
                "rule": rule,
                "service": service_name,
                "entryPoints": ["web"]
            }
        }

    return config
```

### 1.4 路由启用流程

```
用户点击"启用"
    ↓
API: PUT /api/route/{id}/enable
    ↓
RouteService.enable_route()
    → route.enable() (领域方法)
    → save() (数据库)
    → deploy_cert() (如果有证书，写入文件)
    → deploy_route() (写入 YAML 配置)
    ↓
Traefik 自动重载配置
```

**代码实现** (`application/route_service.py:56-70`)：
```python
def enable_route(self, route_id: str) -> None:
    route = self.route_repo.find_by_id(route_id)

    # 1. 调用领域方法
    route.enable()
    self.route_repo.save(route)

    # 2. 下发证书文件（如果有）
    if route.https_enabled and route.cert_pem and route.cert_key:
        self.traefik_manager.deploy_cert(route.name, route.cert_pem, route.cert_key)

    # 3. 下发路由配置
    self.traefik_manager.deploy_route(route)
```

## 2. HTTPS 证书机制

### 2.1 证书存储架构

**双重存储策略**：

1. **数据库存储**（主数据源）：
   ```sql
   ALTER TABLE route ADD COLUMN cert_pem TEXT;
   ALTER TABLE route ADD COLUMN cert_key TEXT;
   ```
   - 支持备份、恢复、同步

2. **文件系统存储**（运行时）：
   ```
   backend/data/applications/traefik/data/certs/
   ├── {route_name}.pem       # 证书文件
   └── {route_name}-key.pem   # 私钥文件
   ```
   - Traefik 从文件系统读取证书

### 2.2 证书上传流程

```
用户上传 PEM 文件
    ↓
API: POST /api/route/{id}/cert
    ↓
Certificate.parse_pem() (领域层解析)
    → 拆分证书和私钥
    ↓
RouteService.enable_https()
    → route.enable_https() (领域方法)
    → save() (保存到数据库)
    → deploy_cert() (写入文件系统)
    → deploy_route() (更新路由配置)
    ↓
Traefik 自动重载配置
```

**API 实现** (`interfaces/api/route.py:182-205`)：
```python
@router.post("/{route_id}/cert")
async def upload_cert(route_id: str, pem: UploadFile) -> RouteResp:
    # 1. 读取上传的 PEM 文件
    cert_content = await pem.read()

    # 2. 解析 PEM（领域层）
    cert_pem, cert_key = traefik_manager.parse_cert(cert_content)

    # 3. 启用 HTTPS（应用层）
    route = route_service.enable_https(route_id, cert_pem, cert_key)
    return RouteResp.model_validate(RouteMapper.to_orm(route))
```

### 2.3 证书解析（领域层）

**Certificate 值对象** (`domain/certificate.py:8-36`)：
```python
class Certificate:
    @staticmethod
    def parse_pem(pem_content: bytes) -> tuple[str, str]:
        """解析合并的 PEM 文件，返回 (cert_pem, cert_key)"""
        text = pem_content.decode("utf-8")
        cert_blocks: list[str] = []
        key_blocks: list[str] = []

        # 遍历 PEM 文本，识别块边界
        for line in text.splitlines(keepends=True):
            if line.strip().startswith("-----BEGIN "):
                # 开始新块
            elif line.strip().startswith("-----END "):
                # 结束块，根据 "PRIVATE KEY" 关键字分类

        return "".join(cert_blocks), "".join(key_blocks)
```

**输入格式**（顺序不限）：
```
-----BEGIN CERTIFICATE-----
...
-----END CERTIFICATE-----
-----BEGIN PRIVATE KEY-----
...
-----END PRIVATE KEY-----
```

### 2.4 证书部署（基础设施层）

**deploy_cert** (`infrastructure/traefik_manager.py:115-118`)：
```python
def deploy_cert(self, route_name: str, cert_pem: str, cert_key: str) -> None:
    """部署证书文件"""
    (self.cert_dir / f"{route_name}.pem").write_bytes(cert_pem.encode())
    (self.cert_dir / f"{route_name}-key.pem").write_bytes(cert_key.encode())
```

**enable_https** (`application/route_service.py:94-110`)：
```python
def enable_https(self, route_id: str, cert_pem: str, cert_key: str) -> Route:
    route = self.route_repo.find_by_id(route_id)

    # 1. 调用领域方法
    route.enable_https(cert_pem, cert_key)
    self.route_repo.save(route)

    # 2. 下发证书文件
    self.traefik_manager.deploy_cert(route.name, cert_pem, cert_key)

    # 3. 如果路由已启用，重新下发路由配置
    if route.enabled:
        self.traefik_manager.deploy_route(route)

    return route
```

### 2.5 证书同步机制

**sync_routes** (`application/route_service.py:82-92`)：
```python
def sync_routes(self) -> None:
    """同步路由配置（含证书文件重建）"""
    routes = self.route_repo.find_all()
    for route in routes:
        # 1. 从数据库重建证书文件
        if route.https_enabled and route.cert_pem and route.cert_key:
            self.traefik_manager.restore_cert(route.name, route.cert_pem, route.cert_key)

        # 2. 部署或撤销路由配置
        if route.enabled:
            self.traefik_manager.deploy_route(route)
        else:
            self.traefik_manager.revoke_route(route)
```

**使用场景**：
- 服务器重启后恢复证书文件
- 证书文件意外丢失
- 迁移到新环境

## 3. 关键设计决策

### 3.1 数据库是唯一数据源

**优势**：
- ✅ 支持备份和恢复
- ✅ 支持同步和迁移
- ✅ 证书内容不会丢失
- ✅ 可以从数据库重建所有文件

**实现**：
- 上传时：解析 → 入库 → 写文件
- 启用时：从数据库读取 → 写文件
- 同步时：从数据库重建所有文件

### 3.2 零停机更新

**实现**：
- Traefik 监听文件变化（`watch: true`）
- 文件变化时自动重载配置
- 无需重启容器

**Windows 特殊处理**：
- 发送 SIGHUP 信号强制重载
- Linux/Mac 不需要
