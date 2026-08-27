# CD 产品模型
最后修改时间: 2026-08-26 23:01:37

Doc role: living product model

## 核心对象

| 对象 | 职责 |
| --- | --- |
| Application | 通用应用元数据和项目归属 |
| Version / Component | 可编辑、可 fork 的 Compose 拓扑声明 |
| Service | Version 的运行实例和运行时覆盖 |
| Deployment | 异步部署任务及其不可变执行输入 |
| GatewayConfig | 一个 Gateway Application 的业务属性和 ACME profile 选择 |
| Route | 自定义 HTTP/TCP 入口和受管 target |

## Gateway

Gateway 是普通 Application。创建时生成 base、HTTP-01、DNS-01、HTTP+DNS 四个普通 Version 和一个停止态 default Service。它们与任何普通 Version/Component 一样可以查看、编辑、fork 和部署。

GatewayConfig 保存控制面、base domain、Component label 默认策略、ACME profile、email 和 DNS token。它不保存镜像、mount、endpoint、TCP listener 或 resolver 布局。profile 选择对应的绑定 Version；空 profile 使用 base Version。

| 能力 | 来源 |
| --- | --- |
| Traefik 端口、socket、cert/acme mount、resolver YAML | Gateway Version / Component |
| Component Docker label 默认 entrypoint/TLS | GatewayConfig |
| HTTP-01 / DNS-01 可用性 | GatewayConfig profile，DNS 另需 Gateway token |
| TCP Route listener | 选中 Gateway Version endpoint |

DNS token 是 Gateway 属性，直接返回、快照并作为 `CF_DNS_API_TOKEN` 写入 DNS profile 的 Compose environment。本期不提供全局 token、secret workspace、Compose secret 或脱敏接口。
