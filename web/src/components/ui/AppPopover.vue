<template>
  <PopoverRoot>
    <PopoverTrigger as-child>
      <slot name="trigger" />
    </PopoverTrigger>
    <PopoverPortal>
      <PopoverContent
        :side="side"
        :align="align"
        :side-offset="6"
        class="app-popover-content w-[min(24rem,calc(100vw-32px))] overflow-y-auto p-3"
      >
        <slot />
      </PopoverContent>
    </PopoverPortal>
  </PopoverRoot>
</template>

<script setup lang="ts">
  import { PopoverContent, PopoverPortal, PopoverRoot, PopoverTrigger } from 'reka-ui';

  withDefaults(
    defineProps<{
      side?: 'top' | 'right' | 'bottom' | 'left';
      align?: 'start' | 'center' | 'end';
    }>(),
    {
      side: 'top',
      align: 'start',
    }
  );
</script>
