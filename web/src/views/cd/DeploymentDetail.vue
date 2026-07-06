<template>
  <div class="flex h-full min-h-0 flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h1 class="text-xl font-semibold text-foreground">部署详情</h1>
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-if="deployment && !isTerminalStatus(deployment.status)"
          class="app-button-danger h-9 px-3"
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
              <router-link :to="`/cd/applications/${deployment.application_id}`" class="app-link">
                {{ deployment.application_name || deployment.application_id }}
              </router-link>
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
              {{
                deployment.operation_type === 'deploy'
                  ? '部署'
                  : deployment.operation_type === 'stop'
                    ? '停止'
                    : '重启'
              }}
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">触发方式</dt>
            <dd class="text-foreground">{{ deployment.trigger_type }}</dd>
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
              <span class="block truncate" :title="deployment.error_message">
                {{ deployment.error_message }}
              </span>
            </dd>
          </div>
        </dl>
      </div>

      <!-- 日志卡片 -->
      <div class="app-surface flex min-h-[360px] flex-1 flex-col">
        <div class="app-section-header flex shrink-0 items-center justify-between">
          <div>
            <h2 class="font-semibold text-foreground">容器日志</h2>
            <p v-if="containerLogSource === 'tail'" class="mt-1 text-xs text-muted-foreground">
              当前展示最近容器日志，可能包含本次操作前的历史输出。
            </p>
          </div>
          <button
            v-if="showsContainerLogs"
            class="app-button inline-flex h-8 items-center gap-2 px-3"
            :class="isLogPolling ? 'text-primary' : ''"
            @click="toggleLogPolling"
          >
            <Loader2 class="size-4" :class="isLogPolling ? 'animate-spin' : ''" />
            {{ isLogPolling ? '自动刷新' : '暂停刷新' }}
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
              <p v-else-if="logStatus === 'loading'" class="mt-2 text-sm">加载容器日志中...</p>
              <p v-else-if="logStatus === 'streaming'" class="mt-2 text-sm">容器日志刷新中...</p>
              <p v-else-if="logStatus === 'empty'" class="text-sm">暂无容器日志输出</p>
              <div v-else-if="logStatus === 'error'">
                <p class="text-sm text-destructive">容器日志加载失败</p>
                <button class="app-link mt-2 text-sm" @click="fetchContainerLogs">重试</button>
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
        <button type="button" class="app-button" @click="isCancelDialogOpen = false">取消</button>
        <button type="button" class="app-button-destructive" @click="handleCancel">确认取消</button>
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Loader2, X } from 'lucide-vue-next';
  import { computed, onMounted, onUnmounted, ref } from 'vue';
  import { useRoute, useRouter } from 'vue-router';
  import { deploymentApi } from '@/api/cd/deployments';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { DeploymentResp } from '@/gen/orbit/api/v1/deployment';
  import { isTerminalStatus, statusTone } from '@/utils/status';
  import { delayAsync, formatDuration, formatTime } from '@/utils/time';
  import type { editor } from 'monaco-editor';

  const route = useRoute();
  const router = useRouter();
  const deploymentId = route.params.id as string;
  const toast = useToast();
  const { status, execute } = useStatusAsync();

  const deployment = ref<DeploymentResp>();
  const logText = ref('');
  const containerLogSource = ref('since');
  const isCancelDialogOpen = ref(false);
  const isLogPolling = ref(false);
  const logStatus = ref<'loading' | 'streaming' | 'done' | 'empty' | 'error' | 'not_applicable'>(
    'loading'
  );
  let logAbort: AbortController | null = null;
  let logEditorInstance: editor.IStandaloneCodeEditor | null = null;

  const backButtonText = computed(() => (route.query.from === 'application' ? '返回应用' : '返回'));

  function goBack() {
    if (route.query.from === 'application' && deployment.value?.application_id) {
      router.push(`/cd/applications/${deployment.value.application_id}`);
      return;
    }
    router.push('/cd/deployments');
  }

  const deploymentStatusTone = computed(() =>
    deployment.value ? statusTone(deployment.value.status) : 'default'
  );

  const deploymentStatusLabel = computed(() => (deployment.value ? deployment.value.status : ''));

  async function fetchDeployment() {
    try {
      await execute(async () => {
        const data = await deploymentApi.get(deploymentId);
        deployment.value = data;
      });
    } catch {
      toast.error('获取部署详情失败');
      router.push('/cd/deployments');
    }
  }

  const showsContainerLogs = computed(() => deployment.value?.operation_type !== 'stop');

  async function fetchContainerLogs() {
    if (!deployment.value || !showsContainerLogs.value) {
      logText.value = '';
      logStatus.value = 'not_applicable';
      return;
    }
    try {
      const data = await deploymentApi.getContainerLogs(deploymentId, { tail: 200 });
      logText.value = data.logs;
      containerLogSource.value = data.source;
      logStatus.value = isTerminalStatus(deployment.value.status)
        ? logText.value
          ? 'done'
          : 'empty'
        : 'streaming';
      scrollToBottom();
    } catch (error) {
      console.error('获取容器日志失败:', error);
      logStatus.value = 'error';
    }
  }

  function startLogPolling() {
    if (isLogPolling.value) {
      return;
    }
    logAbort = new AbortController();
    const signal = logAbort.signal;
    isLogPolling.value = true;
    (async () => {
      await fetchContainerLogs();
      if (!deployment.value || !showsContainerLogs.value) {
        isLogPolling.value = false;
        return;
      }
      while (!signal.aborted && showsContainerLogs.value) {
        await delayAsync(2000);
        if (signal.aborted) {
          break;
        }
        try {
          deployment.value = await deploymentApi.get(deploymentId);
          await fetchContainerLogs();
        } catch {
          logStatus.value = 'error';
        }
      }
    })();
  }

  function stopLog() {
    logAbort?.abort();
    logAbort = null;
    isLogPolling.value = false;
  }

  function toggleLogPolling() {
    if (isLogPolling.value) {
      stopLog();
      return;
    }
    startLogPolling();
  }

  async function handleCancel() {
    try {
      await deploymentApi.cancel(deploymentId, {});
      toast.success('已取消部署');
      stopLog();
      deployment.value = await deploymentApi.get(deploymentId);
      isCancelDialogOpen.value = false;
    } catch {
      toast.error('取消失败');
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

  onMounted(async () => {
    await fetchDeployment();
    if (!deployment.value) {
      return;
    }
    if (!showsContainerLogs.value || isTerminalStatus(deployment.value.status)) {
      await fetchContainerLogs();
      return;
    }
    startLogPolling();
  });
  onUnmounted(stopLog);
</script>
