<template>
  <ComboboxRoot
    :model-value="modelValue"
    :disabled="disabled"
    open-on-click
    @update:model-value="emit('update:modelValue', $event as RawValue)"
  >
    <ComboboxAnchor
      class="app-combobox-anchor"
      :class="[widthClass, disabled ? 'cursor-not-allowed opacity-60' : '']"
    >
      <ComboboxInput
        :display-value="displayValue"
        :placeholder="placeholder"
        :disabled="disabled"
        class="min-w-0 grow bg-transparent outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed"
      />
      <button
        v-if="hasValue(modelValue)"
        type="button"
        aria-label="清空选择"
        class="text-muted-foreground transition-colors hover:text-foreground disabled:cursor-not-allowed"
        :disabled="disabled"
        @click.stop="emit('update:modelValue', '')"
      >
        <X class="size-3.5" />
      </button>
      <ComboboxTrigger as-child>
        <button
          type="button"
          class="text-muted-foreground transition-colors hover:text-foreground disabled:cursor-not-allowed"
          :disabled="disabled"
        >
          <ChevronDown class="size-4" />
        </button>
      </ComboboxTrigger>
    </ComboboxAnchor>
    <ComboboxPortal>
      <ComboboxContent
        position="popper"
        align="start"
        class="app-popover-content w-[var(--reka-combobox-trigger-width)] overflow-y-auto"
        :side-offset="4"
      >
        <ComboboxEmpty class="px-3 py-2 text-sm text-muted-foreground">
          {{ emptyText }}
        </ComboboxEmpty>
        <ComboboxItem
          v-for="value in values"
          :key="String(value)"
          :value="value"
          :text-value="String(value)"
          class="app-option-item"
        >
          <span class="min-w-0">
            <span class="block truncate">{{ value }}</span>
          </span>
          <ComboboxItemIndicator>
            <Check class="size-4 text-primary" />
          </ComboboxItemIndicator>
        </ComboboxItem>
      </ComboboxContent>
    </ComboboxPortal>
  </ComboboxRoot>
</template>

<script setup lang="ts">
  import { Check, ChevronDown, X } from 'lucide-vue-next';
  import {
    ComboboxAnchor,
    ComboboxContent,
    ComboboxEmpty,
    ComboboxInput,
    ComboboxItem,
    ComboboxItemIndicator,
    ComboboxPortal,
    ComboboxRoot,
    ComboboxTrigger,
  } from 'reka-ui';

  export type RawValue = string | number;

  withDefaults(
    defineProps<{
      modelValue?: RawValue;
      values: RawValue[];
      placeholder?: string;
      disabled?: boolean;
      emptyText?: string;
      widthClass?: string;
    }>(),
    {
      placeholder: '请选择',
      disabled: false,
      emptyText: '暂无数据',
      widthClass: 'w-full',
    }
  );

  const emit = defineEmits<{
    'update:modelValue': [value: RawValue];
  }>();

  function displayValue(value: unknown) {
    return value === undefined || value === null ? '' : String(value);
  }

  function hasValue(value: unknown) {
    return value !== undefined && value !== null && value !== '';
  }
</script>
