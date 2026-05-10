# Frontend 技术方案对比

本文档对比三个前端项目的技术选型和实现方式。

## 项目概览

| 项目 | 框架 | UI 库 | 端口 | 状态 |
|------|------|-------|------|------|
| frontend | Vue 3 + Vite | DaisyUI | 10002 | 生产中 |
| frontend | Vue 3 + Vite | Reka UI | 10002 | 开发中 |

## 详细对比

### 1. 框架层面

#### frontend (Vue 3 + Vite)
- **优势**:
  - 轻量灵活，完全控制
  - 构建速度快
  - 配置简单直观
  - 适合小型到中型项目
- **劣势**:
  - 需要手动配置路由、状态管理等
  - SSR 需要额外配置
  - 缺少约定式路由

#### frontend-unaui (Nuxt 3)
- **优势**:
  - 开箱即用的全栈框架
  - 文件路由系统（零配置）
  - 内置 SSR/SSG 支持
  - 自动导入组件和组合式函数
  - 适合中大型项目
- **劣势**:
  - 学习曲线稍陡
  - 构建产物较大
  - 框架约束较多

### 2. UI 库对比

#### Reka UI (frontend)
- **类型**: 无样式 (Unstyled) 组件库
- **特点**:
  - 完全的样式自由度
  - 需要自己编写所有样式
  - 基于 Radix UI 的 Vue 移植版
  - 专注于可访问性和行为
- **适用场景**:
  - 需要高度定制化的设计系统
  - 有专业设计师和充足时间
  - 追求独特的视觉风格

**示例代码**:
```vue
<!-- 需要自己写样式 -->
<RButton class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">
  点击我
</RButton>
```

#### Nuxt UI (frontend-unaui)
- **类型**: 预样式 (Pre-styled) 组件库
- **特点**:
  - 开箱即用的美观组件
  - 基于 Tailwind CSS
  - 完整的主题系统
  - 深色模式内置
- **适用场景**:
  - 快速开发 MVP
  - 标准化的企业应用
  - 小团队或个人项目

**示例代码**:
```vue
<!-- 开箱即用 -->
<UButton color="primary">
  点击我
</UButton>
```

### 3. 开发体验

#### frontend

**项目结构**:
```
src/
├── components/
│   └── ui/           # 手动创建的 UI 组件
├── views/            # 页面组件
├── router/           # 路由配置
├── stores/           # Pinia stores
└── style.css         # 全局样式（CSS 变量）
```

**路由配置**:
```typescript
// 需要手动配置
const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: () => import('@/views/Home.vue') },
    { path: '/login', component: () => import('@/views/Login.vue') }
  ]
})
```

**组件导入**:
```typescript
// 需要配置 unplugin-vue-components
import { Button } from '@/components/ui'
```

#### frontend-unaui

**项目结构**:
```
app/
├── components/       # 自动导入
├── composables/      # 自动导入
├── layouts/          # 布局系统
├── pages/            # 文件路由
├── stores/           # Pinia stores
└── app.config.ts     # 主题配置
```

**路由配置**:
```
# 零配置，基于文件
pages/
├── index.vue         # /
├── login.vue         # /login
└── users/
    ├── index.vue     # /users
    └── [id].vue      # /users/:id
```

**组件导入**:
```vue
<!-- 自动导入，无需 import -->
<UButton>点击</UButton>
<UserCard />
```

### 4. 样式方案

#### frontend
- **方案**: CSS 变量 + Tailwind CSS v4
- **主题切换**: 通过切换 CSS 变量实现
- **自定义**: 完全自由，但需要更多工作

```css
/* style.css */
:root {
  --color-primary: #3b82f6;
  --color-bg: #ffffff;
}

[data-theme="dark"] {
  --color-primary: #60a5fa;
  --color-bg: #0f172a;
}
```

```vue
<button class="bg-(--color-primary) text-white">
  按钮
</button>
```

#### frontend-unaui
- **方案**: Tailwind CSS 工具类
- **主题切换**: Nuxt UI 内置
- **自定义**: 通过 app.config.ts

```typescript
// app.config.ts
export default defineAppConfig({
  ui: {
    colors: {
      primary: 'blue',
      neutral: 'slate'
    }
  }
})
```

```vue
<UButton color="primary">
  按钮
</UButton>
```

### 5. 性能对比

| 指标 | frontend | frontend-unaui |
|------|----------------|----------------|
| 首次加载 | 快 | 中等 |
| 构建速度 | 快 | 中等 |
| 包大小 | 小 | 中等 |
| SSR 支持 | 需配置 | 内置 |
| 代码分割 | 手动 | 自动 |

### 6. 学习曲线

#### frontend
- **难度**: ⭐⭐⭐
- **需要掌握**:
  - Vue 3 基础
  - Reka UI 组件 API
  - CSS 样式编写
  - 路由配置
  - 状态管理

#### frontend-unaui
- **难度**: ⭐⭐⭐⭐
- **需要掌握**:
  - Vue 3 基础
  - Nuxt 3 约定
  - Nuxt UI 组件
  - 文件路由系统
  - Nuxt 生命周期

### 7. 适用场景建议

#### 选择 frontend 如果:
- ✅ 需要高度定制的设计系统
- ✅ 有专业的 UI/UX 设计师
- ✅ 追求最小的包体积
- ✅ 项目规模较小
- ✅ 团队熟悉 Vue 3 生态

#### 选择 frontend-unaui 如果:
- ✅ 需要快速开发和迭代
- ✅ 标准化的企业应用
- ✅ 需要 SSR/SSG 支持
- ✅ 小团队或个人项目
- ✅ 希望减少配置工作

### 8. 迁移建议

#### 从 frontend 迁移到 frontend
1. **优势**: 保持 Vue 3 + Vite 技术栈，学习成本低
2. **工作量**: 需要重写所有组件样式
3. **时间**: 约 3-4 周

#### 从 frontend 迁移到 frontend-unaui
1. **优势**: 组件开箱即用，开发速度快
2. **工作量**: 需要适应 Nuxt 约定，但组件迁移简单
3. **时间**: 约 2-3 周

### 9. 代码示例对比

#### 创建一个用户列表页

**frontend**:
```vue
<!-- src/views/Users.vue -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'

const users = ref([])
const loading = ref(false)

onMounted(async () => {
  loading.value = true
  const { data } = await axios.get('/api/users')
  users.value = data
  loading.value = false
})
</script>

<template>
  <div class="p-6">
    <h1 class="text-2xl font-bold mb-4">用户列表</h1>
    
    <div v-if="loading">加载中...</div>
    
    <div v-else class="space-y-2">
      <div
        v-for="user in users"
        :key="user.id"
        class="p-4 border border-(--color-border) rounded-lg"
      >
        {{ user.name }}
      </div>
    </div>
  </div>
</template>
```

**frontend-unaui**:
```vue
<!-- app/pages/users/index.vue -->
<script setup lang="ts">
definePageMeta({
  title: '用户列表'
})

const { data: users, pending } = await useApi('/api/users')

const columns = [
  { key: 'id', label: 'ID' },
  { key: 'name', label: '姓名' },
  { key: 'email', label: '邮箱' }
]
</script>

<template>
  <div>
    <h1 class="text-2xl font-bold mb-4">用户列表</h1>
    
    <UTable
      :rows="users"
      :columns="columns"
      :loading="pending"
    />
  </div>
</template>
```

### 10. 总结

| 维度 | frontend | frontend-unaui | 推荐 |
|------|----------------|----------------|------|
| 开发速度 | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | unaui |
| 定制能力 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | rekaui |
| 学习成本 | ⭐⭐⭐ | ⭐⭐⭐⭐ | rekaui |
| 包体积 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | rekaui |
| 维护成本 | ⭐⭐⭐ | ⭐⭐⭐⭐ | unaui |
| 团队协作 | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | unaui |

**最终建议**:
- 如果是**快速迭代的业务项目**，推荐 **frontend-unaui**
- 如果是**需要独特设计的产品**，推荐 **frontend**
- 对于 Pomelo Orbit 这样的内部工具，**frontend-unaui** 更合适
