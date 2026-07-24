<template>
  <div class="flex h-full min-h-0 flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h1 class="text-xl font-semibold text-foreground">部署详情</h1>
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-if="isCancelable"
          class="app-button-danger h-9 px-3"
          :disabled="isCancelling"
          @click="isCancelDialogOpen = true"
        >
          <X class="size-4" />
          取消部署
        </button>
        <button class="app-button h-9 px-4" @click="goBack">
          <ArrowLeft class="size-4" />
          {{ backButtonText }}
        </button>
      </div>
    </div>

    <!-- 加载状态 -->
    <AppSpinner v-if="status === 'loading'" class="py-12" />

    <!-- 内容 -->
    <div v-else-if="deployment" class="flex min-h-0 flex-1 flex-col gap-4">
      <!-- 基本信息卡片 -->
      <div class="app-surface shrink-0">
        <div class="app-section-header">
          <h2 class="font-semibold text-foreground">基本信息</h2>
        </div>
        <dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">部署 ID</dt>
            <dd class="min-w-0 break-all text-foreground">{{ deploymentId }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">应用</dt>
            <dd>
              <router-link
                v-if="deployment.application_id"
                :to="`/application/${deployment.application_id}`"
                class="app-link"
              >
                {{ deployment.application_name || deployment.application_id }}
              </router-link>
              <span v-else class="text-muted-foreground">—</span>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">状态</dt>
            <dd>
              <AppBadge variant="pill" :tone="deploymentStatusTone">
                {{ deploymentStatusLabel }}
              </AppBadge>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">操作类型</dt>
            <dd class="text-foreground">
              {{ operationTypeLabel }}
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">触发方式</dt>
            <dd class="text-foreground">{{ triggerTypeLabel }}</dd>
          </div>
          <div class="flex gap-2 sm:col-span-2">
            <dt class="w-32 shrink-0 text-muted-foreground">执行命令</dt>
            <dd class="min-w-0 break-all font-mono text-xs text-foreground">
              {{ deployment.command_text || '未记录' }}
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">耗时</dt>
            <dd class="text-muted-foreground">
              {{ formatDuration(deployment.started_at, deployment.finished_at) }}
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">创建时间</dt>
            <dd class="text-muted-foreground">{{ formatTime(deployment.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">开始时间</dt>
            <dd class="text-muted-foreground">{{ formatTime(deployment.started_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">完成时间</dt>
            <dd class="text-muted-foreground">{{ formatTime(deployment.finished_at) }}</dd>
          </div>
          <div v-if="deployment.error_message" class="flex gap-2 sm:col-span-2">
            <dt class="w-32 shrink-0 text-muted-foreground">错误信息</dt>
            <dd class="min-w-0 text-destructive">
              <pre class="whitespace-pre-wrap break-words font-mono text-xs">{{
                deployment.error_message
              }}</pre>
            </dd>
          </div>
        </dl>
      </div>

      <!-- 日志卡片 -->
      <div class="app-surface flex min-h-[360px] flex-1 flex-col">
        <div class="app-section-header flex shrink-0 items-center justify-between">
          <div>
            <h2 class="font-semibold text-foreground">{{ logSectionTitle }}</h2>
            <p v-if="logMode === 'operation'" class="mt-1 text-xs text-muted-foreground">
              部署失败时展示 compose 操作日志，便于定位命令失败原因。
            </p>
            <p
              v-else-if="logMode === 'container' && containerLogSource === 'tail'"
              class="mt-1 text-xs text-muted-foreground"
            >
              当前展示最近容器日志，可能包含本次操作前的历史输出。
            </p>
          </div>
          <button
            v-if="showsContainerLogs && !isTerminalDeployment"
            class="app-button inline-flex h-8 items-center gap-2 px-3"
            :class="isAutoRefreshing ? 'text-primary' : ''"
            @click="toggleAutoRefresh"
          >
            <Loader2 class="size-4" :class="isAutoRefreshing ? 'animate-spin' : ''" />
            {{ isAutoRefreshing ? '自动刷新' : '暂停刷新' }}
          </button>
        </div>
        <div class="min-h-0 flex-1 p-5">
          <div
            v-if="!logText"
            class="flex h-full min-h-[240px] items-center justify-center text-muted-foreground"
          >
            <div class="text-center">
              <AppSpinner v-if="logStatus === 'loading' || logStatus === 'streaming'" />
              <p v-if="logStatus === 'not_applicable'" class="text-sm">
                停止操作不展示实时容器日志。
              </p>
              <p v-else-if="logStatus === 'loading'" class="mt-2 text-sm">{{ logLoadingText }}</p>
              <p v-else-if="logStatus === 'streaming'" class="mt-2 text-sm">容器日志刷新中...</p>
              <p v-else-if="logStatus === 'empty'" class="text-sm">{{ logEmptyText }}</p>
              <div v-else-if="logStatus === 'error'">
                <p class="text-sm text-destructive">{{ logErrorText }}</p>
                <button class="app-link mt-2 text-sm" @click="retryLogs">重试</button>
              </div>
            </div>
          </div>
          <MonacoEditor
            v-else
            :model-value="logText"
            language="plaintext"
            height="100%"
            :readonly="true"
            squared
            @mount="handleEditorMount"
          />
        </div>
      </div>
    </div>

    <AppDialog
      v-model:open="isCancelDialogOpen"
      title="确认取消"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">确定要取消此部署吗？</p>
      <template #footer>
        <button
          type="button"
          class="app-button"
          :disabled="isCancelling"
          @click="isCancelDialogOpen = false"
        >
          取消
        </button>
        <button
          type="button"
          class="app-button-destructive"
          :disabled="isCancelling"
          @click="handleCancel"
        >
          确认取消
        </button>
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Loader2, X } from 'lucide-vue-next';
  import { computed, onMounted, onUnmounted, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { deploymentApi } from '@/api/deployment/deployment';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
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
  const { t, te } = useI18n();
  const deploymentId = computed(() => String(route.params.id ?? ''));
  const toast = useToast();
  const { status, execute } = useStatusAsync();
  const { loading: isCancelling, execute: executeCancel } = useStatusAsync();

  const deployment = ref<DeploymentResp>();
  const logText = ref('');
  const logMode = ref<'container' | 'operation' | 'none'>('container');
  const containerLogSource = ref('since');
  const isCancelDialogOpen = ref(false);
  const isAutoRefreshing = ref(false);
  const logStatus = ref<'loading' | 'streaming' | 'done' | 'empty' | 'error' | 'not_applicable'>(
    'loading'
  );
  let refreshAbort: AbortController | null = null;
  let refreshGeneration = 0;
  let logEditorInstance: editor.IStandaloneCodeEditor | null = null;

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

  const deploymentStatusLabel = computed(() => {
    if (!deployment.value) {
      return '';
    }
    const key = `deployment.status.${deployment.value.status}`;
    return te(key) ? t(key) : deployment.value.status;
  });
  const operationTypeLabel = computed(() => {
    if (!deployment.value) {
      return '';
    }
    const key = `deployment.operationType.${deployment.value.operation_type}`;
    return te(key) ? t(key) : deployment.value.operation_type;
  });
  const triggerTypeLabel = computed(() => {
    if (!deployment.value) {
      return '';
    }
    const key = `deployment.triggerType.${deployment.value.trigger_type}`;
    return te(key) ? t(key) : deployment.value.trigger_type;
  });
  const isTerminalDeployment = computed(() => isTerminalStatus(deployment.value?.status ?? ''));
  const isFaultedDeployment = computed(() => deployment.value?.status === 'faulted');
  const isCancelable = computed(() =>
    deployment.value ? ['waiting_to_run', 'running'].includes(deployment.value.status) : false
  );
  const showsContainerLogs = computed(
    () =>
      !!deployment.value &&
      deployment.value.operation_type !== 'stop' &&
      deployment.value.status !== 'faulted'
  );
  const logSectionTitle = computed(() => {
    if (logMode.value === 'operation') {
      return '操作日志';
    }
    if (logMode.value === 'none') {
      return '日志';
    }
    return '容器日志';
  });
  const logLoadingText = computed(() =>
    logMode.value === 'operation' ? '加载操作日志中...' : '加载容器日志中...'
  );
  const logEmptyText = computed(() =>
    logMode.value === 'operation' ? '暂无操作日志输出' : '暂无容器日志输出'
  );
  const logErrorText = computed(() =>
    logMode.value === 'operation' ? '操作日志加载失败' : '容器日志加载失败'
  );

  function isCurrentRefresh(generation: number, signal: AbortSignal) {
    return !signal.aborted && generation === refreshGeneration;
  }

  async function fetchOperationLogs(generation?: number, signal?: AbortSignal) {
    logMode.value = 'operation';
    try {
      const data = await deploymentApi.getLogs(deploymentId.value, 0, { signal });
      if (generation !== undefined && signal && !isCurrentRefresh(generation, signal)) {
        return;
      }
      logText.value = data.logs;
      logStatus.value = logText.value ? 'done' : 'empty';
      scrollToBottom();
    } catch {
      if (generation === undefined || !signal || isCurrentRefresh(generation, signal)) {
        logStatus.value = 'error';
      }
    }
  }

  async function fetchContainerLogs(generation?: number, signal?: AbortSignal) {
    if (!deployment.value || deployment.value.operation_type === 'stop') {
      logMode.value = 'none';
      logText.value = '';
      logStatus.value = 'not_applicable';
      return;
    }
    logMode.value = 'container';
    try {
      const data = await deploymentApi.getContainerLogs(
        deploymentId.value,
        { tail: 200 },
        { signal }
      );
      if (generation !== undefined && signal && !isCurrentRefresh(generation, signal)) {
        return;
      }
      logText.value = data.logs;
      containerLogSource.value = data.source;
      logStatus.value = isTerminalDeployment.value
        ? logText.value
          ? 'done'
          : 'empty'
        : 'streaming';
      scrollToBottom();
    } catch {
      if (generation === undefined || !signal || isCurrentRefresh(generation, signal)) {
        logStatus.value = 'error';
      }
    }
  }

  async function fetchLogs(generation?: number, signal?: AbortSignal) {
    if (!deployment.value) {
      return;
    }
    if (deployment.value.operation_type === 'stop' && !isFaultedDeployment.value) {
      logMode.value = 'none';
      logText.value = '';
      logStatus.value = 'not_applicable';
      return;
    }
    if (isFaultedDeployment.value) {
      await fetchOperationLogs(generation, signal);
      return;
    }
    await fetchContainerLogs(generation, signal);
  }

  function retryLogs() {
    void fetchLogs();
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
          if (showsContainerLogs.value) {
            await fetchContainerLogs(generation, signal);
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
    deployment.value = undefined;
    logText.value = '';
    logMode.value = 'container';
    containerLogSource.value = 'since';
    logStatus.value = 'loading';
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
    try {
      await executeCancel(async () => {
        deployment.value = await deploymentApi.cancel(deploymentId.value, {});
        stopAutoRefresh();
        isCancelDialogOpen.value = false;
        toast.success(t('deployment.toast.cancelSuccess'));
      });
    } catch {
      toast.error(t('deployment.toast.cancelFailed'));
    }
  }

  function scrollToBottom() {
    if (logEditorInstance) {
      const lineCount = logEditorInstance.getModel()?.getLineCount() || 0;
      if (lineCount > 0) {
        logEditorInstance.revealLine(lineCount);
      }
    }
  }

  function handleEditorMount(editor: editor.IStandaloneCodeEditor) {
    logEditorInstance = editor;
    scrollToBottom();
  }

  onMounted(loadDeployment);
  onUnmounted(stopAutoRefresh);
</script>
