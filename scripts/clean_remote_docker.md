# `clean_remote_docker.py` — 远程 Docker 可回收空间清理

Doc role: local script reference。与代码冲突时以代码为准。

## 用途

通过 SSH 检查远程主机根分区和 Docker 磁盘占用；显式授权后，清理全部 BuildKit 构建缓存以及未被任何容器使用的镜像。

## 前置条件

`scripts/.env` 必须定义：

- `SSH_HOST`
- `SSH_USER`

本机需要可执行的 `ssh` 命令；远程主机需要 Docker，并允许当前 SSH 用户执行 Docker 命令。

## 用法

```bash
# 默认只检查：根分区、Docker 占用、运行中容器
python scripts/clean_remote_docker.py

# 执行清理
python scripts/clean_remote_docker.py --execute
```

## 执行效果与安全边界

默认模式不会删除远程数据。

`--execute` 会在远程依次执行：

```bash
docker builder prune --all --force
docker image prune --all --force
```

因此它会删除全部 BuildKit 缓存，并删除未被任何容器引用的镜像；不会删除容器、被任意容器使用的镜像或 Docker 卷。后续构建可能需要重新拉取或构建缓存层。

配置缺失、SSH 不可用或远程 Docker 命令失败时，脚本记录错误并以非零状态退出。