<template>
  <form class="space-y-4" @submit.prevent="submit">
    <div class="app-section-header app-detail-section-header">
      <h2 class="app-detail-section-title">{{ title }}</h2>
      <div v-if="editable" class="flex shrink-0 items-center gap-2">
        <button
          type="button"
          class="app-button h-9 px-3"
          :disabled="disabled || editingRowId !== null"
          @click="addRow"
        >
          <Plus class="size-4" />
          {{ t('common.add') }}
        </button>
        <button
          type="submit"
          class="app-button-primary h-9 px-3"
          :disabled="disabled || !dirty || editingRowId !== null"
        >
          <Save class="size-4" />
          {{ t('common.save') }}
        </button>
      </div>
    </div>

    <AppEmptyState v-if="rows.length === 0" size="compact" />
    <div v-else class="overflow-x-auto">
      <table class="app-data-table min-w-[640px] table-fixed">
        <colgroup>
          <col :class="editable ? 'w-[32%]' : 'w-[40%]'" />
          <col :class="editable ? 'w-[52%]' : 'w-[60%]'" />
          <col v-if="editable" class="w-[16%]" />
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
            <td>
              <template v-if="editingRowId === row.id && newRowId === row.id">
                <input
                  :id="keyInputId(row.id)"
                  :value="editingKey"
                  class="app-input h-9"
                  :class="errors[row.id] ? 'app-input-error' : ''"
                  :aria-label="t('environment.fields.key')"
                  :aria-invalid="errors[row.id] ? 'true' : undefined"
                  :aria-describedby="errors[row.id] ? keyErrorId(row.id) : undefined"
                  :disabled="disabled"
                  @input="updateEditingKey"
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
              <span v-else class="flex min-h-9 items-center break-all text-foreground">
                {{ row.key }}
              </span>
            </td>
            <td>
              <div class="relative min-h-9">
                <input
                  v-if="editingRowId === row.id"
                  :value="editingValue"
                  :type="maskValues && !valueVisible[row.id] ? 'password' : 'text'"
                  class="app-input h-9"
                  :class="maskValues ? 'pr-10' : ''"
                  :aria-label="t('environment.fields.value')"
                  :disabled="disabled"
                  @input="updateEditingValue"
                />
                <span v-else class="flex min-h-9 items-center break-all text-foreground">
                  {{
                    maskValues && row.value && !valueVisible[row.id] ? maskValue : row.value || '-'
                  }}
                </span>
                <button
                  v-if="maskValues"
                  type="button"
                  class="absolute right-1 top-1/2 inline-flex size-9 -translate-y-1/2 items-center justify-center text-muted-foreground hover:text-foreground"
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
            <td v-if="editable" class="whitespace-nowrap">
              <div v-if="editingRowId === row.id" class="flex h-9 items-center gap-2">
                <button type="button" class="app-link" :disabled="disabled" @click="applyEdit(row)">
                  {{ t('common.save') }}
                </button>
                <button
                  type="button"
                  class="text-muted-foreground hover:text-foreground"
                  :disabled="disabled"
                  @click="cancelEdit(row.id)"
                >
                  {{ t('common.cancel') }}
                </button>
              </div>
              <div v-else class="flex h-9 items-center gap-2">
                <button
                  type="button"
                  class="app-link"
                  :disabled="disabled || editingRowId !== null"
                  @click="startEdit(row)"
                >
                  {{ t('common.edit') }}
                </button>
                <button
                  type="button"
                  class="app-link-danger"
                  :disabled="disabled || editingRowId !== null"
                  :aria-label="t('common.delete')"
                  :title="t('common.delete')"
                  @click="removeRow(row.id)"
                >
                  {{ t('common.delete') }}
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </form>
</template>

<script setup lang="ts">
  import { computed, nextTick, reactive, ref, watch } from 'vue';
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
  const editingRowId = ref<string | null>(null);
  const editingKey = ref('');
  const editingValue = ref('');
  const editingValueWasVisible = ref(false);
  const newRowId = ref<string | null>(null);
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

  function updateEditingKey(event: Event) {
    editingKey.value = (event.target as HTMLInputElement).value;
    if (editingRowId.value) {
      clearError(editingRowId.value);
    }
  }

  function updateEditingValue(event: Event) {
    editingValue.value = (event.target as HTMLInputElement).value;
    if (editingRowId.value) {
      clearError(editingRowId.value);
    }
  }

  function addRow() {
    const id = `environment-variable-new-${newRowIndex++}`;
    emit('update:rows', [...props.rows, { id, key: '', value: '' }]);
    editingRowId.value = id;
    editingKey.value = '';
    editingValue.value = '';
    newRowId.value = id;
    void nextTick(() => document.getElementById(keyInputId(id))?.focus());
  }

  function removeRow(id: string) {
    emit(
      'update:rows',
      props.rows.filter((item) => item.id !== id)
    );
    clearError(id);
    delete valueVisible[id];
    if (editingRowId.value === id) {
      clearEditState();
    }
  }

  function clearEditState() {
    const id = editingRowId.value;
    if (id && props.maskValues && !editingValueWasVisible.value) {
      valueVisible[id] = false;
    }
    editingRowId.value = null;
    editingKey.value = '';
    editingValue.value = '';
    editingValueWasVisible.value = false;
    newRowId.value = null;
  }

  function startEdit(row: EnvironmentVariableListRow) {
    const isNewRow = newRowId.value === row.id;
    editingValueWasVisible.value = Boolean(valueVisible[row.id]);
    if (props.maskValues) {
      valueVisible[row.id] = true;
    }
    editingRowId.value = row.id;
    editingKey.value = row.key;
    editingValue.value = row.value;
    newRowId.value = isNewRow ? row.id : null;
    clearError(row.id);
  }

  function cancelEdit(id: string) {
    if (newRowId.value === id) {
      removeRow(id);
      return;
    }
    clearEditState();
  }

  function applyEdit(row: EnvironmentVariableListRow) {
    const key = newRowId.value === row.id ? editingKey.value : row.key;
    const rows = props.rows.map((item) =>
      item.id === row.id ? { ...item, key, value: editingValue.value } : item
    );
    const result = validateEnvironmentVariableRows(rows, props.validateKey);
    Object.keys(errors).forEach((id) => delete errors[id]);
    Object.assign(errors, result.errors);
    if (errors[row.id]) {
      return;
    }
    updateRow(row.id, { key, value: editingValue.value });
    clearEditState();
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
        const row = props.rows.find((item) => item.id === firstInvalidId);
        if (row) {
          startEdit(row);
        }
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
