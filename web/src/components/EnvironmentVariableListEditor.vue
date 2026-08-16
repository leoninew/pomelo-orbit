<template>
  <form class="app-surface app-detail-card space-y-4" @submit.prevent="submit">
    <div class="app-section-header app-detail-section-header">
      <h2 class="app-detail-section-title">{{ title }}</h2>
      <div class="app-detail-section-actions">
        <div v-if="rows.length > 0" class="relative w-64 max-w-full">
          <Search
            class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
          />
          <input
            v-model="search"
            type="text"
            class="app-input-search"
            :placeholder="t('environment.searchPlaceholder')"
            :aria-label="t('environment.searchPlaceholder')"
          />
          <button
            v-if="search"
            type="button"
            class="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-muted-foreground transition-colors hover:text-foreground"
            :aria-label="t('common.clearSearch')"
            @click="clearSearch"
          >
            <X class="size-4" />
          </button>
        </div>
        <div v-if="editable" class="flex shrink-0 items-center gap-2">
          <button
            type="submit"
            class="app-button-primary h-9 px-3"
            :disabled="disabled || !dirty || hasEditingRows"
          >
            <Save class="size-4" />
            {{ t('common.save') }}
          </button>
        </div>
      </div>
    </div>

    <p v-if="formError" class="app-field-error px-6" role="alert">
      {{ formError }}
    </p>

    <AppEmptyState v-if="rows.length === 0" size="compact" />
    <div
      v-else-if="filteredRows.length === 0"
      class="px-6 py-8 text-center text-sm text-muted-foreground"
    >
      {{ t('environment.noResults') }}
    </div>
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
          <tr v-for="row in filteredRows" :key="row.id">
            <td>
              <template v-if="isEditing(row.id) && isNewRow(row.id)">
                <input
                  :id="keyInputId(row.id)"
                  :value="editingRows[row.id]?.key || ''"
                  class="app-input h-9"
                  :class="errors[row.id] ? 'app-input-error' : ''"
                  :aria-label="t('environment.fields.key')"
                  :aria-invalid="errors[row.id] ? 'true' : undefined"
                  :aria-describedby="errors[row.id] ? keyErrorId(row.id) : undefined"
                  :disabled="disabled"
                  @input="updateEditingKey(row.id, $event)"
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
                  v-if="isEditing(row.id)"
                  :value="editingRows[row.id]?.value || ''"
                  :type="maskValues && !valueVisible[row.id] ? 'password' : 'text'"
                  class="app-input h-9"
                  :class="maskValues ? 'pr-10' : ''"
                  :aria-label="t('environment.fields.value')"
                  :disabled="disabled"
                  @input="updateEditingValue(row.id, $event)"
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
              <div v-if="isEditing(row.id)" class="flex h-9 items-center gap-2">
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
                <button type="button" class="app-link" :disabled="disabled" @click="startEdit(row)">
                  {{ t('common.edit') }}
                </button>
                <button
                  type="button"
                  class="app-link-danger"
                  :disabled="disabled"
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
        <tfoot v-if="editable">
          <tr>
            <td :colspan="editable ? 3 : 2">
              <button
                type="button"
                class="app-link inline-flex items-center gap-1"
                :disabled="disabled"
                @click="addRow"
              >
                <Plus class="size-3.5" />
                {{ t('common.add') }}
              </button>
            </td>
          </tr>
        </tfoot>
      </table>
    </div>
    <div v-if="editable && (rows.length === 0 || filteredRows.length === 0)" class="px-6 pb-6">
      <button
        type="button"
        class="app-link inline-flex items-center gap-1"
        :disabled="disabled"
        @click="addRow"
      >
        <Plus class="size-3.5" />
        {{ t('common.add') }}
      </button>
    </div>
  </form>
</template>

<script setup lang="ts">
  import { computed, nextTick, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { Eye, EyeOff, Plus, Save, Search, X } from '@lucide/vue';
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
      formError?: string;
    }>(),
    {
      disabled: false,
      editable: true,
      maskValues: false,
      validateKey: undefined,
      formError: undefined,
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
  const editingRows = reactive<
    Record<string, { key: string; value: string; isNew: boolean; valueWasVisible: boolean }>
  >({});
  let newRowIndex = 0;

  const dirty = computed(() => !environmentVariableRowsEqual(props.rows, props.savedRows));
  const hasEditingRows = computed(() => Object.keys(editingRows).length > 0);
  const search = ref('');
  const filteredRows = computed(() => {
    const keyword = search.value.trim().toLowerCase();
    if (!keyword) {
      return props.rows;
    }
    return props.rows.filter(
      (row) => row.key.toLowerCase().includes(keyword) || row.value.toLowerCase().includes(keyword)
    );
  });

  function clearSearch() {
    search.value = '';
  }

  watch(
    () => props.savedRows,
    () => {
      Object.keys(errors).forEach((id) => delete errors[id]);
      Object.keys(editingRows).forEach((id) => clearEditState(id));
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

  function isEditing(id: string) {
    return Boolean(editingRows[id]);
  }

  function isNewRow(id: string) {
    return editingRows[id]?.isNew ?? false;
  }

  function updateEditingKey(id: string, event: Event) {
    const row = editingRows[id];
    if (!row) return;
    row.key = (event.target as HTMLInputElement).value;
    clearError(id);
  }

  function updateEditingValue(id: string, event: Event) {
    const row = editingRows[id];
    if (!row) return;
    row.value = (event.target as HTMLInputElement).value;
    clearError(id);
  }

  function addRow() {
    search.value = '';
    const id = `environment-variable-new-${newRowIndex++}`;
    editingRows[id] = { key: '', value: '', isNew: true, valueWasVisible: false };
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
    clearEditState(id);
  }

  function clearEditState(id: string) {
    const row = editingRows[id];
    if (row && props.maskValues && !row.valueWasVisible) {
      valueVisible[id] = false;
    }
    delete editingRows[id];
  }

  function startEdit(row: EnvironmentVariableListRow) {
    if (editingRows[row.id]) return;
    const valueWasVisible = Boolean(valueVisible[row.id]);
    if (props.maskValues) {
      valueVisible[row.id] = true;
    }
    editingRows[row.id] = {
      key: row.key,
      value: row.value,
      isNew: false,
      valueWasVisible,
    };
    clearError(row.id);
  }

  function cancelEdit(id: string) {
    if (editingRows[id]?.isNew) {
      removeRow(id);
      return;
    }
    clearEditState(id);
  }

  function rowsWithEditingDrafts() {
    return props.rows.map((row) => {
      const draft = editingRows[row.id];
      if (!draft) return row;
      return {
        ...row,
        key: draft.isNew ? draft.key : row.key,
        value: draft.value,
      };
    });
  }

  function applyEdit(row: EnvironmentVariableListRow) {
    const draft = editingRows[row.id];
    if (!draft) return;
    const rows = rowsWithEditingDrafts();
    const result = validateEnvironmentVariableRows(rows, props.validateKey);
    Object.keys(errors).forEach((id) => delete errors[id]);
    Object.assign(errors, result.errors);
    if (errors[row.id]) return;
    updateRow(row.id, {
      key: draft.isNew ? draft.key : row.key,
      value: draft.value,
    });
    clearEditState(row.id);
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
