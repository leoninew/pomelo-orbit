<template>
  <form class="space-y-4" @submit.prevent="submit">
    <div class="app-section-header app-detail-section-header">
      <h2 class="app-detail-section-title">{{ title }}</h2>
      <div v-if="editable" class="flex shrink-0 items-center gap-2">
        <button type="button" class="app-button h-9 px-3" :disabled="disabled" @click="addRow">
          <Plus class="size-4" />
          {{ t('common.add') }}
        </button>
        <button type="submit" class="app-button-primary h-9 px-3" :disabled="disabled || !dirty">
          <Save class="size-4" />
          {{ t('common.save') }}
        </button>
      </div>
    </div>

    <AppEmptyState v-if="rows.length === 0" size="compact" />
    <div v-else class="overflow-x-auto">
      <table class="app-data-table min-w-[640px] table-fixed">
        <colgroup>
          <col :class="editable ? 'w-[34%]' : 'w-[40%]'" />
          <col :class="editable ? 'w-[52%]' : 'w-[60%]'" />
          <col v-if="editable" class="w-[14%]" />
        </colgroup>
        <thead>
          <tr>
            <th>
              {{ t('environment.fields.key') }}
              <span class="ml-1 text-destructive">*</span>
            </th>
            <th>{{ t('environment.fields.value') }}</th>
            <th v-if="editable" class="w-20">{{ t('common.operation') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.id">
            <td class="align-top">
              <template v-if="editable">
                <input
                  :id="keyInputId(row.id)"
                  :value="row.key"
                  class="app-input h-9"
                  :class="errors[row.id] ? 'app-input-error' : ''"
                  :aria-label="t('environment.fields.key')"
                  :aria-invalid="errors[row.id] ? 'true' : undefined"
                  :aria-describedby="errors[row.id] ? keyErrorId(row.id) : undefined"
                  :disabled="disabled"
                  @input="updateKey(row.id, $event)"
                />
                <p
                  v-if="errors[row.id]"
                  :id="keyErrorId(row.id)"
                  class="app-field-error"
                  role="alert"
                >
                  {{ errorMessage(errors[row.id]) }}
                </p>
              </template>
              <span v-else class="block break-all text-foreground">{{ row.key }}</span>
            </td>
            <td class="align-top">
              <div class="relative">
                <input
                  v-if="editable"
                  :value="row.value"
                  :type="maskValues && !valueVisible[row.id] ? 'password' : 'text'"
                  class="app-input h-9"
                  :class="maskValues ? 'pr-10' : ''"
                  :aria-label="t('environment.fields.value')"
                  :disabled="disabled"
                  @input="updateValue(row.id, $event)"
                />
                <span v-else class="block break-all text-foreground">
                  {{
                    maskValues && row.value && !valueVisible[row.id] ? maskValue : row.value || '-'
                  }}
                </span>
                <button
                  v-if="maskValues"
                  type="button"
                  class="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-muted-foreground hover:text-foreground"
                  :aria-label="
                    valueVisible[row.id]
                      ? t('environment.actions.hideValue')
                      : t('environment.actions.showValue')
                  "
                  :title="
                    valueVisible[row.id]
                      ? t('environment.actions.hideValue')
                      : t('environment.actions.showValue')
                  "
                  :disabled="disabled"
                  @click="toggleValueVisibility(row.id)"
                >
                  <EyeOff v-if="valueVisible[row.id]" class="size-4" />
                  <Eye v-else class="size-4" />
                </button>
              </div>
            </td>
            <td v-if="editable" class="align-top whitespace-nowrap">
              <button
                type="button"
                class="app-link-danger inline-flex h-9 items-center"
                :disabled="disabled"
                :aria-label="t('common.delete')"
                :title="t('common.delete')"
                @click="removeRow(row.id)"
              >
                {{ t('common.delete') }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </form>
</template>

<script setup lang="ts">
  import { computed, nextTick, reactive, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { Eye, EyeOff, Plus, Save } from 'lucide-vue-next';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import {
    environmentVariableRowsEqual,
    type EnvironmentVariableEntry,
    type EnvironmentVariableKeyValidator,
    type EnvironmentVariableListRow,
    validateEnvironmentVariableRows,
  } from '@/components/environmentVariableList';

  const props = withDefaults(
    defineProps<{
      rows: EnvironmentVariableListRow[];
      savedRows: EnvironmentVariableListRow[];
      title: string;
      disabled?: boolean;
      editable?: boolean;
      maskValues?: boolean;
      validateKey?: EnvironmentVariableKeyValidator;
    }>(),
    {
      disabled: false,
      editable: true,
      maskValues: false,
      validateKey: undefined,
    }
  );

  const emit = defineEmits<{
    'update:rows': [rows: EnvironmentVariableListRow[]];
    save: [entries: EnvironmentVariableEntry[]];
  }>();

  const { t } = useI18n();
  const errors = reactive<Record<string, string>>({});
  const valueVisible = reactive<Record<string, boolean>>({});
  const maskValue = '********';
  let newRowIndex = 0;

  const dirty = computed(() => !environmentVariableRowsEqual(props.rows, props.savedRows));

  watch(
    () => props.savedRows,
    () => {
      Object.keys(errors).forEach((id) => delete errors[id]);
    }
  );

  function keyInputId(id: string) {
    return `environment-variable-${id}-key`;
  }

  function keyErrorId(id: string) {
    return `environment-variable-${id}-key-error`;
  }

  function errorMessage(error: string) {
    if (error === 'required') return t('environment.validation.required');
    if (error === 'duplicate') return t('environment.validation.duplicate');
    return error;
  }

  function clearError(id: string) {
    delete errors[id];
  }

  function updateRow(id: string, update: Partial<EnvironmentVariableListRow>) {
    emit(
      'update:rows',
      props.rows.map((row) => (row.id === id ? { ...row, ...update } : row))
    );
    clearError(id);
  }

  function updateKey(id: string, event: Event) {
    updateRow(id, { key: (event.target as HTMLInputElement).value });
  }

  function updateValue(id: string, event: Event) {
    updateRow(id, { value: (event.target as HTMLInputElement).value });
  }

  function addRow() {
    const id = `environment-variable-new-${newRowIndex++}`;
    emit('update:rows', [...props.rows, { id, key: '', value: '' }]);
    void nextTick(() => document.getElementById(keyInputId(id))?.focus());
  }

  function removeRow(id: string) {
    emit(
      'update:rows',
      props.rows.filter((item) => item.id !== id)
    );
    clearError(id);
    delete valueVisible[id];
  }

  function toggleValueVisibility(id: string) {
    valueVisible[id] = !valueVisible[id];
  }

  function submit() {
    const result = validateEnvironmentVariableRows(props.rows, props.validateKey);
    Object.keys(errors).forEach((id) => delete errors[id]);
    Object.assign(errors, result.errors);
    if (!result.valid) {
      const firstInvalidId = Object.keys(result.errors)[0];
      if (firstInvalidId) {
        void nextTick(() => document.getElementById(keyInputId(firstInvalidId))?.focus());
      }
      return;
    }
    if (!environmentVariableRowsEqual(result.rows, props.rows)) {
      emit('update:rows', result.rows);
    }
    if (!environmentVariableRowsEqual(result.rows, props.savedRows)) {
      emit('save', result.entries);
    }
  }
</script>
