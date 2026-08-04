<template>
  <div class="flex justify-center" :class="paddingClass">
    <AppSpinner :label="label" />
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue';
  import AppSpinner from '@/components/AppSpinner.vue';

  const props = withDefaults(
    defineProps<{
      label?: string;
      size?: 'page' | 'section' | 'compact';
    }>(),
    {
      label: '',
      size: 'page',
    }
  );

  const paddingClass = computed(() => {
    if (props.size === 'compact') {
      return 'py-8';
    }
    if (props.size === 'section') {
      return 'py-12';
    }
    return 'py-16';
  });
</script>
