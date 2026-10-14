<template>
  <div
    class="app-textarea app-textarea-control"
    :class="$attrs.class as HTMLAttributes['class']"
    :style="$attrs.style as StyleValue"
    :data-disabled="disabled ? '' : undefined"
  >
    <textarea
      ref="textarea"
      v-model="value"
      v-bind="inputAttrs()"
      :disabled="disabled"
      :readonly="readonly"
      class="app-textarea-element"
    />
    <AppClearButton
      v-if="value && !disabled && !readonly"
      :label="t('common.clearInput')"
      @click="clear"
    />
  </div>
</template>

<script setup lang="ts">
  import { computed, ref, useAttrs, type HTMLAttributes, type StyleValue } from 'vue';
  import { useI18n } from 'vue-i18n';
  import AppClearButton from '@/components/ui/AppClearButton.vue';

  defineOptions({ inheritAttrs: false });
  const props = defineProps<{
    modelValue?: string;
    modelModifiers?: { trim?: boolean };
    disabled?: boolean;
    readonly?: boolean;
  }>();
  const emit = defineEmits<{ 'update:modelValue': [value: string] }>();
  const attrs = useAttrs();
  const { t } = useI18n();
  const textarea = ref<HTMLTextAreaElement>();
  const value = computed({
    get: () => props.modelValue ?? '',
    set: (next: string) =>
      emit('update:modelValue', props.modelModifiers?.trim ? next.trim() : next),
  });

  function inputAttrs() {
    return Object.fromEntries(
      Object.entries(attrs).filter(([key]) => key !== 'class' && key !== 'style')
    );
  }

  function clear() {
    if (!textarea.value) {
      return;
    }
    textarea.value.value = '';
    textarea.value.dispatchEvent(new Event('input', { bubbles: true }));
    textarea.value.focus();
  }
</script>
