<template>
  <button
    type="button"
    class="app-clear-button"
    :aria-label="label"
    :title="label"
    :disabled="disabled"
    @pointerdown.prevent
  >
    <X class="size-4" />
  </button>
</template>

<script setup lang="ts">
  import { X } from '@lucide/vue';

  defineProps<{ label: string; disabled?: boolean }>();
</script>
