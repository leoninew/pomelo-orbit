# 开发问题总结

## 1. Nuxt API 代理配置问题

### 问题描述
- 使用 `nuxt.config.ts` 中的 `routeRules` 配置代理不工作
- 后端收到的请求路径缺少 `/api` 前缀
- 导致所有 API 请求返回 404

### 错误配置
```typescript
// nuxt.config.ts - 不工作
routeRules: {
  '/api/**': { proxy: { to: 'http://localhost:10001/api/**' } }
}
```

### 正确方案
创建 `server/api/[...].ts` 使用 `proxyRequest`：
```typescript
export default defineEventHandler(async (event) => {
  const path = event.path
  const target = 'http://127.0.0.1:10001'

  if (path.startsWith('/api/') || path.startsWith('/hooks/')) {
    return proxyRequest(event, `${target}${path}`)
  }
})
```

### 教训
- Nuxt 的 `routeRules` 代理在开发环境不可靠
- 应该使用 Nitro 的 `proxyRequest` API
- 参考 Vite 的代理配置（frontend 项目）更直观

---

## 2. Nuxt 路由结构问题

### 问题描述
- 创建了 `pages/ci/repository.vue` 和 `pages/ci/repository/[id].vue`
- 导致详情页显示的是列表内容
- 路由冲突

### 错误结构
```
pages/
  ci/
    repository.vue          # 列表页 - 错误
    repository/
      [id].vue              # 详情页
```

### 正确结构
```
pages/
  ci/
    repository/
      index.vue             # 列表页 - 正确
      [id].vue              # 详情页
```

### 教训
- Nuxt 的文件路由系统：`xxx.vue` 和 `xxx/` 目录会冲突
- 列表页应该放在 `xxx/index.vue`
- 详情页放在 `xxx/[id].vue`

---

## 3. 全局认证中间件问题

### 问题描述
- 全局中间件 `auth.global.ts` 在服务端渲染时执行
- 服务端无法访问 `localStorage`
- 导致所有页面都跳转到登录页

### 错误实现
```typescript
export default defineNuxtRouteMiddleware((to) => {
  const { isAuthenticated } = useAuth()  // useState 在 SSR 时为空
  
  if (!isAuthenticated.value) {
    return navigateTo('/login')
  }
})
```

### 正确实现
```typescript
export default defineNuxtRouteMiddleware((to) => {
  // 只在客户端检查
  if (import.meta.server) {
    return
  }

  const publicPages = ['/login']
  if (publicPages.includes(to.path)) {
    return
  }

  // 直接读取 localStorage
  const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null
  
  if (!token) {
    return navigateTo('/login')
  }
})
```

### 教训
- 全局中间件在 SSR 和 CSR 都会执行
- 认证检查应该只在客户端进行
- 不要依赖 `useState`，直接读取 `localStorage`

---

## 4. TypeScript 类型不匹配

### 问题描述
- API 返回类型定义错误
- `repositoryApi.list()` 返回 `Repository[]` 但实际应该是 `RepositoryListItem[]`
- 导致类型检查失败

### 错误定义
```typescript
list(params) {
  return api.get<PaginatedResp<Repository>>('/api/ci/repository', params)
}
```

### 正确定义
```typescript
list(params) {
  return api.get<PaginatedResp<RepositoryListItem>>('/api/ci/repository', params)
}
```

### 教训
- 列表接口和详情接口返回的数据结构不同
- 需要定义两个类型：`XxxListItem` 和 `Xxx`
- 严格按照后端 DTO 定义前端类型

---

## 5. 没有参考现有实现

### 问题描述
- 仓库详情页自己瞎编，直接在页面里显示流水线记录列表
- 实际上 frontend 只是放了一个链接跳转到流水线记录页
- 导致代码复杂且用错了接口

### 错误做法
```typescript
// 在仓库详情页直接获取流水线记录
const runs = ref<PipelineRun[]>([])
await pipelineRunApi.list({ repository_id: repositoryId })
```

### 正确做法
```vue
<!-- 只放一个链接 -->
<NuxtLink :to="`/ci/run?repository_id=${repository.id}`">
  查看所有记录
</NuxtLink>
```

### 教训
- **不要瞎编功能**，先看 frontend 是怎么实现的
- 详情页应该简洁，只显示基本信息
- 列表数据应该在专门的列表页展示

---

## 总结

### 核心问题
1. **不熟悉 Nuxt 的约定和机制**（路由、中间件、SSR）
2. **没有先查看文档和参考实现**（frontend 项目）
3. **类型定义不严谨**（没有对照后端 DTO）
4. **没有理解 SSR 和 CSR 的区别**

### 改进方向
1. 遇到问题先查文档或参考 frontend 实现
2. 理解 Nuxt 的文件路由、中间件、SSR 机制
3. 严格按照后端 API 定义前端类型
4. 每次修改后立即运行 `yarn lint --fix` 和 `yarn typecheck`
5. 不要瞎编，不确定就问或查代码
