<template>
  <AppDialog
    :open="open"
    :title="t('route.syncTitle')"
    width-class="w-[min(760px,calc(100vw-32px))]"
    @update:open="handleOpenChange"
  >
    <AppLoadingState v-if="previewLoading" size="compact" />
    <div v-else-if="preview" class="space-y-3">
      <p v-if="preview.matched" class="text-sm text-muted-foreground">
        {{ t(certificatePending ? 'route.syncCertificatePending' : 'route.syncMatched') }}
      </p>
      <div v-else class="max-h-64 overflow-auto">
        <table class="app-data-table min-w-[600px] table-fixed">
          <colgroup>
            <col class="w-[16%]" />
            <col class="w-[20%]" />
            <col class="w-[64%]" />
          </colgroup>
          <thead>
            <tr>
              <th>{{ t('route.syncAction') }}</th>
              <th>{{ t('route.syncSource') }}</th>
              <th>{{ t('route.syncRule') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in previewRows" :key="row.key">
              <td
                v-if="row.showAction"
                :rowspan="row.actionRowspan"
                class="align-top break-words text-foreground"
              >
                <p>{{ t(`route.syncActions.${row.action}`) }}</p>
              </td>
              <td class="align-top break-words text-foreground">
                <p>{{ t(`route.syncSources.${row.source}`) }}</p>
              </td>
              <td class="align-top break-words text-foreground">
                <p v-if="row.rule.match">{{ row.rule.match }}</p>
                <p v-if="row.rule.target">{{ row.rule.match ? '-> ' : '' }}{{ row.rule.target }}</p>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    <p v-if="syncError" class="app-field-error break-words" role="alert">{{ syncError }}</p>
    <p v-if="previewError" class="app-field-error break-words" role="alert">
      {{ previewError }}
    </p>
    <button
      v-if="!preview && !previewLoading && (needsFreshPreview || previewError)"
      type="button"
      class="app-button h-9 px-3"
      @click="loadPreview(needsFreshPreview)"
    >
      <RefreshCw class="size-4" />
      {{ t('route.retryPreview') }}
    </button>
    <template #footer>
      <AppDialogActions
        :busy="syncOperating"
        :confirm-disabled="!preview || previewLoading"
        :confirm-label="t('route.syncAll')"
        @cancel="close"
        @confirm="confirm"
      />
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';
  import { RefreshCw } from '@lucide/vue';
  import { useI18n } from 'vue-i18n';
  import { routeApi } from '@/api/route/route';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import type {
    RouteSyncChange,
    RouteSyncPreviewResp,
    RouteSyncRuleResp,
  } from '@/gen/proto/orbit/v1/route/route';
  import { useProjectStore } from '@/stores/project';
  import { useToast } from '@/composables/useToast';
  import { ApiError } from '@/utils/request';

  const props = defineProps<{
    open: boolean;
    changes: RouteSyncChange[];
    certificatePending?: boolean;
  }>();

  const emit = defineEmits<{
    'update:open': [open: boolean];
    synced: [];
    saved: [];
  }>();

  const { t } = useI18n();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { loading: previewLoading, execute: executePreview } = useStatusAsync();
  const { loading: syncOperating, execute: executeSync } = useStatusAsync();
  const preview = ref<RouteSyncPreviewResp>();
  const previewError = ref('');
  const syncError = ref('');
  const needsFreshPreview = ref(false);
  const syncChanges = ref<RouteSyncChange[]>([]);
  const previewRows = computed(() =>
    (preview.value?.differences ?? []).flatMap((difference) => {
      const rows: Array<{ source: 'customRoute' | 'dockerLabel'; rule: RouteSyncRuleResp }> = [];
      if (difference.business) {
        rows.push({ source: 'customRoute', rule: difference.business });
      }
      if (difference.traefik) {
        rows.push({ source: 'dockerLabel', rule: difference.traefik });
      }
      return rows.map((row, index) => ({
        ...row,
        key: `${difference.route_name}-${difference.field}-${row.source}`,
        action: difference.action,
        showAction: index === 0,
        actionRowspan: rows.length,
      }));
    })
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
    syncError.value = '';
    needsFreshPreview.value = false;
    syncChanges.value = [];
  }

  function close() {
    emit('update:open', false);
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
    syncChanges.value = savedState ? [] : props.changes.map((change) => ({ ...change }));
    preview.value = undefined;
    previewError.value = '';
    try {
      await executePreview(async () => {
        preview.value = await routeApi.previewSync(projectId, { changes: syncChanges.value });
      });
      needsFreshPreview.value = false;
      syncError.value = '';
    } catch (error) {
      previewError.value = error instanceof Error ? error.message : t('route.syncPreviewFailed');
    }
  }

  async function confirm() {
    const projectId = projectStore.activeProjectId;
    const currentPreview = preview.value;
    if (!projectId || !currentPreview) {
      return;
    }
    previewError.value = '';
    syncError.value = '';
    try {
      await executeSync(async () => {
        await routeApi.confirmSync(projectId, {
          changes: syncChanges.value,
          business_hash: currentPreview.business_hash,
          traefik_hash: currentPreview.traefik_hash,
        });
        toast.success(t('route.syncSuccess'));
        emit('synced');
        close();
      });
    } catch (error) {
      if (
        error instanceof ApiError &&
        (error.code === 'route_sync_publish_failed' ||
          error.code === 'route_sync_publish_permission_denied')
      ) {
        syncError.value = t(
          error.code === 'route_sync_publish_permission_denied'
            ? 'route.syncPermissionDenied'
            : 'route.syncPublishFailed'
        );
        if (error.requestId) {
          syncError.value += ` (${t('route.syncRequestId')}: ${error.requestId})`;
        }
        preview.value = undefined;
        needsFreshPreview.value = true;
        emit('saved');
      } else if (error instanceof ApiError && error.code === 'route_sync_preview_expired') {
        syncError.value = error.message;
        await loadPreview();
      } else {
        syncError.value = error instanceof Error ? error.message : t('route.syncFailed');
      }
    }
  }
</script>
