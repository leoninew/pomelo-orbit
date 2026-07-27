# 结构化运行时状态响应
最后修改时间: 2026-07-27 11:55:21

Review status: Accepted

## Background

`GET /api/application/:app_id/status` 曾将 `docker compose ps --format json` 的完整输出放入单个字符串字段，Web 需要自行再次解析，健康状态边界不清。

## Goal

由后端将受管 Compose 运行时状态转换为结构化容器列表；Web 直接消费该列表。健康状态使用 Compose JSON 的 `Health` 字段。

## Non-goal

- 不新增主动 HTTP/TCP 健康探测。
- 不修改 Compose healthcheck 配置或 Service 生命周期状态机。
- 不接收任意文本或从 `Status` 文本推断健康状态；仅支持该 Compose 命令输出的 JSON 数组或逐对象 JSON record stream。

## User scenarios

服务详情加载组件状态时，用户看到每个容器的名称、组件、运行状态、状态描述、健康状态和镜像，不会收到需要浏览器解析的大段 Compose JSON 字符串。

## Acceptance

1. 状态接口返回 `containers` 数组，每项含 `id`、`name`、`service`、`state`、`status`、`health`、`image`。
2. 后端解码 `docker compose ps --format json` 的 JSON 数组或逐对象 JSON record stream，并直接映射 `Health`。
3. 无效或非 JSON 运行时输出不能作为成功响应返回。
4. Web 不再保留 Compose `ps` 字符串解析层。
5. Go、Proto 与 Web 生成契约一致，并通过相关检查。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- 使用轻量模式 / light；用户已要求直接记录并验证，Requirement 视为 Accepted。
- Docker Compose `v5.3.0` 的实际单容器输出为单个 JSON 对象；适配器因此支持受控命令的 JSON 数组和逐对象 JSON record stream，不支持任意文本或动态字段解析。

## Risk

假设受管运行时使用的 Docker Compose 支持既有 `ps --format json` 命令并输出 JSON 数组或对象 record stream。若 CLI 输出不符合该契约，接口返回内部错误，避免展示不完整或错误的健康信息。
