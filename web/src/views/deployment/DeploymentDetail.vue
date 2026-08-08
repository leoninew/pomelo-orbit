<template>
  <div class="flex h-full min-h-0 flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <h1 class="app-detail-page-title min-w-0 break-words">部署详情</h1>
        <DetailHeaderMeta v-if="deployment">
          <AppBadge variant="status" :tone="deploymentStatusTone">{{ deployment.status }}</AppBadge>
        </DetailHeaderMeta>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-if="isCancelable"
          class="app-button-danger h-9 px-3"
          :disabled="isCancelling"
          @click="openCancelDialog"
        >
          <X class="size-4" />
          取消
        </button>
        <button class="app-button h-9 px-4" @click="goBack">
          <ArrowLeft class="size-4" />
          {{ backButtonText }}
        </button>
      </div>
    </div>

    <!-- 加载状态 -->
    <AppLoadingState v-if="status === 'loading'" size="section" />

    <!-- 内容 -->
    <div v-else-if="deployment" class="flex min-h-0 flex-1 flex-col gap-4">
      <!-- 基本信息卡片 -->
      <div class="app-surface shrink-0 app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">基本信息</h2>
        </div>
        <dl class="app-detail-info-grid">
          <div v-if="deployment.application_id" class="flex gap-2">
            <dt>部署 ID</dt>
            <dd class="min-w-0 break-all text-foreground">{{ deploymentId }}</dd>
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
          <div class="flex gap-2">
            <dt>耗时</dt>
            <dd class="text-muted-foreground">
              {{ formatDuration(deployment.started_at, deployment.finished_at) }}
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>创建时间</dt>
            <dd class="text-muted-foreground">{{ formatTime(deployment.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>开始时间</dt>
            <dd class="text-muted-foreground">{{ formatTime(deployment.started_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>完成时间</dt>
            <dd class="text-muted-foreground">{{ formatTime(deployment.finished_at) }}</dd>
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
      </div>

      <div class="app-surface flex min-h-[360px] min-w-0 flex-1 flex-col app-detail-card">
        <TabsRoot default-value="operation" class="flex min-h-0 flex-1 flex-col">
          <div class="app-section-header shrink-0 app-detail-section-header">
            <TabsList aria-label="日志类型" class="flex h-9 gap-1">
              <TabsTrigger
                value="operation"
                class="inline-flex h-9 items-center px-3 text-sm text-muted-foreground hover:text-foreground data-[state=active]:border-b-2 data-[state=active]:border-primary data-[state=active]:text-foreground"
              >
                操作日志
              </TabsTrigger>
              <TabsTrigger
                value="container"
                class="inline-flex h-9 items-center px-3 text-sm text-muted-foreground hover:text-foreground data-[state=active]:border-b-2 data-[state=active]:border-primary data-[state=active]:text-foreground"
              >
                容器日志
              </TabsTrigger>
            </TabsList>
            <button
              v-if="!isTerminalDeployment"
              class="app-button inline-flex h-9 items-center gap-2 px-3"
              :class="isAutoRefreshing ? 'text-primary' : ''"
              @click="toggleAutoRefresh"
            >
              <Loader2 class="size-4" :class="isAutoRefreshing ? 'animate-spin' : ''" />
              {{ isAutoRefreshing ? '自动刷新' : '暂停刷新' }}
            </button>
          </div>
          <TabsContent value="operation" class="min-h-0 flex-1 p-5 outline-none">
            <div
              v-if="!operationLogText"
              class="flex h-full min-h-[240px] items-center justify-center text-muted-foreground"
            >
              <div class="text-center">
                <AppSpinner
                  v-if="operationLogStatus === 'loading' || operationLogStatus === 'streaming'"
                />
                <p v-if="operationLogStatus === 'loading'" class="mt-2 text-sm">
                  加载操作日志中...
                </p>
                <p v-else-if="operationLogStatus === 'streaming'" class="mt-2 text-sm">
                  操作日志刷新中...
                </p>
                <p v-else-if="operationLogStatus === 'empty'" class="text-sm">暂无操作日志输出</p>
                <div v-else-if="operationLogStatus === 'error'">
                  <p class="text-sm text-destructive">操作日志加载失败</p>
                  <button class="app-link mt-2 text-sm" @click="retryOperationLogs">重试</button>
                </div>
              </div>
            </div>
            <MonacoEditor
              v-else
              :model-value="operationLogText"
              language="plaintext"
              height="100%"
              :readonly="true"
              @mount="handleOperationLogEditorMount"
            />
          </TabsContent>

          <TabsContent value="container" class="min-h-0 flex-1 p-5 outline-none">
            <p v-if="containerLogSource === 'tail'" class="mt-1 text-xs text-muted-foreground">
              当前展示最近容器日志，可能包含本次操作前的历史输出。
            </p>
            <div
              v-if="!containerLogText"
              class="flex h-full min-h-[240px] items-center justify-center text-muted-foreground"
            >
              <div class="text-center">
                <AppSpinner
                  v-if="containerLogStatus === 'loading' || containerLogStatus === 'streaming'"
                />
                <p v-if="containerLogStatus === 'not_applicable'" class="text-sm">
                  停止操作不展示容器日志。
                </p>
                <p v-else-if="containerLogStatus === 'waiting_for_operation'" class="text-sm">
                  等待操作完成后拉取容器日志。
                </p>
                <p v-else-if="containerLogStatus === 'loading'" class="mt-2 text-sm">
                  加载容器日志中...
                </p>
                <p v-else-if="containerLogStatus === 'streaming'" class="mt-2 text-sm">
                  容器日志刷新中...
                </p>
                <p v-else-if="containerLogStatus === 'empty'" class="text-sm">暂无容器日志输出</p>
                <div v-else-if="containerLogStatus === 'error'">
                  <p class="text-sm text-destructive">容器日志加载失败</p>
                  <button class="app-link mt-2 text-sm" @click="retryContainerLogs">重试</button>
                </div>
              </div>
            </div>
            <MonacoEditor
              v-else
              :model-value="containerLogText"
              language="plaintext"
              height="100%"
              :readonly="true"
              @mount="handleContainerLogEditorMount"
            />
          </TabsContent>
        </TabsRoot>
      </div>
    </div>

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
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Loader2, X } from 'lucide-vue-next';
  import { computed, onMounted, onUnmounted, ref } from 'vue';
  import { TabsContent, TabsList, TabsRoot, TabsTrigger } from 'reka-ui';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { deploymentApi } from '@/api/deployment/deployment';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import DetailHeaderMeta from '@/components/DetailHeaderMeta.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { DeploymentResp } from '@/gen/proto/orbit/v1/deployment/deployment';
  import { isTerminalStatus, statusTone } from '@/utils/status';
  import { delayAsync, formatDuration, formatTime } from '@/utils/time';
  import type { editor } from 'monaco-editor';

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const deploymentId = computed(() => String(route.params.id ?? ''));
  const toast = useToast();
  const { status, execute } = useStatusAsync();
  const { loading: isCancelling, execute: executeCancel } = useStatusAsync();

  const deployment = ref<DeploymentResp>();
  const deploymentServiceLabel = computed(
    () =>
      deployment.value?.application_name ||
      deployment.value?.application_id ||
      deployment.value?.service_id ||
      ''
  );
  type LogStatus =
    | 'loading'
    | 'streaming'
    | 'done'
    | 'empty'
    | 'error'
    | 'not_applicable'
    | 'waiting_for_operation';

  const operationLogText = ref('');
  const operationLogOffset = ref(0);
  const operationLogStatus = ref<LogStatus>('loading');
  const containerLogText = ref('');
  const containerLogSource = ref('since');
  const containerLogStatus = ref<LogStatus>('loading');
  const isCancelDialogOpen = ref(false);
  const cancelSubmitError = ref('');
  const isAutoRefreshing = ref(false);
  let refreshAbort: AbortController | null = null;
  let refreshGeneration = 0;
  let operationLogEditor: editor.IStandaloneCodeEditor | null = null;
  let containerLogEditor: editor.IStandaloneCodeEditor | null = null;

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

  const isTerminalDeployment = computed(() => isTerminalStatus(deployment.value?.status ?? ''));
  const isCancelable = computed(() =>
    deployment.value ? ['waiting_to_run', 'running'].includes(deployment.value.status) : false
  );

  function isCurrentRefresh(generation: number, signal: AbortSignal) {
    return !signal.aborted && generation === refreshGeneration;
  }

  async function fetchOperationLogs(generation?: number, signal?: AbortSignal) {
    const offset = operationLogOffset.value;
    try {
      const data = await deploymentApi.getLogs(deploymentId.value, offset, { signal });
      if (generation !== undefined && signal && !isCurrentRefresh(generation, signal)) {
        return;
      }
      if (operationLogOffset.value !== offset) {
        return;
      }
      operationLogText.value += data.logs;
      operationLogOffset.value = data.offset;
      operationLogStatus.value = isTerminalDeployment.value
        ? operationLogText.value
          ? 'done'
          : 'empty'
        : 'streaming';
      scrollOperationLogsToBottom();
    } catch {
      if (generation === undefined || !signal || isCurrentRefresh(generation, signal)) {
        operationLogStatus.value = 'error';
      }
    }
  }

  async function fetchContainerLogs(generation?: number, signal?: AbortSignal) {
    if (!deployment.value || deployment.value.operation_type === 'stop') {
      containerLogText.value = '';
      containerLogStatus.value = 'not_applicable';
      return;
    }
    if (!isTerminalDeployment.value) {
      containerLogText.value = '';
      containerLogStatus.value = 'waiting_for_operation';
      return;
    }
    try {
      const data = await deploymentApi.getContainerLogs(
        deploymentId.value,
        { tail: 200 },
        { signal }
      );
      if (generation !== undefined && signal && !isCurrentRefresh(generation, signal)) {
        return;
      }
      containerLogText.value = data.logs;
      containerLogSource.value = data.source;
      containerLogStatus.value = isTerminalDeployment.value
        ? containerLogText.value
          ? 'done'
          : 'empty'
        : 'streaming';
      scrollContainerLogsToBottom();
    } catch {
      if (generation === undefined || !signal || isCurrentRefresh(generation, signal)) {
        containerLogStatus.value = 'error';
      }
    }
  }

  async function fetchLogs(generation?: number, signal?: AbortSignal) {
    if (!deployment.value) {
      return;
    }
    await fetchOperationLogs(generation, signal);
    await fetchContainerLogs(generation, signal);
  }

  function retryOperationLogs() {
    void fetchOperationLogs();
  }

  function retryContainerLogs() {
    void fetchContainerLogs();
  }

  function stopAutoRefresh() {
    refreshGeneration++;
    refreshAbort?.abort();
    refreshAbort = null;
    isAutoRefreshing.value = false;
  }

  function startAutoRefresh() {
    if (isAutoRefreshing.value || !deployment.value || isTerminalDeployment.value) {
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
          const data = await deploymentApi.get(deploymentId.value, { signal });
          if (!isCurrentRefresh(generation, signal)) {
            break;
          }
          deployment.value = data;
          if (isTerminalStatus(data.status)) {
            await fetchLogs(generation, signal);
            break;
          }
          await fetchLogs(generation, signal);
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
    deployment.value = undefined;
    operationLogText.value = '';
    operationLogOffset.value = 0;
    operationLogStatus.value = 'loading';
    containerLogText.value = '';
    containerLogSource.value = 'since';
    containerLogStatus.value = 'loading';
    isCancelDialogOpen.value = false;
  }

  async function loadDeployment() {
    resetState();
    try {
      const currentDeployment = await execute(() => deploymentApi.get(deploymentId.value));
      deployment.value = currentDeployment;
    } catch {
      toast.error(t('deployment.toast.loadFailed'));
      router.push('/deployments');
      return;
    }
    await fetchLogs();
    if (!isTerminalDeployment.value) {
      startAutoRefresh();
    }
  }

  async function handleCancel() {
    cancelSubmitError.value = '';
    try {
      await executeCancel(async () => {
        deployment.value = await deploymentApi.cancel(deploymentId.value, {});
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

  function scrollToBottom(logEditor: editor.IStandaloneCodeEditor | null) {
    if (logEditor) {
      const lineCount = logEditor.getModel()?.getLineCount() || 0;
      if (lineCount > 0) {
        logEditor.revealLine(lineCount);
      }
    }
  }

  function scrollOperationLogsToBottom() {
    scrollToBottom(operationLogEditor);
  }

  function scrollContainerLogsToBottom() {
    scrollToBottom(containerLogEditor);
  }

  function handleOperationLogEditorMount(editor: editor.IStandaloneCodeEditor) {
    operationLogEditor = editor;
    scrollOperationLogsToBottom();
  }

  function handleContainerLogEditorMount(editor: editor.IStandaloneCodeEditor) {
    containerLogEditor = editor;
    scrollContainerLogsToBottom();
  }

  onMounted(loadDeployment);
  onUnmounted(stopAutoRefresh);
</script>
