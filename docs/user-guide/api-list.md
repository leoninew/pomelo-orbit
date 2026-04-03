# API 地址列表

## 认证 (`/api/auth`)

| 方法   | 路径                        | 说明             |
| ------ | --------------------------- | ---------------- |
| POST   | `/api/auth/login`           | 用户登录         |
| POST   | `/api/auth/logout`          | 用户登出         |
| GET    | `/api/auth/me`              | 获取当前用户信息 |
| PUT    | `/api/auth/password`        | 修改密码         |
| GET    | `/api/auth/login-history`   | 列出登录历史     |

## 持续集成 (`/api/ci`)

### Credentials

| 方法   | 路径                              | 说明               |
| ------ | --------------------------------- | ------------------ |
| GET    | `/api/ci/credentials`             | 列出所有凭据（分页）|
| POST   | `/api/ci/credentials`             | 创建凭据           |
| PUT    | `/api/ci/credentials/{credential_id}` | 更新凭据       |
| DELETE | `/api/ci/credentials/{credential_id}` | 删除凭据       |

### Pipeline Templates

| 方法   | 路径                                      | 说明                 |
| ------ | ----------------------------------------- | -------------------- |
| GET    | `/api/ci/templates`                       | 列出所有模板（分页） |
| POST   | `/api/ci/templates`                       | 创建流水线模板       |
| GET    | `/api/ci/templates/{template_id}`         | 获取模板详情         |
| PUT    | `/api/ci/templates/{template_id}`         | 更新模板             |
| DELETE | `/api/ci/templates/{template_id}`         | 删除模板             |
| GET    | `/api/ci/templates/{template_id}/snapshots` | 列出模板的快照     |

### Pipeline Snapshots

| 方法   | 路径                                      | 说明             |
| ------ | ----------------------------------------- | ---------------- |
| GET    | `/api/ci/snapshots/{snapshot_id}`         | 获取快照详情     |

### Projects

| 方法   | 路径                                      | 说明                     |
| ------ | ----------------------------------------- | ------------------------ |
| GET    | `/api/ci/projects`                        | 列出项目（分页）         |
| POST   | `/api/ci/projects`                        | 创建项目                 |
| GET    | `/api/ci/projects/{project_id}`           | 获取项目详情             |
| PUT    | `/api/ci/projects/{project_id}`           | 更新项目                 |
| DELETE | `/api/ci/projects/{project_id}`           | 删除项目                 |
| GET    | `/api/ci/projects/{project_id}/runs`      | 列出项目的 pipeline runs |
| POST   | `/api/ci/projects/{project_id}/trigger`   | 手动触发 pipeline        |

### Pipeline Runs

| 方法   | 路径                                      | 说明                         |
| ------ | ----------------------------------------- | ---------------------------- |
| GET    | `/api/ci/runs`                            | 列出所有 pipeline runs       |
| GET    | `/api/ci/runs/{run_id}`                   | 获取 pipeline run 详情       |
| GET    | `/api/ci/runs/{run_id}/artifacts`         | 列出 pipeline run 的所有制品 |
| GET    | `/api/ci/runs/{run_id}/jobs`              | 列出 pipeline run 的所有 jobs|
| POST   | `/api/ci/runs/{run_id}/cancel`            | 取消 pipeline run           |
| POST   | `/api/ci/runs/{run_id}/retry`             | 重试失败的 pipeline run     |

### Jobs

| 方法   | 路径                              | 说明           |
| ------ | --------------------------------- | -------------- |
| GET    | `/api/ci/jobs/{job_id}/logs`      | 获取 job 日志  |

### Webhooks（无需认证）

| 方法   | 路径                        | 说明                           |
| ------ | --------------------------- | ------------------------------ |
| POST   | `/api/ci/webhooks/git`      | 接收 Git webhook（GitHub / GitLab） |

## 持续部署 (`/api/cd`)

### 应用管理

| 方法   | 路径                                      | 说明                   |
| ------ | ----------------------------------------- | ---------------------- |
| GET    | `/api/cd/applications`                    | 列出所有应用           |
| POST   | `/api/cd/applications`                    | 创建应用               |
| POST   | `/api/cd/applications/import`             | 导入应用               |
| GET    | `/api/cd/applications/{app_id}`           | 获取应用详情           |
| GET    | `/api/cd/applications/{app_id}/export`    | 导出应用               |
| PUT    | `/api/cd/applications/{app_id}`           | 更新应用基本信息       |
| DELETE | `/api/cd/applications/{app_id}`           | 删除应用               |
| GET    | `/api/cd/applications/{app_id}/files`     | 获取应用配置文件列表   |
| POST   | `/api/cd/applications/{app_id}/file`      | 创建应用配置文件       |
| GET    | `/api/cd/applications/{app_id}/file/{file_id}` | 读取应用配置文件       |
| PUT    | `/api/cd/applications/{app_id}/file/{file_id}` | 写入应用配置文件       |
| DELETE | `/api/cd/applications/{app_id}/file/{file_id` | 删除应用配置文件       |
| POST   | `/api/cd/applications/{app_id}/deploy`    | 手动触发部署           |
| POST   | `/api/cd/applications/{app_id}/stop`      | 停止应用               |
| POST   | `/api/cd/applications/{app_id}/restart`   | 重启应用               |
| GET    | `/api/cd/applications/{app_id}/status`    | 获取应用运行状态       |
| GET    | `/api/cd/applications/{app_id}/logs`      | 获取应用日志           |

### 部署操作

| 方法   | 路径                                      | 说明                     |
| ------ | ----------------------------------------- | ------------------------ |
| GET    | `/api/cd/deployments`                     | 列出所有部署             |
| GET    | `/api/cd/deployments/{deployment_id}`     | 获取部署详情             |
| GET    | `/api/cd/deployments/{deployment_id}/logs`| 获取部署日志（增量读取） |
| POST   | `/api/cd/deployments/{deployment_id}/cancel` | 取消正在进行的部署    |

### 路由管理

| 方法   | 路径                                      | 说明                     |
| ------ | ----------------------------------------- | ------------------------ |
| GET    | `/api/cd/routes`                          | 列出所有路由             |
| POST   | `/api/cd/routes`                          | 创建路由                 |
| GET    | `/api/cd/routes/{route_id}`               | 获取路由详情             |
| PUT    | `/api/cd/routes/{route_id}`               | 更新路由                 |
| DELETE | `/api/cd/routes/{route_id}`               | 删除路由（要求路由已停用）|
| PUT    | `/api/cd/routes/{route_id}/enable`        | 启用路由                 |
| PUT    | `/api/cd/routes/{route_id}/disable`       | 停用路由                 |
| POST   | `/api/cd/routes/sync`                     | 同步路由配置             |
| POST   | `/api/cd/routes/{route_id}/cert`          | 上传 SSL 证书 (PEM 格式) |
| DELETE | `/api/cd/routes/{route_id}/https`         | 禁用 HTTPS               |
| POST   | `/api/cd/routes/{route_id}/letsencrypt`   | 启用 Let's Encrypt 自动证书 |
| POST   | `/api/cd/routes/{route_id}/mkcert`        | 启用 mkcert 本地证书     |

### Traefik 路由

| 方法   | 路径                              | 说明                 |
| ------ | --------------------------------- | -------------------- |
| GET    | `/api/cd/traefik-routes/config`   | 获取 Traefik 配置    |
| GET    | `/api/cd/traefik-routes`          | 列出所有 Traefik 路由|

## 系统配置 (`/api/settings`)

| 方法   | 路径                      | 说明                           |
| ------ | ------------------------- | ------------------------------ |
| GET    | `/api/settings/config`     | 获取运行时配置列表             |
| PUT    | `/api/settings/config`     | 新增或更新单个配置项           |
| DELETE | `/api/settings/config`     | 重置指定配置项为默认值         |
