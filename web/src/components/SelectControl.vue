<template>
  <SelectRoot
    :model-value="modelValue"
    :disabled="disabled"
    @update:model-value="emit('update:modelValue', $event as SelectOptionValue)"
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
            v-for="option in options"
            :key="String(option.value)"
            :value="option.value"
            :text-value="option.label"
            :disabled="option.disabled"
            class="app-option-item"
          >
            <SelectItemText>{{ option.label }}</SelectItemText>
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

  export type SelectOptionValue = string | number;

  export interface SelectOption {
    value: SelectOptionValue;
    label: string;
    disabled?: boolean;
  }

  withDefaults(
    defineProps<{
      modelValue?: SelectOptionValue;
      options: SelectOption[];
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
    'update:modelValue': [value: SelectOptionValue];
  }>();
</script>
