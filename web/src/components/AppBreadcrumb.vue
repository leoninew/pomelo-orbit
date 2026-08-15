<template>
  <nav
    v-if="items.length > 0"
    :aria-label="t('app.breadcrumbAria')"
    class="min-w-0 max-w-full overflow-x-auto py-0.5 text-[11px] leading-4 text-muted-foreground"
  >
    <ol class="flex min-w-max items-center gap-1.5 whitespace-nowrap">
      <li
        v-for="(item, index) in items"
        :key="item.label ?? item.labelKey"
        class="flex items-center gap-1.5"
      >
        <ChevronRight v-if="index > 0" class="size-3 shrink-0" aria-hidden="true" />
        <RouterLink
          v-if="item.to"
          :to="item.to"
          class="block max-w-60 truncate rounded-sm outline-none transition-colors hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
        >
          {{ item.label ?? t(item.labelKey!) }}
        </RouterLink>
        <span v-if="!item.to" class="block max-w-60 truncate">
          {{ item.label ?? t(item.labelKey!) }}
        </span>
      </li>
    </ol>
  </nav>
</template>

<script setup lang="ts">
  import { ChevronRight } from '@lucide/vue';
  import { useI18n } from 'vue-i18n';

  defineProps<{
    items: Array<{
      label?: string;
      labelKey?: string;
      to?: string;
    }>;
  }>();

  const { t } = useI18n({ useScope: 'global' });
</script>
