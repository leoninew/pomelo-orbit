# 密钥管理

实例主密钥与外部服务凭据分别保管，恢复数据库时同时恢复匹配的密钥。

| 值 | 用途 |
| --- | --- |
| `jwt__secret_key` | Web JWT 签名，以及仓库凭据、环境 SSH 私钥、PAT 可恢复原值、Google 临时会话的加密 |
| `mcp__access_token` | 本地 MCP / CLI 使用的个人 PAT |
| `llm__api_key` | 模型供应商认证 |
| 仓库 / SSH / Cloudflare 凭据 | 访问对应外部资源 |

## 实例主密钥

`POMELO_ORBIT_JWT__SECRET_KEY` 必须为 URL-safe Base64 编码的 32 字节随机 Fernet 密钥。只在新实例创建时生成一次，后续重启与恢复使用原值。

可使用任意可信的密码学随机工具生成。以下是两种等价方法，按本机已有工具任选一种；Python 命令仅用于生成密钥，Python 不是平台的安装或运行依赖。

已有 OpenSSL 时：

```bash
openssl rand -base64 32 | tr '+/' '-_'
```

已有 Python 3 时，仅使用标准库：

```bash
python3 -c "import base64, os; print(base64.urlsafe_b64encode(os.urandom(32)).decode())"
```

将生成的值填入[安装配置](../getting-started/installation.md)或开发实例的启动配置，按秘密保管，不提交到源码仓库。

替换密钥会使旧 JWT 验签失败，已有加密凭据也无法正常解密。当前没有自动重加密旧数据的通用轮换流程。

PAT 认证使用哈希，更换主密钥不会统一撤销已持有的 PAT；到[访问令牌](../guides/access-tokens.md)显式撤销。

## 保管与恢复

- 启动 dotenv、进程环境、`overwrite.env` 及备份可能保存明文秘密。
- 设置页的遮罩仅控制显示，有 `setting:read` 的用户仍可读取原值。
- `settings.secret_keys` 决定页面遮罩项，修改它不会加密原始配置。
- 配置写入恢复材料在启动目录的 `data/config/`，同样需要访问控制和备份。

发生解密错误时，先核对主密钥、数据库、profile 和配置根目录是否匹配。按[备份与恢复](./backup.md)保全原数据后处理。
