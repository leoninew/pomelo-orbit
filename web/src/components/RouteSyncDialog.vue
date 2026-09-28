<template>
  <AppDialog
    :open="open"
    :title="t('route.syncTitle')"
    width-class="w-[min(720px,calc(100vw-32px))]"
    @update:open="handleOpenChange"
  >
    <div v-if="route?.protocol === 'http'" class="mb-5 space-y-3 border-b border-border pb-5">
      <label for="route-sync-certificate" class="app-field-label block">
        {{ t('route.syncCertificateMode') }}
      </label>
      <SelectControl
        id="route-sync-certificate"
        :model-value="selectedMode"
        :options="certificateOptions"
        :disabled="previewLoading || syncOperating"
        :invalid="Boolean(certificateError)"
        @update:model-value="updateCertificateMode"
      />
      <div v-if="selectedMode === 'manual'" class="flex flex-wrap items-center gap-3">
        <button
          type="button"
          class="app-button h-9 px-3"
          :disabled="previewLoading || syncOperating"
          @click="certificateInput?.click()"
        >
          <Upload class="size-4" />
          {{ t('route.selectPemFile') }}
        </button>
        <span class="text-sm text-muted-foreground">
          {{
            certificateFileName ||
            t(currentMode === 'manual' ? 'route.savedPemHint' : 'route.pemFileHint')
          }}
        </span>
        <input
          ref="certificateInput"
          type="file"
          accept=".pem"
          class="hidden"
          @change="readCertificateFile"
        />
      </div>
      <p v-if="certificateError" class="app-field-error text-xs" role="alert">
        {{ certificateError }}
      </p>
    </div>
    <AppLoadingState v-if="previewLoading" size="compact" />
    <div v-else-if="preview" class="space-y-4">
      <p class="text-sm text-foreground">
        {{ preview.matched ? t('route.syncMatched') : t('route.syncDifferencesFound') }}
      </p>
      <div v-if="!preview.matched" class="overflow-x-auto">
        <table class="app-data-table min-w-[640px] table-fixed">
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
                <p>{{ row.rule }}</p>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-if="preview.pending.length" class="space-y-2">
        <p class="text-sm font-medium text-foreground">{{ t('route.syncPendingConfig') }}</p>
        <ul class="space-y-1 text-sm text-muted-foreground">
          <li v-for="item in preview.pending" :key="item.route_name">
            {{ item.route_name }}: {{ pendingCertificateLabel(item.mode, item.challenge) }}
            <span v-if="!item.publishes_now">({{ t('route.syncDisabledConfig') }})</span>
          </li>
        </ul>
      </div>
      <p v-if="syncChanges.length" class="text-sm text-muted-foreground">
        {{ t('route.syncPendingChanges', { count: syncChanges.length }) }}
      </p>
      <p v-if="savedChanges" class="text-sm text-muted-foreground">
        {{ t('route.syncSavedChanges') }}
      </p>
      <p v-if="!preview.matched" class="text-sm text-amber-800 dark:text-amber-100">
        {{ t('route.syncOverwriteWarning') }}
      </p>
      <p v-if="previewError" class="app-field-error" role="alert">{{ previewError }}</p>
      <p v-if="syncError" class="app-field-error" role="alert">{{ syncError }}</p>
    </div>
    <p v-else-if="previewError" class="app-field-error" role="alert">{{ previewError }}</p>
    <p v-if="syncError && !preview" class="app-field-error" role="alert">{{ syncError }}</p>
    <template #footer>
      <AppDialogActions
        :busy="syncOperating"
        :confirm-disabled="!preview || previewLoading || Boolean(certificateError)"
        :confirm-label="t('route.syncAll')"
        @cancel="close"
        @confirm="confirm"
      />
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';
  import { Upload } from '@lucide/vue';
  import { useI18n } from 'vue-i18n';
  import { routeApi } from '@/api/route/route';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import type {
    RouteResp,
    RouteSyncCertificateChange,
    RouteSyncChange,
    RouteSyncPreviewResp,
  } from '@/gen/proto/orbit/v1/route/route';
  import { useProjectStore } from '@/stores/project';
  import { useToast } from '@/composables/useToast';
  import { ApiError } from '@/utils/request';

  const props = defineProps<{
    open: boolean;
    changes: RouteSyncChange[];
    route?: RouteResp;
    savedChanges?: boolean;
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
  const syncChanges = ref<RouteSyncChange[]>([]);
  type CertificateMode = 'http' | 'letsencrypt-http' | 'letsencrypt-dns' | 'mkcert' | 'manual';
  const selectedMode = ref<CertificateMode>('http');
  const certificateInput = ref<HTMLInputElement>();
  const certificateFileName = ref('');
  const certificatePEM = ref('');
  const certificateError = ref('');
  const currentMode = computed<CertificateMode>(() => {
    const route = props.route;
    if (!route?.https_enabled) {
      return 'http';
    }
    if (route.cert_type === 'letsencrypt') {
      return route.acme_challenge === 'dns' ? 'letsencrypt-dns' : 'letsencrypt-http';
    }
    return route.cert_type === 'mkcert' ? 'mkcert' : 'manual';
  });
  const canUseLetsEncrypt = computed(() => {
    const domain = props.route?.domain ?? '';
    return (
      domain !== 'localhost' &&
      !domain.endsWith('.localhost') &&
      !domain.endsWith('.lvh.me') &&
      !/^\d+\.\d+\.\d+\.\d+$/.test(domain)
    );
  });
  const certificateOptions = computed(() => [
    { value: 'http', label: 'HTTP' },
    {
      value: 'letsencrypt-http',
      label: `HTTPS · Let's Encrypt · HTTP-01`,
      disabled: !canUseLetsEncrypt.value || !props.route?.http01_available,
    },
    {
      value: 'letsencrypt-dns',
      label: `HTTPS · Let's Encrypt · DNS-01`,
      disabled: !canUseLetsEncrypt.value || !props.route?.dns01_available,
    },
    { value: 'mkcert', label: 'HTTPS · mkcert' },
    { value: 'manual', label: 'HTTPS · PEM' },
  ]);
  const certificateChange = computed<RouteSyncCertificateChange | undefined>(() => {
    if (!props.route || (selectedMode.value === currentMode.value && !certificatePEM.value)) {
      return undefined;
    }
    const mode = selectedMode.value;
    return {
      mode: mode.startsWith('letsencrypt') ? 'letsencrypt' : mode,
      challenge: mode === 'letsencrypt-dns' ? 'dns' : mode === 'letsencrypt-http' ? 'http' : '',
      pem: mode === 'manual' ? certificatePEM.value : '',
    };
  });
  const previewRows = computed(() =>
    (preview.value?.differences ?? []).flatMap((difference) => {
      const rows: Array<{ source: 'customRoute' | 'dockerLabel'; rule: string }> = [];
      if (difference.business_value) {
        rows.push({ source: 'customRoute', rule: difference.business_value });
      }
      if (difference.traefik_value) {
        rows.push({ source: 'dockerLabel', rule: difference.traefik_value });
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
        selectedMode.value = currentMode.value;
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
    syncChanges.value = [];
    certificatePEM.value = '';
    certificateFileName.value = '';
    certificateError.value = '';
  }

  function close() {
    emit('update:open', false);
  }

  function handleOpenChange(open: boolean) {
    if (!open) {
      close();
    }
  }

  function updateCertificateMode(value: string | number) {
    if (!certificateOptions.value.some((option) => option.value === value && !option.disabled)) {
      return;
    }
    selectedMode.value = value as CertificateMode;
    certificatePEM.value = '';
    certificateFileName.value = '';
    certificateError.value = '';
    void loadPreview();
  }

  function pendingCertificateLabel(mode: string, challenge: string) {
    if (mode === 'http') {
      return 'HTTP';
    }
    if (mode === 'letsencrypt') {
      return `HTTPS · Let's Encrypt · ${challenge.toUpperCase()}-01`;
    }
    return mode === 'mkcert' ? 'HTTPS · mkcert' : 'HTTPS · PEM';
  }

  async function readCertificateFile(event: Event) {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) {
      return;
    }
    try {
      const content = await file.text();
      if (selectedMode.value !== 'manual') {
        return;
      }
      certificateFileName.value = file.name;
      certificatePEM.value = content;
      certificateError.value = '';
      void loadPreview();
    } catch {
      certificateError.value = t('route.pemReadFailed');
    } finally {
      input.value = '';
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
    if (!savedState && props.route && certificateChange.value) {
      const existing = syncChanges.value.find((change) => change.route_id === props.route?.id);
      if (existing) {
        existing.certificate = certificateChange.value;
      } else {
        syncChanges.value.push({ route_id: props.route.id, certificate: certificateChange.value });
      }
    }
    preview.value = undefined;
    previewError.value = '';
    if (certificateChange.value?.mode === 'manual' && !certificatePEM.value) {
      certificateError.value = t('route.pemRequired');
      return;
    }
    try {
      await executePreview(async () => {
        preview.value = await routeApi.previewSync(projectId, { changes: syncChanges.value });
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
      syncError.value = error instanceof Error ? error.message : t('route.syncFailed');
      if (error instanceof ApiError && error.code === 'route_sync_publish_failed') {
        emit('saved');
        await loadPreview(true);
      } else if (error instanceof ApiError && error.code === 'route_sync_preview_expired') {
        await loadPreview();
      }
    }
  }
</script>
