# 前端架构
最后修改时间: 2026-10-14

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
│   │   ├── ui/           # 通用样式与交互，不依赖领域状态
│   │   │   ├── AppDialog.vue
│   │   │   ├── AppInput.vue
│   │   │   └── ComboboxSelect.vue
│   │   ├── layout/       # 应用顶栏与导航
│   │   ├── application/  # 组件身份规则
│   │   ├── deployment/   # 部署目录与拉取策略
│   │   ├── environment/  # 环境目标字段、表单规则与变量编辑
│   │   ├── log/          # 日志订阅状态、查找与跟随
│   │   ├── pipeline/     # 阶段视图、DAG 与变量声明
│   │   ├── project/      # 项目初始化交互
│   │   ├── repository/   # 仓库与凭证查询、回显
│   │   ├── route/        # 路由目标与同步
│   │   └── service/      # 容器日志资源与订阅
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
│   │   ├── application/ # 应用页面
│   │   │   └── components/ # 应用创建表单与版本组件卡片
│   │   ├── pipeline/    # 流水线页面
│   │   └── route/       # 路由页面与页面内证书弹窗
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

**组件 (Components)**
- `src/components/ui/` 提炼可复用的展示与交互模式，不依赖业务 API、业务 store、领域类型或业务组件。`ComboboxSelect` 负责搜索、选中、清空和键盘交互；`AppInput` / `AppTextarea` 负责字段样式、框内清空、输入事件和焦点恢复；`SearchControl` 组合 `AppInput` 的字段交互，增加搜索图标、搜索按钮与查询事件。
- 组件粒度由重复的样式与交互决定，不逐个包装 HTML 标签。只有样式的普通按钮继续使用原生 `button` 与共享样式类；清空按钮因其统一的焦点、无障碍标签与交互行为使用 `AppClearButton`。
- `src/components/layout/` 负责应用导航、认证与项目切换等外壳交互。
- `src/components/<领域或功能>/` 放置跨页面复用的业务组件。`repository/RepositorySelect`、`RepositoryCredentialSelect` 查询业务资源并映射、回显选项；`deployment/DeploymentDirectoryField` 判断部署目录变更并展示对应警告；`pipeline/` 处理阶段和变量声明；`environment/` 共享环境目标表单及其规则。业务组件依赖通用 UI，通用 UI 不反向依赖业务组件。
- 组件即使不直接调用 API，只要依赖领域状态也属于业务功能。例如 `log/LogView`、`LogDrawer`、`LogActions` 使用日志流的状态、增量内容、历史缺口和跟随行为，放在共享日志功能目录；它们使用的 `MonacoEditor` 仍属于通用 UI。
- 仅单页使用的业务组件保留在对应 `views/<领域>/components/`，组件测试与辅助文件就近放置。不建立旧路径转发或组件别名。
- 审视提炼粒度时，先统计非测试代码中的直接引用文件数，再检查实际职责、间接使用和依赖方向。引用一次的 `AppToaster` 是全局反馈入口，`AppBreadcrumb` 通过 `DetailPageHeader` 服务多个页面，仍有独立职责；只转发参数和事件的包装层应移除，无使用场景的组件不预留为组件库。

**Reka UI 原语**
- 通过显式 import 导入
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

### 2. 组件依赖

共享组件使用显式 import，按职责目录导入所需组件，不通过统一入口或旧路径转发隐藏依赖。

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

编写、审查与清理测试的具体实践见项目内 [frontend-unit-testing skill](../../.agents/skills/frontend-unit-testing/SKILL.md)，包括警告排查和实现变更后的测试漂移检查。

前端只保留轻量单元测试，验证业务规则、数据转换、请求契约和简单组件的输入输出。优先主要正向流程与真实、高风险的失败边界，不追求覆盖率，不补齐每个组件的测试。

不编写并发、性能、集成、端到端或复杂 UI 测试。外部 API 和复杂子组件使用替身；不通过整页挂载串联 Router、Store 和 API，不模拟终端或日志会话的完整生命周期，也不验证 Tooltip 的布局、焦点、Portal 和底层 UI 库行为。

测试使用项目正式语言包，避免因手写部分翻译制造警告。需要模拟路由不存在时，提供未匹配的路由解析结果，不触发 Router 的开发警告；不通过全局静默掩盖警告或错误日志。

### 单元测试
```typescript
// components/ui/SearchControl.test.ts
import { createApp, h } from 'vue';
import { expect, it, vi } from 'vitest';
import i18n from '@/i18n';
import SearchControl from './SearchControl.vue';

it('emits search when clicked', () => {
  const onSearch = vi.fn();
  const root = document.createElement('div');
  const app = createApp({ render: () => h(SearchControl, { modelValue: '', onSearch }) });
  app.use(i18n).mount(root);
  root.querySelector<HTMLButtonElement>('.app-search-button')?.click();
  expect(onSearch).toHaveBeenCalledOnce();
  app.unmount();
});
```

运行 `yarn --cwd web test`。根目录 `task test -- --cov` 仅收集 Go 覆盖率，前端仍运行普通单元测试。

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
- [编码规范](../../AGENTS.md)
