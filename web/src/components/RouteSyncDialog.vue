<template>
  <AppDialog
    :open="open"
    :title="t('route.syncTitle')"
    width-class="w-[min(720px,calc(100vw-32px))]"
    @update:open="handleOpenChange"
  >
    <AppLoadingState v-if="previewLoading" size="compact" />
    <div v-else-if="preview" class="space-y-4">
      <p class="text-sm text-foreground">
        {{ preview.matched ? t('route.syncMatched') : t('route.syncDifferencesFound') }}
      </p>
      <ul v-if="preview.matched" class="list-disc pl-5 text-sm text-muted-foreground">
        <li>{{ t('route.syncPendingChanges', { count: syncChanges.length }) }}</li>
      </ul>
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[640px] table-fixed">
          <colgroup>
            <col class="w-[16%]" />
            <col class="w-[84%]" />
          </colgroup>
          <thead>
            <tr>
              <th>{{ t('route.syncAction') }}</th>
              <th>{{ t('route.syncRule') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="difference in preview.differences" :key="difference.route_name">
              <td class="align-top break-words text-foreground">
                <p>{{ t(`route.syncActions.${difference.action}`) }}</p>
              </td>
              <td class="align-top">
                <div class="space-y-1 break-words text-foreground">
                  <p v-if="difference.business_value">
                    <span class="text-muted-foreground">{{ t('route.syncBusinessValue') }}</span>
                    {{ difference.business_value }}
                  </p>
                  <p v-if="difference.traefik_value">
                    <span class="text-muted-foreground">{{ t('route.syncTraefikValue') }}</span>
                    {{ difference.traefik_value }}
                  </p>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-if="!preview.matched" class="text-sm text-amber-800 dark:text-amber-100">
        {{ t('route.syncOverwriteWarning') }}
      </p>
      <p v-if="previewError" class="app-field-error" role="alert">{{ previewError }}</p>
    </div>
    <p v-else-if="previewError" class="app-field-error" role="alert">{{ previewError }}</p>
    <template #footer>
      <AppDialogActions
        :busy="syncOperating"
        :confirm-disabled="!preview"
        @cancel="close"
        @confirm="confirm"
      />
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { routeApi } from '@/api/route/route';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import type { RouteSyncChange, RouteSyncPreviewResp } from '@/gen/proto/orbit/v1/route/route';
  import { useProjectStore } from '@/stores/project';
  import { useToast } from '@/composables/useToast';

  const props = defineProps<{
    open: boolean;
    changes: RouteSyncChange[];
  }>();

  const emit = defineEmits<{
    'update:open': [open: boolean];
    synced: [];
  }>();

  const { t } = useI18n();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { loading: previewLoading, execute: executePreview } = useStatusAsync();
  const { loading: syncOperating, execute: executeSync } = useStatusAsync();
  const preview = ref<RouteSyncPreviewResp>();
  const previewError = ref('');
  const syncChanges = ref<RouteSyncChange[]>([]);

  watch(
    () => props.open,
    (open) => {
      if (open) {
        void loadPreview();
      } else {
        reset();
      }
    }
  );

  function reset() {
    preview.value = undefined;
    previewError.value = '';
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

  async function loadPreview() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('route.toast.selectProjectRequired'));
      close();
      return;
    }
    syncChanges.value = props.changes.map((change) => ({ ...change }));
    preview.value = undefined;
    previewError.value = '';
    try {
      await executePreview(async () => {
        preview.value = await routeApi.previewSync(
          { changes: syncChanges.value },
          { project_id: projectId }
        );
      });
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
    try {
      await executeSync(async () => {
        await routeApi.confirmSync(
          {
            changes: syncChanges.value,
            business_hash: currentPreview.business_hash,
            traefik_hash: currentPreview.traefik_hash,
          },
          { project_id: projectId }
        );
        toast.success(t('route.syncSuccess'));
        emit('synced');
        close();
      });
    } catch (error) {
      previewError.value = error instanceof Error ? error.message : t('route.syncFailed');
    }
  }
</script>
