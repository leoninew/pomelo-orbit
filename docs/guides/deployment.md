# Pomelo Orbit 部署原理

## 概述

Pomelo Orbit 是一个自举式部署平台，它可以管理自己的部署，也可以管理其他应用的部署。

## 核心理念

### 配置即数据

所有应用配置存储在数据库中，部署时动态生成配置文件。

**优势**：
- 配置集中管理
- 支持多环境（Linux/Windows）
- 可追溯的变更历史
- 便于版本管理和回滚

### 自举式部署

Pomelo Orbit 可以管理自己的部署：

```
Pomelo Orbit 容器
├── 运行 Web 服务
├── 挂载 Docker socket（控制宿主机 Docker）
├── 挂载数据目录（持久化数据）
└── 可以部署/更新自己
```

## 部署流程

### 1. 配置存储

应用配置存储在数据库的 `application_config_file` 表中：

```
├── docker-compose.yml  - Docker Compose 配置
├── .env.linux         - Linux 环境变量
├── .env.windows       - Windows 环境变量
└── init.sh            - 初始化脚本（可选）
```

### 2. 部署执行

```
用户点击"部署"
  ↓
从数据库读取配置文件
  ↓
写入到 /app/data/applications/{app_name}/
  ↓
物化 logical 挂载源（mkdir / 文件 seed|sync content 或 touch 空文件如 acme.json；替代历史 init.sh）
  ↓
docker compose up -d
  ↓
记录部署日志和状态
```

### 3. 部署状态

应用和部署记录各有独立的状态机，详见 [应用状态机设计](application-state-machine.md)。

**Application 状态**：`undeployed` → `deploying` → `deployed` / `deploy_failed`

**Deployment 状态**：`waiting_to_run` → `running` → `ran_to_completion` / `faulted` / `canceled`

## 预置应用

系统初始化时会创建三个预置应用：

### Traefik / Gateway（反向代理）
- 产品面：`Application(kind=gateway)` + `gateway_config`（`rest_api_url`、`base_domain`、可选 `image`）
- 保存 Gateway 时 compile 未发布 Version（组件 `traefik`：端口、docker.sock、静态 `traefik.yml`）
- 平台动态路由：`providers.rest` PUT；应用 Host：`{app_code}.{gateway.base_domain}`
- 部署与 standard 应用同一 Version / Deploy 管线

### nginx（示例应用）
- 演示应用部署流程
- 端口：8081

### Pomelo Orbit（自举部署）
- 平台自身
- 可以通过 Web UI 更新自己

## 多环境支持

### 环境配置文件

```bash
# Linux 服务器
.env.linux
POMELO_ORBIT_DATA_DIR=/opt/pomelo-orbit/data

# Windows + WSL2
.env.windows
POMELO_ORBIT_DATA_DIR=/d/SourceCodes/.../backend/data
```

部署时选择对应的环境配置文件。

## 部署触发方式

### 手动部署
- 在 Web UI 点击"部署"按钮
- 选择环境配置文件

### Webhook 自动部署
- Git 仓库推送触发
- 镜像仓库推送触发（Docker Registry）
- 自动匹配应用并触发部署

## 部署日志

每次部署的日志保存在：
```
/app/data/applications/{app_name}/deployments/{deployment_id}.log
```

可以在 Web UI 实时查看部署日志。

## 回滚机制

```
1. 查找上一次成功的部署记录
2. 读取该部署的配置（镜像版本、环境变量等）
3. 使用历史配置重新部署
```

**注意**：只回滚镜像版本和环境变量，数据库变更需要手动处理。

## 扩展性

### 挂载源物化（替代历史 init.sh）

平台部署前对 Version 声明的 **logical** 挂载自动准备宿主机源：

- 目录型：`mkdir -p`
- 文件型（如 `acme.json`）：父目录创建 + 不存在则建空文件（0600，不覆盖已有）

**不再**执行 `init.sh`。路由由 Traefik `providers.rest` 全量 PUT，不为路由强制 `data/dynamic` 目录。

### 支持完整的 Docker Compose 功能

- 多容器编排
- 网络配置
- 卷挂载
- 健康检查
- 资源限制

## 安全设计

### 凭据加密
- 敏感信息（密码、Token）加密存储
- 使用 Fernet 对称加密

### JWT 认证
- 用户登录后颁发 JWT Token
- 所有 API 请求需要携带有效 Token

### 密钥配置验证
- 部署前检查必需的密钥配置
- 拒绝使用默认密钥

## 故障排查

### 部署失败

1. 查看部署日志
2. 检查 Docker 镜像是否存在
3. 检查端口是否被占用
4. 检查环境变量配置

### 容器无法启动

1. 查看容器日志：`docker logs {container_name}`
2. 检查配置文件语法
3. 检查挂载路径是否存在

