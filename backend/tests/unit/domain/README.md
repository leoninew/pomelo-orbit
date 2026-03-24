# 领域层单元测试

## 测试覆盖概览

本目录包含 Pomelo Orbit 项目领域层（Domain Layer）的单元测试，遵循 DDD（领域驱动设计）原则，从核心业务逻辑开始测试。

### 当前测试覆盖

#### ✅ 已完成（105个测试用例）

1. **领域实体 (test_entities.py)** - 44个测试
   - Application 实体及其关联实体（Credential, GitSource, ImageSource, ConfigFile）
   - Deployment 实体（部署记录）
   - User 实体和 LoginHistory
   - Route 实体（路由配置）
   - WebhookEvent 实体（Webhook 事件）
   - 所有枚举类型验证

2. **领域异常 (test_exceptions.py)** - 12个测试
   - BusinessError 基类
   - AuthenticationError（认证异常）
   - AuthorizationError（授权异常）

3. **值对象 (test_value_objects.py)** - 13个测试
   - ApplicationStatus（应用状态）
   - OperationType（操作类型）
   - DeployStatus（部署状态）
   - ImagePullPolicy（镜像拉取策略）

4. **领域服务 (test_route_service.py)** - 26个测试
   - RouteDomainService（路由领域服务）
   - 路由配置生成
   - 路由文件部署和撤销
   - Traefik 重载逻辑

5. **数据映射器 (test_mappers.py)** - 10个测试
   - 实体与数据模型之间的双向转换
   - 枚举类型保持

### 测试覆盖率

```
Name                                    Stmts   Miss  Cover
-----------------------------------------------------------
domain/__init__.py                          4      0   100%
domain/entities.py                        100      0   100%
domain/exceptions.py                       11      0   100%
domain/route_service.py                    66      4    94%
domain/value_objects.py                    18      0   100%
-----------------------------------------------------------
核心领域逻辑覆盖率: 94%+
```

## 测试原则

1. **从内向外测试**：从领域核心（实体、值对象）开始，逐步向外扩展
2. **业务逻辑优先**：重点测试业务规则和领域逻辑
3. **隔离性**：单元测试不依赖外部资源（数据库、网络等）
4. **可读性**：测试名称清晰描述测试意图
5. **完整性**：覆盖正常流程、边界条件和异常情况

## 运行测试

```bash
# 运行所有领域层测试
uv run pytest tests/unit/domain/ -v

# 运行特定测试文件
uv run pytest tests/unit/domain/test_entities.py -v

# 生成覆盖率报告
uv run pytest tests/unit/domain/ --cov=src/pomelo_orbit/domain --cov-report=term-missing
```

## 下一步计划

### 基础设施层测试
- [ ] Repository 实现测试
- [ ] 数据库持久化测试
- [ ] Docker 管理器测试
- [ ] 配置管理测试

### 应用层测试
- [ ] Application Service 测试
- [ ] DTO 转换测试
- [ ] Webhook 解析器测试

### 接口层测试
- [ ] API 端点测试
- [ ] 请求验证测试
- [ ] 响应序列化测试
