<template>
  <div class="flex justify-center" v-bind="$attrs">
    <div class="size-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
  </div>
</template>

<script setup lang="ts">
  defineOptions({ inheritAttrs: false });
</script>
