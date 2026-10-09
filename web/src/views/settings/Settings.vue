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

    <div v-if="config?.pending_restart" class="app-tip border-amber-200 bg-amber-50">
      <p class="text-sm text-amber-800">{{ t('settings.restartWarning') }}</p>
    </div>
    <p v-if="config?.next_config_error" class="app-field-error" role="alert">
      {{ config.next_config_error }}
    </p>
    <p v-if="submitError" class="app-field-error" role="alert">{{ submitError }}</p>

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
            <tr v-for="item in filteredConfig" :key="item.key" :data-config-key="item.key">
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
                <SensitiveValue
                  v-if="item.secret"
                  :value="displayConfigValue(item.default)"
                  :label="item.key"
                  :show-label="t('settings.showValue')"
                  :hide-label="t('settings.hideValue')"
                />
                <AppTruncatedText v-else :text="displayConfigValue(item.default)" />
              </td>
              <td class="max-w-0">
                <div v-if="drafts[item.key]" class="space-y-2">
                  <RawValueSelect
                    v-if="item.type === 'boolean'"
                    v-model="drafts[item.key]!.text"
                    :values="booleanValues"
                    :disabled="operating"
                    :aria-label="item.key"
                    width-class="w-28"
                  />
                  <textarea
                    v-else-if="item.type === 'string_list'"
                    v-model="drafts[item.key]!.text"
                    class="app-input min-h-24 w-full"
                    :aria-label="item.key"
                    :disabled="operating"
                  />
                  <input
                    v-else
                    :value="drafts[item.key]!.text"
                    :type="item.type === 'integer' ? 'number' : item.secret ? 'password' : 'text'"
                    step="1"
                    class="app-input h-9 max-w-full"
                    :class="{ 'app-input-error': fieldErrors[item.key] }"
                    :aria-label="item.key"
                    :aria-invalid="!!fieldErrors[item.key]"
                    :disabled="operating"
                    @input="updateText(item.key, $event)"
                  />
                  <p v-if="fieldErrors[item.key]" class="app-field-error" role="alert">
                    {{ fieldErrors[item.key] }}
                  </p>
                </div>
                <div
                  v-else
                  class="min-w-0"
                  :class="
                    item.is_overridden ? 'text-amber-600 dark:text-amber-400' : 'text-foreground'
                  "
                >
                  <SensitiveValue
                    v-if="item.secret"
                    :value="displayConfigValue(item.value)"
                    :label="item.key"
                    :show-label="t('settings.showValue')"
                    :hide-label="t('settings.hideValue')"
                  />
                  <AppTruncatedText v-else :text="displayConfigValue(item.value)" />
                  <p v-if="resetKeys.has(item.key)" class="mt-1 text-xs text-amber-700">
                    {{ t('settings.resetPending') }}
                  </p>
                </div>
              </td>
              <td v-if="canWriteSettings" class="whitespace-nowrap">
                <div v-if="drafts[item.key] || resetKeys.has(item.key)" class="flex gap-2">
                  <button :disabled="operating" class="app-link" @click="handleSave">
                    {{ changeCount > 1 ? t('settings.saveAll') : t('common.save') }}
                  </button>
                  <button
                    :disabled="operating"
                    class="text-muted-foreground hover:text-foreground"
                    @click="cancelEdit(item.key)"
                  >
                    {{ t('common.cancel') }}
                  </button>
                </div>
                <div v-else class="flex gap-2">
                  <button :disabled="operating" class="app-link" @click="startEdit(item)">
                    {{ t('common.edit') }}
                  </button>
                  <button
                    :disabled="operating || !item.is_overridden"
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
        <AppDialogActions
          :busy="operating"
          @cancel="isResetDialogOpen = false"
          @confirm="stageReset"
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
  import AppTruncatedText from '@/components/AppTruncatedText.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import RawValueSelect from '@/components/RawValueSelect.vue';
  import SensitiveValue from '@/components/SensitiveValue.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { PERMISSIONS } from '@/constants/permissions';
  import { useAuthStore } from '@/stores/auth';
  import type {
    ConfigItemResp,
    ConfigUpdateItem,
    SystemConfigResp,
  } from '@/gen/proto/orbit/v1/settings/settings';

  interface Draft {
    text: string;
  }
  const { t } = useI18n();
  const authStore = useAuthStore();
  const toast = useToast();
  const config = ref<SystemConfigResp>();
  const searchText = ref('');
  const appliedSearch = ref('');
  const drafts = ref<Record<string, Draft>>({});
  const resetKeys = ref(new Set<string>());
  const fieldErrors = ref<Record<string, string>>({});
  const submitError = ref('');
  const isResetDialogOpen = ref(false);
  const pendingResetKey = ref('');
  const booleanValues = ['true', 'false'];
  const {
    status: configStatus,
    error: configError,
    loading: configLoading,
    execute,
  } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();
  const canWriteSettings = computed(() => authStore.hasPermission(PERMISSIONS.SETTING_WRITE));
  const changeCount = computed(() => Object.keys(drafts.value).length + resetKeys.value.size);
  const filteredConfig = computed(() => {
    const search = appliedSearch.value.trim().toLowerCase();
    return (config.value?.items || []).filter(
      (item) =>
        !search ||
        item.key.toLowerCase().includes(search) ||
        item.description.toLowerCase().includes(search)
    );
  });

  function displayConfigValue(value: unknown) {
    if (value === null || value === undefined) {
      return t('settings.unknown');
    }
    if (Array.isArray(value)) {
      return JSON.stringify(value);
    }
    return value === '' ? '""' : String(value);
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
  }

  function startEdit(item: ConfigItemResp) {
    if (!canWriteSettings.value || operating.value) {
      return;
    }
    const value = item.is_overridden ? item.override_value : (item.default ?? item.value);
    drafts.value[item.key] = {
      text: Array.isArray(value) ? value.join('\n') : String(value ?? ''),
    };
    submitError.value = '';
  }

  function cancelEdit(key: string) {
    delete drafts.value[key];
    resetKeys.value.delete(key);
    delete fieldErrors.value[key];
    submitError.value = '';
  }

  function updateText(key: string, event: Event) {
    const draft = drafts.value[key];
    if (draft) {
      draft.text = (event.target as HTMLInputElement).value;
    }
    delete fieldErrors.value[key];
  }

  function confirmReset(key: string) {
    if (!canWriteSettings.value || operating.value) {
      return;
    }
    pendingResetKey.value = key;
    isResetDialogOpen.value = true;
  }

  function stageReset() {
    resetKeys.value.add(pendingResetKey.value);
    delete drafts.value[pendingResetKey.value];
    delete fieldErrors.value[pendingResetKey.value];
    isResetDialogOpen.value = false;
  }

  async function handleSave() {
    if (!canWriteSettings.value || !config.value || !changeCount.value || operating.value) {
      return;
    }
    fieldErrors.value = {};
    submitError.value = '';
    const updates: ConfigUpdateItem[] = [];
    for (const item of config.value.items) {
      const draft = drafts.value[item.key];
      if (!draft) {
        continue;
      }
      let value: unknown = draft.text;
      if (item.type === 'boolean') {
        value = draft.text === 'true';
      } else if (item.type === 'integer') {
        const number = Number(draft.text);
        if (!draft.text.trim() || !Number.isSafeInteger(number)) {
          fieldErrors.value[item.key] = t('settings.integerRequired');
          continue;
        }
        value = number;
      } else if (item.type === 'string_list') {
        value = draft.text === '' ? [] : draft.text.split('\n');
      }
      updates.push({ key: item.key, value });
    }
    if (Object.keys(fieldErrors.value).length) {
      return;
    }
    const request = { revision: config.value.revision, updates, reset_keys: [...resetKeys.value] };
    try {
      await executeOp(async () => {
        config.value = await settingApi.updateConfig(request);
        drafts.value = {};
        resetKeys.value = new Set();
        if (config.value.pending_restart) {
          toast.warning(t('settings.saveSuccess'));
        } else {
          toast.success(t('settings.saved'));
        }
      });
    } catch (error) {
      submitError.value = error instanceof Error ? error.message : t('settings.saveFailed');
    }
  }

  onMounted(fetchConfig);
</script>
