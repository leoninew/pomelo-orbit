# CD 部署原理（How-to / 概览）
最后修改时间: 2026-07-24 10:47:37

Doc role: living guide  
权威领域模型见 [CD 领域模型](../product/cd-model.md)、[CD 运行时](../architecture/cd-runtime.md)。与代码冲突时以代码为准。

## 概述

Pomelo Orbit 可在本机 Docker 上部署 **standard** 业务应用与 **gateway** 应用。规格在数据库中以 **Version / Component / Expose** 存储，部署时 **Render** 生成 compose 并执行。

## 核心理念

1. **规格即数据**：Version 结构化存储，不是「仅托管 compose 文件包」为唯一模型。  
2. **运行绑定是 Service**：`application + environment + instance_key` 绑定某一 Version。  
3. **Deployment 是操作流水**：API 入队，同进程 worker 执行 Docker。  
4. **Gateway 配置即接入 SoT**：`rest_api_url`、`base_domain`、TLS/入口；非全局 traefik env 产品路径。

## 部署流程（逻辑）

```text
选择 Version + Environment (+ instance_key，默认 default)
  → 创建/更新 Service 与 Deployment / Task
  → worker: Render(Version, kind) → 写入 CD workspace
  → docker compose up/down/…
  → 更新 Service / Deployment 状态与日志
```

### 状态（现行）

| 对象 | 状态 |
|------|------|
| Service | `deploying` / `running` / `stopped` / `faulted` |
| Deployment | `waiting_to_run` / `running` / `ran_to_completion` / `faulted` / `canceled` |

**不要**再使用已归档的 Application `undeployed/deployed` 状态机（`docs/archive/guides/application-state-machine.md`）。

## Gateway

- 产品面：`Application(kind=gateway)` + `gateway_config`  
- 与 standard 共用 Version / Deploy 管线  
- 平台 Route：`providers.rest` PUT 到 `rest_api_url`  
- 应用 public Host：`{app_code}.{gateway.base_domain}`  
- 同时仅一个 active gateway Service（deploying/running）

## 相关指南

- [路由与证书](./routing-and-certificates.md)  
- [Docker 部署](./docker-deployment.md)  
- [服务器部署](./server-deployment.md)  
