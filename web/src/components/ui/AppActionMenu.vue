<template>
  <DropdownMenuRoot>
    <DropdownMenuTrigger as-child>
      <slot name="trigger" />
    </DropdownMenuTrigger>
    <DropdownMenuPortal>
      <DropdownMenuContent
        class="app-popover-content min-w-40 max-w-[calc(100vw-32px)] overflow-y-auto"
        align="end"
        :side-offset="4"
      >
        <DropdownMenuItem
          v-for="option in options"
          :key="option.value"
          class="app-option-item justify-start"
          :class="option.destructive ? 'text-destructive' : ''"
          :disabled="option.disabled"
          @select="emit('select', option.value)"
        >
          {{ option.label }}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenuPortal>
  </DropdownMenuRoot>
</template>

<script setup lang="ts">
  import {
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuPortal,
    DropdownMenuRoot,
    DropdownMenuTrigger,
  } from 'reka-ui';

  defineProps<{
    options: { value: string; label: string; destructive?: boolean; disabled?: boolean }[];
  }>();

  const emit = defineEmits<{
    select: [value: string];
  }>();
</script>
