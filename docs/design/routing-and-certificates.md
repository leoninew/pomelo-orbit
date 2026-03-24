# Pomelo Orbit 路由和证书原理

## 概述

Pomelo Orbit 使用 Traefik 作为反向代理网关，实现动态路由和 HTTPS 证书管理。路由配置和证书数据存储在数据库中，通过文件同步机制应用到 Traefik。

## 路由系统架构

```
用户配置路由（Web UI）
  ↓
保存到数据库（route 表）
  ↓
同步到文件系统（dynamic/*.yml）
  ↓
Traefik 自动加载（watch: true）
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
  "enabled": true,
  "https_enabled": true
}
```

### 配置说明

| 字段 | 说明 | 示例 |
|------|------|------|
| name | 路由名称 | my-app |
| domain | 域名 | app.example.com |
| path_prefix | 路径前缀 | / 或 /api |
| target_url | 目标 URL | http://host.docker.internal:8081 |
| enabled | 是否启用 | true/false |
| https_enabled | 是否启用 HTTPS | true/false |

## Traefik 配置格式

### HTTP 路由

```yaml
http:
  routers:
    my-app:
      rule: "Host(`app.example.com`) && PathPrefix(`/`)"
      service: my-app-service
      entryPoints:
        - web

  services:
    my-app-service:
      loadBalancer:
        servers:
          - url: "http://host.docker.internal:8081"
```

### HTTPS 路由

```yaml
http:
  routers:
    my-app-https:
      rule: "Host(`app.example.com`)"
      service: my-app-service
      entryPoints:
        - websecure
      tls:
        certResolver: default
```

## 证书管理

### 证书存储

证书内容存储在数据库的 `route` 表中：

- `cert_pem` - 证书内容（PEM 格式）
- `cert_key` - 证书私钥（PEM 格式）

### 证书同步流程

```
1. 用户上传证书（Web UI）
   ↓
2. 保存到数据库
   route.cert_pem  - 证书内容
   route.cert_key  - 私钥内容
   ↓
3. 同步到文件系统
   /app/data/applications/traefik/data/certs/
   ├── {domain}.crt
   └── {domain}.key
   ↓
4. 挂载到 Traefik 容器
   /certs/{domain}.crt
   /certs/{domain}.key
   ↓
5. Traefik 引用证书
   tls:
     certificates:
       - certFile: /certs/app.example.com.crt
         keyFile: /certs/app.example.com.key
```

### 证书文件命名

```
域名: app.example.com
证书文件: app.example.com.crt
私钥文件: app.example.com.key
```

## 动态配置更新

### 文件监听

Traefik 配置了文件监听，自动加载配置变化：

```yaml
providers:
  file:
    directory: /etc/traefik/dynamic
    watch: true  # 自动监听文件变化
```

### 更新流程

```
1. 用户修改路由配置
   ↓
2. Pomelo Orbit 更新 dynamic/routes.yml
   ↓
3. Traefik 检测到文件变化
   ↓
4. 自动重新加载配置
   ↓
5. 无需重启容器
```

## 路由规则

### Host 规则

匹配域名：

```yaml
rule: "Host(`app.example.com`)"
```

### PathPrefix 规则

匹配路径前缀：

```yaml
rule: "Host(`app.example.com`) && PathPrefix(`/api`)"
```

### 多域名

```yaml
rule: "Host(`app.example.com`) || Host(`www.app.example.com`)"
```

## 入口点配置

Traefik 静态配置定义了两个入口点：

```yaml
entryPoints:
  web:
    address: ":80"      # HTTP
  websecure:
    address: ":443"     # HTTPS
```

## HTTP 到 HTTPS 重定向

```yaml
http:
  routers:
    my-app-http:
      rule: "Host(`app.example.com`)"
      entryPoints:
        - web
      middlewares:
        - redirect-to-https

  middlewares:
    redirect-to-https:
      redirectScheme:
        scheme: https
        permanent: true
```

## 证书自动续期

### Let's Encrypt 集成

```yaml
certificatesResolvers:
  letsencrypt:
    acme:
      email: admin@example.com
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web
```

### 使用自动证书

```yaml
http:
  routers:
    my-app-https:
      rule: "Host(`app.example.com`)"
      entryPoints:
        - websecure
      tls:
        certResolver: letsencrypt
```

## 监控和日志

### Traefik Dashboard

访问地址：`http://localhost:8080/dashboard/`

可以查看：
- 路由列表
- 服务状态
- 中间件配置
- 实时请求

### 访问日志

```yaml
accessLog:
  filePath: /var/log/traefik/access.log
  format: json
```

## 故障排查

### 路由不生效

1. 检查 Traefik 是否运行：`docker ps | grep traefik`
2. 检查路由配置文件：`cat /app/data/applications/traefik/data/dynamic/routes.yml`
3. 检查 Traefik 日志：`docker logs traefik`
4. 检查域名解析：`nslookup app.example.com`

### 证书错误

1. 检查证书文件是否存在：`ls /app/data/applications/traefik/data/certs/`
2. 检查证书有效期：`openssl x509 -in cert.crt -noout -dates`
3. 检查证书和私钥匹配：
   ```bash
   openssl x509 -noout -modulus -in cert.crt | openssl md5
   openssl rsa -noout -modulus -in cert.key | openssl md5
   ```

### HTTPS 无法访问

1. 检查端口 443 是否开放
2. 检查防火墙规则
3. 检查证书配置
4. 检查 Traefik 入口点配置

## 安全最佳实践

### 1. 使用 HTTPS

生产环境所有路由应启用 HTTPS。

### 2. 证书私钥保护

- 证书私钥加密存储
- 限制文件系统访问权限
- 定期轮换证书

### 3. 添加安全头

```yaml
http:
  middlewares:
    security-headers:
      headers:
        customResponseHeaders:
          X-Frame-Options: "DENY"
          X-Content-Type-Options: "nosniff"
```

## 示例配置

### 完整的 HTTPS 路由

```yaml
http:
  routers:
    # HTTP 重定向到 HTTPS
    my-app-http:
      rule: "Host(`app.example.com`)"
      entryPoints:
        - web
      middlewares:
        - redirect-to-https
      service: my-app-service

    # HTTPS 路由
    my-app-https:
      rule: "Host(`app.example.com`)"
      entryPoints:
        - websecure
      service: my-app-service
      tls:
        certResolver: letsencrypt

  services:
    my-app-service:
      loadBalancer:
        servers:
          - url: "http://host.docker.internal:8081"

  middlewares:
    redirect-to-https:
      redirectScheme:
        scheme: https
        permanent: true

tls:
  certificates:
    - certFile: /certs/app.example.com.crt
      keyFile: /certs/app.example.com.key
```

