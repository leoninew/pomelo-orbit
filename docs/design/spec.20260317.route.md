# Pomelo Orbit 路由系统重新设计规范

**日期:** 2026-03-17
**状态:** 部分实施
**背景:** 从纯 File Provider 过渡到 Docker Label + File Provider 混合架构

## 实施进度

### 已完成 ✅

**后端（50%）:**
- ✅ 混合架构基础（File Provider + Docker Provider）
- ✅ Traefik 配置（两个独立应用：Windows bridge 模式 + Linux host 模式）
- ✅ Docker Label 路由支持（nginx、pomelo-orbit）
- ✅ Traefik API 集成（读取路由状态）
- ✅ 数据迁移重组（v0.4.x 系列）

**前端（50%）:**
- ✅ 路由管理基础功能
- ✅ Traefik 路由查看页面
- ✅ 应用配置文件编辑（Monaco 编辑器）

### 待完成 ❌

**后端:**
- ❌ 证书管理（集中式 TLS 配置生成）
- ❌ 易用性改进（配置验证、错误提示）

**前端:**
- ❌ docker-compose.yml 编排 UI 化（Traefik Labels Helper）
- ❌ 表单验证增强
- ❌ 配置模板和向导

### 下一阶段目标

1. **证书管理实现** - 生成集中式 tls.yml，支持 Docker Label 路由的证书配置
2. **Traefik Labels Helper** - 模态框辅助生成 Traefik labels
3. **docker-compose.yml UI 化** - 表单化编辑常见配置项

---

## 概述

Pomelo Orbit 已实现 Docker Label 支持用于基于容器的路由，创建了混合架构。本规范记录设计决策和实施状态：

1. 平台特定的 Traefik 部署策略（Linux host 模式 vs Windows bridge 模式）
2. 混合 Docker Label + File Provider 场景的路由能力
3. docker-compose.yml 配置的用户体验改进

---

## 1. 平台特定的 Traefik 部署 ✅

### 实施方案：两个独立应用

**已实现:**
- 数据库中两个独立应用：Traefik Windows 和 Traefik Linux
- 用户根据平台选择对应应用
- 每个应用有独立的配置文件和路由记录

**Traefik Windows (bridge 模式):**
- 使用端口映射：`ports: ["80:80", "443:443", "8080:8080"]`
- 适用于 Windows/macOS（Docker Desktop）
- 使用 traefik 网络

**Traefik Linux (host 模式):**
- 使用 `network_mode: host`
- 适用于 Linux 原生 Docker
- 无 NAT 开销，保留客户端真实 IP

### 设计考虑

**为什么选择两个独立应用:**
- 明确的平台区分，用户清楚选择哪个
- 配置简单，无需平台检测逻辑
- 便于维护和调试

---

## 2. 路由能力 ⚠️

### 当前实现状态

**混合架构（已实现）:**
- Traefik 同时启用 File Provider 和 Docker Provider
- 部分应用使用 Docker Labels（nginx、pomelo-orbit）
- 部分路由使用 File Provider（traefik dashboard）
- Docker Label 路由不在数据库中创建记录

**当前设计:**
- Docker Label 路由由 Traefik 自动发现，不需要数据库记录
- File Provider 路由存储在数据库，生成 YAML 文件
- 两种路由方式独立工作，互不干扰

### 证书管理（待实现）❌

**目标:**
- 为 Docker Label 路由提供证书支持
- 生成集中式 `tls.yml` 配置文件
- 证书内容存储在数据库的 route 表中

**实现方案:**
```python
def generate_tls_config():
    """生成集中式 TLS 配置"""
    routes = route_repo.find_all()
    certificates = []

    for route in routes:
        if route.cert_pem and route.cert_key:
            certificates.append({
                "certFile": f"/certs/{route.name}.pem",
                "keyFile": f"/certs/{route.name}-key.pem"
            })

    return {"tls": {"certificates": certificates}}
```

**使用方式:**
1. 在数据库中创建 route 记录（enabled=false）
2. 上传证书到该记录
3. 执行同步操作，生成 tls.yml
4. 证书自动应用到对应域名的所有路由

### mkcert 集成（当前状态）

**CLI 工具:**
- `scripts/cert.py` 提供 mkcert 命令行工具
- 命令：`new -n <domain>`（生成）、`check -n <domain>`（验证）
- 输出到 `scripts/cert/{domain}.pem`
- 用于开发环境生成本地证书

**未来改进:**
- API 端点集成
- UI 一键生成

---


## 3. 用户体验改进

### 当前状态：原始文本编辑

**docker-compose.yml 编辑:**
- Monaco 代码编辑器（语法高亮）
- 原始 YAML 文本输入
- 部署前无验证
- Traefik labels 容易出错（长字符串，容易打错）

### 建议方案：分阶段方法

**阶段 1（立即）：带 Traefik Helper 的智能编辑器**
- 保持 Monaco 编辑器
- 添加 "Traefik Labels Helper" 按钮/模态框
- 生成常见标签模式
- 保存时添加 YAML 验证

**Traefik Labels Helper UI:**
```
[按钮: 添加 Traefik 路由]

模态框打开:
  域名: [app.localhost]
  路径: [/]
  入口点: [○ HTTP ○ HTTPS]
  容器端口: [80]

  [生成标签] → 插入到编辑器:

  labels:
    - "traefik.enable=true"
    - "traefik.http.routers.app.rule=Host(`app.localhost`)"
    - "traefik.http.routers.app.entrypoints=web"
    - "traefik.http.services.app.loadbalancer.server.port=80"
```

**阶段 2（短期）：增强验证**
- 验证 Traefik label 语法
- 检查常见错误（拼写错误、缺少引号）
- 内联错误高亮
- 标签键的自动完成

**阶段 3（长期）：结构化表单 UI**
- 研究开源解决方案（Portainer、Yacht）
- 为常见场景设计基于表单的编辑器
- 保持原始编辑器作为后备
- 实现表单 ↔ YAML 双向同步


---

## 总结

**已完成的核心功能:**
- ✅ 混合架构基础（File Provider + Docker Provider）
- ✅ 两个平台特定的 Traefik 应用（Windows bridge + Linux host）
- ✅ Docker Label 路由支持（nginx、pomelo-orbit）
- ✅ Traefik API 集成（读取路由状态）
- ✅ 数据迁移重组（v0.4.x 系列）

**下一阶段重点:**
1. **证书管理** - 生成集中式 tls.yml，支持 Docker Label 路由的证书配置
2. **Traefik Labels Helper** - 模态框辅助生成 Traefik labels
3. **docker-compose.yml UI 化** - 表单化编辑常见配置项

**设计原则:**
- 保持简洁，只实现必需功能
- 混合架构，两种路由方式共存
- 渐进式改进，根据实际需求迭代

