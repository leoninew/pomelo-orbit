<template>
  <nav class="flex-1 overflow-x-auto overflow-y-hidden p-3 md:overflow-y-auto md:p-4">
    <!-- Collapsed: level-1 only, jump to first child -->
    <div v-if="collapsed" class="flex gap-2 md:flex-col md:gap-1">
      <RouterLink
        v-for="branch in items"
        :key="branch.key"
        :to="branch.path"
        class="flex h-10 shrink-0 items-center justify-center rounded-md px-3 text-sm transition-colors md:px-0"
        :class="
          isBranchActive(branch)
            ? 'bg-primary/10 text-primary'
            : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'
        "
        :title="branch.label"
      >
        <component :is="branch.icon" class="size-4 shrink-0" />
      </RouterLink>
    </div>

    <!-- Expanded: true two-level menu (L1 expandable, L2 routes) -->
    <div v-else class="flex gap-3 md:block md:space-y-1">
      <CollapsibleRoot
        v-for="branch in items"
        :key="branch.key"
        class="min-w-[10rem] shrink-0 md:min-w-0"
        :open="isOpen(branch.key)"
        @update:open="(open) => setOpen(branch.key, open)"
      >
        <CollapsibleTrigger
          class="flex h-10 w-full items-center gap-3 rounded-md px-3 text-sm transition-colors outline-none focus-visible:bg-muted/60 focus-visible:text-foreground"
          :class="
            isBranchActive(branch)
              ? 'bg-muted/60 text-foreground'
              : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'
          "
        >
          <component :is="branch.icon" class="size-4 shrink-0" />
          <span class="min-w-0 flex-1 truncate text-left">{{ branch.label }}</span>
          <ChevronDown
            class="size-4 shrink-0 transition-transform duration-200"
            :class="isOpen(branch.key) ? 'rotate-0' : '-rotate-90'"
          />
        </CollapsibleTrigger>

        <CollapsibleContent class="overflow-hidden data-[state=closed]:animate-none">
          <div class="mt-1 space-y-0.5 border-l border-border/70 ml-5 pl-2">
            <RouterLink
              v-for="child in branch.children"
              :key="child.key"
              :to="child.path"
              class="flex h-9 items-center rounded-md px-2.5 text-sm transition-colors"
              :class="
                selectedKey === child.key
                  ? 'bg-primary/10 text-primary'
                  : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'
              "
            >
              <span class="truncate">{{ child.label }}</span>
            </RouterLink>
          </div>
        </CollapsibleContent>
      </CollapsibleRoot>
    </div>
  </nav>
</template>

<script setup lang="ts">
  import { ChevronDown } from '@lucide/vue';
  import { CollapsibleContent, CollapsibleRoot, CollapsibleTrigger } from 'reka-ui';
  import { reactive, watch } from 'vue';
  import type { ResolvedNavigationBranch } from '@/navigation';

  const props = defineProps<{
    items: ResolvedNavigationBranch[];
    selectedKey: string;
    collapsed?: boolean;
  }>();

  const openState = reactive<Record<string, boolean>>({});

  function isOpen(key: string) {
    return openState[key] ?? true;
  }

  function setOpen(key: string, open: boolean) {
    openState[key] = open;
  }

  function isBranchActive(branch: ResolvedNavigationBranch) {
    return branch.children.some((child) => child.key === props.selectedKey);
  }

  function syncOpenWithSelection() {
    for (const branch of props.items) {
      if (isBranchActive(branch)) {
        openState[branch.key] = true;
      }
    }
  }

  watch(
    () => [props.items, props.selectedKey] as const,
    () => {
      syncOpenWithSelection();
    },
    { immediate: true, deep: true }
  );
</script>
