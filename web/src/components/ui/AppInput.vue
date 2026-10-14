<template>
  <div
    class="app-input app-input-control"
    :class="$attrs.class as HTMLAttributes['class']"
    :style="$attrs.style as StyleValue"
    :data-disabled="disabled ? '' : undefined"
  >
    <slot name="leading" />
    <input
      ref="input"
      v-model="value"
      v-bind="inputAttrs()"
      :disabled="disabled"
      :readonly="readonly"
      class="app-input-element"
    />
    <AppClearButton
      v-if="value !== '' && !disabled && !readonly"
      :label="clearLabel || t('common.clearInput')"
      @click="clear"
    />
    <slot name="trailing" />
  </div>
</template>

<script setup lang="ts" generic="T extends string | number = string">
  import { computed, ref, useAttrs, type HTMLAttributes, type StyleValue } from 'vue';
  import { useI18n } from 'vue-i18n';
  import AppClearButton from '@/components/ui/AppClearButton.vue';

  defineOptions({ inheritAttrs: false });
  const props = defineProps<{
    modelValue?: T;
    modelModifiers?: { number?: boolean; trim?: boolean };
    disabled?: boolean;
    readonly?: boolean;
    clearLabel?: string;
  }>();
  const emit = defineEmits<{ 'update:modelValue': [value: T]; clear: [] }>();
  const attrs = useAttrs();
  const { t } = useI18n();
  const input = ref<HTMLInputElement>();
  const value = computed({
    get: () => props.modelValue ?? '',
    set: (next: string | number) => {
      if (props.modelModifiers?.trim && typeof next === 'string') {
        next = next.trim();
      }
      if (props.modelModifiers?.number && typeof next === 'string') {
        const number = Number.parseFloat(next);
        next = Number.isNaN(number) ? next : number;
      }
      emit('update:modelValue', next as T);
    },
  });

  function inputAttrs() {
    return Object.fromEntries(
      Object.entries(attrs).filter(([key]) => key !== 'class' && key !== 'style')
    );
  }

  function clear() {
    if (!input.value) {
      return;
    }
    input.value.value = '';
    input.value.dispatchEvent(new Event('input', { bubbles: true }));
    input.value.focus();
    emit('clear');
  }

  defineExpose({ focus: () => input.value?.focus() });
</script>
