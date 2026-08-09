<template>
  <EnvironmentVariableListEditor
    :rows="rows"
    :saved-rows="savedRows"
    :title="t('environment.title')"
    :disabled="disabled"
    :validate-key="validateKey"
    @update:rows="emit('update:rows', $event)"
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

  defineProps<{
    rows: EnvironmentVariableListRow[];
    savedRows: EnvironmentVariableListRow[];
    disabled: boolean;
    validateKey: EnvironmentVariableKeyValidator;
  }>();

  const emit = defineEmits<{
    'update:rows': [rows: EnvironmentVariableListRow[]];
    save: [entries: EnvironmentVariableEntry[]];
  }>();

  const { t } = useI18n();
</script>
