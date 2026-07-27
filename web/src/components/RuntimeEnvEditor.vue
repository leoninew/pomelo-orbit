<template>
  <div class="space-y-2">
    <div class="flex items-center justify-between gap-3">
      <label class="app-field-label">{{ t('credential.runtimeEnv.variables') }}</label>
      <button type="button" class="app-link inline-flex items-center gap-1 text-sm" @click="addRow">
        <Plus class="size-3.5" />
        {{ t('credential.runtimeEnv.add') }}
      </button>
    </div>
    <div
      v-for="(row, index) in modelValue"
      :key="`runtime-env-${index}`"
      class="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_1.2fr_auto]"
    >
      <input
        :value="row.key"
        type="text"
        class="app-input text-xs"
        :placeholder="t('credential.runtimeEnv.keyPlaceholder')"
        @input="updateRow(index, 'key', $event)"
      />
      <input
        :value="row.value"
        type="text"
        class="app-input text-xs"
        :placeholder="t('credential.runtimeEnv.valuePlaceholder')"
        @input="updateRow(index, 'value', $event)"
      />
      <button
        type="button"
        class="app-link-danger inline-flex items-center justify-center"
        :aria-label="t('credential.runtimeEnv.remove')"
        @click="removeRow(index)"
      >
        <Trash2 class="size-4" />
      </button>
    </div>
    <p v-if="error" class="app-field-error text-xs">{{ error }}</p>
  </div>
</template>

<script setup lang="ts">
  import { Plus, Trash2 } from 'lucide-vue-next';
  import { useI18n } from 'vue-i18n';
  import type { RuntimeEnvRow } from '@/utils/runtimeEnv';

  const props = withDefaults(
    defineProps<{
      modelValue: RuntimeEnvRow[];
      error?: string;
    }>(),
    { error: '' }
  );

  const emit = defineEmits<{
    'update:modelValue': [rows: RuntimeEnvRow[]];
  }>();

  const { t } = useI18n();

  function addRow() {
    emit('update:modelValue', [...props.modelValue, { key: '', value: '' }]);
  }

  function removeRow(index: number) {
    emit(
      'update:modelValue',
      props.modelValue.filter((_, rowIndex) => rowIndex !== index)
    );
  }

  function updateRow(index: number, field: keyof RuntimeEnvRow, event: Event) {
    const value = (event.target as HTMLInputElement).value;
    emit(
      'update:modelValue',
      props.modelValue.map((row, rowIndex) =>
        rowIndex === index ? { ...row, [field]: value } : row
      )
    );
  }
</script>
