# 运行时配置与 OAuth 配置指南

本文说明前端运行时配置、后端环境变量和 Google OAuth 回调地址的职责边界。

## 配置分层

后端配置由 Dynaconf 加载，优先级从低到高：

```text
backend/config.defaults.yaml -> backend/config.yaml -> backend/.env -> 系统环境变量
```

后端环境变量统一使用 `POMELO_ORBIT_` 前缀，例如：

```env
POMELO_ORBIT_GOOGLE__CLIENT_ID=...
POMELO_ORBIT_GOOGLE__CLIENT_SECRET=...
POMELO_ORBIT_GOOGLE__REDIRECT_URI=http://localhost:9002/google/callback
```

前端有两类配置：

```text
VITE_*                 构建期配置，打包时写入 bundle
window.__CONFIG__      运行时配置，由 Nginx/entrypoint 在加载应用前注入
```

运行时配置优先于 `VITE_*`。不要把密钥放进前端配置，`window.__CONFIG__` 对浏览器用户完全可见。

## API Base URL 约定

默认部署模型是同源 API：

```text
前端页面: http://localhost:9002
API 路径: /api/...
```

开发环境由 Vite proxy 把 `/api` 转发到后端 `http://127.0.0.1:9001`；生产镜像由 FastAPI 同源提供静态前端和 `/api`。

远程开发模式也使用同一个 proxy。如果本机存在 SSH 隧道：

```text
ssh -L 9001:localhost:9001 ...
```

那么 `http://localhost:9002/api/...` 实际会进入远程 `9001`。此时前端本地改动不会改变远程后端行为，需要同时部署或更新远程后端。

`apiBaseUrl` 表示 API 路径前缀，不是 API 根路径本身：

| 配置值 | 含义 | 结果示例 |
| --- | --- | --- |
| 空字符串 | 同源 | `/api/auth/login` |
| `/` | 同源 | `/api/auth/login` |
| `https://api.example.com` | 跨域 API | `https://api.example.com/api/auth/login` |
| `/backend` | 同源路径前缀 | `/backend/api/auth/login` |

不要配置成 `/api`，因为业务 API 路径已经包含 `/api`，会变成 `/api/api/...`。

推荐开发环境保持：

```env
VITE_API_BASE_URL=
```

## Nginx 运行时注入

`frontend/index.html` 里保留了注入点：

```html
<!-- __RUNTIME_CONFIG__ -->
<script>
  window.__CONFIG__ = window.__CONFIG__ || {};
</script>
```

Nginx 镜像或 entrypoint 可以在启动时替换该标记。示例：

```sh
#!/bin/sh
set -eu

API_BASE_URL="${POMELO_ORBIT_PUBLIC_API_BASE_URL:-}"

RUNTIME_CONFIG="<script>window.__CONFIG__ = { apiBaseUrl: \"${API_BASE_URL}\" };</script>"
sed -i "s|<!-- __RUNTIME_CONFIG__ -->|${RUNTIME_CONFIG}|" /usr/share/nginx/html/index.html

exec nginx -g "daemon off;"
```

如果前端和后端同源部署，不需要注入 `apiBaseUrl`。如果 API 独立部署，注入完整 origin：

```env
POMELO_ORBIT_PUBLIC_API_BASE_URL=https://api.example.com
```

同源部署时，Nginx 必须先代理 API，再把其他路径交给 SPA fallback：

```nginx
server {
    listen 80;
    server_name _;
    root /usr/share/nginx/html;

    location /api/ {
        proxy_pass http://backend:9001/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /hooks/ {
        proxy_pass http://backend:9001/hooks/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

不要让 `/api/...` 落到 `try_files ... /index.html`。否则 `/api/auth/google` 会被当成前端页面返回，浏览器随后加载 `/assets/index-*.js`，表现为白页或静态资源 404。

## Google OAuth

Google 登录起跳是浏览器导航，不是 Axios 请求：

```text
window.location.assign(buildApiUrl('/api/auth/google'))
```

同源开发环境下会访问：

```text
http://localhost:9002/api/auth/google
```

再由 Vite proxy 转发到后端：

```text
http://127.0.0.1:9001/api/auth/google
```

后端 `/api/auth/google` 生成 Google 授权 URL，并使用后端配置的 `POMELO_ORBIT_GOOGLE__REDIRECT_URI` 作为 Google 回调地址。

开发环境应配置：

```env
POMELO_ORBIT_GOOGLE__REDIRECT_URI=http://localhost:9002/google/callback
```

该地址必须和 Google Cloud Console 里的 Authorized redirect URI 完全一致。端口使用前端端口 `9002`，不是后端端口，也不是旧的 `9006`。

跨域 API 部署时还需要：

- `window.__CONFIG__.apiBaseUrl` 指向 API origin。
- 后端 CORS 允许前端 origin。
- Google Console 的 redirect URI 仍然配置为前端回调页，例如 `https://app.example.com/google/callback`。

## 配置原则

- 后端密钥只进后端配置：JWT secret、Google client secret、数据库凭据。
- 前端运行时配置只放公开值：API origin、环境标签、公开 feature flag。
- OAuth redirect URI 是后端 OAuth 配置，不从前端 API base URL 推导。
- 默认走同源 `/api`，只有跨域部署才配置 `apiBaseUrl`。
