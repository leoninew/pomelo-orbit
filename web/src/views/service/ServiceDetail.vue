<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <h1 class="min-w-0 break-words text-xl font-semibold text-foreground">
          {{ t('service.detail.title') }}
        </h1>
        <DetailHeaderMeta v-if="service">
          <AppBadge variant="status" :tone="appStatusTone(service.status)">
            {{ service.status }}
          </AppBadge>
        </DetailHeaderMeta>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-if="service"
          class="app-button-primary h-9 px-3"
          :disabled="operating || isDeploying"
          @click="openDeployDialog"
        >
          <Rocket class="size-4" />
          {{ t('service.actions.deploy') }}
        </button>
        <button
          v-if="service"
          class="app-button-danger h-9 px-3"
          :disabled="operating || !canStop"
          @click="openStopDialog"
        >
          <Square class="size-4" />
          {{ t('service.actions.stop') }}
        </button>
        <button
          v-if="canDelete"
          class="app-button-danger h-9 px-3"
          :disabled="operating"
          @click="isDeleteDialogOpen = true"
        >
          <Trash2 class="size-4" />
          {{ t('service.actions.delete') }}
        </button>
        <button
          v-if="service"
          class="app-button h-9 px-3"
          :disabled="operating"
          @click="openLogsDrawer()"
        >
          <ScrollText class="size-4" />
          {{ t('service.actions.logsAll') }}
        </button>
        <button class="app-button h-9 px-4" @click="router.push('/services')">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <AppSpinner v-if="status === 'loading' && !service" class="py-12" />

    <template v-else-if="service">
      <div class="app-surface">
        <div class="app-section-header">
          <h2 class="font-semibold text-foreground">{{ t('service.detail.sections.basic') }}</h2>
        </div>
        <dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">
              {{ t('service.fields.application') }}
            </dt>
            <dd class="min-w-0">
              <router-link :to="`/application/${service.application_id}`" class="app-link">
                {{ service.application_name || service.application_id }}
              </router-link>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">
              {{ t('service.fields.instanceKey') }}
            </dt>
            <dd class="text-foreground">{{ service.instance_key || 'default' }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('service.fields.version') }}</dt>
            <dd>
              <button class="app-link" @click="router.push(`/version/${service.version_id}`)">
                {{ service.version_label || service.version_id }}
              </button>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('common.status') }}</dt>
            <dd>
              <AppBadge variant="status" :tone="appStatusTone(service.status)">
                {{ service.status }}
              </AppBadge>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">
              {{ t('service.fields.lastSuccessfulVersion') }}
            </dt>
            <dd class="text-foreground">
              {{ service.last_successful_version_label || '—' }}
            </dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('common.createdAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(service.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt class="w-32 shrink-0 text-muted-foreground">{{ t('common.updatedAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(service.updated_at) }}</dd>
          </div>
        </dl>
      </div>

      <div class="app-surface">
        <div class="app-section-header flex items-center justify-between gap-3">
          <h2 class="font-semibold text-foreground">{{ t('service.runtimeEnv.title') }}</h2>
          <button
            class="app-button inline-flex h-8 items-center gap-2 px-3"
            :disabled="runtimeEnvLoading"
            @click="loadRuntimeEnv"
          >
            <RefreshCw class="size-4" :class="{ 'animate-spin': runtimeEnvLoading }" />
            {{ t('common.refresh') }}
          </button>
        </div>
        <div class="px-5 py-4">
          <AppSpinner v-if="runtimeEnvLoading && !runtimeEnv" class="py-8" />
          <div v-else-if="runtimeEnvError" class="flex flex-wrap items-center gap-3">
            <p class="text-sm text-destructive">{{ runtimeEnvError }}</p>
            <button class="app-link text-sm" @click="loadRuntimeEnv">
              {{ t('service.runtimeEnv.retry') }}
            </button>
          </div>
          <AppEmptyState
            v-else-if="!runtimeEnv || runtimeEnv.items.length === 0"
            :message="t('service.runtimeEnv.empty')"
          />
          <div v-else class="overflow-x-auto">
            <table class="app-table-detail min-w-[900px]">
              <thead>
                <tr>
                  <th>{{ t('service.fields.component') }}</th>
                  <th>{{ t('service.runtimeEnv.envKey') }}</th>
                  <th>{{ t('service.runtimeEnv.credential') }}</th>
                  <th>{{ t('service.runtimeEnv.dataKey') }}</th>
                  <th>{{ t('service.runtimeEnv.value') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="item in runtimeEnv.items"
                  :key="`${item.component_name}:${item.env_key}`"
                >
                  <td class="text-foreground">{{ item.component_name }}</td>
                  <td class="break-all text-sm text-foreground">{{ item.env_key }}</td>
                  <td>
                    <router-link :to="`/credential/${item.credential_id}`" class="app-link">
                      {{ item.credential_name }}
                    </router-link>
                  </td>
                  <td class="break-all text-sm text-foreground">{{ item.data_key }}</td>
                  <td class="min-w-[240px] whitespace-pre-wrap break-all text-sm text-foreground">
                    {{ item.value }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <div class="app-surface">
        <div class="app-section-header flex items-center justify-between gap-3">
          <h2 class="font-semibold text-foreground">
            {{ t('service.detail.sections.components') }}
          </h2>
          <button
            class="app-button inline-flex h-8 items-center gap-2 px-3"
            :disabled="containersLoading"
            @click="loadContainers"
          >
            <RefreshCw class="size-4" :class="{ 'animate-spin': containersLoading }" />
            {{ t('common.refresh') }}
          </button>
        </div>
        <div class="px-5 py-4">
          <AppSpinner v-if="containersLoading && containers.length === 0" class="py-8" />
          <p v-else-if="containersError" class="text-sm text-destructive">{{ containersError }}</p>
          <AppEmptyState
            v-else-if="containers.length === 0"
            :message="t('service.containers.empty')"
          />
          <div v-else class="overflow-x-auto">
            <table class="app-table-list min-w-[960px]">
              <thead>
                <tr>
                  <th>{{ t('service.fields.component') }}</th>
                  <th>{{ t('service.fields.container') }}</th>
                  <th>{{ t('common.status') }}</th>
                  <th>{{ t('service.fields.runtime') }}</th>
                  <th>{{ t('service.fields.health') }}</th>
                  <th>{{ t('service.fields.image') }}</th>
                  <th>{{ t('common.operation') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="container in containers"
                  :key="container.id || container.name || container.service"
                >
                  <td class="text-sm text-foreground">
                    {{ container.service || '—' }}
                  </td>
                  <td class="text-xs text-muted-foreground">
                    {{ container.name || container.id || '—' }}
                  </td>
                  <td>
                    <AppBadge variant="pill" :tone="containerStateTone(container.state)">
                      {{ container.state || '—' }}
                    </AppBadge>
                  </td>
                  <td
                    class="max-w-[280px] truncate text-xs text-muted-foreground"
                    :title="container.status"
                  >
                    {{ container.status || '—' }}
                  </td>
                  <td class="text-sm text-foreground">{{ container.health || '—' }}</td>
                  <td
                    class="max-w-[280px] truncate text-xs text-muted-foreground"
                    :title="container.image"
                  >
                    {{ container.image || '—' }}
                  </td>
                  <td>
                    <button
                      class="app-link"
                      :disabled="!container.service"
                      @click="openLogsDrawer(container.service)"
                    >
                      {{ t('service.actions.logs') }}
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </template>

    <AppEmptyState v-else :message="t('service.detail.notFound')" />

    <AppDialog v-model:open="isDeployDialogOpen" :title="t('service.deploy.dialogTitle')">
      <div class="space-y-4">
        <p class="text-sm text-muted-foreground">{{ t('service.deploy.description') }}</p>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('service.fields.version') }}
            <span class="text-destructive">*</span>
          </label>
          <ComboboxSelect
            :model-value="deployForm.version_id"
            :options="versionSelectOptions"
            :placeholder="t('service.deploy.selectVersion')"
            width-class="w-full"
            @update:model-value="handleDeployVersionChange"
          />
          <p v-if="deployError" class="app-field-error mt-1 text-xs">{{ deployError }}</p>
        </div>
        <label class="flex items-center gap-2">
          <input v-model="deployForm.force_recreate" type="checkbox" class="app-checkbox" />
          <span class="text-sm text-foreground">{{ t('service.deploy.forceRecreate') }}</span>
        </label>
      </div>
      <template #footer>
        <button class="app-button" @click="isDeployDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="operating" @click="handleDeployOk">
          {{ t('service.actions.deploy') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isStopDialogOpen"
      :title="t('service.stop.dialogTitle')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="mb-4 text-sm text-muted-foreground">{{ t('service.stop.confirm') }}</p>
      <label class="flex items-center gap-2">
        <input v-model="stopRemoveVolumes" type="checkbox" class="app-checkbox" />
        <span class="text-sm text-foreground">{{ t('service.stop.removeVolumes') }}</span>
      </label>
      <template #footer>
        <button class="app-button" @click="isStopDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-danger" :disabled="operating" @click="handleStopOk">
          {{ t('service.actions.stop') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteDialogOpen"
      :title="t('service.delete.dialogTitle')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">
        {{ t('service.delete.confirm', { instance: service?.instance_key || 'default' }) }}
      </p>
      <template #footer>
        <button class="app-button" @click="isDeleteDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-destructive" :disabled="operating" @click="handleDeleteOk">
          {{ t('common.delete') }}
        </button>
      </template>
    </AppDialog>

    <AppDrawer
      :open="logsDrawerOpen"
      :title="logsDrawerTitle"
      width-class="w-[min(960px,100vw)]"
      body-class="min-h-0 flex-1 overflow-hidden p-0"
      @update:open="handleLogsDrawerOpenChange"
    >
      <div class="flex h-full flex-col gap-3 p-6">
        <div class="flex shrink-0 items-center justify-between gap-3">
          <p class="text-sm text-muted-foreground">
            {{ t('service.logs.description') }}
          </p>
          <button
            class="app-button inline-flex h-8 items-center gap-2 px-3"
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
  import {
    ArrowLeft,
    Loader2,
    RefreshCw,
    Rocket,
    ScrollText,
    Square,
    Trash2,
  } from 'lucide-vue-next';
  import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/application/application';
  import { serviceApi } from '@/api/service/service';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailHeaderMeta from '@/components/DetailHeaderMeta.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ComboboxSelect, { type ComboboxOptionValue } from '@/components/ComboboxSelect.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { ApplicationContainerStatusResp } from '@/gen/proto/orbit/v1/application/application';
  import type { VersionResp } from '@/gen/proto/orbit/v1/application/version';
  import type { ServiceResp, ServiceRuntimeEnvResp } from '@/gen/proto/orbit/v1/service/service';
  import { appStatusTone, containerStateTone } from '@/utils/status';
  import { delayAsync, formatTime } from '@/utils/time';

  type LogStatus = 'loading' | 'streaming' | 'done' | 'empty' | 'error';

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const toast = useToast();
  const { status, execute } = useStatusAsync();
  const { status: opStatus, execute: executeOp } = useStatusAsync();

  const service = ref<ServiceResp | null>(null);
  const containers = ref<ApplicationContainerStatusResp[]>([]);
  const containersLoading = ref(false);
  const containersError = ref('');
  const runtimeEnv = ref<ServiceRuntimeEnvResp | null>(null);
  const runtimeEnvLoading = ref(false);
  const runtimeEnvError = ref('');
  const versions = ref<VersionResp[]>([]);

  const isDeployDialogOpen = ref(false);
  const deployError = ref('');
  const deployForm = reactive({
    version_id: '',
    force_recreate: false,
  });

  const isStopDialogOpen = ref(false);
  const stopRemoveVolumes = ref(false);
  const isDeleteDialogOpen = ref(false);

  const logsDrawerOpen = ref(false);
  const logsComponent = ref('');
  const logText = ref('');
  const logStatus = ref<LogStatus>('loading');
  const logError = ref('');
  const isLogsAutoRefreshing = ref(false);
  let logsRefreshAbort: AbortController | null = null;
  let logsRefreshGeneration = 0;

  const serviceId = computed(() => String(route.params.id || ''));
  const operating = computed(() => opStatus.value === 'loading');
  const isDeploying = computed(() => service.value?.status === 'deploying');
  const canStop = computed(() => {
    const s = service.value?.status;
    return s === 'running' || s === 'faulted';
  });
  const canDelete = computed(() => service.value?.status === 'stopped');

  const versionSelectOptions = computed(() =>
    versions.value.map((v) => ({
      value: v.id,
      label: v.label,
      description: v.status,
    }))
  );

  const logsDrawerTitle = computed(() => {
    if (!service.value) {
      return t('service.logs.title');
    }
    const app = service.value.application_name || service.value.application_id;
    const instance = service.value.instance_key || 'default';
    if (logsComponent.value) {
      return t('service.logs.titleWithComponent', {
        app,
        instance,
        component: logsComponent.value,
      });
    }
    return t('service.logs.titleWithTarget', { app, instance });
  });

  async function fetchService() {
    if (!serviceId.value) {
      service.value = null;
      return;
    }
    try {
      await execute(async () => {
        service.value = await serviceApi.get(serviceId.value);
      });
    } catch (err: unknown) {
      service.value = null;
      toast.error(err instanceof Error ? err.message : t('service.toast.loadDetailFailed'));
    }
  }

  async function loadContainers() {
    if (!service.value) {
      containers.value = [];
      containersError.value = '';
      return;
    }
    containersLoading.value = true;
    containersError.value = '';
    try {
      const resp = await applicationApi.getStatus(service.value.application_id, {
        service_id: service.value.id,
      });
      containers.value = resp.containers;
    } catch (err: unknown) {
      containers.value = [];
      containersError.value =
        err instanceof Error ? err.message : t('service.containers.loadFailed');
    } finally {
      containersLoading.value = false;
    }
  }

  async function loadRuntimeEnv() {
    if (!service.value) {
      runtimeEnv.value = null;
      runtimeEnvError.value = '';
      return;
    }
    runtimeEnvLoading.value = true;
    runtimeEnvError.value = '';
    try {
      runtimeEnv.value = await serviceApi.getRuntimeEnv(service.value.id);
    } catch (err: unknown) {
      runtimeEnv.value = null;
      runtimeEnvError.value =
        err instanceof Error ? err.message : t('service.runtimeEnv.loadFailed');
    } finally {
      runtimeEnvLoading.value = false;
    }
  }

  async function loadVersions() {
    if (!service.value) {
      versions.value = [];
      return;
    }
    try {
      const resp = await applicationApi.listVersions(service.value.application_id, {
        per_page: 100,
      });
      versions.value = (resp.items ?? []).filter((v) => v.status === 'published');
      if (
        service.value.version_id &&
        !versions.value.some((v) => v.id === service.value?.version_id)
      ) {
        // keep current bound version selectable even if unpublished edge case
        versions.value = [
          {
            id: service.value.version_id,
            application_id: service.value.application_id,
            label: service.value.version_label || service.value.version_id,
            status: 'published',
            created_at: '',
            updated_at: '',
          } as VersionResp,
          ...versions.value,
        ];
      }
    } catch {
      versions.value = [];
    }
  }

  async function handleRefresh() {
    await fetchService();
    if (service.value) {
      await Promise.all([loadContainers(), loadVersions(), loadRuntimeEnv()]);
      return;
    }
    runtimeEnv.value = null;
    runtimeEnvError.value = '';
  }

  function openDeployDialog() {
    if (!service.value) {
      return;
    }
    deployForm.version_id = service.value.version_id;
    deployForm.force_recreate = false;
    deployError.value = '';
    void loadVersions();
    isDeployDialogOpen.value = true;
  }

  function handleDeployVersionChange(value: ComboboxOptionValue) {
    deployForm.version_id = String(value || '');
    deployError.value = '';
  }

  async function handleDeployOk() {
    const current = service.value;
    if (!current) {
      return;
    }
    if (!deployForm.version_id) {
      deployError.value = t('service.deploy.versionRequired');
      return;
    }
    try {
      await executeOp(async () => {
        const result = await applicationApi.deploy(current.application_id, {
          version_id: deployForm.version_id,
          instance_key: current.instance_key || 'default',
          force_recreate: deployForm.force_recreate,
          runtime_config: {},
        });
        toast.success(t('service.toast.deployQueued'));
        isDeployDialogOpen.value = false;
        if (result.deployment_id) {
          router.push(`/deployment/${result.deployment_id}`);
          return;
        }
        await handleRefresh();
      });
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : t('service.toast.deployFailed'));
    }
  }

  function openStopDialog() {
    stopRemoveVolumes.value = false;
    isStopDialogOpen.value = true;
  }

  async function handleStopOk() {
    const current = service.value;
    if (!current) {
      return;
    }
    try {
      await executeOp(async () => {
        const result = await applicationApi.stop(current.application_id, {
          service_id: current.id,
          remove_volumes: stopRemoveVolumes.value,
        });
        toast.success(t('service.toast.stopQueued'));
        isStopDialogOpen.value = false;
        if (result.deployment_id) {
          router.push(`/deployment/${result.deployment_id}`);
          return;
        }
        await handleRefresh();
      });
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : t('service.toast.stopFailed'));
    }
  }

  async function handleDeleteOk() {
    const current = service.value;
    if (!current) {
      return;
    }
    try {
      await executeOp(async () => {
        await serviceApi.remove(current.id);
        toast.success(t('service.toast.deleteSuccess'));
        isDeleteDialogOpen.value = false;
        await router.push('/services');
      });
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : t('service.toast.deleteFailed'));
    }
  }

  function isCurrentLogsRefresh(generation: number, signal: AbortSignal) {
    return !signal.aborted && generation === logsRefreshGeneration;
  }

  async function fetchServiceLogs(generation?: number, signal?: AbortSignal) {
    if (!service.value) {
      return;
    }
    if (generation === undefined) {
      logStatus.value = logText.value ? 'streaming' : 'loading';
      logError.value = '';
    }
    try {
      const data = await applicationApi.getLogs(service.value.application_id, {
        tail: 200,
        service_id: service.value.id,
        component: logsComponent.value || undefined,
      });
      if (generation !== undefined && signal && !isCurrentLogsRefresh(generation, signal)) {
        return;
      }
      logText.value = data.logs || '';
      logStatus.value = logText.value
        ? isLogsAutoRefreshing.value
          ? 'streaming'
          : 'done'
        : 'empty';
    } catch (err: unknown) {
      if (generation === undefined || !signal || isCurrentLogsRefresh(generation, signal)) {
        logStatus.value = 'error';
        logError.value = err instanceof Error ? err.message : t('service.logs.loadFailed');
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
    if (isLogsAutoRefreshing.value || !service.value) {
      return;
    }
    const generation = ++logsRefreshGeneration;
    const controller = new AbortController();
    const { signal } = controller;
    logsRefreshAbort = controller;
    isLogsAutoRefreshing.value = true;
    void (async () => {
      while (isCurrentLogsRefresh(generation, signal)) {
        await delayAsync(2000, signal);
        if (!isCurrentLogsRefresh(generation, signal)) {
          break;
        }
        await fetchServiceLogs(generation, signal);
      }
      if (isCurrentLogsRefresh(generation, signal)) {
        logsRefreshAbort = null;
        isLogsAutoRefreshing.value = false;
        if (logText.value && logStatus.value === 'streaming') {
          logStatus.value = 'done';
        }
      }
    })();
  }

  function toggleLogsAutoRefresh() {
    if (isLogsAutoRefreshing.value) {
      stopLogsAutoRefresh();
      if (logText.value && logStatus.value === 'streaming') {
        logStatus.value = 'done';
      }
      return;
    }
    startLogsAutoRefresh();
  }

  function openLogsDrawer(component?: string) {
    stopLogsAutoRefresh();
    logsComponent.value = (component || '').trim();
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
    if (!open) {
      stopLogsAutoRefresh();
      logsComponent.value = '';
      logText.value = '';
      logError.value = '';
      logStatus.value = 'loading';
    }
  }

  function retryLogs() {
    void fetchServiceLogs();
  }

  watch(serviceId, () => {
    handleLogsDrawerOpenChange(false);
    void handleRefresh();
  });

  onMounted(() => {
    void handleRefresh();
  });

  onUnmounted(() => {
    stopLogsAutoRefresh();
  });
</script>
