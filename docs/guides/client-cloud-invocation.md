# 客户端与云端调用机制

最后修改时间: 2026-08-25

## 业务定位

Pomelo Orbit 是一个自托管的 CI/CD 与平台管理控制面。用户通过浏览器或 AI/MCP 客户端操作，Orbit 服务负责身份认证、权限校验、业务编排、任务调度和结果查询；后台 Worker 负责调用 Docker Compose 完成实际部署。

这里的“云端”指 Orbit 所在的服务端，可以部署在公有云、私有云或用户自己的服务器上，并不限定为公共 SaaS。

## 总体架构

```mermaid
flowchart LR
    A[浏览器 Web 客户端] -->|HTTPS REST API| B[Orbit 服务端]
    C[AI / MCP 客户端] -->|本地 stdio 或远程 MCP| B

    B --> D[认证与 RBAC]
    B --> E[业务服务层]
    E --> F[(数据库)]
    E --> G[任务队列]
    G --> H[后台 Worker]
    H --> I[Docker Compose / Docker]
    H --> J[Gateway / Traefik]
    B -.可选.-> K[LLM 服务]
```

## Web 客户端调用

浏览器负责页面展示和提交业务操作，不直接访问数据库、Docker 或运行时环境。

```mermaid
sequenceDiagram
    participant U as 用户
    participant W as Web 浏览器
    participant O as Orbit API
    participant DB as 数据库
    participant Q as 任务队列
    participant WK as Worker
    participant D as Docker

    U->>W: 登录或打开页面
    W->>O: 登录并获取访问令牌
    O-->>W: 返回 Bearer Token
    W->>O: 携带 Token 调用业务 API
    O->>O: 认证、权限和业务校验
    O->>DB: 读取或写入业务数据
    O-->>W: 返回资源或 Deployment ID

    W->>O: 查询部署状态和日志
    O-->>W: 返回状态和日志

    W->>O: 发起部署
    O->>DB: 保存 Deployment 快照
    O->>Q: 创建后台任务
    Q->>WK: Worker 领取任务
    WK->>D: 渲染 Compose 并执行部署
    D-->>WK: 返回执行结果
    WK->>DB: 更新状态和日志
    W->>O: 轮询或读取最终结果
```

部署请求只负责创建任务和保存部署快照，不在 HTTP 请求中同步执行 Docker。客户端通过 Deployment 状态、日志和取消接口跟踪任务。

## MCP / AI 客户端调用

### 本地 stdio 模式

本地 AI 客户端启动 Orbit MCP 进程，通过标准输入输出调用工具。首次实际调用时，MCP 进程打开浏览器完成授权：浏览器使用已有登录态申请一次性授权码，MCP 进程通过本机 loopback 回调接收授权码并交换 Bearer Token，随后将凭据保存到本地配置目录。

本地 stdio 工具直接复用 Orbit 的 Go 业务服务，不再回环调用 Orbit HTTP API；但仍使用相同的权限和业务规则。

```mermaid
sequenceDiagram
    participant C as AI/MCP 客户端
    participant S as 本地 MCP 进程
    participant B as 浏览器
    participant O as Orbit 服务端
    participant D as Docker

    C->>S: 启动 stdio MCP Server
    C->>S: 第一次实际调用工具
    S->>B: 打开授权页面
    B->>O: 使用浏览器登录态申请授权码
    O-->>B: 返回一次性授权码
    B-->>S: loopback 回调传递授权码
    S->>O: 交换 Bearer Token
    O-->>S: 返回访问令牌
    C->>S: 调用 orbit_deploy 等工具
    S->>D: 通过部署用例创建任务或执行运行时操作
    S-->>C: 返回结构化结果
```

### 远程 MCP 模式

远程客户端连接 Orbit 的 `/mcp` 端点，使用 Bearer Token 建立 Streamable HTTP 会话。服务端在连接时校验身份，并创建绑定当前用户的 MCP 工具集合。

```mermaid
sequenceDiagram
    participant C as 远程 MCP 客户端
    participant M as Orbit /mcp
    participant A as 认证服务
    participant S as 业务服务
    participant Q as Worker

    C->>M: Streamable HTTP + Bearer Token
    M->>A: 校验 Token 和用户身份
    A-->>M: 当前用户
    C->>M: 调用 orbit_deploy
    M->>S: 执行业务用例
    S->>Q: 创建 Deployment 任务
    M-->>C: 返回部署 ID 和操作摘要
    C->>M: 调用 orbit_wait_deployment
    M-->>C: 返回终态和日志摘要
```

MCP 工具覆盖应用、版本、服务、Gateway、Route、Preview、Deploy、Stop、Restart、日志和运行时验证。客户端不能直接执行任意 Docker 命令，只能通过受管工具访问部署能力。

## 对话式部署

部署对话页面由 Orbit 服务端连接 LLM，并把受管 MCP 工具作为模型工具提供。浏览器只调用对话 API，不直接连接 LLM、MCP 或 Docker。

```mermaid
flowchart LR
    A[用户自然语言] --> B[Web SSE 请求]
    B --> C[Orbit 对话服务]
    C --> D[读取 MCP 工具定义]
    C --> E[调用 OpenAI 兼容 LLM]
    E -->|选择工具| C
    C --> F[执行受管 MCP 工具]
    F --> G[读取或修改 Orbit 资源]
    G --> H[必要时创建 Deployment]
    H --> I[等待部署终态]
    I --> C
    C -->|SSE 进度和结果| B
```

对话服务会把工具调用结果反馈给 LLM，形成有限轮次的编排；部署类操作要求等待 Deployment 进入终态后再返回最终答复。

## 一句话总结

客户端负责表达意图、发起调用和展示结果；Orbit 服务端负责认证、权限和业务编排；Worker 负责实际 Docker 执行。Web 和 MCP 是两种入口，但最终共享同一套业务模型、权限边界和部署流程。
