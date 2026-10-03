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
          <col class="w-28" />
          <col class="w-28" />
          <col class="w-28" />
          <col />
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
              <AppTooltip
                :content="
                  item.cert_type ? certificateLabel(item.cert_type, item.acme_challenge) : undefined
                "
              >
                <span
                  class="inline-flex items-center gap-1 whitespace-nowrap"
                  :class="item.cert_type ? 'cursor-help' : undefined"
                  :tabindex="item.cert_type ? 0 : undefined"
                >
                  {{ item.rule?.protocol.toUpperCase() }}
                  <Info v-if="item.cert_type" class="size-3 shrink-0" aria-hidden="true" />
                </span>
              </AppTooltip>
            </td>
            <td class="align-top break-words">
              <p v-if="item.rule?.match">{{ item.rule.match }}</p>
              <p v-if="item.rule?.target">-&gt; {{ item.rule.target }}</p>
            </td>
            <td class="align-top break-words">
              <AppTooltip :content="item.status === 'failed' ? item.reason : undefined">
                <AppBadge
                  variant="status"
                  :tone="syncStatusTone(item.status)"
                  :class="item.status === 'failed' ? 'cursor-help gap-1' : undefined"
                  :tabindex="item.status === 'failed' ? 0 : undefined"
                >
                  {{ t(`route.syncStatuses.${item.status}`) }}
                  <Info
                    v-if="item.status === 'failed'"
                    class="size-3 shrink-0"
                    aria-hidden="true"
                  />
                </AppBadge>
              </AppTooltip>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-else class="text-sm text-muted-foreground">{{ t('route.syncEmpty') }}</p>
    </div>
    <p v-if="previewError" class="app-field-error break-words" role="alert">
      {{ previewError }}
    </p>
    <template #footer>
      <AppDialogActions
        v-if="syncOperating || !syncRows.length || needsFreshPreview || preview"
        :busy="syncOperating"
        :confirm-disabled="
          previewLoading || (!canRetryPreview && (!preview || preview.route_ids.length === 0))
        "
        :confirm-label="canRetryPreview ? t('route.retryPreview') : t('route.syncAll')"
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
  import { computed, ref, watch } from 'vue';
  import { Info } from '@lucide/vue';
  import { useI18n } from 'vue-i18n';
  import { routeApi } from '@/api/route/route';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import AppTooltip from '@/components/AppTooltip.vue';
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
    finished: [];
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
  const canRetryPreview = computed(
    () => !preview.value && (needsFreshPreview.value || previewError.value !== '')
  );
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
      previewError.value = syncFailureReason(
        error instanceof ApiError ? (error.code ?? '') : '',
        error instanceof Error ? error.message : t('route.syncPreviewFailed')
      );
      toast.error(previewError.value);
    }
  }

  async function confirm() {
    if (syncOperating.value || previewLoading.value) {
      return;
    }
    if (canRetryPreview.value) {
      await loadPreview(needsFreshPreview.value);
      return;
    }
    const projectId = projectStore.activeProjectId;
    const currentPreview = preview.value;
    if (!projectId || !currentPreview) {
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
    try {
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
              row.reason = syncFailureReason(
                result.code,
                result.error,
                result.cleanup,
                result.recovery
              );
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
          if (row.status === 'failed') {
            toast.error(`${row.route_name || row.route_id}: ${row.reason}`);
          }
        }
        retryRouteIds.value = syncRows.value
          .filter((item) => item.status === 'failed')
          .map((item) => item.route_id);
        needsFreshPreview.value = retryRouteIds.value.length > 0;
        if (!needsFreshPreview.value) {
          toast.success(t('route.syncConfigurationComplete'));
          emit('synced');
        }
      });
    } finally {
      emit('finished');
    }
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

  function syncFailureReason(code: string, error: string, cleanup?: string, recovery?: string) {
    if (cleanup === 'failed') {
      return t('route.syncCleanupFailed');
    }
    const key = `route.syncErrorCodes.${code}`;
    const reason = te(key) ? t(key) : error || t('route.syncFailed');
    return recovery === 'failed' ? t('route.syncRecoveryFailed', { reason }) : reason;
  }
</script>
