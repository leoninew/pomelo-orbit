<template>
  <div class="space-y-6">
    <ToolbarRoot class="flex items-center justify-between gap-6" aria-label="系统设置工具栏">
      <SearchControl
        v-model="searchText"
        :placeholder="t('settings.searchPlaceholder')"
        :loading="configLoading"
      />
    </ToolbarRoot>

    <!-- Restart Warning -->
    <div v-if="needsRestart" class="app-tip border-amber-200 bg-amber-50">
      <p class="text-sm text-amber-800">{{ t('settings.restartWarning') }}</p>
    </div>

    <!-- Config Table -->
    <div v-if="configLoading" class="app-surface">
      <AppSpinner class="py-16" />
    </div>
    <div v-else-if="filteredConfig.length === 0" class="app-surface">
      <AppEmptyState />
    </div>
    <div v-else class="app-surface">
      <div class="overflow-x-auto">
        <table class="app-table-list min-w-[1120px]">
          <thead>
            <tr>
              <th>{{ t('settings.configKey') }}</th>
              <th>{{ t('common.description') }}</th>
              <th>{{ t('settings.currentValue') }}</th>
              <th>{{ t('settings.defaultValue') }}</th>
              <th>{{ t('settings.updatedAt') }}</th>
              <th v-if="canWriteSettings">{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in filteredConfig" :key="item.key">
              <td class="text-foreground">{{ item.key }}</td>
              <td class="text-muted-foreground">{{ item.description || '-' }}</td>
              <td>
                <!-- Editing Mode -->
                <div v-if="editingState.key === item.key">
                  <!-- Boolean -->
                  <SelectControl
                    v-if="typeof item.default === 'boolean'"
                    v-model="editingState.boolValue"
                    :options="booleanOptions"
                    width-class="w-28"
                  />
                  <!-- Select -->
                  <SelectControl
                    v-else-if="selectOptions[item.key]"
                    v-model="editingState.stringValue"
                    :options="getSettingOptions(item.key)"
                    width-class="w-40"
                  />
                  <!-- Text -->
                  <input
                    v-else
                    v-model="editingState.stringValue"
                    :type="secretKeys.has(item.key) ? 'password' : 'text'"
                    :placeholder="secretKeys.has(item.key) ? t('common.emptyKeepUnchanged') : ''"
                    class="app-input h-9"
                  />
                </div>
                <!-- Display Mode -->
                <div v-else class="text-foreground">
                  <span v-if="secretKeys.has(item.key)">••••••••</span>
                  <span v-else-if="typeof item.value === 'boolean'">
                    {{ item.value ? 'true' : 'false' }}
                  </span>
                  <span v-else>{{ item.value || '-' }}</span>
                </div>
              </td>
              <td class="text-muted-foreground">
                <span v-if="typeof item.default === 'boolean'">
                  {{ item.default ? 'true' : 'false' }}
                </span>
                <span v-else>{{ item.default || '-' }}</span>
              </td>
              <td class="text-muted-foreground">
                {{ item.updated_at ? formatTime(item.updated_at) : '-' }}
              </td>
              <td v-if="canWriteSettings">
                <div v-if="editingState.key === item.key" class="flex justify-end gap-2">
                  <button :disabled="operating" class="app-link" @click="handleSave(item)">
                    {{ t('common.save') }}
                  </button>
                  <button class="text-muted-foreground hover:text-foreground" @click="cancelEdit">
                    {{ t('common.cancel') }}
                  </button>
                </div>
                <div v-else class="flex justify-end gap-2">
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
      <template #footer>
        <button class="app-button" @click="isResetDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="operating" @click="handleReset">
          {{ t('common.reset') }}
        </button>
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref } from 'vue';
  import { ToolbarRoot } from 'reka-ui';
  import { useI18n } from 'vue-i18n';
  import { settingApi } from '@/api/settings';
  import AppDialog from '@/components/AppDialog.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { PERMISSIONS } from '@/constants/permissions';
  import { useAuthStore } from '@/stores/auth';
  import type { ConfigItemResp, SystemConfigResp } from '@/types/cd/settings';
  import { formatTime } from '@/utils/time';

  const { t } = useI18n();
  const authStore = useAuthStore();
  const toast = useToast();

  const config = ref<SystemConfigResp>();
  const searchText = ref('');
  const { loading: configLoading, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();
  const needsRestart = ref(false);
  const canWriteSettings = computed(() => authStore.hasPermission(PERMISSIONS.SETTING_WRITE));

  const filteredConfig = computed(() => {
    if (!config.value?.items) {
      return [];
    }
    if (!searchText.value.trim()) {
      return config.value.items;
    }
    const search = searchText.value.toLowerCase();
    return config.value.items.filter(
      (item) =>
        item.key.toLowerCase().includes(search) || item.description?.toLowerCase().includes(search)
    );
  });

  const selectOptions: Record<string, string[]> = {
    cert__letsencrypt__challenge: ['http', 'dns'],
  };
  const booleanOptions = [
    { value: 'true', label: 'true' },
    { value: 'false', label: 'false' },
  ];
  const secretKeys = new Set(['jwt__secret_key']);

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

  function getSettingOptions(key: string) {
    return (selectOptions[key] ?? []).map((option) => ({
      value: option,
      label: option,
    }));
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

  function startEdit(record: ConfigItemResp) {
    if (!canWriteSettings.value) {
      return;
    }
    const boolValue = record.value ?? record.default;
    editingState.value = {
      key: record.key,
      stringValue:
        typeof record.default === 'boolean' || secretKeys.has(record.key)
          ? ''
          : String(record.value ?? ''),
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
    if (secretKeys.has(record.key) && !currentEditing.stringValue) {
      cancelEdit();
      return;
    }
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

  function confirmReset(key: string) {
    if (!canWriteSettings.value) {
      return;
    }
    pendingResetKey.value = key;
    isResetDialogOpen.value = true;
  }

  async function handleReset() {
    if (!canWriteSettings.value) {
      return;
    }
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
      toast.error(t('settings.resetFailed'));
    }
  }

  onMounted(fetchConfig);
</script>
