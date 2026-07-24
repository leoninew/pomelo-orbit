# Reka UI Integration Guide
最后修改时间: 2026-07-24 10:47:37

Doc role: living guide。与代码冲突时以代码为准。

> **For LLM Readers**: This document describes integration patterns and practices specific to this project. For Reka UI component API reference, see [reka-llms.txt](./reka-llms.txt).

## Quick Reference

- **Reka UI Docs**: See [reka-llms.txt](./reka-llms.txt) for complete component catalog
- **Official Site**: https://reka-ui.com/
- **Key Concept**: Reka UI provides unstyled, accessible primitives. We style them with Tailwind CSS v4.

---

## Project Setup

### Dependencies

```json
{
  "reka-ui": "^2.9.6",
  "lucide-vue-next": "^1.0.0",
  "@tailwindcss/vite": "^4.2.0",
  "tailwindcss": "^4.2.0"
}
```

### Auto-Import Configuration

```typescript
// wxt.config.ts or vite.config.ts
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

**Result**: No manual imports needed, components auto-imported on usage.

---

## Styling Strategy

### 1. Tailwind CSS v4 Setup

```css
/* assets/main.css */
@import 'tailwindcss';

@variant dark (&:where(.dark, .dark *));
```

### 2. CSS Variable Theme System

Define semantic tokens for consistent theming:

```css
:root {
  /* Surfaces */
  --color-bg: theme(colors.slate.200 / 70%);
  --color-surface: theme(colors.white);
  --color-surface-raised: theme(colors.white);
  
  /* Borders */
  --color-border: theme(colors.slate.200);
  --color-border-subtle: theme(colors.slate.100);
  
  /* Text */
  --color-text: theme(colors.slate.700);
  --color-text-muted: theme(colors.slate.400);
  --color-text-faint: theme(colors.slate.300);
  
  /* Interactive States */
  --color-hover: theme(colors.slate.100);
  --color-active: theme(colors.blue.50);
  --color-active-hover: theme(colors.blue.100);
  --color-input-bg: theme(colors.slate.50);
  
  /* Effects */
  --color-shadow: theme(colors.slate.900 / 8%);
}

.dark {
  --color-bg: theme(colors.slate.950);
  --color-surface: theme(colors.slate.900);
  --color-surface-raised: theme(colors.slate.800);
  --color-border: theme(colors.slate.700);
  --color-border-subtle: theme(colors.slate.800);
  --color-text: theme(colors.slate.200);
  --color-text-muted: theme(colors.slate.400);
  --color-text-faint: theme(colors.slate.600);
  --color-hover: theme(colors.slate.800);
  --color-active: theme(colors.blue.900 / 60%);
  --color-active-hover: theme(colors.blue.900);
  --color-input-bg: theme(colors.slate.800);
  --color-shadow: theme(colors.black / 30%);
}
```

### 3. Using Theme Variables

Tailwind v4 supports CSS variables directly in class names:

```vue
<div class="bg-(--color-surface) text-(--color-text) border-(--color-border)">
  Content
</div>
```

### 4. Data Attribute Styling

Reka UI uses data attributes for component states. Use Tailwind's arbitrary variant syntax:

```vue
<ToolbarToggleItem
  class="px-3 py-1.5 
         hover:bg-(--color-hover)
         data-[state=on]:bg-blue-500 
         data-[state=on]:text-white"
>
  Toggle
</ToolbarToggleItem>
```

**Common data attributes:**
- `data-state="open" | "closed"` - Open/closed state
- `data-state="on" | "off"` - Toggle state
- `data-state="checked" | "unchecked"` - Checkbox state
- `data-highlighted` - Keyboard/mouse highlight
- `data-disabled` - Disabled state

---

## Component Integration Patterns

### Dialog & AlertDialog

**Files**: `components/FolderCreateModal.vue`, `components/BookmarkDeleteModal.vue`

```vue
<template>
  <DialogRoot v-model:open="isOpen">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 bg-black/50 z-50" />
      <DialogContent
        class="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 
               bg-(--color-surface) rounded-lg p-6 shadow-xl z-51 min-w-[400px]"
      >
        <DialogTitle class="text-lg font-semibold mb-4 text-(--color-text)">
          Title
        </DialogTitle>
        <DialogDescription class="sr-only">
          Description for accessibility
        </DialogDescription>
        
        <!-- Content -->
        
        <div class="flex justify-end gap-2 mt-4">
          <DialogClose as-child>
            <button class="px-4 py-2 rounded bg-(--color-hover)">
              Cancel
            </button>
          </DialogClose>
          <button class="px-4 py-2 rounded bg-blue-500 text-white">
            Confirm
          </button>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<script setup lang="ts">
import { ref } from 'vue';

const isOpen = ref(false);

const open = () => {
  isOpen.value = true;
};

const close = () => {
  isOpen.value = false;
};

defineExpose({ open, close });
</script>
```

**Key points:**
- Always use `DialogPortal` for proper z-index layering
- `DialogTitle` is required for accessibility (use `sr-only` class to hide visually)
- Use `as-child` prop to merge props with custom elements
- Expose `open`/`close` methods for parent control

**AlertDialog difference:**
- Cannot dismiss by clicking overlay (enforced focus trap)
- Use `AlertDialogCancel` and `AlertDialogAction` instead of generic buttons
- Better for destructive actions

### Combobox (Searchable Select)

**File**: `components/FolderSelect.vue`

```vue
<template>
  <ComboboxRoot
    v-model="selected"
    :ignore-filter="true"
    @update:model-value="emit('update:modelValue', $event?.id ?? null)"
  >
    <ComboboxAnchor class="flex items-center gap-2 w-full px-3 py-2 border rounded-lg">
      <ComboboxInput
        :display-value="(item) => item?.label ?? ''"
        class="flex-1 bg-transparent outline-none"
        placeholder="Select..."
        @update:model-value="searchTerm = $event"
        @keydown.capture="handleKeydown"
      />
      <ComboboxCancel v-if="selected" as-child>
        <button @click="handleClear">×</button>
      </ComboboxCancel>
      <ComboboxTrigger>▼</ComboboxTrigger>
    </ComboboxAnchor>

    <ComboboxPortal>
      <ComboboxContent
        position="popper"
        :side-offset="4"
        class="z-50 w-[var(--reka-combobox-trigger-width)] max-h-64 
               overflow-y-auto rounded-lg border bg-(--color-surface) shadow-lg"
      >
        <ComboboxViewport class="p-1">
          <ComboboxEmpty class="py-6 text-center text-sm text-(--color-text-muted)">
            No results
          </ComboboxEmpty>
          <ComboboxItem
            v-for="item in filteredItems"
            :key="item.id"
            :value="item"
            class="px-3 py-2 rounded cursor-pointer outline-none
                   data-[highlighted]:bg-(--color-hover)
                   data-[state=checked]:text-blue-600"
          >
            {{ item.label }}
            <ComboboxItemIndicator>✓</ComboboxItemIndicator>
          </ComboboxItem>
        </ComboboxViewport>
      </ComboboxContent>
    </ComboboxPortal>
  </ComboboxRoot>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { useFilter } from 'reka-ui';

const selected = ref(null);
const searchTerm = ref('');
const { contains } = useFilter({ sensitivity: 'base' });

const filteredItems = computed(() => {
  if (!searchTerm.value) return items.value;
  return items.value.filter(item => 
    contains(item.label, searchTerm.value)
  );
});

// Prevent Combobox from handling text editing keys
const handleKeydown = (e: KeyboardEvent) => {
  const textEditKeys = ['Home', 'End', 'ArrowLeft', 'ArrowRight'];
  if (e.shiftKey || textEditKeys.includes(e.key)) {
    e.stopPropagation();
  }
};
</script>
```

**Key points:**
- Use `ignore-filter="true"` for custom filtering with `useFilter` composable
- `--reka-combobox-trigger-width` CSS variable for width matching
- Stop propagation of text editing keys to preserve cursor movement
- `ComboboxItemIndicator` only shows when item is selected

### Autocomplete (Search Input)

**File**: `components/GlobalSearch.vue`

```vue
<template>
  <AutocompleteRoot v-model="query" v-model:open="isOpen">
    <AutocompleteAnchor class="flex items-center gap-2 px-3 py-2 border rounded-lg">
      <AutocompleteInput
        class="flex-1 bg-transparent outline-none"
        placeholder="Search..."
        @keydown.capture="handleKeydown"
      />
      <AutocompleteCancel v-if="query" as-child>
        <button @click="query = ''">×</button>
      </AutocompleteCancel>
    </AutocompleteAnchor>

    <AutocompletePortal>
      <AutocompleteContent
        position="popper"
        align="start"
        :side-offset="6"
        class="z-50 w-[400px] max-h-[400px] overflow-y-auto rounded-lg border bg-(--color-surface) shadow-lg"
      >
        <AutocompleteViewport class="p-1">
          <AutocompleteEmpty class="py-8 text-center text-sm">
            No results
          </AutocompleteEmpty>
          <AutocompleteItem
            v-for="result in results"
            :key="result.id"
            :value="result.id"
            class="px-3 py-2 rounded cursor-pointer data-[highlighted]:bg-(--color-hover)"
            @click="handleSelect(result)"
          >
            <!-- Result content -->
          </AutocompleteItem>
        </AutocompleteViewport>
      </AutocompleteContent>
    </AutocompletePortal>
  </AutocompleteRoot>
</template>
```

**Difference from Combobox:**
- No selection state (search-only)
- Value is the input text, not a selected item
- Better for command palettes and global search

### Toolbar

**File**: `components/BookmarkToolbar.vue`

```vue
<template>
  <ToolbarRoot class="flex items-center gap-2 px-3 py-2 border-b">
    <ToolbarButton
      class="flex items-center gap-1.5 px-3 py-1.5 rounded hover:bg-(--color-hover)"
      @click="handleAction"
    >
      Action
    </ToolbarButton>

    <ToolbarSeparator class="w-px h-5 bg-(--color-border)" />

    <ToolbarToggleGroup
      type="single"
      :model-value="viewMode"
      class="flex rounded-lg border overflow-hidden"
      @update:model-value="viewMode = $event"
    >
      <ToolbarToggleItem
        value="grid"
        class="px-3 py-1.5 hover:bg-(--color-hover)
               data-[state=on]:bg-(--color-surface-raised) 
               data-[state=on]:text-(--color-text)"
      >
        Grid
      </ToolbarToggleItem>
      <ToolbarToggleItem
        value="list"
        class="px-3 py-1.5 hover:bg-(--color-hover)
               data-[state=on]:bg-(--color-surface-raised) 
               data-[state=on]:text-(--color-text)"
      >
        List
      </ToolbarToggleItem>
    </ToolbarToggleGroup>
  </ToolbarRoot>
</template>
```

**Key points:**
- `ToolbarToggleGroup` supports `type="single"` or `type="multiple"`
- Use `data-[state=on]` for active state styling
- `ToolbarSeparator` for visual dividers

### Collapsible (Tree Nodes)

**File**: `components/BookmarkTreeNode.vue`

```vue
<template>
  <CollapsibleRoot v-model:open="isOpen" :disabled="!hasChildren">
    <div class="flex items-center gap-2">
      <CollapsibleTrigger v-if="hasChildren" as-child>
        <button class="p-1 rounded hover:bg-(--color-hover)">
          <Icon :icon="isOpen ? 'chevron-down' : 'chevron-right'" />
        </button>
      </CollapsibleTrigger>
      <span v-else class="w-[18px]"></span>
      <span>{{ node.title }}</span>
    </div>

    <CollapsibleContent v-if="hasChildren">
      <ul class="pl-4">
        <!-- Nested items -->
      </ul>
    </CollapsibleContent>
  </CollapsibleRoot>
</template>
```

**Key points:**
- Use `disabled` prop when no children
- Combine with recursive components for tree structures
- Add CSS transitions to `CollapsibleContent` for smooth animations

### Tree

**File**: `components/EmptyFolderTreeNode.vue`

```vue
<template>
  <li>
    <TreeItem
      v-slot="{ handleSelect, handleToggle, isSelected, isIndeterminate, isExpanded }"
      as-child
      :level="level"
      :value="tree"
    >
      <div class="flex items-center gap-2">
        <button v-if="hasChildren" @click.stop="handleToggle">
          <Icon :icon="isExpanded ? 'chevron-down' : 'chevron-right'" />
        </button>
        
        <button v-if="selectable" @click.stop="handleSelect">
          <Icon v-if="isSelected" icon="check" />
          <Icon v-else-if="isIndeterminate" icon="minus" />
        </button>
        
        <span>{{ tree.title }}</span>
      </div>
    </TreeItem>

    <ul v-if="tree.children">
      <TreeNode
        v-for="child in tree.children"
        :key="child.id"
        :tree="child"
        :level="level + 1"
      />
    </ul>
  </li>
</template>
```

**Key points:**
- Use `TreeRoot` as container
- `TreeItem` provides selection and expansion state via slot props
- `level` prop for indentation
- Combine with recursive components for nested trees

### Tooltip

**File**: `components/BookmarkPreviewTooltip.vue`

```vue
<template>
  <TooltipProvider :delay-duration="500">
    <TooltipRoot v-model:open="isOpen">
      <TooltipTrigger as-child>
        <slot></slot>
      </TooltipTrigger>
      <TooltipPortal>
        <TooltipContent
          :side="side"
          :side-offset="8"
          class="z-[300] rounded-lg border bg-(--color-surface) shadow-xl p-3"
        >
          <!-- Tooltip content -->
          <TooltipArrow class="fill-(--color-surface)" />
        </TooltipContent>
      </TooltipPortal>
    </TooltipRoot>
  </TooltipProvider>
</template>
```

**Key points:**
- `TooltipProvider` manages delay for multiple tooltips
- `side` prop: `'top' | 'right' | 'bottom' | 'left'`
- Use high z-index (300+) to appear above other overlays

### Toast

**File**: `components/AppToast.vue`

```vue
<template>
  <ToastProvider :duration="4000" swipe-direction="right">
    <ToastRoot
      v-for="toast in toasts"
      :key="toast.id"
      v-model:open="openMap[toast.id]"
      type="foreground"
      class="flex items-start gap-3 px-4 py-3 rounded-lg shadow-lg border
             data-[state=open]:animate-in data-[state=closed]:animate-out"
      @update:open="(v) => handleOpenChange(toast.id, v)"
    >
      <ToastDescription>{{ toast.message }}</ToastDescription>
      <ToastClose as-child>
        <button>×</button>
      </ToastClose>
    </ToastRoot>

    <ToastViewport class="fixed bottom-4 right-4 flex flex-col gap-2 z-[200]" />
  </ToastProvider>
</template>

<script setup lang="ts">
import { reactive } from 'vue';

const toasts = ref([]);
const openMap = reactive<Record<number, boolean>>({});

const handleOpenChange = (id: number, open: boolean) => {
  if (!open) {
    toasts.value = toasts.value.filter(t => t.id !== id);
    delete openMap[id];
  }
};
</script>
```

**Key points:**
- `ToastProvider` manages global settings
- `swipe-direction` enables swipe-to-dismiss
- Track open state per toast for controlled animations
- `ToastViewport` defines container position

---

## Best Practices

### 1. Always Use `as-child` for Custom Elements

```vue
<!-- ❌ Bad: Creates extra wrapper -->
<DialogClose>
  <button>Cancel</button>
</DialogClose>

<!-- ✅ Good: No wrapper, props merged -->
<DialogClose as-child>
  <button>Cancel</button>
</DialogClose>
```

### 2. Provide Accessible Labels

```vue
<!-- ✅ Required -->
<DialogTitle>Title</DialogTitle>
<DialogDescription>Description</DialogDescription>

<!-- If you need to hide visually -->
<DialogTitle class="sr-only">Hidden Title</DialogTitle>
```

### 3. Handle Keyboard Events in Form Components

For Combobox and Autocomplete, prevent component from handling text editing keys:

```typescript
const handleKeydown = (e: KeyboardEvent) => {
  const textEditKeys = ['Home', 'End', 'ArrowLeft', 'ArrowRight'];
  
  // Stop propagation for text selection and cursor movement
  if (e.shiftKey || textEditKeys.includes(e.key)) {
    e.stopPropagation();
  }
};
```

**Why**: Reka UI handles Arrow keys for navigation. Without this, cursor movement in input fields won't work.

### 4. Always Use Portals for Overlays

```vue
<!-- ✅ Good: Rendered in document.body -->
<DialogPortal>
  <DialogOverlay />
  <DialogContent>...</DialogContent>
</DialogPortal>
```

**Why**: Prevents z-index and positioning issues.

### 5. Z-Index Layering

```css
/* Recommended z-index scale */
.z-dropdown { z-index: 50; }    /* Combobox, Autocomplete */
.z-modal { z-index: 100; }      /* Dialog, AlertDialog */
.z-toast { z-index: 200; }      /* Toast notifications */
.z-tooltip { z-index: 300; }    /* Tooltips */
```

### 6. Expose Control Methods

```vue
<script setup lang="ts">
const isOpen = ref(false);

const open = () => {
  isOpen.value = true;
};

const close = () => {
  isOpen.value = false;
};

defineExpose({ open, close });
</script>
```

**Usage in parent:**
```vue
<MyModal ref="modalRef" />
<button @click="modalRef?.open()">Open</button>
```

---

## Common Issues

### Issue 1: Keyboard Navigation Not Working in Combobox

**Problem**: Arrow keys don't navigate items.

**Solution**: Only stop propagation of text editing keys, not Arrow Up/Down:

```typescript
const handleKeydown = (e: KeyboardEvent) => {
  // Only stop text editing keys
  const textEditKeys = ['Home', 'End', 'ArrowLeft', 'ArrowRight'];
  if (e.shiftKey || textEditKeys.includes(e.key)) {
    e.stopPropagation();
  }
  // Arrow Up/Down should propagate for navigation
};
```

### Issue 2: Modal Not Closing

**Problem**: AlertDialog doesn't close when clicking overlay.

**Solution**: By design. Use `AlertDialogCancel` button or switch to `Dialog` if you need overlay dismiss.

### Issue 3: Dropdown Cut Off

**Problem**: Dropdown content gets cut off by parent overflow.

**Solution**: Ensure Portal is used and parent doesn't have `overflow: hidden`:

```vue
<ComboboxPortal>
  <ComboboxContent position="popper">
    <!-- Content -->
  </ComboboxContent>
</ComboboxPortal>
```

### Issue 4: TypeScript Errors with Auto-Import

**Problem**: TypeScript can't find component types.

**Solution**: Ensure `dts: true` in Components config and restart TS server:

```typescript
Components({
  dts: true,  // Generates components.d.ts
  resolvers: [RekaResolver()],
})
```

---

## TODO

- [ ] Document animation patterns (CSS transitions, Vue Transition)
- [ ] Add examples for Checkbox component (currently using native input)
- [ ] Document Label component usage with form fields
- [ ] Add examples for Accordion, Tabs, Popover (not currently used)
- [ ] Document testing strategies for Reka UI components
- [ ] Add performance optimization tips (lazy loading, code splitting)

---

## Additional Resources

- **Reka UI Component Catalog**: [archived documentation](../archive/reka-ui-llms.txt)
- **Official Documentation**: https://reka-ui.com/
- **Styling Guide**: https://reka-ui.com/docs/guides/styling.html
- **Composition Guide**: https://reka-ui.com/docs/guides/composition.html
- **Accessibility**: https://reka-ui.com/docs/overview/accessibility.html
- **Project Style Guide**: [style-guide.md](./style-guide.md)
- **Frontend Architecture**: [architecture.md](./architecture.md)
