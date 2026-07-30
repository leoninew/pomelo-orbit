# 运维脚本

## 检查

```bash
python -m mypy scripts/
python -m ruff check scripts/ --fix
python -m ruff format scripts/
```

## CD Compose 代理协调

`reconcile_compose_proxies.py` 在宿主机上协调手工维护的 CD Compose 文件：递归扫描 `/opt/pomelo-orbit/data/cd/**/docker-compose.yml`，为每个服务写入 `host.docker.internal` 映射和 `HTTP_PROXY`、`HTTPS_PROXY`、`ALL_PROXY`、`NO_PROXY`。仅在文件实际变化时才备份、写入并立即执行 `docker compose up -d`。

脚本会自动从 Compose 服务使用的本机 Docker bridge 网络读取 IPv4 gateway，并从 `ss -H -ltnp` 发现**绑定到该 gateway 的 Mihomo listener**。不会将代理 endpoint 写死为特定地址或端口。发现不到、发现多个候选 listener，或通过候选 listener 的 HTTPS 代理探测失败时，脚本会在改动 Compose 文件前退出。

远程前置条件：Python 3.12、PyYAML、Docker Compose v2、宿主 Mihomo 已在每个目标 Docker bridge gateway 上安全监听。脚本不安装或修改 Mihomo，也不会改动 labels、networks、ports、volumes、镜像、Traefik 路由或 `/opt/pomelo-orbit/docker-compose.yml`。

脚本需要读取其他用户运行的 Mihomo socket 进程归属并写入 Compose 文件，通常应以 root 运行。先进行只读预演：

```bash
sudo python3 scripts/reconcile_compose_proxies.py --dry-run
```

默认写入、验证并立即部署变更的项目：

```bash
sudo python3 scripts/reconcile_compose_proxies.py
```

常用选项：

```bash
# 只处理指定 CD 目录；写入但不部署
python3 scripts/reconcile_compose_proxies.py --include typing-island --no-deploy

# 当一个 Docker gateway 上有多个 Mihomo listener 时精确选择端口
python3 scripts/reconcile_compose_proxies.py --network traefik --proxy-port 7891

# 指定扫描根目录并追加内部绕过项
python3 scripts/reconcile_compose_proxies.py \
  --root /opt/pomelo-orbit/data/cd \
  --no-proxy internal.example,10.0.0.0/8
```

已变更文件的原始字节备份默认位于：

```text
<root>/.proxy-reconcile-backups/<UTC-run-id>/
```

候选 Compose 在替换前会执行 `docker compose config -q`；写入后部署失败时，脚本恢复对应备份并尝试以恢复后的 Compose 重新部署。PyYAML 会规范化发生变更的 YAML 文件格式和注释；已经符合目标状态的文件不会重写、不会备份，也不会触发部署。

该脚本管理的是手工 CD Compose 物料。Pomelo Orbit 后续重新渲染其自身管理的部署工作区时，可能覆盖独立的手工配置；它不是 Pomelo Orbit Go 部署渲染器的替代方案。
