# 应用管理易用性分析

日期：2026-06-26

## 背景

当前应用管理已经具备比较完整的底层能力：应用元信息、配置文件存储、导入导出、Docker Compose 部署、服务镜像覆盖、路由托管、部署日志等。但从用户视角看，创建一个可部署应用仍然需要理解并手写大量 Docker Compose 相关配置，易用性不足。

本分析基于当前项目实现，重点审视“为什么用户要费很大功夫才能配置出可部署应用”，并提出改进方向。本文不涉及代码改动方案。

## 当前应用管理的真实模型

当前产品表面叫“应用管理”，但实际模型更接近“部署文件包管理”：一个应用由应用元信息和一组配置文件组成，部署时将配置文件写入工作目录，然后执行 Docker Compose。

### 1. 新建应用只创建元信息

前端新建表单只包含：

- 应用名称
- 应用代码
- 镜像拉取策略
- 路由管理开关

参考：

- `frontend/src/components/ApplicationFormFields.vue`
- `frontend/src/views/cd/ApplicationPage.vue`
- `backend-go/internal/service/cd/service.go`

后端创建应用时只创建 `application` 记录，不自动创建任何可部署配置文件。因此，新建出来的应用默认是“空壳”，还不能部署。

### 2. 用户需要手动添加配置文件

应用详情页提供“配置文件”区域，用户可以添加任意路径和文本内容。

典型配置文件包括：

- `docker-compose.yml`
- `docker-compose.yml.liquid`
- `.env`
- `init.sh`
- 其他被 compose 引用的配置文件

部署前必须至少存在 `docker-compose.yml` 或 `docker-compose.yml.liquid`。否则后端会拒绝部署。

参考：

- `frontend/src/views/cd/ApplicationDetail.vue`
- `backend-go/internal/service/cd/service.go`

### 3. 部署时写入文件并执行 Docker Compose

部署执行流程如下：

1. 读取应用的所有配置文件。
2. 将每个配置文件写入应用工作目录。
3. 如果文件以 `.liquid` 结尾，先进行模板渲染，并去掉 `.liquid` 后缀写出。
4. 如果存在 `init.sh`，部署前执行它。
5. 执行 `docker compose -f docker-compose.yml up -d --remove-orphans --pull <image_pull_policy>`。

参考：

- `backend-go/internal/service/cd/deployment_execution.go`
- `backend-go/internal/service/cd/workspace.go`

### 4. 路由管理依赖 compose 解析

添加路由时，前端会先请求后端解析 compose services。后端读取 compose 的 `services` 节点，返回服务名、默认域名、默认端口。

如果启用路由托管，部署时平台会将应用路由注入为 Traefik labels。

参考：

- `frontend/src/views/cd/ApplicationDetail.vue`
- `backend-go/internal/service/cd/application_extra.go`
- `backend-go/internal/service/cd/deployment_execution.go`

### 5. 镜像配置也是从 compose 派生出来的

应用详情页的“镜像配置”不是独立应用模型，而是解析 compose service 后，允许用户对 service 的 image 进行覆盖。

部署时平台会应用镜像覆盖，生成最终写入的 compose 文件。

参考：

- `frontend/src/views/cd/ApplicationDetail.vue`
- `backend-go/internal/service/cd/application_extra.go`

## 主要易用性断点

### 断点一：新建应用不是创建可部署应用

用户点击“新建应用”后，直觉上会认为自己已经创建了一个可以部署的应用。但当前系统只创建了应用元信息，仍缺少关键部署配置。

用户还需要知道：

- 必须添加 `docker-compose.yml` 或 `docker-compose.yml.liquid`
- `.env` 只是可选辅助文件
- `init.sh` 会在部署前执行
- 路由托管依赖 compose services
- 镜像配置依赖 compose service 解析

这导致“新建应用”与“可部署应用”之间存在很大的隐性距离。

### 断点二：配置文件是核心能力，但 UI 缺少语义

当前 UI 将所有配置文件都当成“路径 + 文本内容”。但不同文件有不同语义：

| 文件 | 用户真正关心的问题 |
| --- | --- |
| `docker-compose.yml` | services 是否有效？镜像、端口、网络、卷是否能部署？ |
| `.env` | 是 compose 变量插值？是否注入容器？是否被 compose 引用？ |
| `init.sh` | 何时执行？失败会怎样？是否幂等？ |
| `*.liquid` | 可用变量有哪些？渲染后是什么？ |

现在虽然有 compose 预览，但主要展示平台渲染、镜像覆盖和路由注入后的 YAML，并没有完整表达 Docker Compose 最终解析结果，也没有对 `.env`、`init.sh` 等文件给出专门解释。

### 断点三：`.env` 行为对用户是隐藏规则

`.env` 会被写入应用工作目录。Docker Compose 默认会读取工作目录下的 `.env`，用于 compose 文件变量插值。

但 `.env` 不会自动变成容器环境变量。若希望容器内部能读取这些变量，compose 中必须显式配置：

```yaml
services:
  app:
    env_file:
      - .env
```

或：

```yaml
services:
  app:
    environment:
      SOME_KEY: ${SOME_KEY}
```

当前 UI 没有将这条规则显式告诉用户。用户添加 `.env` 后，很容易误以为容器内自动拥有这些环境变量。

### 断点四：路由托管有隐藏副作用

启用路由托管后，部署时平台会注入 Traefik labels。当前注入逻辑会清理 service 上已有 labels，再写入平台托管的 labels。

这意味着：如果用户原本在 compose 中手写了自定义 labels，启用平台路由托管后可能被覆盖。

这并不一定是错误设计，但它必须在产品上被明确表达：启用路由托管意味着平台接管该应用服务的路由 labels。

### 断点五：部署失败主要靠用户读日志

当前部署详情页已经具备日志展示和流式更新能力。但日志本身仍然是 Docker/Compose 原始输出，普通用户需要自己判断：

- 是镜像拉取失败？
- 是端口占用？
- 是网络不存在？
- 是 volume 路径问题？
- 是 compose 变量未设置？
- 是 `init.sh` 失败？

这对有 Docker 经验的用户可接受，但对应用管理产品来说门槛偏高。

## 当前实现中影响易用性的契约问题

以下问题不是长期产品优化，而是当前实现中可能直接造成用户困惑的点。

### 1. 新建应用的路由管理开关可能不生效

前端创建应用时发送了 `route_managed`，但 backend-go 的创建请求结构中没有该字段，服务层创建应用时也没有设置该字段。

参考：

- `frontend/src/views/cd/ApplicationPage.vue`
- `backend-go/internal/transport/http/handler/cd/handler.go`
- `backend-go/internal/service/cd/service.go`

结果是：用户在新建时勾选“启用路由管理”，后端可能忽略该设置。

### 2. 删除应用时“同时删除工作目录”前后端不一致

前端删除应用时将 `remove_dir` 放在请求 body 中，而后端读取的是 query 参数。

参考：

- `frontend/src/api/cd/application.ts`
- `backend-go/internal/transport/http/handler/cd/handler.go`

结果是：用户勾选“同时删除应用工作目录”可能不会生效。

### 3. UI 文案中的工作目录路径与 backend-go 实际目录不一致

UI 文案中出现 `data/apps/{code}`，但 backend-go 实际应用工作目录是 `DataRoot/cd/{code}`。

参考：

- `frontend/src/i18n/locales/zh-CN.ts`
- `backend-go/internal/service/cd/workspace.go`

### 4. 服务镜像覆盖重置的前后端行为不一致

前端重置服务镜像覆盖时发送 `image: null`，UI 文案也提示留空会移除覆盖。但后端当前逻辑会将空值视为错误。

参考：

- `frontend/src/views/cd/ApplicationDetail.vue`
- `frontend/src/i18n/locales/zh-CN.ts`
- `backend-go/internal/service/cd/application_extra.go`

这会导致用户无法按 UI 提示完成重置。

## 改进方向

### P0：先做可部署性可见化和契约修复

这是最高优先级。目标不是重构模型，而是让用户知道当前应用是否已经可部署、不可部署时差什么，并修复现有前后端契约不一致。

建议在应用详情顶部增加“部署准备状态”区域：

- 基本信息是否完整
- 是否存在 `docker-compose.yml` 或 `docker-compose.yml.liquid`
- compose 是否可解析
- 是否发现 services
- `.env` 是否存在
- `.env` 是否被 `${VAR}` 或 `env_file` 使用
- 是否启用路由托管
- 启用路由托管后是否已配置路由
- route 指向的 service 是否存在
- 是否存在 `init.sh`
- 配置文件改动是否需要重新部署

同时修复：

- 创建应用时 `route_managed` 生效
- 删除应用时 `remove_dir` 前后端一致
- 工作目录文案与 backend-go 实际路径一致
- 服务镜像覆盖支持真正重置

这一阶段收益最大：不改变底层架构，但能显著降低用户迷路、误解和不信任。

### P1：把新建应用改成创建方式选择

当前“新建应用”只适合高级用户。建议将入口调整为“创建应用”，第一步选择创建方式：

1. 从镜像创建
2. 从 docker-compose 创建
3. 导入应用包
4. 高级空白应用

#### 从镜像创建

适合最常见的单服务应用。

用户输入：

- 应用名称
- 应用代码
- 镜像
- 容器端口
- 可选环境变量
- 可选数据目录
- 可选域名/路由

系统生成：

- `docker-compose.yml`
- 可选 `.env`
- 可选路由配置

#### 从 docker-compose 创建

适合已有 compose 的用户。

用户粘贴或上传 compose 后，系统：

- 解析 services
- 展示镜像、端口、环境变量、卷、网络摘要
- 引导配置 `.env`
- 引导配置路由
- 保存为配置文件

#### 导入应用包

保留当前 JSON 导入能力，但增加导入预览和冲突检查。

#### 高级空白应用

保留当前自由文件模式，面向熟悉 Docker Compose 的用户。

关键原则：不要废掉当前配置文件模型。向导层只负责生成配置文件，底层仍使用现有配置文件存储和部署执行链路。

### P2：增加文件类型专用编辑体验

当前所有配置文件基本都用同一种编辑体验。建议按文件类型增强：

#### `.env` 专用体验

- 表格编辑 key/value
- 支持源码模式
- 检测重复 key
- 检测空 key 或非法 key
- 检测未被 compose 使用的变量
- 提醒 `.env` 默认只参与 compose 插值，不自动注入容器

#### `docker-compose.yml` 专用体验

- YAML 编辑
- 保存前解析
- 展示 services、images、ports、volumes、networks 摘要
- 显示平台渲染后的 compose
- 进一步支持 `docker compose config` 级别的有效配置预览

#### `init.sh` 专用体验

- shell 高亮
- 明确提示部署前执行
- 明确提示失败会中断部署
- 提醒脚本应保持幂等
- 自动规范 CRLF 为 LF 的行为应可见化

### P3：模板库，但先不要做复杂应用市场

项目初始化数据中已经内置了 Traefik 和 FileBrowser 示例配置，这说明项目本身已经有模板雏形。

建议将其产品化为“内置模板”：

- Traefik
- FileBrowser
- Nginx 静态站点
- PostgreSQL
- Redis
- MinIO
- Gitea 等

模板提供变量表单，例如：

- 域名
- 镜像 tag
- 容器端口
- 数据目录
- 管理员账号
- 初始密码

创建后仍然生成普通应用配置文件，用户可以继续编辑底层 compose 和 `.env`。

不建议第一步做复杂的模板市场、远程模板仓库、模板版本管理。这些可以后续再考虑。

### P4：部署前诊断和失败解释

#### 部署前诊断

点击部署前可以先进行预检：

- 是否有 compose 文件
- compose 是否可解析
- 是否发现 services
- route 指向的 service 是否存在
- route 端口是否合理
- `.env` 是否存在语法问题
- compose 中引用的变量是否都已提供
- `init.sh` 是否存在明显问题
- volume 宿主路径是否可能不可写
- 网络是否存在或是否会创建

#### 部署失败解释

部署失败后，除了展示日志，增加“可能原因”摘要：

- 镜像拉取失败
- 端口占用
- Docker 网络不存在
- volume 路径不存在或无权限
- compose 变量未设置
- `init.sh` 执行失败
- 容器启动后立即退出

第一版不需要 AI 分析，可以先用常见错误文本规则分类。

## 不建议的方向

### 不建议马上做完整 Compose 表单化

Docker Compose 能力很大，包括：

- networks
- volumes
- healthcheck
- depends_on
- profiles
- secrets
- configs
- build
- deploy
- labels
- anchors

如果试图全量表单化，很容易变成不完整且难维护的低配 Compose UI。

更好的边界是：

- 常见场景走向导和模板
- 复杂场景保留源码编辑
- 平台负责校验、预览、解释和生成

### 不建议第一步做 AI 生成 compose

AI 可以作为未来增强，但当前最缺的是确定性产品路径：

- 哪些状态可部署
- 哪些配置会生效
- 哪些改动需要重新部署
- `.env`、路由、`init.sh` 的规则是什么

这些不先补齐，AI 生成只会放大不可解释性。

### 不建议现在引入新的应用规格模型替代配置文件

可以想象未来存在 `ApplicationSpec`，再由系统生成 Compose。但当前已有配置文件存储、导入导出、预览和部署执行链路，直接替换成本偏高。

更现实的策略是：在现有配置文件模型上增加向导、模板、预检和解释能力。

## 推荐路线

### 第一阶段：修信任 + 可见性

目标：让用户知道应用是否可部署，并修复明显契约问题。

范围：

- 修复前后端契约问题
- 应用详情增加部署准备状态
- 配置文件按类型显示说明
- 部署按钮前置预检
- 明确“配置文件改动需要重新部署，重启不会重新应用”

### 第二阶段：首次部署向导

目标：解决从零创建应用太难的问题。

范围：

- 新建应用入口改成创建方式选择
- 支持从镜像创建单服务应用
- 支持粘贴 docker-compose 创建应用
- 自动生成 compose / `.env`
- 创建后进入部署准备状态页

### 第三阶段：模板化

目标：沉淀常见应用部署经验。

范围：

- 将内置示例配置抽象为模板
- 模板变量表单生成普通应用
- 支持用户从模板快速创建应用
- 暂不做模板市场和复杂版本管理

## 总结

当前应用管理的问题不是缺少底层功能，而是产品抽象层太低。系统已经有足够灵活的 Docker Compose 部署能力，但用户被迫直接面对文件、路径、模板、环境变量、路由 labels 和 Docker 错误日志。

更合适的方向不是把 Pomelo Orbit 做成 Docker Compose 的低配 UI，而是：

> 把常见应用部署路径产品化，同时保留 Docker Compose 作为高级逃生口。

短期应先解决可部署性可见化和契约一致性；中期再做创建向导；长期沉淀模板库。这样既符合项目当前架构，也能明显降低用户从零部署应用的成本。
