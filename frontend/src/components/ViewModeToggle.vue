<template>
  <ToggleGroupRoot
    v-model="internalValue"
    type="single"
    class="flex h-8 overflow-hidden rounded-md border border-border bg-background"
    aria-label="视图模式"
  >
    <ToggleGroupItem
      value="list"
      class="px-3 text-xs font-medium text-muted-foreground outline-none transition-colors hover:bg-muted/50 hover:text-foreground data-[state=on]:bg-primary data-[state=on]:text-primary-foreground"
      aria-label="列表视图"
    >
      列表
    </ToggleGroupItem>
    <ToggleGroupItem
      value="dag"
      class="px-3 text-xs font-medium text-muted-foreground outline-none transition-colors hover:bg-muted/50 hover:text-foreground data-[state=on]:bg-primary data-[state=on]:text-primary-foreground"
      aria-label="DAG视图"
    >
      DAG
    </ToggleGroupItem>
  </ToggleGroupRoot>
</template>

<script setup lang="ts">
  import { computed } from 'vue';
  import { ToggleGroupItem, ToggleGroupRoot } from 'reka-ui';

  const props = withDefaults(
    defineProps<{
      modelValue: 'list' | 'dag';
    }>(),
    {
      modelValue: 'list',
    }
  );

  const emit = defineEmits<{
    'update:modelValue': [value: 'list' | 'dag'];
  }>();

  const internalValue = computed({
    get: () => props.modelValue,
    set: (value) => emit('update:modelValue', value as 'list' | 'dag'),
  });
</script>
