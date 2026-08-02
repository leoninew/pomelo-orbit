<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <h1 class="app-detail-page-title min-w-0 break-words">
          {{ t('service.detail.title') }}
        </h1>
        <DetailHeaderMeta v-if="service">
          <AppBadge variant="status" :tone="appStatusTone(service.status)">
            {{ service.status }}
          </AppBadge>
        </DetailHeaderMeta>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button v-if="service" class="app-button h-9 px-3" :disabled="operating" @click="preview">
          <FileCode2 class="size-4" />
          {{ t('application.detail.actions.preview') }}
        </button>
        <button
          v-if="service"
          class="app-button-primary h-9 px-3"
          :disabled="operating || service.status === 'deploying'"
          @click="deploy"
        >
          <Rocket class="size-4" />
          {{ t('service.actions.deploy') }}
        </button>
        <button class="app-button h-9 px-4" @click="router.push('/services')">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <AppSpinner v-if="loading && !service" class="py-12" />
    <AppEmptyState v-else-if="!service" :message="t('service.detail.notFound')" />

    <template v-else>
      <ServiceBasicInfoCard :service="service" />
      <ServiceComponentsCard :service="service" @view-logs="openLogsDrawer" />
      <ServiceEnvironmentCard
        :rows="environmentRows"
        :saved-rows="savedEnvironmentRows"
        :disabled="operating"
        :validate-key="validateEnvironmentKey"
        @update:rows="environmentRows = $event"
        @save="persistEnvironment"
      />
    </template>

    <AppDrawer
      :open="previewOpen"
      :title="t('application.detail.drawer.composePreview')"
      width-class="w-[min(960px,100vw)]"
      body-class="min-h-0 flex-1 overflow-hidden p-0"
      @update:open="setPreviewOpen"
    >
      <div class="flex h-full flex-col gap-3 p-6">
        <div v-if="previewLoading" class="flex flex-1 items-center justify-center">
          <AppSpinner />
        </div>
        <div
          v-else-if="previewError"
          class="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
        >
          {{ previewError }}
        </div>
        <div v-else class="min-h-0 flex-1">
          <MonacoEditor
            :model-value="previewContent"
            language="yaml"
            height="100%"
            :readonly="true"
          />
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="setPreviewOpen(false)">
          {{ t('application.detail.actions.close') }}
        </button>
      </template>
    </AppDrawer>

    <AppDrawer
      :open="logsDrawerOpen"
      :title="logsDrawerTitle"
      width-class="w-[min(960px,100vw)]"
      body-class="min-h-0 flex-1 overflow-hidden p-0"
      @update:open="handleLogsDrawerOpenChange"
    >
      <div class="flex h-full min-h-[420px] flex-col gap-3 p-6">
        <div class="flex shrink-0 justify-end">
          <button
            class="app-button inline-flex h-9 items-center gap-2 px-3"
            :class="isLogsAutoRefreshing ? 'text-primary' : ''"
            @click="toggleLogsAutoRefresh"
          >
            <Loader2 class="size-4" :class="isLogsAutoRefreshing ? 'animate-spin' : ''" />
            {{
              isLogsAutoRefreshing
                ? t('service.logs.autoRefreshing')
                : t('service.logs.refreshPaused')
            }}
          </button>
        </div>
        <div class="min-h-0 flex-1">
          <div
            v-if="!logText"
            class="flex h-full min-h-[320px] items-center justify-center text-muted-foreground"
          >
            <div class="text-center">
              <AppSpinner v-if="logStatus === 'loading' || logStatus === 'streaming'" />
              <p v-if="logStatus === 'loading'" class="mt-2 text-sm">
                {{ t('service.logs.loading') }}
              </p>
              <p v-else-if="logStatus === 'streaming'" class="mt-2 text-sm">
                {{ t('service.logs.streaming') }}
              </p>
              <p v-else-if="logStatus === 'empty'" class="text-sm">{{ t('service.logs.empty') }}</p>
              <div v-else-if="logStatus === 'error'">
                <p class="text-sm text-destructive">
                  {{ logError || t('service.logs.loadFailed') }}
                </p>
                <button type="button" class="app-link mt-2 text-sm" @click="retryLogs">
                  {{ t('service.logs.retry') }}
                </button>
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
          />
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="handleLogsDrawerOpenChange(false)">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="logStatus === 'loading'" @click="retryLogs">
          <RefreshCw class="size-4" :class="{ 'animate-spin': logStatus === 'loading' }" />
          {{ t('common.refresh') }}
        </button>
      </template>
    </AppDrawer>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, FileCode2, Loader2, RefreshCw, Rocket } from 'lucide-vue-next';
  import { computed, onMounted, onUnmounted, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/application/application';
  import { serviceApi } from '@/api/service/service';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailHeaderMeta from '@/components/DetailHeaderMeta.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import {
    cloneEnvironmentVariableRows,
    environmentVariableRowsFromEntries,
    type EnvironmentVariableEntry,
    type EnvironmentVariableListRow,
  } from '@/components/environmentVariableList';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { ServiceResp } from '@/gen/proto/orbit/v1/service/service';
  import { appStatusTone } from '@/utils/status';
  import { delayAsync } from '@/utils/time';
  import ServiceBasicInfoCard from './components/ServiceBasicInfoCard.vue';
  import ServiceComponentsCard from './components/ServiceComponentsCard.vue';
  import ServiceEnvironmentCard from './components/ServiceEnvironmentCard.vue';

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const toast = useToast();
  const { loading, execute } = useStatusAsync();
  const { loading: operating, execute: executeOperation } = useStatusAsync();
  const { loading: previewLoading, execute: executePreview } = useStatusAsync();
  const service = ref<ServiceResp>();
  const previewOpen = ref(false);
  const previewContent = ref('');
  const previewError = ref('');
  const serviceId = String(route.params.id || '');
  const environmentRows = ref<EnvironmentVariableListRow[]>([]);
  const savedEnvironmentRows = ref<EnvironmentVariableListRow[]>([]);
  const environmentKeyPattern = /^[A-Za-z_][A-Za-z0-9_]*$/;
  const logsDrawerOpen = ref(false);
  const logsComponent = ref('');
  const logText = ref('');
  const logStatus = ref<LogStatus>('loading');
  const logError = ref('');
  const isLogsAutoRefreshing = ref(false);
  let logsRefreshAbort: AbortController | null = null;
  let logsRefreshGeneration = 0;

  type LogStatus = 'loading' | 'streaming' | 'done' | 'empty' | 'error';

  const logsDrawerTitle = computed(() => {
    if (!service.value) return t('service.logs.title');
    const app = service.value.application_name || service.value.application_id;
    const instance = service.value.instance_key || 'default';
    return t('service.logs.titleWithComponent', {
      app,
      instance,
      component: logsComponent.value,
    });
  });

  function setEnvironmentRows(value: ServiceResp) {
    const rows = environmentVariableRowsFromEntries(value.env, 'service-environment');
    environmentRows.value = rows;
    savedEnvironmentRows.value = cloneEnvironmentVariableRows(rows);
  }

  async function load() {
    try {
      await execute(async () => {
        service.value = await serviceApi.get(serviceId);
        setEnvironmentRows(service.value);
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('service.toast.loadDetailFailed'));
      await router.push('/services');
    }
  }

  function validateEnvironmentKey(key: string) {
    return environmentKeyPattern.test(key) ? undefined : t('environment.validation.invalidKey');
  }

  async function persistEnvironment(entries: EnvironmentVariableEntry[]) {
    try {
      await executeOperation(async () => {
        const updated = await serviceApi.updateEnv(serviceId, {
          env: entries,
        });
        service.value = updated;
        setEnvironmentRows(updated);
        toast.success(t('environment.saved'));
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('environment.saveFailed'));
    }
  }

  async function preview() {
    previewContent.value = '';
    previewError.value = '';
    previewOpen.value = true;
    try {
      await executePreview(async () => {
        const result = await serviceApi.preview(serviceId);
        previewContent.value = result.compose_yaml;
      });
    } catch (error) {
      previewError.value =
        error instanceof Error ? error.message : t('application.toast.loadPreviewFailed');
    }
  }

  function setPreviewOpen(open: boolean) {
    previewOpen.value = open;
    if (!open) {
      previewContent.value = '';
      previewError.value = '';
    }
  }

  function isCurrentLogsRefresh(generation: number, signal: AbortSignal) {
    return !signal.aborted && generation === logsRefreshGeneration;
  }

  async function fetchServiceLogs(generation?: number, signal?: AbortSignal) {
    const current = service.value;
    if (!current) return;
    if (generation === undefined) {
      logStatus.value = logText.value ? 'streaming' : 'loading';
      logError.value = '';
    }
    try {
      const data = await applicationApi.getLogs(current.application_id, {
        tail: 200,
        service_id: current.id,
        component: logsComponent.value,
      });
      if (generation !== undefined && signal && !isCurrentLogsRefresh(generation, signal)) return;
      logText.value = data.logs || '';
      logStatus.value = logText.value
        ? isLogsAutoRefreshing.value
          ? 'streaming'
          : 'done'
        : 'empty';
    } catch (error) {
      if (generation === undefined || !signal || isCurrentLogsRefresh(generation, signal)) {
        logStatus.value = 'error';
        logError.value = error instanceof Error ? error.message : t('service.logs.loadFailed');
      }
    }
  }

  function stopLogsAutoRefresh() {
    logsRefreshGeneration++;
    logsRefreshAbort?.abort();
    logsRefreshAbort = null;
    isLogsAutoRefreshing.value = false;
  }

  function startLogsAutoRefresh() {
    if (isLogsAutoRefreshing.value || !service.value) return;
    const generation = ++logsRefreshGeneration;
    const controller = new AbortController();
    const { signal } = controller;
    logsRefreshAbort = controller;
    isLogsAutoRefreshing.value = true;
    void (async () => {
      while (isCurrentLogsRefresh(generation, signal)) {
        await delayAsync(2000, signal);
        if (!isCurrentLogsRefresh(generation, signal)) break;
        await fetchServiceLogs(generation, signal);
      }
      if (isCurrentLogsRefresh(generation, signal)) {
        logsRefreshAbort = null;
        isLogsAutoRefreshing.value = false;
        if (logText.value && logStatus.value === 'streaming') logStatus.value = 'done';
      }
    })();
  }

  function toggleLogsAutoRefresh() {
    if (isLogsAutoRefreshing.value) {
      stopLogsAutoRefresh();
      if (logText.value && logStatus.value === 'streaming') logStatus.value = 'done';
      return;
    }
    startLogsAutoRefresh();
  }

  function openLogsDrawer(component: string) {
    stopLogsAutoRefresh();
    logsComponent.value = component.trim();
    logText.value = '';
    logError.value = '';
    logStatus.value = 'loading';
    logsDrawerOpen.value = true;
    void (async () => {
      await fetchServiceLogs();
      startLogsAutoRefresh();
    })();
  }

  function handleLogsDrawerOpenChange(open: boolean) {
    logsDrawerOpen.value = open;
    if (open) return;
    stopLogsAutoRefresh();
    logsComponent.value = '';
    logText.value = '';
    logError.value = '';
    logStatus.value = 'loading';
  }

  function retryLogs() {
    void fetchServiceLogs();
  }

  async function deploy() {
    try {
      await executeOperation(async () => {
        const result = await serviceApi.deploy(serviceId, { force_recreate: false });
        for (const warning of result.warnings) toast.error(warning);
        toast.success(t('service.toast.deployQueued'));
        if (result.deployment_id) await router.push(`/deployment/${result.deployment_id}`);
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('service.toast.deployFailed'));
    }
  }

  onMounted(load);
  onUnmounted(stopLogsAutoRefresh);
</script>
