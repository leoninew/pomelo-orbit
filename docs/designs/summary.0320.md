# Pomelo Orbit 设计总结（2026-03-20）

参考 spec.20260316、spec.20260317.route，记录当前实现状态。

---

## 一、目录结构

```
data/
├── db/
│   └── pomelo-orbit.db        # SQLite 数据库
├── {app_code}/                # 应用工作目录
│   ├── docker-compose.yml
│   ├── .env
│   ├── data/                  # 应用运行时数据
│   └── deployments/           # 部署日志
├── traefik/                   # Traefik 应用工作目录（结构同上）
│   ├── docker-compose.yml
│   ├── .env
│   ├── data/
│   │   ├── traefik.yml        # 静态配置（部署时由 Jinja2 模板生成）
│   │   ├── dynamic/           # 动态路由配置（Pomelo Orbit 运行时写入）
│   │   ├── certs/             # 证书文件（Pomelo Orbit 运行时写入）
│   │   └── acme.json          # Let's Encrypt 证书存储
│   └── deployments/
└── ...
```

`config.defaults.yaml` 中 `traefik.dynamic_route_dir` 和 `traefik.cert_dir` 分别指向 `data/traefik/data/dynamic` 和 `data/traefik/data/certs`。

`backend/migrations/` 独立于 `data/`，容器内路径为 `/app/migrations`，不受数据卷挂载影响。

---

## 二、配置文件渲染

DooD 场景下 docker-compose.yml 的卷挂载路径必须是宿主机绝对路径。通过 Jinja2 在部署时解决：

- 配置文件以 `.jinja` 结尾时触发渲染，渲染后去掉后缀写入磁盘
- 普通文件原样写入

渲染上下文（`ApplicationManagerImpl._render_template`）：

```python
{
    "app": {
        "physical_data_dir": ...,      # data/ 的宿主机绝对路径
        "physical_app_data_dir": ...,  # data/{code}/data/ 的宿主机绝对路径
    },
    "traefik": {
        "dashboard_domain": ...,
    },
    "cert": {
        "letsencrypt": { "enabled": ..., "email": ..., "challenge": ..., "dns_provider": ... }
    },
}
```

宿主机路径探测（`get_physical_data_dir`）：
- 容器模式：读取 `/proc/self/cgroup` 获取容器 ID，通过 `docker inspect` 找到 `/app/data` 的 bind mount source
- 本地开发模式：项目根目录下的 `data/`

---

## 三、Traefik 部署

数据库预置两个 Traefik 应用（`v0.4.2` 迁移脚本），用户按平台选择：

| 应用 | 网络模式 | 容器名 | 适用场景 |
|---|---|---|---|
| Traefik Windows | bridge，端口映射 80/443/8080 | `traefik` | Windows/macOS（Docker Desktop）|
| Traefik Linux | `network_mode: host` | `traefik-linux` | Linux 原生 Docker |

两个应用的 `traefik.yml.jinja` 均启用混合 Provider：

```yaml
providers:
  file:
    directory: /etc/traefik/dynamic
    watch: true
  docker:
    endpoint: "unix:///var/run/docker.sock"
    exposedByDefault: false
    network: traefik
```

Let's Encrypt 配置在 `traefik.yml.jinja` 中条件渲染，由 `cert.letsencrypt.*` 配置项控制。

Windows 下 Traefik 文件监听失效（跨文件系统边界事件不传播），通过部署/撤销路由后发送 SIGHUP 信号解决（`TraefikManager._reload_traefik`），Linux 依赖 fsnotify 自动重载。

---

## 四、路由系统

混合路由架构：File Provider 管理数据库路由，Docker Provider 自动发现容器路由（通过 docker-compose labels 配置）。

**路由配置生成**（`RouteDomainService.generate_route_config`）：

| 模式 | entrypoint | TLS |
|---|---|---|
| HTTP | `web` | 无 |
| HTTPS + 手动/mkcert 证书 | `websecure` | 引用 `/etc/traefik/certs/{name}.pem` |
| HTTPS + Let's Encrypt | `websecure` | `certResolver: letsencrypt` |

每个路由生成独立的 `{name}.yml` 写入 `dynamic/` 目录。

**路由生命周期**（`RouteService`）：
- 创建/更新：保存后自动 deploy 或 revoke
- 启用/禁用：切换状态并同步文件
- `sync_routes`：全量重建，从数据库恢复证书文件，重新生成集中式 `tls.yml`

**待完成**：
- Traefik Labels Helper UI（辅助生成 docker-compose labels）
- docker-compose.yml 表单化编辑

---

## 五、证书管理

证书内容存储在数据库（`route.cert_pem` / `route.cert_key`），文件写入 `data/traefik/data/certs/`，挂载到容器 `/etc/traefik/certs/`。

支持三种证书类型（`CertType`）：

| 类型 | 说明 |
|---|---|
| `manual` | 用户上传 PEM，后端解析拆分后存库并写文件 |
| `mkcert` | `MkcertService.generate_cert()` 自动生成，存库并写文件 |
| `letsencrypt` | Traefik 自动申请，无需存储证书内容 |

`sync_routes` 时调用 `TraefikManager.generate_tls_config()` 生成集中式 `tls.yml`，包含所有有证书的路由，支持 Docker Label 路由的证书配置。

---

## 六、自举部署

`v0.4.3` 迁移脚本预置 `pomelo-orbit` 应用，`docker-compose.yml.jinja` 通过 `{{ app.physical_data_dir }}` 注入宿主机数据目录，通过 Docker Label 接入 Traefik 路由，支持通过 Web UI 零停机更新自身。
