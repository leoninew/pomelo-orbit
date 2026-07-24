# ant-design-vue 图标自动导入配置
最后修改时间: 2026-07-24 10:47:37

Doc role: living guide。与代码冲突时以代码为准。

## 概述

使用 `unplugin-vue-components` 可以实现 ant-design-vue 组件和图标的自动按需导入，无需手动 import。

## 安装依赖

```bash
yarn add -D unplugin-vue-components
yarn add @ant-design/icons-vue
```

## Vite 配置

```typescript
// vite.config.ts
import Components from 'unplugin-vue-components/vite';
import { AntDesignVueResolver } from 'unplugin-vue-components/resolvers';

export default defineConfig({
  plugins: [
    vue(),
    Components({
      resolvers: [
        AntDesignVueResolver({
          importStyle: false,
          resolveIcons: true,
        }),
      ],
    }),
  ],
});
```

## 配置选项说明

### AntDesignVueResolver 选项

| 选项 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `importStyle` | `boolean \| 'css' \| 'less' \| 'css-in-js'` | `'css'` | 组件样式导入方式，`false` 禁用 |
| `resolveIcons` | `boolean` | `false` | 是否自动解析图标组件 |

### resolveIcons

当设置为 `true` 时，自动解析以下命名的图标组件：
- 以 `Outlined` 结尾（如 `AppstoreOutlined`）
- 以 `Filled` 结尾（如 `StarFilled`）
- 以 `TwoTone` 结尾（如 `SmileTwoTone`）
- 以 `Icon` 结尾（如 `LoadingIcon`）

**注意**：需要安装 `@ant-design/icons-vue` 包。

## 使用方式

配置完成后，无需手动导入图标，直接使用：

```vue
<template>
  <AppstoreOutlined />
  <CloudServerOutlined />
  <PlusOutlined />
</template>
```

Vite 会在构建时自动按需导入图标组件。

## 参考

- [unplugin-vue-components](https://github.com/antfu/unplugin-vue-components)
- [@ant-design/icons-vue](https://github.com/ant-design/icons-vue)