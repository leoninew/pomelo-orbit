<template>
  <AppDialog
    :open="open"
    :title="t('route.syncTitle')"
    width-class="w-[min(960px,calc(100vw-32px))]"
    @update:open="handleOpenChange"
  >
    <AppLoadingState v-if="previewLoading && !syncRows.length" size="compact" />
    <div v-else-if="preview || syncRows.length" class="max-h-96 overflow-auto" aria-live="polite">
      <table v-if="syncRows.length" class="app-data-table min-w-[680px] table-fixed">
        <colgroup>
          <col class="w-[18%]" />
          <col class="w-[14%]" />
          <col class="w-[22%]" />
          <col class="w-[30%]" />
          <col class="w-[16%]" />
        </colgroup>
        <thead>
          <tr>
            <th>{{ t('route.fields.name') }}</th>
            <th>{{ t('route.syncAction') }}</th>
            <th>{{ t('route.fields.protocol') }}</th>
            <th>{{ t('route.syncRule') }}</th>
            <th>{{ t('route.syncStatus') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in syncRows" :key="item.route_id">
            <td class="align-top break-words">{{ item.route_name || item.route_id }}</td>
            <td class="align-top break-words">{{ t(`route.syncActions.${item.action}`) }}</td>
            <td class="align-top break-words">
              <p>{{ item.rule?.protocol.toUpperCase() }}</p>
              <p v-if="item.cert_type" class="text-muted-foreground">
                {{ certificateLabel(item.cert_type, item.acme_challenge) }}
              </p>
            </td>
            <td class="align-top break-words">
              <p v-if="item.rule?.match">{{ item.rule.match }}</p>
              <p v-if="item.rule?.target">-&gt; {{ item.rule.target }}</p>
            </td>
            <td class="align-top break-words">
              <AppBadge
                variant="status"
                :tone="syncStatusTone(item.status)"
                :title="item.status === 'failed' ? item.reason : undefined"
                :tabindex="item.status === 'failed' ? 0 : undefined"
              >
                {{ t(`route.syncStatuses.${item.status}`) }}
              </AppBadge>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-else class="text-sm text-muted-foreground">{{ t('route.syncEmpty') }}</p>
    </div>
    <p v-if="previewError" class="app-field-error break-words" role="alert">
      {{ previewError }}
    </p>
    <button
      v-if="!syncOperating && !preview && !previewLoading && (needsFreshPreview || previewError)"
      type="button"
      class="app-button h-9 px-3"
      @click="loadPreview(needsFreshPreview)"
    >
      <RefreshCw class="size-4" />
      {{ t('route.retryPreview') }}
    </button>
    <template #footer>
      <AppDialogActions
        v-if="syncOperating || !syncRows.length || needsFreshPreview || preview"
        :busy="syncOperating"
        :confirm-disabled="!preview || previewLoading || preview.route_ids.length === 0"
        :confirm-label="t('route.syncAll')"
        @cancel="close"
        @confirm="confirm"
      />
      <button v-else type="button" class="app-button" @click="close">
        {{ t('common.close') }}
      </button>
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { ref, watch } from 'vue';
  import { RefreshCw } from '@lucide/vue';
  import { useI18n } from 'vue-i18n';
  import { routeApi } from '@/api/route/route';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import type {
    RouteSyncChange,
    RouteSyncPreviewResp,
    RouteSyncPlanItemResp,
    RouteSyncResultResp,
  } from '@/gen/proto/orbit/v1/route/route';
  import { useProjectStore } from '@/stores/project';
  import { useToast } from '@/composables/useToast';
  import { ApiError } from '@/utils/request';
  import type { BadgeTone } from '@/utils/status';

  type SyncStatus = 'pending' | 'processing' | 'completed' | 'failed';
  interface SyncRow extends RouteSyncPlanItemResp {
    status: SyncStatus;
    reason?: string;
  }

  const props = defineProps<{
    open: boolean;
    scope: 'selected' | 'project';
    routeIds?: string[];
    changes: RouteSyncChange[];
  }>();

  const emit = defineEmits<{
    'update:open': [open: boolean];
    synced: [];
    saved: [results: RouteSyncResultResp[]];
  }>();

  const { t, te } = useI18n();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { loading: previewLoading, execute: executePreview } = useStatusAsync();
  const { loading: syncOperating, execute: executeSync } = useStatusAsync();
  const preview = ref<RouteSyncPreviewResp>();
  const previewError = ref('');
  const needsFreshPreview = ref(false);
  const syncChanges = ref<RouteSyncChange[]>([]);
  const syncRows = ref<SyncRow[]>([]);
  const retryRouteIds = ref<string[]>();
  watch(
    () => props.open,
    (open) => {
      if (open) {
        void loadPreview();
      } else {
        reset();
      }
    },
    { immediate: true }
  );

  function reset() {
    preview.value = undefined;
    previewError.value = '';
    needsFreshPreview.value = false;
    syncChanges.value = [];
    syncRows.value = [];
    retryRouteIds.value = undefined;
  }

  function close() {
    if (!syncOperating.value) {
      emit('update:open', false);
    }
  }

  function handleOpenChange(open: boolean) {
    if (!open) {
      close();
    }
  }

  async function loadPreview(savedState = false) {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('route.toast.selectProjectRequired'));
      close();
      return;
    }
    if (!savedState) {
      syncChanges.value = props.changes.map((change) => ({ ...change }));
    }
    preview.value = undefined;
    previewError.value = '';
    try {
      await executePreview(async () => {
        preview.value = await routeApi.previewSync(projectId, {
          scope: retryRouteIds.value ? 'selected' : props.scope,
          route_ids: retryRouteIds.value ?? props.routeIds ?? [],
          changes: syncChanges.value,
        });
        for (const item of preview.value.items) {
          const row = syncRows.value.find((existing) => existing.route_id === item.route_id);
          if (row) {
            Object.assign(row, item);
          } else {
            syncRows.value.push({ ...item, status: 'pending' });
          }
        }
      });
      needsFreshPreview.value = false;
    } catch (error) {
      previewError.value = error instanceof Error ? error.message : t('route.syncPreviewFailed');
    }
  }

  async function confirm() {
    const projectId = projectStore.activeProjectId;
    const currentPreview = preview.value;
    if (!projectId || !currentPreview || syncOperating.value) {
      return;
    }
    previewError.value = '';
    preview.value = undefined;
    const currentRows: SyncRow[] = [];
    for (const item of currentPreview.items) {
      let row = syncRows.value.find((existing) => existing.route_id === item.route_id);
      if (row) {
        row.status = 'pending';
        row.reason = undefined;
      } else {
        row = { ...item, status: 'pending' };
        syncRows.value.push(row);
      }
      currentRows.push(row);
    }
    await executeSync(async () => {
      for (const row of currentRows) {
        row.status = 'processing';
        try {
          const response = await routeApi.confirmSync(projectId, {
            route_ids: [row.route_id],
            publication_hash: row.publication_hash,
            changes: syncChanges.value.filter((change) => change.route_id === row.route_id),
            business_hash: row.business_hash,
          });
          const result = response.results.find((entry) => entry.route_id === row.route_id);
          if (!result) {
            throw new Error(t('route.syncFailed'));
          }
          row.status = result.code === 'route_sync_completed' ? 'completed' : 'failed';
          if (row.status === 'failed') {
            row.reason = syncFailureReason(result.code, result.error, result.cleanup);
          }
          if (result.business_save === 'saved' || row.status === 'completed') {
            syncChanges.value = syncChanges.value.filter(
              (change) => change.route_id !== row.route_id
            );
          }
          emit('saved', [result]);
        } catch (error) {
          row.status = 'failed';
          row.reason = syncFailureReason(
            error instanceof ApiError ? (error.code ?? '') : '',
            error instanceof Error ? error.message : t('route.syncFailed')
          );
        }
      }
      retryRouteIds.value = syncRows.value
        .filter((item) => item.status === 'failed')
        .map((item) => item.route_id);
      needsFreshPreview.value = retryRouteIds.value.length > 0;
      if (needsFreshPreview.value) {
        toast.error(t('route.syncIncomplete'));
      } else {
        toast.success(t('route.syncConfigurationComplete'));
        emit('synced');
      }
    });
  }

  function certificateLabel(type: string, challenge: string) {
    if (type === 'manual') {
      return 'PEM';
    }
    if (type === 'mkcert') {
      return 'mkcert';
    }
    return `Let's Encrypt · ${challenge === 'dns' ? 'DNS-01' : 'HTTP-01'}`;
  }

  function syncStatusTone(status: SyncStatus): BadgeTone {
    const tones: Record<SyncStatus, BadgeTone> = {
      pending: 'default',
      processing: 'info',
      completed: 'success',
      failed: 'error',
    };
    return tones[status];
  }

  function syncFailureReason(code: string, error: string, cleanup?: string) {
    if (cleanup === 'failed') {
      return t('route.syncCleanupFailed');
    }
    const key = `route.syncErrorCodes.${code}`;
    return te(key) ? t(key) : error || t('route.syncFailed');
  }
</script>
