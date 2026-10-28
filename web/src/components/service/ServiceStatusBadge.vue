<template>
  <AppBadge variant="status" :tone="appStatusTone(status)">
    <CircleMinus v-if="status === 'stopped'" class="size-4 shrink-0" aria-hidden="true" />
    {{ status }}
  </AppBadge>
</template>

<script setup lang="ts">
  import { CircleMinus } from '@lucide/vue';
  import AppBadge from '@/components/ui/AppBadge.vue';
  import { appStatusTone } from '@/utils/status';

  defineProps<{ status: string }>();
</script>
