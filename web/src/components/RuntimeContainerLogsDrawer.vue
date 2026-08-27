<template>
  <AppDrawer
    :open="open"
    :title="target.title"
    width-class="w-[min(960px,100vw)]"
    body-class="min-h-0 flex-1 overflow-hidden p-0"
    @update:open="emit('update:open', $event)"
  >
    <ContainerLogView
      class="min-h-0 p-6"
      :logs="logText"
      :status="status"
      :error="logError"
      message=""
      :auto-refreshing="isAutoRefreshing"
      :show-auto-refresh="true"
      @retry="retry"
      @toggle-auto-refresh="toggleAutoRefresh"
    />
    <template #footer>
      <button class="app-button" @click="emit('update:open', false)">
        {{ t('common.cancel') }}
      </button>
      <button class="app-button-primary" :disabled="status === 'loading'" @click="retry">
        <RefreshCw class="size-4" :class="{ 'animate-spin': status === 'loading' }" />
        {{ t('common.refresh') }}
      </button>
    </template>
  </AppDrawer>
</template>

<script setup lang="ts">
  import { RefreshCw } from '@lucide/vue';
  import { onMounted, onUnmounted, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { applicationApi } from '@/api/application/application';
  import AppDrawer from '@/components/AppDrawer.vue';
  import ContainerLogView from '@/components/ContainerLogView.vue';
  import { delayAsync } from '@/utils/time';
  import type { RuntimeContainerLogTarget } from './runtimeContainerLogs';

  const props = defineProps<{
    open: boolean;
    target: RuntimeContainerLogTarget;
  }>();

  const emit = defineEmits<{
    'update:open': [open: boolean];
  }>();

  type LogStatus = 'loading' | 'streaming' | 'done' | 'empty' | 'error';

  const { t } = useI18n({ useScope: 'global' });
  const logText = ref('');
  const status = ref<LogStatus>('loading');
  const logError = ref('');
  const isAutoRefreshing = ref(false);
  let refreshAbort: AbortController | null = null;
  let refreshGeneration = 0;

  function isCurrentRefresh(generation: number, signal: AbortSignal) {
    return !signal.aborted && generation === refreshGeneration;
  }

  async function fetchLogs(generation?: number, signal?: AbortSignal) {
    if (generation === undefined) {
      status.value = logText.value ? 'streaming' : 'loading';
      logError.value = '';
    }
    try {
      const data = await applicationApi.getLogs(
        props.target.applicationId,
        {
          tail: 200,
          service_id: props.target.serviceId,
          component: props.target.component,
        },
        { signal }
      );
      if (generation !== undefined && signal && !isCurrentRefresh(generation, signal)) return;
      logText.value = data.logs;
      status.value = logText.value ? (isAutoRefreshing.value ? 'streaming' : 'done') : 'empty';
    } catch (error) {
      if (signal?.aborted) return;
      if (generation === undefined || !signal || isCurrentRefresh(generation, signal)) {
        status.value = 'error';
        logError.value = error instanceof Error ? error.message : t('service.logs.loadFailed');
      }
    }
  }

  function stopAutoRefresh() {
    refreshGeneration++;
    refreshAbort?.abort();
    refreshAbort = null;
    isAutoRefreshing.value = false;
  }

  function startAutoRefresh() {
    if (isAutoRefreshing.value) return;
    const generation = ++refreshGeneration;
    const controller = new AbortController();
    const { signal } = controller;
    refreshAbort = controller;
    isAutoRefreshing.value = true;
    void (async () => {
      while (isCurrentRefresh(generation, signal)) {
        await delayAsync(2000, signal);
        if (!isCurrentRefresh(generation, signal)) break;
        await fetchLogs(generation, signal);
      }
      if (isCurrentRefresh(generation, signal)) {
        refreshAbort = null;
        isAutoRefreshing.value = false;
        if (logText.value && status.value === 'streaming') status.value = 'done';
      }
    })();
  }

  function resetLogState() {
    logText.value = '';
    logError.value = '';
    status.value = 'loading';
  }

  function openLogs() {
    stopAutoRefresh();
    resetLogState();
    startAutoRefresh();
    void fetchLogs(refreshGeneration, refreshAbort?.signal);
  }

  function closeLogs() {
    stopAutoRefresh();
    resetLogState();
  }

  function toggleAutoRefresh() {
    if (isAutoRefreshing.value) {
      stopAutoRefresh();
      if (logText.value && status.value === 'streaming') status.value = 'done';
      return;
    }
    startAutoRefresh();
  }

  function retry() {
    void fetchLogs(
      isAutoRefreshing.value ? refreshGeneration : undefined,
      isAutoRefreshing.value ? refreshAbort?.signal : undefined
    );
  }

  watch([() => props.open, () => props.target], ([open, target], [wasOpen, previousTarget]) => {
    if (!open) {
      closeLogs();
      return;
    }
    if (!wasOpen || target !== previousTarget) openLogs();
  });

  onMounted(() => {
    if (props.open) openLogs();
  });
  onUnmounted(stopAutoRefresh);
</script>
