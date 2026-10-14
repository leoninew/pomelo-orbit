<template>
  <SelectRoot
    v-model:open="open"
    :model-value="modelValue"
    :disabled="disabled"
    @update:model-value="emit('update:modelValue', $event as SelectOptionValue)"
  >
    <div ref="control" class="relative" :class="widthClass">
      <SelectTrigger
        :id="id"
        v-bind="$attrs"
        class="app-select-trigger w-full"
        :class="[invalid ? 'app-input-error' : '', hasValue ? 'gap-10' : '']"
      >
        <slot name="trigger">
          <SelectValue class="min-w-0 truncate" :placeholder="placeholder" />
          <ChevronDown class="size-4 shrink-0 text-muted-foreground" />
        </slot>
      </SelectTrigger>
      <AppClearButton
        v-if="hasValue && !disabled"
        class="absolute right-8 top-1/2 -translate-y-1/2"
        :label="t(resetValue === undefined ? 'common.clearSelection' : 'common.reset')"
        @click="clear"
      />
    </div>
    <SelectPortal>
      <SelectContent
        position="popper"
        class="app-popover-content min-w-[var(--reka-select-trigger-width)] overflow-hidden"
        :side-offset="4"
      >
        <SelectViewport>
          <SelectItem
            v-for="option in selectOptions"
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
  import { Check, ChevronDown } from '@lucide/vue';
  import { computed, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import AppClearButton from '@/components/ui/AppClearButton.vue';
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

  defineOptions({ inheritAttrs: false });
  const props = withDefaults(
    defineProps<{
      modelValue?: SelectOptionValue;
      options: (SelectOption | SelectOptionValue)[];
      id?: string;
      placeholder?: string;
      disabled?: boolean;
      invalid?: boolean;
      widthClass?: string;
      resetValue?: SelectOptionValue;
    }>(),
    {
      placeholder: '请选择',
      disabled: false,
      invalid: false,
      widthClass: 'w-full',
    }
  );

  const emit = defineEmits<{
    'update:modelValue': [value: SelectOptionValue];
  }>();
  const { t } = useI18n();
  const control = ref<HTMLElement>();
  const open = ref(false);
  const hasValue = computed(() => props.modelValue !== undefined && props.modelValue !== '');
  const selectOptions = computed(() =>
    props.options.map((option) =>
      typeof option === 'object' ? option : { value: option, label: String(option) }
    )
  );

  function clear() {
    open.value = false;
    emit('update:modelValue', props.resetValue ?? '');
    control.value?.querySelector<HTMLButtonElement>('[role="combobox"]')?.focus();
  }
</script>
