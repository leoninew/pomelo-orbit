# Vue 文件块顺序和缩进规范

## 规范说明

项目中所有 Vue 单文件组件（SFC）的块顺序统一为：

```vue
<template>
  <!-- 模板内容 -->
</template>

<script setup lang="ts">
	// 脚本内容（注意：script 和 style 块内容需要缩进一级）
	import { ref } from 'vue';
	
	const count = ref(0);
</script>

<style scoped>
	/* 样式内容（同样需要缩进一级）*/
	.container {
		padding: 1rem;
	}
</style>
```

## 关键规则

1. **块顺序**：template → script → style
2. **script/style 缩进**：`<script>` 和 `<style>` 标签内的内容需要缩进一级（使用 tab）
3. **template 不缩进**：`<template>` 标签内的内容从第一级开始，不额外缩进

## 自动修复

项目已配置 ESLint 规则 `vue/block-order`，可以自动修复块顺序问题。

### 修复单个文件

```bash
yarn eslint --fix src/views/Home.vue
```

### 修复所有 Vue 文件

```bash
yarn eslint --fix "src/**/*.vue"
```

### 在保存时自动修复

如果你使用 VS Code，可以在 `.vscode/settings.json` 中配置：

```json
{
  "editor.codeActionsOnSave": {
    "source.fixAll.eslint": true
  }
}
```

## 配置位置

### ESLint 规则配置

规则配置在 `eslint.config.js` 中：

```javascript
rules: {
  // 块顺序：template → script → style
  'vue/block-order': ['error', { order: ['template', 'script', 'style'] }],
  
  // template 内容缩进（使用 tab）
  'vue/html-indent': ['error', 'tab'],
  
  // script 和 style 内容缩进（使用 tab，基础缩进为 1）
  'vue/script-indent': ['error', 'tab', { baseIndent: 1 }],
}
```

### Prettier 配置

在 `.prettierrc` 中：

```json
{
  "vueIndentScriptAndStyle": true,  // 启用 script 和 style 块缩进
  "useTabs": true,                   // 使用 tab 缩进
  "tabWidth": 2                      // tab 宽度为 2 个空格
}
```

## 为什么这样做

### 块顺序（template 在前）

1. **可读性优先**：template 是组件的视图层，最直观，放在最前面便于快速理解组件结构
2. **符合直觉**：从上到下依次是"看到什么"→"如何实现"→"如何呈现"
3. **团队一致性**：统一的顺序减少代码审查时的认知负担
4. **工具支持**：Vue 官方推荐的顺序，大多数工具和插件都默认支持

### script/style 缩进

1. **视觉层次**：缩进使得块的边界更清晰，一眼就能看出哪些代码属于哪个块
2. **编辑器友好**：大多数编辑器的代码折叠功能在有缩进时工作得更好
3. **一致性**：与其他支持嵌套语法的文件格式（如 HTML、XML）保持一致
4. **Vue 官方推荐**：Vue 官方文档和 Vue 3 项目模板都使用这种缩进方式
