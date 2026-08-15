<template>
  <div class="flex min-w-0 flex-wrap items-center gap-2">
    <AppBreadcrumb :items="items" />
    <ChevronRight
      v-if="items.length > 0"
      class="size-3 shrink-0 text-muted-foreground"
      aria-hidden="true"
    />
    <h1 class="app-detail-page-title min-w-0 flex-1 break-words">{{ title }}</h1>
  </div>
</template>

<script setup lang="ts">
  import { ChevronRight } from '@lucide/vue';
  import AppBreadcrumb from '@/components/AppBreadcrumb.vue';

  defineProps<{
    items: Array<{
      label?: string;
      labelKey?: string;
      to?: string;
    }>;
    title: string;
  }>();
</script>
