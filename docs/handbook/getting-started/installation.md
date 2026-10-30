# 自托管安装

本文提供单机试用的安装路径：在独立 Linux Docker 主机上运行 Pomelo Orbit，使用 SQLite 保存平台数据，先仅允许宿主机本地访问，再完成首次登录。业务容器与平台实例不是同一回事，安装后还需配置项目执行环境。

本手册跟随 `develop` 分支。安装前记录所用提交或发布版本；不要把本文视为任意历史版本的安装保证，也不要直接用于已有生产数据库。

## 选择安装来源

- **已有可信发布包：** 可使用与目标系统匹配的完整包，见本文末尾“使用已取得的程序包”。包应来自项目维护者的可信交付渠道。
- **没有可用发布包：** 使用下方源码构建镜像路径。构建过程在 Docker 内完成，不要求在宿主机搭建 Go、Node 或前端开发环境。
- [GitHub Releases](https://github.com/leoninew/pomelo-orbit/releases)是发布资产入口；没有对应版本的资产时，不要把 GitHub 自动提供的源码压缩包当作运行包。镜像仓库中是否存在某标签、是否允许拉取，需另行确认，不能仅凭仓库名称推断。

本文主路径不依赖预构建公共镜像。构建失败时保留所用提交和脱敏错误，先解决构建问题，不换成来源不明的镜像。

## 安装前准备

- 一台隔离的 Linux 主机，已安装可用的 Docker Engine、`docker compose`、Git 和 Python 3；执行用户有 Docker 权限以及安装目录的读写权限。
- 能访问源码仓库、Dockerfile 所需基础镜像和构建依赖源。为镜像、日志、构建文件和业务数据预留磁盘空间。
- 宿主机 `9023` 端口未被占用。下面只绑定 `127.0.0.1`；首次改密前不要公开入口。
- 一个**新的、无现有数据**的安装目录。已有实例应先阅读[数据保留与升级](../operations/data-upgrades.md)，不要套用首次安装步骤重建密钥或覆盖配置。

此路径把宿主机 Docker socket 交给 Orbit，意味着平台及有执行权限的用户能控制该 Docker 主机，接近宿主机管理员权限。不要将它当作不可信租户的安全沙盒；不要通过公开 Docker TCP 端口解决连接问题。

## 1. 获取源码并构建安装镜像

在选定的源码父目录执行：

```sh
git clone --branch develop --single-branch https://github.com/leoninew/pomelo-orbit.git
cd pomelo-orbit
git rev-parse HEAD
docker build -t pomelo-orbit:local .
```

记录输出的提交编号和本地镜像 ID。`develop` 会变化；以后升级应使用独立版本标签并保留旧镜像，不把重复使用 `local` 标签当成版本记录。这里的构建只产生镜像，不安装数据库或启动实例。

## 2. 准备运行目录和稳定密钥

离开源码目录，在自己选定的安装父目录创建新的 `orbit-instance`。以下命令中的 `mkdir` 若提示目录已存在，先停止并确认是否为已有实例，不覆盖其中的文件：

```sh
umask 077
mkdir orbit-instance && cd orbit-instance && mkdir data logs && touch overwrite.env
```

在这个目录创建 `.env`，仅包含本次新实例所需的启动秘密。下面命令会以独占创建方式写文件，已有 `.env` 时会拒绝；不输出密钥：

```sh
python3 -c "import base64,os; key=base64.urlsafe_b64encode(os.urandom(32)).decode(); f=open('.env','x'); f.write('POMELO_ORBIT_JWT__SECRET_KEY='+key+'\n'); f.close()" && chmod 600 .env overwrite.env
```

只在全新实例生成一次，并安全备份 `.env`。这把密钥不仅签名登录令牌，还用于解密平台保存的凭据；后续重启、容器重建、恢复数据库时必须使用匹配的原密钥，不要重新运行生成步骤。不要把运行目录加入 Git 或分享其内容。

## 3. 创建 Compose 配置

在 `orbit-instance` 中创建 `compose.yaml`：

```yaml
services:
  orbit:
    image: pomelo-orbit:local
    restart: unless-stopped
    ports:
      - "127.0.0.1:9023:80"
    env_file:
      - .env
    environment:
      POMELO_ORBIT_SERVER__HOST: "0.0.0.0"
      POMELO_ORBIT_SERVER__PORT: "80"
      POMELO_ORBIT_DATABASE__URL: "sqlite:///data/db/pomelo-orbit.db"
      POMELO_ORBIT_WORKSPACE__ROOT: "/app/data"
      POMELO_ORBIT_LOGGING__DEPLOYMENT_ROOT: "/app/data/deployment-logs"
      POMELO_ORBIT_PROJECT_INITIALIZATION__ENVIRONMENT__LOCAL_WORKSPACE_ROOT: "/app/data/projects/first-example"
      POMELO_ORBIT_TURNSTILE__ENABLED: "false"
    volumes:
      - type: bind
        source: ./data
        target: /app/data
      - type: bind
        source: ./logs
        target: /app/logs
      - type: bind
        source: ./overwrite.env
        target: /app/overwrite.env
      - type: bind
        source: /var/run/docker.sock
        target: /var/run/docker.sock
```

各挂载的作用：

| 宿主机内容 | 容器内用途 | 保留要求 |
| --- | --- | --- |
| `data/` | SQLite 数据库、部署日志、配置恢复记录及放在其下的项目工作区 | 整体保留，不能只保留数据库文件 |
| `logs/` | 平台运行日志 | 按日志保留策略管理，反馈前脱敏 |
| `overwrite.env` | 系统设置保存的应用覆盖 | 必须预先是可写文件，不是目录；与 `data/config/` 一并保留 |
| Docker socket | 查询当前容器挂载并管理宿主机 Docker | 高权限入口，仅用于可信主机与可信使用者 |

不设置自定义 `hostname`，也不把本机 Docker socket 换成另一台主机的 Docker daemon：容器运行时需要查询自身挂载来映射路径。平台工作区和部署日志目录必须落在可映射的挂载内；项目初始化时也应使用 `/app/data/` 下独立目录，而不是容器内未挂载的家目录。

`overwrite.env` 的优先级高于 Compose 注入的环境。空文件适用于首次安装；以后修改 Compose 环境不能保证覆盖已保存的系统设置，详见[配置与密钥](../operations/configuration.md)。

此示例关闭 Turnstile，仅用于回环入口的隔离试用；没有配置 LLM、Google 登录或公网 TLS。不要把试用配置直接改成公网监听来交付生产服务。

## 4. 启动并检查安装结果

确认当前目录是 `orbit-instance` 后执行：

```sh
docker compose up -d
docker compose ps
docker compose logs --tail=100 orbit
```

`serve` 默认同时启动 Web 和后台 Worker，启动时会自动执行数据库迁移。**首次启动不是只读检查**；不要为了“看看能否启动”把新版本指向生产数据库，也不需要为这个单进程试用另外启动 Worker。

按以下顺序检查：

1. 容器没有持续退出或重启，日志没有配置、挂载、密钥或数据库迁移错误。
2. 从 Docker 宿主机浏览器访问 `http://127.0.0.1:9023/`，能够加载登录页面及静态资源。
3. 在其他电脑访问时，通过你已有的安全管理通道转发宿主机回环端口；其他电脑的 `127.0.0.1` 不是服务器，不要为首登临时开放公网管理入口。
4. 完成下一节登录和改密，退出后用新密码重新登录。

页面可打开只证明 Web 入口可用；Docker 业务执行、镜像拉取、项目环境、网关与业务访问仍需分别检查。

## 5. 首次登录并立即改密

当前版本在**全新数据库完成初始迁移后**创建的初始管理员使用：

| 字段 | 初始值 |
| --- | --- |
| 邮箱 | `admin@lvh.me` |
| 密码 | `admin@lvh.me` |

这是公开的初始化凭据，不是生产秘密。它不适用于已经改密、从其他实例恢复、种子数据不同或管理员已调整的数据库；升级也不会把密码重置成这个值。

1. 在尚未公开的入口登录，注意登录字段是邮箱，不是 `admin` 用户名。
2. 打开顶部用户菜单，选择“修改密码”，填写旧密码和新的独立强密码并保存。
3. 退出并使用新密码重新登录，保留可恢复的管理员访问方式。
4. 需要多人使用时，创建各自账号、分配所需角色并加入项目，不共享初始管理员。详见[用户、权限与访问令牌](../guides/users-access.md)。

登录失败时先核对所用版本、迁移是否完成和数据库路径。不要删除数据库重新试密码，也不要修改已执行的迁移或直接覆盖用户密码哈希。

## 使用已取得的程序包

程序包是另一种入口，不需要与 Docker 主路径同时运行。当前打包形式为 Windows / Linux / macOS 的 x64 ZIP；镜像的架构支持不能推断为原生程序包也支持 ARM。

1. 在新的专用目录解压完整包，保留 `configs/`、`static/`、启动脚本及 `VERSION`，不能只取可执行文件。记录包版本和可信来源。
2. 编辑包根目录的 `.env.release`，为新实例填写一次性生成并保留的 `POMELO_ORBIT_JWT__SECRET_KEY`；已有实例使用原密钥。可用前述 Python 的随机生成方法，将结果安全写入该字段，不覆盖整个模板。
3. 首次试用将 `POMELO_ORBIT_SERVER__HOST` 改为 `127.0.0.1`，保留端口 `9023`，确认 Turnstile 关闭。包模板原本监听所有接口，不应原样用于公开首登。
4. Windows PowerShell 运行 `./pomelo-orbit.cmd start`；Linux / macOS 运行 `./pomelo-orbit.sh start`。启动器会切换到包根目录并选择 `release` profile，读取 `.env.release`，不是 `.env`。如果另有 `overwrite.env`，先核对其最高优先级覆盖。
5. 程序以前台方式运行，不自动安装系统服务；后续按你的进程管理方案管理生命周期。按前面的检查和首登步骤验收；需要本机部署时，另行准备程序执行用户可用的 Docker / Compose。

直接运行二进制不会替你选择 `release` profile 或切换工作目录。不要将启动器的 `status` 当作业务健康检查，也不要在仍有任务运行时使用 `restart`：停止流程可能在等待后强制结束进程。

## 安装失败时

| 症状 | 检查方向 |
| --- | --- |
| 缺少配置或页面空白 | 是否保留完整包，工作目录是否正确，是否错误挂载覆盖了 `/app/configs` 或 `/app/static` |
| 密钥为空或格式错误 | `.env` 是否被 Compose 注入、程序包是否读到 `.env.release`，是否被应用覆盖替换；不能用任意字符串代替 Fernet 密钥 |
| 工作区挂载或自身容器查询失败 | Docker socket、当前 Docker daemon、自定义 hostname、`/app/data` 挂载及路径配置 |
| SQLite 或覆盖文件无法写入 | 实际运行身份、目录 / 文件类型和权限，磁盘空间；不要通过删除数据库或放宽全部权限试探 |
| 端口已占用 | 先检查占用者，选择空闲宿主机端口并相应调整访问地址，不停止未知服务 |
| 迁移失败或 dirty 状态 | 保留日志与数据，由维护者核对版本和恢复方案，不手工跳过迁移或清除版本标记 |

安装完成后进入[首个应用部署](./first-deployment.md)；进一步排障见[故障定位](../operations/troubleshooting.md)。
