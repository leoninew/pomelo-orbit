<template>
  <div class="flex w-full min-w-0 flex-wrap items-center gap-2 sm:w-auto">
    <AppBreadcrumb :items="items" />
    <ChevronRight
      v-if="items.length > 0"
      class="hidden size-3 shrink-0 text-muted-foreground sm:block"
      aria-hidden="true"
    />
    <h1 class="app-detail-page-title min-w-0 flex-1 basis-full break-words sm:basis-auto">
      {{ title }}
    </h1>
  </div>
</template>

<script setup lang="ts">
  import { ChevronRight } from '@lucide/vue';
  import AppBreadcrumb from '@/components/ui/AppBreadcrumb.vue';

  defineProps<{
    items: Array<{
      label?: string;
      labelKey?: string;
      to?: string;
    }>;
    title: string;
  }>();
</script>
