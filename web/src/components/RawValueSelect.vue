<template>
  <SelectRoot
    :model-value="modelValue"
    :disabled="disabled"
    @update:model-value="emit('update:modelValue', $event as RawValue)"
  >
    <SelectTrigger class="app-select-trigger" :class="widthClass">
      <SelectValue :placeholder="placeholder" />
      <ChevronDown class="size-4 shrink-0 text-muted-foreground" />
    </SelectTrigger>
    <SelectPortal>
      <SelectContent
        position="popper"
        class="app-popover-content min-w-[var(--reka-select-trigger-width)] overflow-hidden"
        :side-offset="4"
      >
        <SelectViewport>
          <SelectItem
            v-for="value in values"
            :key="String(value)"
            :value="value"
            :text-value="String(value)"
            class="app-option-item"
          >
            <SelectItemText>{{ value }}</SelectItemText>
            <SelectItemIndicator>
              <Check class="size-4 text-primary" />
            </SelectItemIndicator>
          </SelectItem>
        </SelectViewport>
      </SelectContent>
    </SelectPortal>
  </SelectRoot>
</template>

<script setup lang="ts">
  import { Check, ChevronDown } from 'lucide-vue-next';
  import {
    SelectContent,
    SelectItem,
    SelectItemIndicator,
    SelectItemText,
    SelectPortal,
    SelectRoot,
    SelectTrigger,
    SelectValue,
    SelectViewport,
  } from 'reka-ui';

  export type RawValue = string | number;

  withDefaults(
    defineProps<{
      modelValue?: RawValue;
      values: RawValue[];
      placeholder?: string;
      disabled?: boolean;
      widthClass?: string;
    }>(),
    {
      placeholder: '请选择',
      disabled: false,
      widthClass: 'w-full',
    }
  );

  const emit = defineEmits<{
    'update:modelValue': [value: RawValue];
  }>();
</script>
