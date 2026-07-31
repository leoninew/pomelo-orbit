# `reconcile_compose_proxies.py` — 协调手工 CD Compose 代理配置

Doc role: local script reference。与代码冲突时以代码为准。

## 用途

递归扫描 `/opt/pomelo-orbit/data/cd` 下的非符号链接 `docker-compose.yml`，为符合条件的服务写入 `host.docker.internal` 映射以及 `HTTP_PROXY`、`HTTPS_PROXY`、`ALL_PROXY`、`NO_PROXY`。脚本从服务使用的本机 Docker bridge 网络读取 gateway，并从 `ss -H -ltnp` 识别绑定于该 gateway 的 Mihomo listener；不会将代理地址或端口写死。

## 前置条件

远程宿主机需要 Python 3.12、PyYAML、Docker Compose v2 和已安全监听在目标 Docker bridge gateway 的 Mihomo。脚本通常需要以 root 运行，以读取 Mihomo socket 的进程归属并写入 Compose 文件。

## 先预演

```bash
sudo python3 scripts/reconcile_compose_proxies.py --dry-run
```

运行前，脚本会验证目标 Compose 配置、发现唯一候选 listener，并通过该代理执行 HTTPS 探测。发现不到或发现多个 listener，或探测失败时，会在写入前退出。

## 用法

```bash
# 默认：写入、验证并对已变更项目执行 docker compose up -d
sudo python3 scripts/reconcile_compose_proxies.py

# 写入但不部署
python3 scripts/reconcile_compose_proxies.py --include typing-island --no-deploy

# 指定扫描根、网络或 listener
python3 scripts/reconcile_compose_proxies.py \
  --root /opt/pomelo-orbit/data/cd \
  --network traefik \
  --proxy-port 7891
```

支持选项：`--root`、`--host`、`--network`、`--proxy-listen`、`--proxy-port`、`--no-proxy`、`--include`、`--backup-dir`、`--dry-run`、`--no-deploy`、`--continue-on-error`、`--compose-bin`。使用 `--help` 查看当前参数语义和限制。

## 写入与恢复边界

候选文件在写入前会执行 `docker compose config -q`。变更时，原始字节默认备份到 `<root>/.proxy-reconcile-backups/<UTC-run-id>/`；部署失败则恢复备份，并尝试以恢复后的 Compose 再次部署。PyYAML 会规范化发生变更文件的 YAML 格式和注释；未发生配置变化的文件不会重写、备份或部署。

该工具仅管理手工维护的 CD Compose 物料；不替代 Pomelo Orbit 部署渲染器，也不修改 labels、networks、ports、volumes、镜像、Traefik 路由或 `/opt/pomelo-orbit/docker-compose.yml`。