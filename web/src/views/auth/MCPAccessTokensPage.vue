<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-simple" aria-label="MCP access token toolbar">
      <SearchControl
        v-model="searchText"
        class="shrink-0"
        :placeholder="t('mcpAccessTokens.searchPlaceholder')"
        :loading="status === 'loading'"
        @search="handleSearch"
      />
      <button class="app-button-primary px-5" @click="openCreateDialog">
        <Plus class="size-4" aria-hidden="true" />
        {{ t('mcpAccessTokens.create') }}
      </button>
    </ToolbarRoot>

    <div class="app-list-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <div v-else-if="status === 'error'" class="py-16 text-center text-destructive">
        <p class="text-sm">{{ error || t('mcpAccessTokens.loadFailed') }}</p>
      </div>
      <AppEmptyState v-else-if="filteredTokens.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[760px]">
          <thead>
            <tr>
              <th>{{ t('mcpAccessTokens.name') }}</th>
              <th>{{ t('mcpAccessTokens.token') }}</th>
              <th>{{ t('mcpAccessTokens.expiresAt') }}</th>
              <th>{{ t('mcpAccessTokens.createdAt') }}</th>
              <th class="w-16">&nbsp;</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="token in filteredTokens" :key="token.id">
              <td class="text-foreground">{{ token.name }}</td>
              <td><MCPAccessTokenValue :value="token.token" /></td>
              <td>
                <AppTableTime v-if="token.expires_at" :time="token.expires_at" />
                <span v-else>{{ t('mcpAccessTokens.neverExpires') }}</span>
              </td>
              <td><AppTableTime :time="token.created_at" /></td>
              <td>
                <button
                  type="button"
                  class="app-icon-button text-destructive"
                  :title="t('mcpAccessTokens.revoke')"
                  :aria-label="t('mcpAccessTokens.revoke')"
                  @click="openRevokeDialog(token.id)"
                >
                  <Trash2 class="size-4" aria-hidden="true" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <AppDialog v-model:open="showCreateDialog" :title="t('mcpAccessTokens.createDialogTitle')">
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label block" for="mcp-token-name">
            {{ t('mcpAccessTokens.name') }}
            <span class="text-destructive">*</span>
          </label>
          <AppInput
            id="mcp-token-name"
            v-model="form.name"
            type="text"
            maxlength="100"
            :class="formError ? 'app-input-error' : ''"
            :aria-invalid="formError ? 'true' : undefined"
            :placeholder="t('mcpAccessTokens.namePlaceholder')"
            @input="formError = ''"
          />
          <p v-if="formError" class="app-field-error">{{ formError }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block" for="mcp-token-expiry">
            {{ t('mcpAccessTokens.expiresIn') }}
          </label>
          <SelectControl
            id="mcp-token-expiry"
            :model-value="form.expires_in_days"
            :options="expiryOptions"
            :reset-value="0"
            @update:model-value="form.expires_in_days = Number($event)"
          />
        </div>
      </div>
      <p v-if="createError" class="app-field-error mt-3" role="alert">{{ createError }}</p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('mcpAccessTokens.create')"
          @cancel="showCreateDialog = false"
          @confirm="createToken"
        />
      </template>
    </AppDialog>

    <AppDialog v-model:open="showRevokeDialog" :title="t('mcpAccessTokens.revokeDialogTitle')">
      <p class="text-sm text-foreground">{{ t('mcpAccessTokens.revokeConfirm') }}</p>
      <p v-if="revokeError" class="app-field-error mt-3" role="alert">{{ revokeError }}</p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('mcpAccessTokens.revoke')"
          variant="destructive"
          @cancel="showRevokeDialog = false"
          @confirm="revokeToken"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import AppInput from '@/components/ui/AppInput.vue';
  import { Plus, Trash2 } from '@lucide/vue';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { authApi } from '@/api/auth/auth';
  import AppDialog from '@/components/ui/AppDialog.vue';
  import AppDialogActions from '@/components/ui/AppDialogActions.vue';
  import AppEmptyState from '@/components/ui/AppEmptyState.vue';
  import AppLoadingState from '@/components/ui/AppLoadingState.vue';
  import AppTableTime from '@/components/ui/AppTableTime.vue';
  import SearchControl from '@/components/ui/SearchControl.vue';
  import SelectControl, { type SelectOption } from '@/components/ui/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { MCPAccessTokenResp } from '@/gen/proto/orbit/v1/auth/auth';
  import MCPAccessTokenValue from './components/MCPAccessTokenValue.vue';
  import { ToolbarRoot } from 'reka-ui';

  const { t } = useI18n();
  const toast = useToast();
  const { status, error, execute: executeLoad } = useStatusAsync();
  const { loading: operating, execute: executeOperation } = useStatusAsync();
  const tokens = ref<MCPAccessTokenResp[]>([]);
  const searchText = ref('');
  const appliedSearch = ref('');
  const showCreateDialog = ref(false);
  const showRevokeDialog = ref(false);
  const pendingRevokeId = ref('');
  const form = reactive({ name: '', expires_in_days: 0 });
  const formError = ref('');
  const createError = ref('');
  const revokeError = ref('');
  const expiryOptions = computed<SelectOption[]>(() => [
    { value: 0, label: t('mcpAccessTokens.neverExpires') },
    { value: 30, label: t('mcpAccessTokens.expiresIn30Days') },
    { value: 90, label: t('mcpAccessTokens.expiresIn90Days') },
    { value: 365, label: t('mcpAccessTokens.expiresIn365Days') },
  ]);
  const filteredTokens = computed(() => {
    const search = appliedSearch.value.trim().toLowerCase();
    return tokens.value.filter((token) => token.name.toLowerCase().includes(search));
  });

  function handleSearch() {
    appliedSearch.value = searchText.value;
  }

  async function loadTokens() {
    try {
      await executeLoad(async () => {
        const response = await authApi.listMcpAccessTokens();
        tokens.value = response.items;
      });
    } catch {
      toast.error(t('mcpAccessTokens.loadFailed'));
    }
  }

  function openCreateDialog() {
    Object.assign(form, { name: '', expires_in_days: 0 });
    formError.value = '';
    createError.value = '';
    showCreateDialog.value = true;
  }

  async function createToken() {
    formError.value = form.name.trim() ? '' : t('mcpAccessTokens.nameRequired');
    if (formError.value) {
      return;
    }
    createError.value = '';
    try {
      await executeOperation(async () => {
        await authApi.createMcpAccessToken({
          name: form.name.trim(),
          expires_in_days: form.expires_in_days,
        });
        showCreateDialog.value = false;
        await loadTokens();
      });
      toast.success(t('mcpAccessTokens.createSuccess'));
    } catch (error) {
      createError.value =
        error instanceof Error ? error.message : t('mcpAccessTokens.createFailed');
    }
  }

  function openRevokeDialog(tokenId: string) {
    pendingRevokeId.value = tokenId;
    revokeError.value = '';
    showRevokeDialog.value = true;
  }

  async function revokeToken() {
    revokeError.value = '';
    try {
      await executeOperation(async () => {
        await authApi.revokeMcpAccessToken(pendingRevokeId.value);
        showRevokeDialog.value = false;
        pendingRevokeId.value = '';
        await loadTokens();
      });
      toast.success(t('mcpAccessTokens.revokeSuccess'));
    } catch (error) {
      revokeError.value =
        error instanceof Error ? error.message : t('mcpAccessTokens.revokeFailed');
    }
  }

  onMounted(loadTokens);
</script>
