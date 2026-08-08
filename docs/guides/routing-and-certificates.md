# Pomelo Orbit 路由和证书原理
最后修改时间: 2026-08-08 13:29:04

Doc role: living guide  
领域总览见 [CD 模型](../product/cd-model.md)。与代码冲突时以代码为准。

## 概述

Pomelo Orbit 使用 Traefik 作为反向代理网关，实现动态路由和 HTTPS 证书管理。

- **平台路由**（CD Route 表）：经 Traefik **`providers.rest`** 全量 PUT，控制面 URL 来自 **Gateway config `rest_api_url`**。
- **应用暴露**（Version Expose）：部署时写入 Docker labels；每个 `gateway_http` endpoint 的 Host 为 `{component_name}.{service_code}.{gateway.base_domain}`，其中 Service code 在创建窗预填并提交后不可修改。
- **证书文件**：平台写入证书目录供 TLS 路由引用（见下文）。
- **接入配置 SoT**：`gateway_config`（非全局 `traefik.api_url` / `domain_suffix` 产品路径）。

## 路由系统架构

```
用户配置平台路由（Web UI）
  ↓
保存到数据库（route 表）
  ↓
ResolveActiveGatewayConfig → rest_api_url
  ↓
PUT {rest_api_url}/api/providers/rest  （全量快照）
  ↓
路由生效
```

## 路由配置

### 基本配置

在 Web UI 中配置路由：

```json
{
  "name": "my-app",
  "domain": "app.example.com",
  "path_prefix": "/",
  "target_url": "http://host.docker.internal:8081",
  "enabled": true
}
```

### 配置说明

| 字段 | 说明 | 示例 |
|------|------|------|
| name | 路由名称（同时作为配置文件名和证书文件名前缀） | my-app |
| domain | 域名 | app.example.com |
| path_prefix | 路径前缀 | / 或 /api |
| target_url | 目标 URL | http://host.docker.internal:8081 |
| enabled | 是否启用 | true/false |

## Traefik 配置格式

平台路由不再写 `dynamic/*.yml` 文件。`RouteManager` 组装与 providers.rest 契约一致的 JSON 全量配置后 PUT 到 Gateway 的管理面。网关静态配置（entryPoints / providers.rest / docker）由 **Gateway 保存时 compile** 写入 Version 挂载 `traefik.yml`（content_mode=sync）。

以下为概念形态（历史 file provider 文档示例；实现已 rest 化）：

### HTTP 路由（概念）

```yaml
http:
  routers:
    my-app-route:
      rule: "Host(`app.example.com`)"
      service: my-app-service
      entryPoints:
        - web
  services:
    my-app-service:
      loadBalancer:
        servers:
          - url: "http://host.docker.internal:8081"
```

### HTTPS 路由（手动证书 / mkcert）

```yaml
http:
  routers:
    my-app-route:
      rule: "Host(`app.example.com`)"
      service: my-app-service
      entryPoints:
        - websecure
      tls: {}
  services:
    my-app-service:
      loadBalancer:
        servers:
          - url: "http://host.docker.internal:8081"
tls:
  certificates:
    - certFile: /etc/traefik/certs/my-app.pem
      keyFile: /etc/traefik/certs/my-app-key.pem
```

### HTTPS 路由（Let's Encrypt）

```yaml
http:
  routers:
    my-app-route:
      rule: "Host(`app.example.com`)"
      service: my-app-service
      entryPoints:
        - websecure
      tls:
        certResolver: letsencrypt
  services:
    my-app-service:
      loadBalancer:
        servers:
          - url: "http://host.docker.internal:8081"
```

## 证书管理

### 证书存储

证书内容存储在数据库的 `route` 表中：

- `cert_pem` - 证书内容（PEM 格式）
- `cert_key` - 证书私钥（PEM 格式）
- `cert_type` - 证书类型：`manual` | `letsencrypt` | `mkcert`

### 证书文件命名

按路由名称命名（非域名）：

```
路由名: my-app
证书文件: my-app.pem
私钥文件: my-app-key.pem
```

### 证书同步流程（手动 / mkcert）

```
1. 用户上传证书或生成 mkcert 证书
   ↓
2. 保存到数据库（route.cert_pem / route.cert_key）
   ↓
3. 写入文件系统
   data/traefik/data/certs/
   ├── {route-name}.pem
   └── {route-name}-key.pem
   ↓
4. 挂载到 Traefik 容器
   /etc/traefik/certs/{route-name}.pem
   /etc/traefik/certs/{route-name}-key.pem
   ↓
5. 路由 yml 中直接引用证书路径
   tls:
     certificates:
       - certFile: /etc/traefik/certs/{route-name}.pem
         keyFile: /etc/traefik/certs/{route-name}-key.pem
```

### Let's Encrypt 流程

```
1. 用户启用 Let's Encrypt
   ↓
2. 路由 yml 写入 tls.certResolver: letsencrypt
   ↓
3. Traefik 自动向 Let's Encrypt 申请证书
   ↓
4. 证书存储在 acme.json（由 Traefik 管理）
```

### 组件级域名的证书覆盖

`{component_name}.{service_code}.{base_domain}` 含有两级子域名。因此 `*.{base_domain}` 不能匹配该地址。使用 TLS 时，部署者需要让 ACME 按实际 Host 申请证书，或提供覆盖精确 SAN / `*.{service_code}.{base_domain}` 的证书；同时 DNS 必须解析这些 Host。

## 动态配置更新

Traefik 配置了文件监听，自动加载配置变化：

```yaml
providers:
  file:
    directory: /etc/traefik/dynamic
    watch: true
```

在 Linux/macOS 上 Traefik 通过 inotify 自动检测文件变化；Windows 上需要发送 SIGHUP 信号触发重载（由 `TraefikManager._reload_traefik` 处理）。

## 路由规则

### Host 规则

```yaml
rule: "Host(`app.example.com`)"
```

### Host + PathPrefix 规则（path_prefix 不为 `/` 时）

```yaml
rule: "Host(`app.example.com`) && PathPrefix(`/api`)"
```

## 入口点配置

```yaml
entryPoints:
  web:
    address: ":80"      # HTTP
  websecure:
    address: ":443"     # HTTPS
```

## 证书类型对比

| 类型 | 适用场景 | 证书来源 | 文件管理 |
|------|----------|----------|----------|
| manual | 内网域名、自签名 | 用户上传 PEM | Pomelo Orbit 管理 |
| mkcert | 本地开发 | mkcert 生成 | Pomelo Orbit 管理 |
| letsencrypt | 公网域名 | Traefik 自动申请 | Traefik 管理（acme.json） |

## 故障排查

### 路由不生效

1. 检查 Traefik 是否运行：`docker ps | grep traefik`
2. 检查路由配置文件：`ls data/traefik/data/dynamic/`
3. 检查 Traefik 日志：`docker logs traefik`
4. 检查域名解析：`nslookup app.example.com`

### 证书错误

1. 检查证书文件是否存在：`ls data/traefik/data/certs/`
2. 检查证书有效期：`openssl x509 -in my-app.pem -noout -dates`
3. 检查证书和私钥匹配：
   ```bash
   openssl x509 -noout -modulus -in my-app.pem | openssl md5
   openssl rsa -noout -modulus -in my-app-key.pem | openssl md5
   ```
