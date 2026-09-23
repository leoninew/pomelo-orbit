# Docker 部署与服务器首次部署指南
最后修改时间: 2026-09-23 19:59:36

Doc role: living guide（运维向）。与代码冲突时以代码为准；领域模型见 [CD 模型](../product/cd-model.md)。

本指南统一说明 Pomelo Orbit 的 Docker 部署、服务器首次部署、更新和目录权限要求。服务器首次部署不再维护另一套重复步骤；旧入口见 [服务器首次部署兼容入口](./server-deployment.md)。

## 前置条件

- Docker 已安装并运行
- 服务器可访问 `ghcr.io`，或已提前拉取目标镜像
- Traefik 已部署，或准备与 Pomelo Orbit 一起接入
- 宿主机可以访问 Docker socket

Pomelo Orbit 需要 Docker socket、统一工作区根目录、数据库目录和应用日志目录。local Environment 的 `workspace_root` 还必须选择一个同时对 Orbit 容器和 Docker daemon 可见的路径。

## 1. 创建部署目录

```bash
mkdir -p /opt/pomelo-orbit
cd /opt/pomelo-orbit
```

## 2. 准备 docker-compose.yml

下面的 Compose 配置同时覆盖控制面启动、Docker socket、数据持久化和 Traefik 接入。将 `pomelo-orbit.example.com` 替换为实际域名；不使用 Traefik 时可以移除 `networks`、`ports` 和 `labels` 中不需要的部分。

```bash
cat > /opt/pomelo-orbit/docker-compose.yml << 'EOF'
services:
  pomelo-orbit:
    image: ghcr.io/leoninew/pomelo-orbit:latest
    container_name: pomelo-orbit
    user: "1000:1000"
    group_add:
      - "${DOCKER_SOCKET_GID:?Set DOCKER_SOCKET_GID in .env}"
    restart: unless-stopped
    networks:
      - traefik
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /opt/pomelo-orbit/data:/app/data
      - /opt/pomelo-orbit/logs:/app/logs
    environment:
      - HOME=/app/data
    ports:
      - "9003:80"
    env_file:
      - /opt/pomelo-orbit/.env
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.pomelo-orbit.rule=Host(`pomelo-orbit.example.com`)"
      - "traefik.http.routers.pomelo-orbit.entrypoints=web"
      - "traefik.http.routers.pomelo-orbit.service=pomelo-orbit"
      - "traefik.http.services.pomelo-orbit.loadbalancer.server.port=80"

networks:
  traefik:
    external: true
EOF
```

如果使用本地构建镜像，将 `image` 改为对应的本地镜像标签即可；Compose 其余的数据、socket 和权限配置保持一致。

## 3. 准备 .env 和目录权限

```bash
cat > /opt/pomelo-orbit/.env << 'EOF'
# JWT 密钥（同时用于凭据加密，必须使用 Fernet 格式）
POMELO_ORBIT_JWT__SECRET_KEY=REPLACE_WITH_A_REAL_FERNET_KEY
EOF
printf 'DOCKER_SOCKET_GID=%s\n' "$(stat -c %g /var/run/docker.sock)" >> /opt/pomelo-orbit/.env
mkdir -p /opt/pomelo-orbit/data/db /opt/pomelo-orbit/logs
sudo chown -- 1000:1000 /opt/pomelo-orbit/data /opt/pomelo-orbit/data/db /opt/pomelo-orbit/logs
```

必须将 `POMELO_ORBIT_JWT__SECRET_KEY` 替换为真实的 Fernet 密钥，否则启动后无法登录。生成方法（需要 Python 和 `cryptography`）：

```bash
python3 -c "from cryptography.fernet import Fernet; print(Fernet.generate_key().decode())"
```

`DOCKER_SOCKET_GID` 必须与宿主机 `/var/run/docker.sock` 的 GID 一致；也可以用以下命令单独确认：

```bash
stat -c %g /var/run/docker.sock
```

已有实例切换到非 root 用户前，应单独迁移 Orbit 写入的日志、CI 工作区和 CD 受管文件。不要对 `data/deployment` 整体递归执行 `chown` 或 `chmod`：其中还可能包含 PostgreSQL 等应用数据卷，以及 Traefik 的 ACME 存储与证书目录。路由证书/私钥由业务逻辑以 `0600` 写入，ACME 存储应保留 Traefik 所需的 `0600` 权限；只按各目录的实际写入者调整属主。

## 4. 确保 Traefik 网络存在

使用上面的外部 `traefik` 网络时执行：

```bash
docker network inspect traefik || docker network create traefik
```

## 5. 首次启动

```bash
cd /opt/pomelo-orbit
docker compose pull
docker compose up -d
```

验证容器状态和启动日志：

```bash
docker compose ps
docker compose logs -f --tail=50
```

正常启动后访问 `http://pomelo-orbit.example.com`，使用初始管理员账号登录。

## 6. 工作区与 Gateway 配置

默认 YAML 的 `workspace.root=data` 与 `logging.deployment_root=data/deployment-logs` 分别解析为容器内的 `/app/data` 和 `/app/data/deployment-logs`。每个 Project 的 `Environment.workspace_root` 下固定使用：

- `pipeline/`：CI Pipeline checkout、Stage 日志和 Run 文件制品
- `deployment/<service-code>/`：CD 服务运行时和受管文件

Wizard 的 local 默认工作目录是 `~/.pomelo-orbit`，配置和界面原样展示，使用时才展开为容器内进程用户的主目录。若工作区需要落在其他已挂载路径上，应在 Wizard 中填写容器内绝对路径，并保证该路径同时对 Orbit 容器和 Docker daemon 可见；不要只挂载 Docker socket。

Traefik 控制面 URL 与业务域名后缀在 CD Gateway config 中配置（`rest_api_url`、`base_domain`）。创建或编辑 Gateway 时选择 ACME profile；DNS-01 token 在 Gateway 配置中填写，部署 DNS profile 时由 Orbit 作为 `CF_DNS_API_TOKEN` 写入 Traefik Compose environment，无需为 Orbit 配置全局 Cloudflare token。

## 7. 更新镜像

```bash
cd /opt/pomelo-orbit
docker compose pull
docker compose up -d
```

正常升级只替换镜像并保留 `/opt/pomelo-orbit/data`、`/opt/pomelo-orbit/logs` 和 `.env`。如果数据库迁移出现 checksum 错误，先确认备份和故障原因；只有明确确认数据库已损坏时才执行以下破坏性操作：

```bash
docker compose down
rm /opt/pomelo-orbit/data/db/pomelo-orbit.db
docker compose up -d
```

删除数据库会清空所有数据，不应作为普通升级步骤。

## 目录结构

```text
/opt/pomelo-orbit/
├── docker-compose.yml   # 容器编排配置（手动维护）
├── .env                 # 宿主机环境变量，通过 env_file 注入容器
├── data/                # 统一工作区根（含 db、pipeline、deployment 与部署日志）
└── logs/                # 控制面应用日志
```

`.env` 有两个层面，但内容相同：

- 宿主机 `/opt/pomelo-orbit/.env`：通过 `env_file` 注入容器环境变量，供应用读取
- Pomelo Orbit 管理的应用配置模板：部署时由系统渲染写入，路径由系统管理

首次手动部署时只需关注宿主机的 `.env`。
