# 前端架构
最后修改时间: 2026-07-24 10:47:37

Doc role: living SoT（前端）。目录以仓库 **`web/`** 为准。与代码冲突时以代码为准。

## 技术栈

### 核心框架
- **Vue 3** - 渐进式 JavaScript 框架
- **Vite** - 下一代前端构建工具
- **TypeScript** - 类型安全的 JavaScript 超集
- **Pinia** - Vue 3 状态管理
- **Vue Router** - 官方路由管理器

### UI 层
- **Reka UI** - 无样式可访问组件库（基于 Radix UI 的 Vue 移植）
- **Tailwind CSS v4** - 实用优先的 CSS 框架
- **Lucide Vue Next** - 图标库
- **Monaco Editor** - 代码编辑器（用于脚本和配置编辑）

### 工具链
- **ESLint** - 代码质量检查
- **Prettier** - 代码格式化
- **Vitest** - 单元测试框架
- **TypeScript ESLint** - TypeScript 代码检查

## 项目结构

```
web/
├── src/
│   ├── api/              # API 客户端
│   │   ├── ci/          # CI 模块 API
│   │   └── cd/          # CD 模块 API
│   ├── components/       # 共享组件
│   │   ├── AppDialog.vue
│   │   ├── AppDrawer.vue
│   │   ├── AppToaster.vue
│   │   ├── ComboboxSelect.vue
│   │   ├── SearchControl.vue
│   │   ├── SelectControl.vue
│   │   └── MonacoEditor.vue
│   ├── composables/      # 组合式函数
│   │   └── useToast.ts
│   ├── router/           # 路由配置
│   │   └── index.ts
│   ├── stores/           # Pinia stores
│   │   └── auth.ts
│   ├── types/            # TypeScript 类型定义
│   ├── utils/            # 工具函数
│   │   ├── request.ts   # Axios 封装
│   │   └── time.ts      # 时间处理工具
│   ├── views/            # 页面组件
│   │   ├── ci/          # CI 模块页面
│   │   └── cd/          # CD 模块页面
│   ├── App.vue
│   ├── main.ts
│   └── style.css        # 全局样式和 CSS 变量
├── public/
├── eslint.config.js
├── tsconfig.json
├── vite.config.ts
└── package.json
```

## 设计模式

### 1. 组件分层

**页面组件 (Views)**
- 位于 `src/views/`
- 负责数据获取、状态管理、业务逻辑编排
- 使用共享组件构建 UI

**共享组件 (Components)**
- 位于 `src/components/`
- 封装 Reka UI 原语，提供项目级别的默认样式和行为
- 可复用、无业务逻辑

**Reka UI 原语**
- 通过 auto-import 自动导入
- 提供无样式的可访问组件基础
- 页面层避免直接使用，优先使用共享组件

### 2. 状态管理

**本地状态**
- 使用 Vue 3 Composition API (`ref`, `reactive`, `computed`)
- 适用于单个组件或父子组件通信

**全局状态**
- 使用 Pinia stores
- 当前主要用于：
  - 用户认证状态 (`auth.ts`)
  - Toast 通知 (`useToast` composable)

### 3. API 调用

**统一封装**
```typescript
// utils/request.ts
import axios from 'axios';

const request = axios.create({
  baseURL: config.apiBaseUrl,
  timeout: 30000,
});

// 请求拦截器：添加 token
request.interceptors.request.use(config => {
  const authStore = useAuthStore();
  if (authStore.token) {
    config.headers.Authorization = `Bearer ${authStore.token}`;
  }
  return config;
});

// 响应拦截器：统一错误处理
request.interceptors.response.use(
  response => response.data,
  error => {
    if (error.response?.status === 401) {
      // 跳转登录
    }
    return Promise.reject(new Error(error.response?.data?.detail));
  }
);
```

**API 模块**
```typescript
// api/ci/repository.ts
export const repositoryApi = {
  list(params: ListParams): Promise<PaginatedResp<Repository>> {
    return request.get('/api/ci/repository', { params });
  },
  get(id: string): Promise<Repository> {
    return request.get(`/api/ci/repository/${id}`);
  },
  create(data: RepositoryCreateReq): Promise<Repository> {
    return request.post('/api/ci/repository', data);
  },
};
```

### 4. 路由管理

**手动配置路由**
```typescript
// router/index.ts
const routes = [
  {
    path: '/',
    component: () => import('@/views/Home.vue'),
  },
  {
    path: '/ci/repository',
    component: () => import('@/views/ci/RepositoryPage.vue'),
  },
  {
    path: '/ci/repository/:id',
    component: () => import('@/views/ci/RepositoryDetail.vue'),
  },
];
```

**命名约定**
- 列表页：`*Page.vue`（如 `RepositoryPage.vue`）
- 详情页：`*Detail.vue`（如 `RepositoryDetail.vue`）
- 子组件：功能描述（如 `ApplicationFormFields.vue`）

## 样式系统

### CSS 变量主题

```css
/* style.css */
:root {
  /* 表面 */
  --color-bg: theme(colors.slate.200 / 70%);
  --color-surface: theme(colors.white);
  --color-surface-raised: theme(colors.white);

  /* 边框 */
  --color-border: theme(colors.slate.200);
  --color-border-subtle: theme(colors.slate.100);

  /* 文本 */
  --color-text: theme(colors.slate.700);
  --color-text-muted: theme(colors.slate.400);

  /* 交互状态 */
  --color-hover: theme(colors.slate.100);
  --color-active: theme(colors.blue.50);
}

.dark {
  --color-bg: theme(colors.slate.950);
  --color-surface: theme(colors.slate.900);
  /* ... */
}
```

### 共享样式类

详见 [风格一致性规范](./style-guide.md)

## 性能优化

### 1. 路由懒加载
```typescript
{
  path: '/ci/repository',
  component: () => import('@/views/ci/RepositoryPage.vue'),
}
```

### 2. 组件自动导入
```typescript
// vite.config.ts
import Components from 'unplugin-vue-components/vite';
import RekaResolver from 'reka-ui/resolver';

export default defineConfig({
  plugins: [
    Components({
      dts: true,
      resolvers: [RekaResolver()],
    }),
  ],
});
```

### 3. 图标按需加载
```vue
<script setup>
import { Plus, Edit, Trash } from 'lucide-vue-next';
</script>
```

## 开发规范

### 1. TypeScript 使用

**严格模式**
```json
{
  "compilerOptions": {
    "strict": true,
    "noImplicitAny": true,
    "strictNullChecks": true
  }
}
```

**类型定义**
```typescript
// types/ci.ts
export interface Repository {
  id: string
  name: string
  code: string
  repository_url: string
  created_at: string
}

export interface PaginatedResp<T> {
  items: T[]
  total: number
  page: number
  per_page: number
}
```

### 2. 组件编写规范

**Composition API + `<script setup>`**
```vue
<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { repositoryApi } from '@/api/ci/repository';
import type { Repository } from '@/types/ci';

const repositories = ref<Repository[]>([]);
const loading = ref(false);

const filteredRepositories = computed(() => {
  // ...
});

onMounted(async () => {
  await fetchRepositories();
});

async function fetchRepositories() {
  loading.value = true;
  try {
    const data = await repositoryApi.list();
    repositories.value = data.items;
  } catch (error) {
    console.error(error);
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div>
    <!-- ... -->
  </div>
</template>
```

### 3. 错误处理

**统一 Toast 提示**
```typescript
import { useToast } from '@/composables/useToast';

const toast = useToast();

try {
  await repositoryApi.create(form);
  toast.success('创建成功');
} catch (error: unknown) {
  toast.error(error instanceof Error ? error.message : '操作失败');
}
```

### 4. 时间处理

**统一使用工具函数**
```typescript
import { formatTime, formatRelativeTime } from '@/utils/time';

// UTC 时间转本地时间显示
const displayTime = formatTime(utcTime);

// 相对时间（如"3分钟前"）
const relativeTime = formatRelativeTime(utcTime);
```

## 测试策略

### 单元测试
```typescript
// components/SearchControl.test.ts
import { mount } from '@vue/test-utils';
import SearchControl from './SearchControl.vue';

describe('SearchControl', () => {
  it('emits search event when button clicked', async () => {
    const wrapper = mount(SearchControl);
    await wrapper.find('button').trigger('click');
    expect(wrapper.emitted('search')).toBeTruthy();
  });
});
```

### E2E 测试
- 待补充

## 构建和部署

### 开发环境
```bash
yarn dev
```

### 生产构建
```bash
yarn build
```

### 代码检查
```bash
yarn lint
yarn lint:fix
yarn typecheck
```

## 相关文档

- [Reka UI 使用指南](./reka-ui-guide.md)
- [风格一致性规范](./style-guide.md)
- [编码规范](../../CLAUDE.md)
