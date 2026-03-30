# Pomelo Orbit 设计总结（2026-03-30）

---

## 一、整体架构

Pomelo Orbit 是一个基于 Docker 的自托管应用部署平台，采用 DooD（Docker-outside-of-Docker）模式运行，通过挂载宿主机 Docker socket 管理宿主机上的容器。

后端采用 FastAPI + SQLite，前端采用 Vue 3 + Ant Design Vue。所有应用配置存储在数据库中，部署时动态生成配置文件写入磁盘。

平台本身也作为一个受管应用运行，可以通过 Web UI 零停机更新自身（自举部署）。

---

## 二、应用管理

应用是平台的核心管理单元，支持两种来源类型：

- Git 源：关联仓库地址和部署分支，支持 Webhook 自动触发
- 镜像源：直接指定 Docker 镜像，支持镜像仓库 Webhook 触发

每个应用持有一组配置文件（docker-compose.yml、.env 等），部署时写入磁盘后执行 `docker compose up`。配置文件支持 Jinja2 模板渲染，用于注入宿主机路径等运行时信息。

支持应用的导入 / 导出（JSON 格式，含全部配置文件），便于迁移和备份。

### 应用状态机

```
undeployed ──部署──► deploying ──成功──► deployed
                         │                  │
                    失败/取消           停止/停止失败
                         ▼                  ▼
                   deploy_failed       undeployed / deploy_failed

deployed ──重启──► deploying ──成功──► deployed
```

每次部署/停止/重启操作创建一条 Deployment 记录，异步执行（停止为同步）。Deployment 有独立状态机（waiting_to_run → running → 终态），终态结果回写到 Application 状态。

---

## 三、路由与证书

平台以 Traefik 作为反向代理网关，采用混合路由架构：

- File Provider：管理在平台中手动配置的路由，配置文件由平台写入 `dynamic/` 目录
- Docker Provider：自动发现带有 Traefik Label 的容器路由

每条路由独立生成一个配置文件，支持三种 HTTPS 模式：

| 模式 | 适用场景 |
|------|----------|
| HTTP | 内网或不需要加密的场景 |
| 手动证书 / mkcert | 内网域名、自签名、本地开发 |
| Let's Encrypt | 公网域名，Traefik 自动申请和续期 |

证书内容（手动/mkcert）存储在数据库，同步时恢复到磁盘。Let's Encrypt 证书由 Traefik 全权管理（acme.json）。

Windows 下 Traefik 文件监听跨文件系统失效，平台在写入路由配置后主动发送 SIGHUP 触发重载；Linux 依赖 fsnotify 自动重载。

预置两套 Traefik 应用供选择：bridge 网络模式（适用于 Docker Desktop）和 host 网络模式（适用于 Linux 原生 Docker）。

---

## 四、Webhook 自动部署

支持接收 GitHub Webhook（push / release / ping 事件），根据仓库地址和分支匹配应用，自动触发部署。所有事件均记录状态（matched / ignored / error），便于排查。

Webhook 鉴权使用全局 secret。

---

## 五、运行时配置

平台配置分两层：`config.defaults.yaml` 提供默认值，数据库存储运行时覆盖值。支持任意 key 的 CRUD，被覆盖的配置项在 UI 中有视觉标记。

---

## 六、DooD 路径注入

容器内路径与宿主机路径不同，卷挂载必须使用宿主机绝对路径。平台通过读取 `/proc/self/cgroup`（支持 cgroup v1/v2）获取自身容器 ID，再通过 `docker inspect` 反查 bind mount source，得到宿主机数据目录路径，在 Jinja2 渲染时注入配置文件。

---

## 七、运维脚本

`scripts/manage.py` 提供本地运维能力：

- `deploy`：通过 rsync + SSH 将配置同步到远程服务器并执行部署
- `backup`：备份远程数据目录
- `tunnel`：建立 SSH 隧道（ControlMaster，支持多端口转发）

---

## 八、待完成事项

- Traefik Labels Helper UI（辅助生成 docker-compose labels）
- docker-compose.yml 表单化编辑

---

## 附：自 0320 以来的变更

- 应用与部署状态机统一，集中管理状态转换规则
- 移除集中式 `tls.yml`，证书配置内联到各路由文件
- cgroup v2 容器 ID 检测支持
- 新增应用导入 / 导出（含预览弹窗）
- 运行时配置支持开放式 key 的 CRUD，被覆盖项有视觉标记
- 移除 credential 功能，Webhook 简化为全局 secret
- 登录拦截未激活 / 未验证邮箱账户，注册强制密码强度校验
- 应用卡片重新设计，增加状态指示器和快捷操作按钮
- `scripts/manage.py` 整合 deploy / backup / tunnel
- 新增 `Dockerfile.cn` 国内镜像构建

