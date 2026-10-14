<template>
  <AppDialog
    :open="open"
    :title="t('service.statusCheck.title')"
    width-class="w-[min(960px,calc(100vw-32px))]"
    @update:open="handleOpenChange"
  >
    <AppLoadingState v-if="loading" size="compact" :label="t('service.statusCheck.querying')" />
    <template v-else-if="preview">
      <div class="max-h-96 overflow-auto" aria-live="polite">
        <table v-if="rows.length" class="app-data-table min-w-[680px] table-fixed">
          <colgroup>
            <col />
            <col class="w-28" />
            <col class="w-28" />
            <col class="w-[16%]" />
          </colgroup>
          <thead>
            <tr>
              <th>{{ t('service.fields.code') }}</th>
              <th>{{ t('service.statusCheck.recordedStatus') }}</th>
              <th>{{ t('service.statusCheck.observedStatus') }}</th>
              <th>{{ t('service.statusCheck.result') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in rows" :key="row.service_id">
              <td class="align-top break-words">{{ row.service_code }}</td>
              <td class="align-top break-words">
                <AppBadge variant="status" :tone="appStatusTone(row.recorded_status)">
                  {{ row.recorded_status }}
                </AppBadge>
              </td>
              <td class="align-top break-words">
                <AppBadge variant="status" :tone="appStatusTone(row.observed_status)">
                  {{ row.observed_status }}
                </AppBadge>
              </td>
              <td class="align-top break-words">
                <AppTooltip :content="row.result === 'failed' ? row.reason : undefined">
                  <AppBadge
                    variant="status"
                    :tone="resultTone(row.result)"
                    :class="row.result === 'failed' ? 'cursor-help gap-1' : undefined"
                    :tabindex="row.result === 'failed' ? 0 : undefined"
                  >
                    {{ t(`service.statusCheck.results.${row.result}`) }}
                    <Info
                      v-if="row.result === 'failed'"
                      class="size-3 shrink-0"
                      aria-hidden="true"
                    />
                  </AppBadge>
                </AppTooltip>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-else class="text-sm text-muted-foreground">{{ t('service.statusCheck.empty') }}</p>
      </div>
    </template>
    <p v-if="error" class="app-field-error break-words" role="alert">{{ error }}</p>
    <template #footer>
      <AppDialogActions
        :busy="processing || loading"
        :confirm-disabled="!projectId"
        :cancel-label="t('common.close')"
        :confirm-label="
          canApply ? t('service.statusCheck.apply') : t('service.statusCheck.recheck')
        "
        @cancel="handleOpenChange(false)"
        @confirm="canApply ? apply() : loadPreview()"
      />
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { Info } from '@lucide/vue';
  import { computed, onBeforeUnmount, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { serviceApi } from '@/api/service/service';
  import AppBadge, { type BadgeTone } from '@/components/ui/AppBadge.vue';
  import AppDialog from '@/components/ui/AppDialog.vue';
  import AppDialogActions from '@/components/ui/AppDialogActions.vue';
  import AppLoadingState from '@/components/ui/AppLoadingState.vue';
  import AppTooltip from '@/components/ui/AppTooltip.vue';
  import type {
    ServiceStatusCheckItemResp,
    ServiceStatusCheckPreviewResp,
  } from '@/gen/proto/orbit/v1/service/service';
  import { ApiError } from '@/utils/request';
  import { appStatusTone } from '@/utils/status';

  type RowResult = 'pending' | 'processing' | 'completed' | 'failed' | 'unchanged';
  interface CheckRow extends ServiceStatusCheckItemResp {
    result: RowResult;
    reason?: string;
  }

  const props = defineProps<{ open: boolean; projectId: string }>();
  const emit = defineEmits<{
    'update:open': [open: boolean];
    finished: [projectId: string];
  }>();
  const { t } = useI18n();
  const preview = ref<ServiceStatusCheckPreviewResp>();
  const rows = ref<CheckRow[]>([]);
  const loading = ref(false);
  const processing = ref(false);
  const error = ref('');
  let generation = 0;

  const canApply = computed(
    () => Boolean(preview.value) && rows.value.some((row) => row.result === 'pending')
  );

  watch(
    () => [props.open, props.projectId] as const,
    ([open, projectId]) => {
      generation++;
      preview.value = undefined;
      rows.value = [];
      error.value = '';
      loading.value = false;
      processing.value = false;
      if (open && projectId) {
        void loadPreview();
      }
    },
    { immediate: true }
  );
  onBeforeUnmount(() => generation++);

  function handleOpenChange(open: boolean) {
    if (!processing.value) {
      emit('update:open', open);
    }
  }

  function resultTone(result: RowResult): BadgeTone {
    const tones: Record<RowResult, BadgeTone> = {
      pending: 'default',
      processing: 'info',
      completed: 'success',
      failed: 'error',
      unchanged: 'default',
    };
    return tones[result];
  }

  function errorMessage(cause: unknown): string {
    if (cause instanceof ApiError) {
      const messages: Record<string, string> = {
        service_status_check_expired: 'service.statusCheck.expired',
        service_status_check_changed: 'service.statusCheck.changed',
        service_status_check_unavailable: 'service.statusCheck.unavailable',
      };
      if (cause.code && messages[cause.code]) {
        return t(messages[cause.code]);
      }
    }
    return cause instanceof Error ? cause.message : t('service.statusCheck.failed');
  }

  async function loadPreview() {
    if (loading.value || processing.value || !props.open || !props.projectId) {
      return;
    }
    const current = ++generation;
    const projectId = props.projectId;
    loading.value = true;
    error.value = '';
    preview.value = undefined;
    rows.value = [];
    try {
      const result = await serviceApi.previewStatusCheck(projectId);
      if (current !== generation) {
        return;
      }
      preview.value = result;
      rows.value = result.items.map((item) => ({
        ...item,
        result: item.recorded_status === item.observed_status ? 'unchanged' : 'pending',
      }));
    } catch (cause) {
      if (current === generation) {
        error.value = errorMessage(cause);
      }
    } finally {
      if (current === generation) {
        loading.value = false;
      }
    }
  }

  async function apply() {
    if (processing.value || loading.value || !preview.value || !canApply.value) {
      return;
    }
    const current = generation;
    const projectId = props.projectId;
    const token = preview.value.token;
    const pending = rows.value.filter((row) => row.result === 'pending');
    processing.value = true;
    try {
      for (const row of pending) {
        if (current !== generation) {
          break;
        }
        row.result = 'processing';
        try {
          await serviceApi.confirmStatusCheck(projectId, { token, service_id: row.service_id });
          if (current !== generation) {
            break;
          }
          row.result = 'completed';
        } catch (cause) {
          if (current !== generation) {
            break;
          }
          row.result = 'failed';
          row.reason = errorMessage(cause);
        }
      }
    } finally {
      if (current === generation) {
        processing.value = false;
      }
      emit('finished', projectId);
    }
  }
</script>
