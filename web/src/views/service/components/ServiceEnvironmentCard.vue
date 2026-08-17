<template>
  <EnvironmentVariableListEditor
    :rows="rows"
    :saved-rows="savedRows"
    :title="t('environment.title')"
    :disabled="disabled"
    :validate-key="validateKey"
    :allow-add="allowAdd"
    :allow-remove="allowRemove"
    :default-values="defaultValues"
    :default-value-label="defaultValueLabel"
    :value-label="valueLabel"
    :deleted-row-ids="deletedRowIds"
    :resettable-row-ids="resettableRowIds"
    :reset-label="resetLabel"
    :dirty="dirty"
    @update:rows="emit('update:rows', $event)"
    @reset:row="emit('reset:row', $event)"
    @save="emit('save', $event)"
  />
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n';
  import EnvironmentVariableListEditor from '@/components/EnvironmentVariableListEditor.vue';
  import type {
    EnvironmentVariableEntry,
    EnvironmentVariableKeyValidator,
    EnvironmentVariableListRow,
  } from '@/components/environmentVariableList';

  withDefaults(
    defineProps<{
      rows: EnvironmentVariableListRow[];
      savedRows: EnvironmentVariableListRow[];
      disabled?: boolean;
      validateKey?: EnvironmentVariableKeyValidator;
      allowAdd?: boolean;
      allowRemove?: boolean;
      defaultValues?: Record<string, string>;
      defaultValueLabel?: string;
      valueLabel?: string;
      deletedRowIds?: string[];
      resettableRowIds?: string[];
      resetLabel?: string;
      dirty?: boolean;
    }>(),
    {
      disabled: false,
      validateKey: undefined,
      allowAdd: true,
      allowRemove: true,
      defaultValues: undefined,
      defaultValueLabel: '',
      valueLabel: '',
      deletedRowIds: () => [],
      resettableRowIds: () => [],
      resetLabel: '',
      dirty: undefined,
    }
  );

  const emit = defineEmits<{
    'update:rows': [rows: EnvironmentVariableListRow[]];
    'reset:row': [row: EnvironmentVariableListRow];
    save: [entries: EnvironmentVariableEntry[]];
  }>();

  const { t } = useI18n();
</script>
