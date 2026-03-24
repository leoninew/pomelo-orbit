# 证书机制说明

## 架构

Traefik 负责 TLS 终止，证书以 PEM 格式通过 pomelo-orbit UI 上传后：

1. 后端解析合并 PEM，拆分为 `{route_name}.pem`（证书）和 `{route_name}-key.pem`（私钥）分别落盘
2. 文件存储路径：`backend/data/applications/traefik/data/certs/`
3. 证书内容同时入库（`route.cert_pem` / `route.cert_key`），支持同步时从数据库重建文件
4. Traefik 动态配置（`dynamic/{route_name}.yml`）引用 `/certs/{route_name}.pem` 和 `/certs/{route_name}-key.pem`
5. Docker volume 挂载：`data/certs:/certs:ro`

## 上传格式

上传的 PEM 文件需同时包含证书和私钥两个块（顺序不限），后端自动拆分：

```
-----BEGIN CERTIFICATE-----
...
-----END CERTIFICATE-----
-----BEGIN PRIVATE KEY-----
...
-----END PRIVATE KEY-----
```

## cert.py 工具

`scripts/cert.py` 提供两个子命令，使用 uv 运行：

```bash
# 查看帮助
uv run --project backend/ python scripts/cert.py -h

# 生成证书（输出到 scripts/cert/{domain}.pem）
uv run --project backend/ python scripts/cert.py new -n pomelo-orbit.localhost

# 检查证书信任链（CA → 叶证书 → TLS 握手，模拟浏览器）
uv run --project backend/ python scripts/cert.py check -n pomelo-orbit.localhost
```

### new

生成 mkcert 证书并输出合并 PEM 到 `scripts/cert/` 目录：
- `{domain}.pem` — cert + key 合并，用于上传到 pomelo-orbit UI

### check

依次验证以下链路，任一失败均会标红：

1. mkcert 根 CA 文件是否存在
2. mkcert CA 是否已导入 Windows 系统信任库（Root store）
3. 本地 `scripts/cert/{domain}.pem` 文件内容（有效期、SAN、签发者）
4. 叶证书是否由当前 mkcert CA 签名（离线验证）
5. 实际 TLS 握手，使用系统信任库验证（与浏览器行为一致）

第 5 步通过 = 浏览器不报红。

## 前置条件

1. 安装 mkcert：https://github.com/FiloSottile/mkcert
2. 安装本地 CA（只需执行一次）：
   ```bash
   mkcert -install
   # Windows 额外执行（将 CA 导入系统信任库）：
   certutil -addstore "Root" "$LOCALAPPDATA/mkcert/rootCA.pem"
   ```
3. 配置 hosts 文件：
   ```
   127.0.0.1  pomelo-orbit.localhost
   ```

## 使用流程

### 1. 生成证书

```bash
# 生成证书（输出到 scripts/cert/{domain}.pem）
uv run --project backend/ python scripts/cert.py new -n app.localhost
```

### 2. 上传证书到 Pomelo Orbit

1. 打开 Pomelo Orbit UI
2. 进入路由管理页面
3. 选择要启用 HTTPS 的路由
4. 点击"上传证书"按钮
5. 选择生成的 `scripts/cert/{domain}.pem` 文件
6. 上传完成后，路由自动启用 HTTPS

### 3. 验证证书

```bash
# 检查证书是否正确安装
uv run --project backend/ python scripts/cert.py check -n app.localhost
```

### 4. 访问应用

在浏览器中访问 `https://app.localhost`，应该看到绿色锁图标，表示证书有效。

## 证书管理最佳实践

### 证书存储

- **数据库**: 证书内容存储在 `route.cert_pem` 和 `route.cert_key` 字段
- **文件系统**: 证书文件存储在 `backend/data/applications/traefik/data/certs/` 目录
- **同步**: 执行"同步路由"操作时，会从数据库重建证书文件

### 证书更新

当证书过期或需要更新时：

1. 生成新证书（使用相同域名）
2. 在 UI 中重新上传证书
3. 系统自动替换旧证书文件
4. Traefik 自动重新加载配置

### 证书删除

删除路由时，相关证书文件会自动删除。

### 多域名证书

如果需要为多个域名使用同一证书（通配符证书）：

1. 生成通配符证书：`mkcert "*.localhost"`
2. 为每个路由分别上传证书
3. 每个路由独立管理证书

## Docker Label 路由的证书管理

从 v0.4.0 开始，支持为 Docker Label 路由上传证书：

1. Docker Label 路由在数据库中有记录（`source_type='docker_label'`）
2. 可以为这些路由上传证书
3. 证书通过集中式 `tls.yml` 配置文件应用
4. 路由规则由 Docker Labels 管理，证书由 File Provider 管理

**注意**: Docker Label 路由本身是只读的（不能编辑域名、路径等），但可以上传和管理证书。

## Chrome 缓存清理

若 Chrome 显示不受信任但隐身模式正常：

1. `chrome://net-internals/#hsts` → Delete domain security policies 输入域名删除
2. `chrome://net-internals/#dns` → Clear host cache
3. 重新访问域名

要是都不能解决，可能要清除浏览器数据了。