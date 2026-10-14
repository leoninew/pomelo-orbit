<template>
  <AppTruncatedText
    :text="formatTime(time)"
    :display-text="formatTime(time, 'YYYY-MM-DD')"
    class="whitespace-nowrap"
  />
</template>

<script setup lang="ts">
  import AppTruncatedText from '@/components/ui/AppTruncatedText.vue';
  import { formatTime } from '@/utils/time';

  defineProps<{ time?: string | null }>();
</script>
