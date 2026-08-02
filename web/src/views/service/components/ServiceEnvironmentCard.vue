<template>
  <section class="app-surface app-detail-card">
    <EnvironmentVariableListEditor
      :rows="rows"
      :saved-rows="savedRows"
      :title="t('environment.title')"
      :disabled="disabled"
      mask-values
      :validate-key="validateKey"
      @update:rows="emit('update:rows', $event)"
      @save="emit('save', $event)"
    />
  </section>
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
