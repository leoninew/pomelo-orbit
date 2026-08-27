# CD 部署原理
最后修改时间: 2026-08-26 23:01:37

Doc role: living guide。权威模型见 [CD 领域模型](../product/cd-model.md) 与 [CD 运行时](../architecture/cd-runtime.md)。

部署从 Service 选择的普通 Version 构建有效计划，渲染 Compose 后由 worker 执行 Docker：

```text
Service + Version + Component + runtime overlay
  -> Deployment snapshot
  -> effective plan
  -> docker compose
```

Gateway 仍使用该通道。它在创建 Deployment 前由 GatewayConfig 的 ACME profile 选择绑定 Version；Deployment 保存 Version ID 与 GatewayConfig snapshot。worker 只在该 Version 已声明的 Traefik static config 中写入 email，并为 DNS profile 注入 `CF_DNS_API_TOKEN`。它不会投影 listener、mount、endpoint 或 resolver。

TCP entrypoint/host port 是 Version Component endpoint。需要新的 TCP 端口时，在 Version 中声明、部署该 Version，再创建 TCP Route。
