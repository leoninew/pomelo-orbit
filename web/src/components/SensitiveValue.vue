<template>
  <div class="flex min-w-0 flex-1 items-start gap-2">
    <span class="min-w-0 whitespace-pre-wrap break-all text-foreground">
      {{ isVisible ? value : mask }}
    </span>
    <button
      type="button"
      class="shrink-0 p-0.5 text-muted-foreground transition-colors hover:text-foreground"
      :aria-label="toggleLabel"
      :title="toggleLabel"
      @click="isVisible = !isVisible"
    >
      <EyeOff v-if="isVisible" class="size-4" />
      <Eye v-else class="size-4" />
    </button>
  </div>
</template>

<script setup lang="ts">
  import { Eye, EyeOff } from 'lucide-vue-next';
  import { computed, ref, watch } from 'vue';

  const props = withDefaults(
    defineProps<{
      value: string;
      label: string;
      mask?: string;
      showLabel?: string;
      hideLabel?: string;
    }>(),
    { mask: '********' }
  );

  const isVisible = ref(false);
  const toggleLabel = computed(() => {
    if (isVisible.value) {
      return props.hideLabel || `隐藏${props.label}`;
    }
    return props.showLabel || `显示${props.label}`;
  });

  watch(
    () => props.value,
    () => {
      isVisible.value = false;
    }
  );
</script>
