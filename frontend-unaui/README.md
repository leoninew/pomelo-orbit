# Frontend UnaUI

基于 Nuxt 3 + Nuxt UI 的 Pomelo Orbit 前端实现。

## 技术栈

- **框架**: Nuxt 3
- **UI 库**: Nuxt UI (基于 Tailwind CSS v4)
- **图标**: Lucide Icons (通过 @iconify-json/lucide)
- **包管理器**: pnpm
- **TypeScript**: 完整类型支持

## 项目结构

```
app/
├── assets/          # 静态资源
├── components/      # 可复用组件
├── composables/     # 组合式函数
│   └── useApi.ts   # API 请求封装
├── layouts/         # 布局组件
│   ├── default.vue # 默认布局（带导航）
│   └── empty.vue   # 空布局（登录页）
├── pages/           # 页面路由
│   ├── index.vue   # 首页
│   └── login.vue   # 登录页
├── stores/          # Pinia 状态管理
│   └── auth.ts     # 认证状态
├── types/           # TypeScript 类型定义
│   └── index.ts    # 通用类型
├── utils/           # 工具函数
│   └── format.ts   # 格式化函数
├── app.config.ts    # 应用配置
└── app.vue          # 根组件
```

## 开发

```bash
# 安装依赖
pnpm install

# 启动开发服务器 (http://localhost:10004)
pnpm dev

# 构建生产版本
pnpm build

# 预览生产构建
pnpm preview

# 类型检查
pnpm typecheck

# 代码检查
pnpm lint
```

## 端口配置

- 开发服务器: `10004`
- 后端 API 代理: `http://127.0.0.1:10001`

## 特性

- ✅ 自动导入组件和组合式函数
- ✅ 基于文件的路由系统
- ✅ 布局系统（默认布局 + 空布局）
- ✅ Pinia 状态管理
- ✅ API 代理配置
- ✅ 深色模式支持
- ✅ TypeScript 类型支持
- ✅ 响应式设计

## Nuxt UI 特点

Nuxt UI 是基于 Tailwind CSS 的高质量 UI 组件库，特点：

1. **开箱即用**: 预配置的组件，无需额外样式
2. **完全可定制**: 通过 `app.config.ts` 自定义主题
3. **可访问性**: 所有组件符合 WCAG 标准
4. **深色模式**: 内置深色模式支持
5. **图标集成**: 支持 Iconify 的所有图标集

## 与 frontend-rekaui 的对比

| 特性     | frontend-rekaui  | frontend-unaui   |
| -------- | ---------------- | ---------------- |
| 框架     | Vue 3 + Vite     | Nuxt 3           |
| UI 库    | Reka UI (无样式) | Nuxt UI (预样式) |
| 路由     | Vue Router       | 文件路由         |
| 状态管理 | Pinia            | Pinia            |
| 组件导入 | 手动/自动        | 自动             |
| 样式方案 | 自定义 CSS 变量  | Tailwind 工具类  |
| 开发体验 | 需要更多配置     | 开箱即用         |

## 下一步

参考 `GETTING_STARTED.md` 了解如何：

- 添加新页面
- 创建新组件
- 使用 Nuxt UI 组件
- 配置主题
- 调用后端 API
