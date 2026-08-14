<template>
  <div class="space-y-6">
    <ToolbarRoot class="flex items-center justify-between gap-6" aria-label="系统设置工具栏">
      <SearchControl
        v-model="searchText"
        :placeholder="t('settings.searchPlaceholder')"
        :loading="configLoading"
        @search="handleSearch"
      />
    </ToolbarRoot>

    <!-- Restart Warning -->
    <div v-if="needsRestart" class="app-tip border-amber-200 bg-amber-50">
      <p class="text-sm text-amber-800">{{ t('settings.restartWarning') }}</p>
    </div>

    <!-- Config Table -->
    <div v-if="configLoading" class="app-surface">
      <AppLoadingState />
    </div>
    <div v-else-if="configStatus === 'error'" class="app-surface">
      <div class="py-16 text-center text-destructive">
        <p class="text-sm">{{ configError || t('settings.loadFailed') }}</p>
      </div>
    </div>
    <div v-else-if="filteredConfig.length === 0" class="app-surface">
      <AppEmptyState />
    </div>
    <div v-else class="app-surface">
      <div class="overflow-x-auto">
        <table class="app-data-table table-fixed min-w-[960px]">
          <colgroup>
            <col class="w-[25%]" />
            <col class="w-[30%]" />
            <col :class="canWriteSettings ? 'w-[30%]' : 'w-[45%]'" />
            <col v-if="canWriteSettings" class="w-[15%]" />
          </colgroup>
          <thead>
            <tr>
              <th>{{ t('settings.configKey') }}</th>
              <th>{{ t('settings.defaultValue') }}</th>
              <th>{{ t('settings.currentValue') }}</th>
              <th v-if="canWriteSettings">{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in filteredConfig" :key="item.key">
              <td
                class="max-w-0 text-foreground"
                :title="item.description ? `${item.key}: ${item.description}` : item.key"
              >
                <span
                  class="block truncate"
                  :class="
                    item.description
                      ? 'cursor-help underline decoration-dotted underline-offset-4'
                      : ''
                  "
                >
                  {{ item.key }}
                </span>
              </td>
              <td class="max-w-0 text-muted-foreground">
                <span class="block truncate" :title="displayConfigValue(item.default)">
                  {{ displayConfigValue(item.default) }}
                </span>
              </td>
              <td class="max-w-0">
                <!-- Editing Mode -->
                <div v-if="editingState.key === item.key">
                  <!-- Boolean -->
                  <RawValueSelect
                    v-if="typeof item.default === 'boolean'"
                    v-model="editingState.boolValue"
                    :values="booleanValues"
                    width-class="w-28"
                  />
                  <!-- Select -->
                  <RawValueSelect
                    v-else-if="selectOptions[item.key]"
                    v-model="editingState.stringValue"
                    :values="getSettingValues(item.key)"
                    width-class="w-40"
                  />
                  <!-- Text -->
                  <input
                    v-else
                    v-model="editingState.stringValue"
                    type="text"
                    class="app-input h-9"
                  />
                </div>
                <!-- Display Mode -->
                <div
                  v-else
                  class="min-w-0"
                  :class="
                    item.is_overridden ? 'text-amber-600 dark:text-amber-400' : 'text-foreground'
                  "
                >
                  <span class="block truncate" :title="displayConfigValue(item.value)">
                    {{ displayConfigValue(item.value) }}
                  </span>
                </div>
              </td>
              <td v-if="canWriteSettings" class="whitespace-nowrap">
                <div v-if="editingState.key === item.key" class="flex gap-2">
                  <button :disabled="operating" class="app-link" @click="handleSave(item)">
                    {{ t('common.save') }}
                  </button>
                  <button class="text-muted-foreground hover:text-foreground" @click="cancelEdit">
                    {{ t('common.cancel') }}
                  </button>
                </div>
                <div v-else class="flex gap-2">
                  <button class="app-link" @click="startEdit(item)">{{ t('common.edit') }}</button>
                  <button
                    class="text-muted-foreground hover:text-foreground"
                    @click="confirmReset(item.key)"
                  >
                    {{ t('common.reset') }}
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <AppDialog
      v-model:open="isResetDialogOpen"
      :title="t('settings.resetDialog.title')"
      width-class="w-[min(400px,calc(100vw-32px))]"
    >
      <p class="text-sm text-muted-foreground">{{ t('settings.resetDialog.description') }}</p>
      <p v-if="resetSubmitError" class="app-field-error mt-3" role="alert">
        {{ resetSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          @cancel="isResetDialogOpen = false"
          @confirm="handleReset"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref } from 'vue';
  import { ToolbarRoot } from 'reka-ui';
  import { useI18n } from 'vue-i18n';
  import { settingApi } from '@/api/settings/settings';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { PERMISSIONS } from '@/constants/permissions';
  import { useAuthStore } from '@/stores/auth';
  import type { ConfigItemResp, SystemConfigResp } from '@/gen/proto/orbit/v1/settings/settings';

  const { t } = useI18n();
  const authStore = useAuthStore();
  const toast = useToast();

  const config = ref<SystemConfigResp>();
  const searchText = ref('');
  const appliedSearch = ref('');
  const {
    status: configStatus,
    error: configError,
    loading: configLoading,
    execute,
  } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();
  const needsRestart = ref(false);
  const canWriteSettings = computed(() => authStore.hasPermission(PERMISSIONS.SETTING_WRITE));

  const filteredConfig = computed(() => {
    if (!config.value?.items) {
      return [];
    }
    if (!appliedSearch.value.trim()) {
      return config.value.items;
    }
    const search = appliedSearch.value.toLowerCase();
    return config.value.items.filter(
      (item) =>
        item.key.toLowerCase().includes(search) || item.description?.toLowerCase().includes(search)
    );
  });

  const selectOptions: Record<string, string[]> = {
    cert__letsencrypt__challenge: ['http', 'dns'],
  };
  const booleanValues = ['true', 'false'];

  interface EditingState {
    key: string | null;
    stringValue: string;
    boolValue: string;
  }

  function createEditingState(): EditingState {
    return {
      key: null,
      stringValue: '',
      boolValue: 'false',
    };
  }

  const editingState = ref<EditingState>(createEditingState());

  function getSettingValues(key: string) {
    return selectOptions[key];
  }

  function displayConfigValue(value: unknown) {
    if (typeof value === 'boolean') {
      return value ? 'true' : 'false';
    }
    if (value === null || value === undefined || value === '') {
      return '-';
    }
    return String(value);
  }

  async function fetchConfig() {
    try {
      await execute(async () => {
        config.value = await settingApi.getConfig();
      });
    } catch {
      toast.error(t('settings.loadFailed'));
    }
  }

  function handleSearch() {
    appliedSearch.value = searchText.value;
    void fetchConfig();
  }

  function startEdit(record: ConfigItemResp) {
    if (!canWriteSettings.value) {
      return;
    }
    const boolValue = record.value ?? record.default;
    editingState.value = {
      key: record.key,
      stringValue: typeof record.default === 'boolean' ? '' : String(record.value ?? ''),
      boolValue: typeof boolValue === 'boolean' ? String(boolValue) : 'false',
    };
  }

  function cancelEdit() {
    editingState.value = createEditingState();
  }

  async function handleSave(record: ConfigItemResp) {
    if (!canWriteSettings.value) {
      return;
    }
    const currentEditing = editingState.value;
    try {
      await executeOp(async () => {
        const value =
          typeof record.default === 'boolean'
            ? currentEditing.boolValue === 'true'
            : currentEditing.stringValue;
        config.value = await settingApi.updateConfig({ key: record.key, value });
        needsRestart.value = true;
        cancelEdit();
        toast.warning(t('settings.saveSuccess'));
      });
    } catch {
      toast.error(t('settings.saveFailed'));
    }
  }

  const isResetDialogOpen = ref(false);
  const pendingResetKey = ref('');
  const resetSubmitError = ref('');

  function confirmReset(key: string) {
    if (!canWriteSettings.value) {
      return;
    }
    pendingResetKey.value = key;
    resetSubmitError.value = '';
    isResetDialogOpen.value = true;
  }

  async function handleReset() {
    if (!canWriteSettings.value) {
      return;
    }
    resetSubmitError.value = '';
    try {
      await executeOp(async () => {
        config.value = await settingApi.resetConfig({
          keys: [pendingResetKey.value],
        });
        needsRestart.value = true;
        isResetDialogOpen.value = false;
        toast.warning(t('settings.resetSuccess'));
      });
    } catch {
      resetSubmitError.value = t('settings.resetFailed');
    }
  }

  onMounted(fetchConfig);
</script>
