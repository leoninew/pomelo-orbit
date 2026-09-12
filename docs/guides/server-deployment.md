# 服务器首次部署指南
最后修改时间: 2026-09-11 14:48:17

Doc role: living guide（运维向）。与代码冲突时以代码为准。

本指南说明如何在新服务器上手动完成 Pomelo Orbit 的首次部署。

## 前置条件

- Docker 已安装并运行
- 服务器可访问 ghcr.io（或已提前拉取镜像）
- Traefik 已部署（或将与 Pomelo Orbit 同步部署）

---

## 1. 创建部署目录

```bash
mkdir -p /opt/pomelo-orbit
cd /opt/pomelo-orbit
```

---

## 2. 准备 docker-compose.yml

```bash
cat > /opt/pomelo-orbit/docker-compose.yml << 'EOF'
services:
  pomelo-orbit:
    image: ghcr.io/leoninew/pomelo-orbit:latest
    container_name: pomelo-orbit
    restart: unless-stopped
    networks:
      - traefik
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /opt/pomelo-orbit/data/db:/app/data/db
      - /opt/pomelo-orbit/ci:/app/data/pipeline
      - /opt/pomelo-orbit/cd:/opt/pomelo-orbit/cd
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

将 `pomelo-orbit.example.com` 替换为实际域名。

---

## 3. 准备 .env

```bash
cat > /opt/pomelo-orbit/.env << 'EOF'
# JWT 密钥（同时用于凭据加密，必须使用 Fernet 格式）
# 生成方法: python -c "import base64, os; print(base64.urlsafe_b64encode(os.urandom(32)).decode())"
POMELO_ORBIT_JWT__SECRET_KEY=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=
EOF
```

**必须**将 `POMELO_ORBIT_JWT__SECRET_KEY` 替换为真实的 Fernet 密钥，否则启动后无法登录。

Traefik 控制面 URL 与业务域名后缀在 **CD 网关（Gateway config）** 中配置（`rest_api_url`、`base_domain`）。

生成方法（需要 Python + cryptography）：

```bash
python3 -c "from cryptography.fernet import Fernet; print(Fernet.generate_key().decode())"
```

---

## 4. 确保 traefik 网络存在

```bash
docker network inspect traefik || docker network create traefik
```

---

## 5. 拉取镜像并启动

```bash
cd /opt/pomelo-orbit
docker compose pull
docker compose up -d
```

---

## 6. 验证

```bash
# 查看容器状态
docker compose ps

# 查看启动日志
docker compose logs -f --tail=50
```

正常启动后访问 `http://pomelo-orbit.example.com`，使用初始管理员账号登录。

---

## 更新镜像

```bash
cd /opt/pomelo-orbit
docker compose pull
docker compose up -d
```

如果数据库迁移出现 checksum 错误，需要删除数据库文件后重启：

```bash
docker compose down
rm /opt/pomelo-orbit/data/db/pomelo-orbit.db
docker compose up -d
```

> 注意：删除数据库会清空所有数据，仅在迁移损坏时使用。

---

## 目录结构说明

```
/opt/pomelo-orbit/
├── docker-compose.yml   # 容器编排配置（手动维护）
├── .env                 # 宿主机环境变量，通过 env_file 注入容器
├── data/db/             # SQLite 数据库（挂载到容器 /app/data/db）
├── ci/                  # CI workspace（挂载到容器 /app/data/pipeline）
└── cd/                  # 可选：给 local Environment.workspace_root 使用的宿主目录
```

`.env` 有两个层面，但内容相同：
- 宿主机 `/opt/pomelo-orbit/.env`：通过 `env_file` 注入容器环境变量，供应用读取
- Pomelo Orbit 管理的应用配置模板（`v0.4.3` 迁移写入）：部署时由系统渲染写入，路径由系统管理

首次手动部署时只需关注宿主机的 `.env`。

默认 YAML 的 `workspace.pipeline=data/pipeline` 在容器内解析为 `/app/data/pipeline`。CD 工作目录不再来自进程配置；打开未初始化 Project 时由 Web Wizard 写入 `Environment.workspace_root`。默认值 `~/.pomelo-orbit` 原样展示和保存，使用时才展开为进程用户主目录。容器部署若要把 CD 根放在已挂载路径上，应在 Wizard 中填写该绝对路径。`logging.deployment_root` 仍是控制面部署执行日志目录。
