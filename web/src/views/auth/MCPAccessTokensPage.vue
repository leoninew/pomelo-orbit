<template>
  <div class="space-y-6">
    <ToolbarRoot class="flex items-center justify-end" aria-label="MCP access token toolbar">
      <button class="app-button-primary px-5" @click="openCreateDialog">
        <Plus class="size-4" aria-hidden="true" />
        {{ t('mcpAccessTokens.create') }}
      </button>
    </ToolbarRoot>

    <div class="app-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <div v-else-if="status === 'error'" class="py-16 text-center text-destructive">
        <p class="text-sm">{{ error || t('mcpAccessTokens.loadFailed') }}</p>
      </div>
      <AppEmptyState v-else-if="tokens.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[760px]">
          <thead>
            <tr>
              <th>{{ t('mcpAccessTokens.name') }}</th>
              <th>{{ t('mcpAccessTokens.expiresAt') }}</th>
              <th>{{ t('mcpAccessTokens.createdAt') }}</th>
              <th class="w-16">&nbsp;</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="token in tokens" :key="token.id">
              <td class="text-foreground">{{ token.name }}</td>
              <td>
                {{
                  token.expires_at
                    ? formatTime(token.expires_at)
                    : t('mcpAccessTokens.neverExpires')
                }}
              </td>
              <td>{{ formatTime(token.created_at) }}</td>
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
          <input
            id="mcp-token-name"
            v-model="form.name"
            type="text"
            maxlength="100"
            class="app-input"
            :class="formError ? 'app-input-error' : ''"
            :aria-invalid="formError ? 'true' : undefined"
            :placeholder="t('mcpAccessTokens.namePlaceholder')"
            @input="formError = ''"
          />
          <p v-if="formError" class="app-field-error text-xs">{{ formError }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block" for="mcp-token-expiry">
            {{ t('mcpAccessTokens.expiresIn') }}
          </label>
          <select id="mcp-token-expiry" v-model.number="form.expires_in_days" class="app-input">
            <option :value="0">{{ t('mcpAccessTokens.neverExpires') }}</option>
            <option :value="30">{{ t('mcpAccessTokens.expiresIn30Days') }}</option>
            <option :value="90">{{ t('mcpAccessTokens.expiresIn90Days') }}</option>
            <option :value="365">{{ t('mcpAccessTokens.expiresIn365Days') }}</option>
          </select>
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

    <AppDialog
      :open="showCreatedDialog"
      :title="t('mcpAccessTokens.createdDialogTitle')"
      @update:open="setCreatedDialogOpen"
    >
      <div class="space-y-3">
        <p class="text-sm text-amber-800">{{ t('mcpAccessTokens.shownOnce') }}</p>
        <div class="flex gap-2">
          <input
            :value="createdToken"
            readonly
            tabindex="-1"
            class="app-input min-w-0 font-mono text-xs"
          />
          <button
            type="button"
            class="app-icon-button shrink-0"
            :title="t('mcpAccessTokens.copy')"
            :aria-label="t('mcpAccessTokens.copy')"
            @click="copyValue(createdToken)"
          >
            <Copy class="size-4" aria-hidden="true" />
          </button>
        </div>
      </div>
      <template #footer>
        <button type="button" class="app-button" @click="copyValue(environmentValue)">
          <Copy class="size-4" aria-hidden="true" />
          {{ t('mcpAccessTokens.copyEnvironment') }}
        </button>
        <button type="button" class="app-button-primary" @click="setCreatedDialogOpen(false)">
          {{ t('common.confirm') }}
        </button>
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
  import { Copy, Plus, Trash2 } from '@lucide/vue';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { authApi } from '@/api/auth/auth';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { MCPAccessTokenResp } from '@/gen/proto/orbit/v1/auth/auth';
  import { formatTime } from '@/utils/time';
  import { ToolbarRoot } from 'reka-ui';

  const { t } = useI18n();
  const toast = useToast();
  const { status, error, execute: executeLoad } = useStatusAsync();
  const { loading: operating, execute: executeOperation } = useStatusAsync();
  const tokens = ref<MCPAccessTokenResp[]>([]);
  const showCreateDialog = ref(false);
  const showCreatedDialog = ref(false);
  const showRevokeDialog = ref(false);
  const pendingRevokeId = ref('');
  const createdToken = ref('');
  const form = reactive({ name: '', expires_in_days: 0 });
  const formError = ref('');
  const createError = ref('');
  const revokeError = ref('');
  const environmentValue = computed(() => `POMELO_ORBIT_MCP__ACCESS_TOKEN=${createdToken.value}`);

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
        const response = await authApi.createMcpAccessToken({
          name: form.name.trim(),
          expires_in_days: form.expires_in_days,
        });
        createdToken.value = response.token;
        showCreateDialog.value = false;
        showCreatedDialog.value = true;
        await loadTokens();
      });
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

  async function copyValue(value: string) {
    try {
      await navigator.clipboard.writeText(value);
      toast.success(t('mcpAccessTokens.copied'));
    } catch {
      toast.error(t('mcpAccessTokens.copyFailed'));
    }
  }

  function setCreatedDialogOpen(open: boolean) {
    showCreatedDialog.value = open;
    if (!open) {
      createdToken.value = '';
    }
  }

  onMounted(loadTokens);
</script>
