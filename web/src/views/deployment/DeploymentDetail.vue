<template>
  <div class="flex h-full min-h-0 flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <DetailPageHeader :items="[]" title="部署详情" />
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-if="deployment && !isCompleteDeployment"
          type="button"
          class="app-icon-button size-9"
          :class="isAutoRefreshing ? 'text-primary' : ''"
          :title="t(isAutoRefreshing ? 'logs.pauseDetailRefresh' : 'logs.resumeDetailRefresh')"
          :aria-label="t(isAutoRefreshing ? 'logs.pauseDetailRefresh' : 'logs.resumeDetailRefresh')"
          :aria-pressed="isAutoRefreshing"
          @click="toggleAutoRefresh"
        >
          <Loader2 class="size-4" :class="isAutoRefreshing ? 'animate-spin' : ''" />
        </button>
        <button
          v-if="isCancelable"
          class="app-button-danger h-9 px-3"
          :disabled="isCancelling"
          @click="openCancelDialog"
        >
          <X class="size-4" />
          取消
        </button>
        <button
          v-if="deployment && isCompleteDeployment"
          class="app-button-danger h-9 px-3"
          :disabled="isDeleting"
          @click="openDeleteDialog"
        >
          <Trash2 class="size-4" />
          {{ t('common.delete') }}
        </button>
        <button class="app-button h-9 px-4" @click="goBack">
          <ArrowLeft class="size-4" />
          {{ backButtonText }}
        </button>
      </div>
    </div>

    <AppLoadingState v-if="status === 'loading'" size="section" />

    <div v-else-if="deployment" class="flex min-h-0 flex-1 flex-col gap-4">
      <DetailInfoCard class="shrink-0" title="部署概况">
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt>部署 ID</dt>
            <dd class="min-w-0 break-all font-mono text-xs text-foreground">{{ deploymentId }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('deployment.fields.service') }}</dt>
            <dd v-if="deployment.service_id">
              <router-link :to="`/service/${deployment.service_id}`" class="app-link">
                {{ deploymentServiceLabel }}
              </router-link>
            </dd>
            <dd v-else class="text-muted-foreground">-</dd>
          </div>
          <div class="flex gap-2">
            <dt>状态</dt>
            <dd>
              <AppBadge variant="pill" :tone="deploymentStatusTone">
                {{ deployment.status }}
              </AppBadge>
            </dd>
          </div>
        </dl>
      </DetailInfoCard>

      <DetailInfoCard class="shrink-0" title="执行配置">
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt>操作类型</dt>
            <dd>
              <AppBadge variant="pill">{{ deployment.operation_type }}</AppBadge>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>触发方式</dt>
            <dd>
              <AppBadge variant="pill">{{ deployment.trigger_type }}</AppBadge>
            </dd>
          </div>
          <div class="flex gap-2 sm:col-span-2">
            <dt>执行命令</dt>
            <dd class="min-w-0 break-all text-xs text-foreground">
              {{ deployment.command_text || '未记录' }}
            </dd>
          </div>
        </dl>
      </DetailInfoCard>

      <DetailInfoCard class="shrink-0" title="时间与结果">
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt>创建时间</dt>
            <dd class="text-muted-foreground">{{ formatTime(deployment.created_at) }}</dd>
          </div>
          <div v-if="deployment.started_at" class="flex gap-2">
            <dt>开始时间</dt>
            <dd class="text-muted-foreground">{{ formatTime(deployment.started_at) }}</dd>
          </div>
          <div v-if="deployment.finished_at" class="flex gap-2">
            <dt>完成时间</dt>
            <dd class="text-muted-foreground">{{ formatTime(deployment.finished_at) }}</dd>
          </div>
          <div v-if="deployment.started_at && deployment.finished_at" class="flex gap-2">
            <dt>耗时</dt>
            <dd class="text-muted-foreground">
              {{ formatDuration(deployment.started_at, deployment.finished_at) }}
            </dd>
          </div>
          <div v-if="deployment.error_message" class="flex gap-2 sm:col-span-2">
            <dt>错误信息</dt>
            <dd class="min-w-0 text-destructive">
              <pre class="whitespace-pre-wrap break-words font-sans text-xs">{{
                deployment.error_message
              }}</pre>
            </dd>
          </div>
        </dl>
      </DetailInfoCard>

      <DetailInfoCard class="flex min-h-[360px] min-w-0 flex-1 flex-col" title="操作日志">
        <template #actions>
          <LogActions
            :state="operationLogState"
            @find="operationLogView?.find()"
            @follow="operationLogView?.follow()"
          />
          <button
            v-if="deployment.operation_type !== 'stop'"
            type="button"
            class="app-icon-button size-8"
            :title="t('service.logs.title')"
            :aria-label="t('service.logs.title')"
            @click="isContainerLogDrawerOpen = true"
          >
            <ScrollText class="size-4" />
          </button>
        </template>
        <LogView ref="operationLogView" :state="operationLogState" />
      </DetailInfoCard>
    </div>

    <LogDrawer
      v-model:open="isContainerLogDrawerOpen"
      :title="t('service.logs.title')"
      :state="containerLogState"
    />

    <AppDialog
      v-model:open="isCancelDialogOpen"
      :title="t('deployment.dialog.confirmCancel')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">
        {{
          t('deployment.dialog.cancelConfirm', {
            name: deploymentServiceLabel || t('deployment.dialog.currentService'),
          })
        }}
      </p>
      <p v-if="cancelSubmitError" class="app-field-error mt-3" role="alert">
        {{ cancelSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="isCancelling"
          :confirm-label="t('common.confirm')"
          variant="destructive"
          @cancel="isCancelDialogOpen = false"
          @confirm="handleCancel"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteDialogOpen"
      :title="t('deployment.dialog.confirmDelete')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">
        {{ t('deployment.dialog.deleteConfirm', { id: deploymentId }) }}
      </p>
      <p v-if="deleteSubmitError" class="app-field-error mt-3" role="alert">
        {{ deleteSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="isDeleting"
          :confirm-label="t('common.delete')"
          variant="destructive"
          @cancel="closeDeleteDialog"
          @confirm="handleDelete"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Loader2, ScrollText, Trash2, X } from '@lucide/vue';
  import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { deploymentApi } from '@/api/deployment/deployment';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import DetailPageHeader from '@/components/DetailPageHeader.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import LogView from '@/components/LogView.vue';
  import LogActions from '@/components/LogActions.vue';
  import LogDrawer from '@/components/LogDrawer.vue';
  import { provideLogStreamCache, useLogStream } from '@/composables/useLogStream';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { DeploymentResp } from '@/gen/proto/orbit/v1/deployment/deployment';
  import { useProjectStore } from '@/stores/project';
  import { isComplete, statusTone } from '@/utils/status';
  import { delayAsync, formatDuration, formatTime } from '@/utils/time';

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const deploymentId = computed(() => String(route.params.id ?? ''));
  const toast = useToast();
  const projectStore = useProjectStore();
  const { status, execute } = useStatusAsync();
  const { loading: isCancelling, execute: executeCancel } = useStatusAsync();
  const { loading: isDeleting, execute: executeDelete } = useStatusAsync();

  const deployment = ref<DeploymentResp>();
  const isContainerLogDrawerOpen = ref(false);
  const operationLogView = ref<InstanceType<typeof LogView>>();
  const deploymentServiceLabel = computed(
    () =>
      deployment.value?.application_name ||
      deployment.value?.application_id ||
      deployment.value?.service_id ||
      ''
  );
  const logCache = provideLogStreamCache(
    () => `${projectStore.activeProjectId}:${deploymentId.value}`
  );
  const operationLogResource = computed(() =>
    projectStore.activeProjectId && deployment.value
      ? {
          projectId: projectStore.activeProjectId,
          path: `/api/deployment/${deploymentId.value}/log/stream`,
          kind: 'file' as const,
        }
      : undefined
  );
  const containerLogResource = computed(() =>
    projectStore.activeProjectId && deployment.value && deployment.value.operation_type !== 'stop'
      ? {
          projectId: projectStore.activeProjectId,
          path: `/api/deployment/${deploymentId.value}/container-log/stream`,
          kind: 'container' as const,
        }
      : undefined
  );
  const { state: operationLogState } = useLogStream(operationLogResource, () => !!deployment.value);
  const { state: containerLogState } = useLogStream(
    containerLogResource,
    () => isContainerLogDrawerOpen.value
  );
  const isCancelDialogOpen = ref(false);
  const cancelSubmitError = ref('');
  const isDeleteDialogOpen = ref(false);
  const deleteSubmitError = ref('');
  const isAutoRefreshing = ref(false);
  let refreshAbort: AbortController | null = null;
  let refreshGeneration = 0;

  function selectedProjectId() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      throw new Error(t('application.toast.selectProjectRequired'));
    }
    return projectId;
  }

  const backButtonText = computed(() => {
    if (route.query.from === 'application') {
      return '返回应用';
    }
    if (route.query.from === 'versions') {
      return '返回版本';
    }
    return '返回';
  });

  function goBack() {
    if (route.query.from === 'application' && deployment.value?.application_id) {
      router.push(`/application/${deployment.value.application_id}`);
      return;
    }
    if (route.query.from === 'versions' && deployment.value?.application_id) {
      router.push({
        path: '/versions',
        query: { application_id: deployment.value.application_id },
      });
      return;
    }
    router.push('/deployments');
  }

  const deploymentStatusTone = computed(() =>
    deployment.value ? statusTone(deployment.value.status) : 'default'
  );

  const isCompleteDeployment = computed(() => isComplete(deployment.value?.status ?? ''));
  const isCancelable = computed(() =>
    deployment.value ? !isComplete(deployment.value.status) : false
  );
  function isCurrentRefresh(generation: number, signal: AbortSignal) {
    return !signal.aborted && generation === refreshGeneration;
  }

  function stopAutoRefresh() {
    refreshGeneration++;
    refreshAbort?.abort();
    refreshAbort = null;
    isAutoRefreshing.value = false;
  }

  function startAutoRefresh() {
    if (isAutoRefreshing.value || !deployment.value || isCompleteDeployment.value) {
      return;
    }
    const generation = ++refreshGeneration;
    const controller = new AbortController();
    const { signal } = controller;
    refreshAbort = controller;
    isAutoRefreshing.value = true;
    void (async () => {
      while (isCurrentRefresh(generation, signal)) {
        await delayAsync(2000, signal);
        if (!isCurrentRefresh(generation, signal)) {
          break;
        }
        try {
          const data = await deploymentApi.get(selectedProjectId(), deploymentId.value, { signal });
          if (!isCurrentRefresh(generation, signal)) {
            break;
          }
          deployment.value = data;
          if (isComplete(data.status)) {
            break;
          }
        } catch {
          // Continue refreshing after a transient detail request failure.
        }
      }
      if (isCurrentRefresh(generation, signal)) {
        refreshAbort = null;
        isAutoRefreshing.value = false;
      }
    })();
  }

  function toggleAutoRefresh() {
    if (isAutoRefreshing.value) {
      stopAutoRefresh();
      return;
    }
    startAutoRefresh();
  }

  function resetState() {
    stopAutoRefresh();
    isContainerLogDrawerOpen.value = false;
    deployment.value = undefined;
    logCache.clear();
    isCancelDialogOpen.value = false;
    isDeleteDialogOpen.value = false;
  }

  async function loadDeployment() {
    resetState();
    try {
      const currentDeployment = await execute(() =>
        deploymentApi.get(selectedProjectId(), deploymentId.value)
      );
      deployment.value = currentDeployment;
    } catch {
      toast.error(t('deployment.toast.loadFailed'));
      router.push('/deployments');
      return;
    }
    if (!isCompleteDeployment.value) {
      startAutoRefresh();
    }
  }

  async function handleCancel() {
    cancelSubmitError.value = '';
    try {
      await executeCancel(async () => {
        deployment.value = await deploymentApi.cancel(selectedProjectId(), deploymentId.value, {});
        stopAutoRefresh();
        isCancelDialogOpen.value = false;
        toast.success(t('deployment.toast.cancelSuccess'));
      });
    } catch {
      cancelSubmitError.value = t('deployment.toast.cancelFailed');
    }
  }

  function openCancelDialog() {
    cancelSubmitError.value = '';
    isCancelDialogOpen.value = true;
  }

  function openDeleteDialog() {
    deleteSubmitError.value = '';
    isDeleteDialogOpen.value = true;
  }

  function closeDeleteDialog() {
    isDeleteDialogOpen.value = false;
    deleteSubmitError.value = '';
  }

  async function handleDelete() {
    deleteSubmitError.value = '';
    try {
      await executeDelete(async () => {
        await deploymentApi.delete(selectedProjectId(), deploymentId.value);
        stopAutoRefresh();
        toast.success(t('deployment.toast.deleteSuccess'));
        await router.replace('/deployments');
      });
    } catch (error) {
      deleteSubmitError.value =
        error instanceof Error ? error.message : t('deployment.toast.deleteFailed');
    }
  }

  watch([deploymentId, () => projectStore.activeProjectId], loadDeployment);

  onMounted(loadDeployment);
  onUnmounted(stopAutoRefresh);
</script>
