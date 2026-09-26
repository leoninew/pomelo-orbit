<template>
  <ComboboxRoot
    :model-value="modelValue"
    :disabled="disabled"
    :ignore-filter="!filterOptions"
    open-on-click
    @update:model-value="emit('update:modelValue', $event as ComboboxOptionValue)"
  >
    <ComboboxAnchor
      class="app-combobox-anchor"
      :class="[
        widthClass,
        disabled ? 'cursor-not-allowed opacity-60' : '',
        invalid ? 'app-input-error' : '',
      ]"
    >
      <ComboboxInput
        :display-value="displayValue"
        :placeholder="placeholder"
        :disabled="disabled"
        class="min-w-0 grow bg-transparent outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed"
        @update:model-value="emit('search', String($event))"
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
    <ComboboxPortal :disabled="!portal">
      <ComboboxContent
        position="popper"
        align="start"
        class="app-popover-content w-[var(--reka-combobox-trigger-width)] overflow-y-auto"
        :side-offset="4"
      >
        <ComboboxEmpty v-if="!loading" class="px-3 py-2 text-sm text-muted-foreground">
          {{ emptyText }}
        </ComboboxEmpty>
        <ComboboxItem
          v-for="option in selectableOptions"
          :key="String(option.value)"
          :value="option.value"
          :text-value="option.label"
          :disabled="option.disabled"
          class="app-option-item"
        >
          <span class="flex min-w-0 flex-1 items-center gap-2 overflow-hidden">
            <span class="min-w-0 truncate" :title="option.label">{{ option.label }}</span>
            <span
              v-if="option.description"
              class="max-w-[45%] shrink-0 truncate text-xs text-muted-foreground"
              :title="option.description"
            >
              {{ option.description }}
            </span>
          </span>
          <ComboboxItemIndicator>
            <Check class="size-4 shrink-0 text-primary" />
          </ComboboxItemIndicator>
        </ComboboxItem>
        <div v-if="loading" class="px-3 py-2 text-sm text-muted-foreground">加载中...</div>
        <button
          v-if="hasMore && !loading"
          type="button"
          class="app-option-item w-full text-left text-sm"
          @pointerdown.prevent
          @click.stop="emit('load-more')"
        >
          加载更多
        </button>
      </ComboboxContent>
    </ComboboxPortal>
  </ComboboxRoot>
</template>

<script setup lang="ts">
  import { Check, ChevronDown, X } from '@lucide/vue';
  import { computed } from 'vue';
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

  export type ComboboxOptionValue = string | number;

  export interface ComboboxOption {
    value: ComboboxOptionValue;
    label: string;
    description?: string;
    disabled?: boolean;
  }

  const props = withDefaults(
    defineProps<{
      modelValue?: ComboboxOptionValue;
      options: ComboboxOption[];
      placeholder?: string;
      disabled?: boolean;
      emptyText?: string;
      portal?: boolean;
      widthClass?: string;
      invalid?: boolean;
      filterOptions?: boolean;
      loading?: boolean;
      hasMore?: boolean;
    }>(),
    {
      placeholder: '请选择',
      disabled: false,
      emptyText: '暂无数据',
      portal: true,
      widthClass: 'w-full',
      invalid: false,
      filterOptions: true,
      loading: false,
      hasMore: false,
    }
  );

  const emit = defineEmits<{
    'update:modelValue': [value: ComboboxOptionValue];
    search: [value: string];
    'load-more': [];
  }>();

  const selectableOptions = computed(() => props.options.filter((item) => item.value !== ''));

  function displayValue(value: unknown) {
    const option = selectableOptions.value.find((item) => item.value === value);
    return option?.label ?? '';
  }

  function hasValue(value: unknown) {
    return value !== undefined && value !== null && value !== '';
  }
</script>
