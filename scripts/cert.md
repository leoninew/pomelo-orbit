# `cert.py` — 本地开发证书工具

Doc role: local script reference。与代码冲突时以代码为准。

## 用途

使用 mkcert 生成可上传的合并 PEM 证书，或检查本地证书、mkcert 根 CA、Windows 信任库和实际 TLS 握手。

## 前置条件

- 已安装 `mkcert`；首次使用通常需要执行 `mkcert -install`。
- 已安装 uv；`cryptography` 由 `scripts/pyproject.toml` 管理。
- `check` 在 Windows 上使用 `certutil -store Root` 检查系统 Root 信任库。
- `check` 会连接 `<domain>:443` 执行系统信任库校验的 TLS 握手；目标域名必须可解析并提供 HTTPS 服务。

## 用法

```bash
# 生成合并 PEM（证书与私钥）
uv run --project scripts python scripts/cert.py new -n app.localhost --cert-dir /srv/orbit/cd/traefik/data/certs

# 检查 CA、PEM 内容、签名链与实际 TLS 握手
uv run --project scripts python scripts/cert.py check -n app.localhost --cert-dir /srv/orbit/cd/traefik/data/certs
```

`new` 的输出文件为：

```text
<cert-dir>/<domain>.pem
```

## 安全边界

`new` 会创建输出目录并覆盖同域名现有 PEM 文件。`check` 不修改证书文件，但会访问目标域名的 443 端口。

## 相关文档

证书上传、Traefik 存储和浏览器排障请参阅 [`../docs/guides/certificate-management.md`](../docs/guides/certificate-management.md)。
