<template>
  <div role="tablist" aria-label="阶段视图" class="mb-4 flex border-b border-border">
    <button
      v-for="tab in tabs"
      :key="tab.value"
      role="tab"
      type="button"
      :aria-selected="modelValue === tab.value"
      class="app-tab"
      @click="emit('update:modelValue', tab.value)"
    >
      {{ tab.label }}
    </button>
  </div>
</template>

<script setup lang="ts">
  type ViewMode = 'list' | 'dag';

  defineProps<{ modelValue: ViewMode }>();

  const emit = defineEmits<{
    'update:modelValue': [value: ViewMode];
  }>();

  const tabs = [
    { value: 'list', label: '阶段表格' },
    { value: 'dag', label: 'DAG' },
  ] as const;
</script>
