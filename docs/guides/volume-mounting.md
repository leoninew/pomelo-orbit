# Pomelo Orbit 目录挂载原理

## 概述

Pomelo Orbit 通过目录挂载实现数据持久化和容器间数据共享。所有应用数据、路由配置和证书文件都通过挂载机制在宿主机和容器之间同步。

## 整体目录结构

```
backend/data/                           # 宿主机数据根目录
├── db/
│   └── pomelo-orbit.db                # SQLite 数据库
├── applications/                       # 应用部署目录
│   ├── traefik/                       # Traefik 应用
│   ├── nginx/                         # Nginx 应用
│   └── pomelo-orbit/                  # Pomelo Orbit 自身
└── migrations/                         # 数据库迁移脚本
```

## Pomelo Orbit 容器挂载

### 挂载配置

```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock  # Docker 控制
  - ${POMELO_ORBIT_DATA_DIR}:/app/data         # 数据目录
```

### 挂载映射

```
宿主机                                    容器内
/path/to/backend/data          →        /app/data
├── db/pomelo-orbit.db         →        /app/data/db/pomelo-orbit.db
├── applications/              →        /app/data/applications/
└── migrations/                →        /app/data/migrations/
```

## 应用数据目录

### 目录结构

每个应用在 `applications/` 下有独立目录：

```
applications/{app_name}/
├── docker-compose.yml         # 从数据库同步
├── .env.linux                 # 从数据库同步
├── .env.windows               # 从数据库同步
├── init.sh                    # 从数据库同步（可选）
├── data/                      # 应用运行时数据
└── deployments/               # 部署日志
    └── {deployment_id}.log
```

### 数据流向

```
数据库 (application_config_file)
  ↓ 部署时读取
容器内 /app/data/applications/{app_name}/
  ↓ 挂载映射
宿主机 backend/data/applications/{app_name}/
```

## Traefik 特殊挂载

### 挂载配置

```yaml
volumes:
  # 静态配置
  - ${POMELO_TRAEFIK_DATA_DIR}/data/traefik.yml:/etc/traefik/traefik.yml

  # 动态路由配置（只读）
  - ${POMELO_TRAEFIK_DATA_DIR}/data/dynamic:/etc/traefik/dynamic:ro

  # 证书目录（只读）
  - ${POMELO_TRAEFIK_DATA_DIR}/data/certs:/certs:ro
```

### 目录结构

```
traefik/
├── data/
│   ├── traefik.yml           # 静态配置（入口点、日志等）
│   ├── dynamic/              # 动态路由配置
│   │   └── routes.yml        # Pomelo Orbit 生成的路由规则
│   └── certs/                # 证书文件
│       ├── example.com.crt
│       └── example.com.key
├── docker-compose.yml
└── .env.linux
```

## 路由配置同步

平台路由 **不再** 依赖 `dynamic/routes.yml` 文件 watch。

### 同步流程（E5/E6）

```
1. 用户在 Web UI 配置路由
   ↓
2. 保存到数据库 route 表
   ↓
3. 解析当前 Gateway → gateway_config.rest_api_url
   ↓
4. PUT {rest_api_url}/api/providers/rest 全量快照
   ↓
5. Traefik rest provider 生效
```

网关静态配置（含 `providers.rest`）由 Gateway compile 生成到 Version 挂载 `traefik.yml`（logical + content_mode=sync），Deploy 时物化。

## 证书文件同步

### 同步流程

```
1. 证书存储在数据库
   route.cert_pem  - 证书内容
   route.cert_key  - 私钥内容
   ↓
2. Pomelo Orbit 同步到文件系统
   /app/data/applications/traefik/data/certs/
   ├── {domain}.crt  ← route.cert_pem
   └── {domain}.key  ← route.cert_key
   ↓
3. 挂载到 Traefik 容器
   /certs/{domain}.crt
   /certs/{domain}.key
   ↓
4. Traefik 动态配置引用
   tls:
     certificates:
       - certFile: /certs/example.com.crt
         keyFile: /certs/example.com.key
```

## 环境变量路径适配

### Windows + WSL2

```bash
POMELO_TRAEFIK_DATA_DIR=/d/SourceCodes/mywork/pkg/creativity/others/pomelo-orbit/backend/data/applications/traefik
```

### Linux

```bash
POMELO_TRAEFIK_DATA_DIR=/opt/pomelo-orbit/data/applications/traefik
```

部署时选择对应的 `.env.linux` 或 `.env.windows` 文件。

## 只读挂载保护

### 为什么使用只读挂载

```yaml
# 动态配置和证书使用只读挂载，防止容器修改
- ${POMELO_TRAEFIK_DATA_DIR}/data/dynamic:/etc/traefik/dynamic:ro
- ${POMELO_TRAEFIK_DATA_DIR}/data/certs:/certs:ro
```

**优势**：
- 防止容器意外修改配置
- 提高安全性
- 配置只能通过 Pomelo Orbit 管理

## 部署日志持久化

### 日志存储

```
/app/data/applications/{app_name}/deployments/{deployment_id}.log
```

每次部署的日志都保存在应用目录下，便于追溯和调试。

### 日志内容

- 配置文件写入日志
- init.sh 执行输出
- docker compose 命令输出
- 错误信息和堆栈跟踪

## 数据持久化策略

### 持久化数据

✅ 以下数据持久化在宿主机：
- 数据库文件：`db/pomelo-orbit.db`
- 应用配置：`applications/{app}/docker-compose.yml`
- 路由配置：`applications/traefik/data/dynamic/`
- 证书文件：`applications/traefik/data/certs/`
- 部署日志：`applications/{app}/deployments/`

### 临时数据

❌ 以下数据不持久化：
- 容器内部状态（容器重启后丢失）
- 容器日志（使用 `docker logs` 查看）

## 完整数据流示意图

```
┌─────────────────────────────────────────────────────────┐
│ Pomelo Orbit 容器                                        │
│                                                          │
│  /app/data/                                             │
│  ├── db/pomelo-orbit.db          ← 数据库              │
│  ├── applications/                                      │
│  │   ├── traefik/                                      │
│  │   │   ├── data/                                     │
│  │   │   │   ├── dynamic/        ← 路由配置           │
│  │   │   │   └── certs/          ← 证书文件           │
│  │   │   └── docker-compose.yml  ← 应用配置           │
│  │   └── nginx/                                        │
│  └── migrations/                                        │
│                                                          │
│  ↕ 挂载映射                                             │
│                                                          │
│  宿主机: backend/data/                                  │
└─────────────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────────┐
│ Traefik 容器                                             │
│                                                          │
│  /etc/traefik/                                          │
│  ├── traefik.yml                ← 静态配置              │
│  └── dynamic/                   ← 动态路由（只读）      │
│                                                          │
│  /certs/                        ← 证书目录（只读）      │
└─────────────────────────────────────────────────────────┘
```

## 关键设计特点

### 1. 双向数据流

```
数据库 ←→ 文件系统 ←→ 容器
  ↑                    ↑
  配置存储            运行时
```

### 2. 配置集中管理

所有配置存储在数据库，通过挂载同步到容器。

### 3. 数据持久化

数据库、配置、证书都持久化在宿主机，容器重启不丢失。

### 4. 动态更新

Traefik 监听配置文件变化，自动重载路由，无需重启容器。

### 5. 安全隔离

使用只读挂载保护关键配置，防止容器意外修改。

## 故障排查

### 挂载路径不存在

**问题**：容器启动失败，提示挂载路径不存在

**解决**：
1. 检查 `.env` 文件中的路径配置
2. 确保宿主机路径存在
3. 检查路径权限

### 配置文件未同步

**问题**：修改平台路由后 Traefik 未生效

**解决**：
1. 确认 Gateway `rest_api_url` 可达（平台 PUT `/api/providers/rest`）
2. 确认网关 Version 已 compile 并部署含 `providers.rest` 的 `traefik.yml`
3. 检查 Traefik 日志：`docker logs <gateway-container>`

### 证书文件无法访问

**问题**：HTTPS 访问失败，提示证书错误

**解决**：
1. 检查证书文件是否存在：`ls /app/data/applications/traefik/data/certs/`
2. 检查文件权限
3. 检查证书内容是否正确

## 最佳实践

### 1. 使用绝对路径

环境变量中使用绝对路径，避免相对路径问题：

```bash
POMELO_ORBIT_DATA_DIR=/opt/pomelo-orbit/data
```

### 2. 定期备份

定期备份数据目录：

```bash
tar -czf backup-$(date +%Y%m%d).tar.gz backend/data/
```

### 3. 权限管理

确保容器有权限访问挂载目录：

```bash
chmod 755 backend/data/applications/
```

### 4. 监控磁盘空间

定期检查磁盘空间，避免日志文件占满磁盘：

```bash
du -sh backend/data/applications/*/deployments/
```
