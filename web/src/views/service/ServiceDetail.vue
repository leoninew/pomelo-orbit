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
          :disabled="operating || service.active_deployment"
          @click="openDeployDialog"
        >
          <Rocket class="size-4" />
          {{ t('service.actions.deploy') }}
        </button>
        <button
          v-if="service"
          class="app-button-danger h-9 px-3"
          :disabled="operating || service.status !== 'stopped'"
          @click="openDeleteDialog"
        >
          <Trash2 class="size-4" />
          {{ t('common.delete') }}
        </button>
        <button class="app-button h-9 px-4" @click="router.push('/services')">
          <ArrowLeft class="size-4" />
          {{ t('common.back') }}
        </button>
      </div>
    </div>

    <AppLoadingState v-if="loading && !service" size="section" />
    <AppEmptyState v-else-if="!service" :message="t('service.detail.notFound')" />

    <template v-else>
      <DetailInfoCard
        :title="t('service.detail.sections.basic')"
        editable
        :disabled="operating"
        @edit="openBasicEditDialog"
      >
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt>{{ t('service.fields.application') }}</dt>
            <dd>
              <router-link :to="`/application/${service.application_id}`" class="app-link">
                {{ service.application_name }}
              </router-link>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('service.fields.code') }}</dt>
            <dd class="text-foreground">{{ service.code }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('service.fields.instanceKey') }}</dt>
            <dd class="text-foreground">{{ service.instance_key }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('service.fields.version') }}</dt>
            <dd>
              <router-link :to="`/version/${service.version_id}`" class="app-link">
                {{ service.version_label }}
              </router-link>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('common.status') }}</dt>
            <dd>
              <AppBadge variant="status" :tone="appStatusTone(service.status)">
                {{ service.status }}
              </AppBadge>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('common.createdAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(service.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>{{ t('common.updatedAt') }}</dt>
            <dd class="text-muted-foreground">{{ formatTime(service.updated_at) }}</dd>
          </div>
        </dl>
      </DetailInfoCard>
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

    <AppDialog
      v-if="service"
      :open="isBasicEditDialogOpen"
      :title="t('service.detail.dialog.editBasic')"
      width-class="w-[min(640px,calc(100vw-32px))]"
      @update:open="setBasicEditDialogOpen"
    >
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label mb-1.5 block">
            {{ t('service.fields.application') }}
          </label>
          <input :value="service.application_name" type="text" class="app-input" disabled />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label mb-1.5 block">
            {{ t('service.fields.code') }}
          </label>
          <input :value="service.code" type="text" class="app-input" disabled />
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label mb-1.5 block">
            {{ t('service.fields.version') }}
            <span class="text-destructive">*</span>
          </label>
          <ComboboxSelect
            :model-value="basicEditForm.version_id"
            :options="basicEditVersionSelectOptions"
            :placeholder="t('service.create.selectVersion')"
            :invalid="Boolean(basicEditErrors.version_id)"
            description-inline
            width-class="w-full"
            @update:model-value="handleBasicEditVersionChange"
          />
          <p v-if="basicEditErrors.version_id" class="app-field-error" role="alert">
            {{ basicEditErrors.version_id }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label mb-1.5 block">
            {{ t('service.fields.instanceKey') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="basicEditForm.instance_key"
            type="text"
            class="app-input"
            :class="basicEditErrors.instance_key ? 'app-input-error' : ''"
            :aria-invalid="basicEditErrors.instance_key ? 'true' : undefined"
            @input="basicEditErrors.instance_key = ''"
          />
          <p v-if="basicEditErrors.instance_key" class="app-field-error" role="alert">
            {{ basicEditErrors.instance_key }}
          </p>
        </div>
      </div>
      <p v-if="basicEditSubmitError" class="app-field-error mt-3" role="alert">
        {{ basicEditSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.save')"
          @cancel="cancelBasicEditing"
          @confirm="saveBasicInfo"
        />
      </template>
    </AppDialog>

    <AppDialog
      :open="isDeployDialogOpen"
      :title="t('service.deploy.dialogTitle')"
      @update:open="handleDeployDialogOpenChange"
    >
      <div class="space-y-4">
        <p class="text-sm text-muted-foreground">
          {{ deployTargetLabel }}
        </p>
        <label class="flex items-center gap-2">
          <input v-model="deployForm.force_recreate" type="checkbox" class="app-checkbox" />
          <span class="text-sm text-foreground">{{ t('service.deploy.forceRecreate') }}</span>
        </label>
      </div>
      <p v-if="deploySubmitError" class="app-field-error mt-3" role="alert">
        {{ deploySubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.deploy')"
          @cancel="closeDeployDialog"
          @confirm="handleDeployOk"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteDialogOpen"
      :title="t('service.detail.dialog.confirmDelete')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-muted-foreground">
        {{
          t('service.detail.dialog.deleteConfirm', {
            instance: service?.instance_key || '-',
          })
        }}
      </p>
      <p v-if="deleteError" class="app-field-error mt-3" role="alert">
        {{ deleteError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.delete')"
          variant="destructive"
          @cancel="closeDeleteDialog"
          @confirm="deleteService"
        />
      </template>
    </AppDialog>

    <AppDrawer
      :open="previewOpen"
      :title="t('application.detail.drawer.composePreview')"
      width-class="w-[min(960px,100vw)]"
      body-class="min-h-0 flex-1 overflow-hidden p-0"
      @update:open="setPreviewOpen"
    >
      <div class="flex h-full flex-col gap-3 p-6">
        <AppLoadingState v-if="previewLoading" class="flex-1 items-center" size="section" />
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
  import { ArrowLeft, FileCode2, Loader2, RefreshCw, Rocket, Trash2 } from '@lucide/vue';
  import { computed, onMounted, onUnmounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { applicationApi } from '@/api/application/application';
  import { serviceApi } from '@/api/service/service';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import DetailHeaderMeta from '@/components/DetailHeaderMeta.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import {
    cloneEnvironmentVariableRows,
    environmentVariableRowsFromEntries,
    type EnvironmentVariableEntry,
    type EnvironmentVariableListRow,
  } from '@/components/environmentVariableList';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { VersionResp } from '@/gen/proto/orbit/v1/application/version';
  import type { ServiceResp } from '@/gen/proto/orbit/v1/service/service';
  import { appStatusTone } from '@/utils/status';
  import { delayAsync, formatTime } from '@/utils/time';
  import ComboboxSelect, { type ComboboxOptionValue } from '@/components/ComboboxSelect.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
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
  const isBasicEditDialogOpen = ref(false);
  const basicEditVersions = ref<VersionResp[]>([]);
  const basicEditForm = reactive({
    version_id: '',
    instance_key: '',
  });
  const basicEditErrors = reactive({
    version_id: '',
    instance_key: '',
  });
  const basicEditSubmitError = ref('');
  const isDeployDialogOpen = ref(false);
  const deployForm = reactive({ force_recreate: false });
  const deploySubmitError = ref('');
  const isDeleteDialogOpen = ref(false);
  const deleteError = ref('');
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
    const app = service.value.application_name;
    const instance = service.value.instance_key || 'default';
    return t('service.logs.titleWithComponent', {
      app,
      instance,
      component: logsComponent.value,
    });
  });
  const deployTargetLabel = computed(() => {
    const current = service.value;
    if (!current) return '';
    const application = current.application_name;
    return t('service.detail.subtitle', {
      instance: `${application} / ${current.instance_key || 'default'}`,
      version: current.version_label,
    });
  });
  const basicEditVersionSelectOptions = computed(() =>
    basicEditVersions.value.map((version) => ({
      value: version.id,
      label: version.label,
      description: version.status,
    }))
  );

  function setEnvironmentRows(value: ServiceResp) {
    const rows = environmentVariableRowsFromEntries(value.env, 'service-environment');
    environmentRows.value = rows;
    savedEnvironmentRows.value = cloneEnvironmentVariableRows(rows);
  }

  function setService(value: ServiceResp) {
    service.value = value;
    setEnvironmentRows(value);
    if (value.effective_error) {
      toast.error(
        t('service.detail.effectiveConfigUnavailable', { error: value.effective_error }),
        8000
      );
    }
  }

  async function load() {
    try {
      await execute(async () => {
        setService(await serviceApi.get(serviceId));
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('service.toast.loadDetailFailed'));
      await router.push('/services');
    }
  }

  function validateEnvironmentKey(key: string) {
    return environmentKeyPattern.test(key) ? undefined : t('environment.validation.invalidKey');
  }

  async function openBasicEditDialog() {
    const current = service.value;
    if (!current) return;
    Object.assign(basicEditForm, {
      version_id: current.version_id,
      instance_key: current.instance_key,
    });
    Object.assign(basicEditErrors, { version_id: '', instance_key: '' });
    basicEditSubmitError.value = '';
    try {
      const page = await applicationApi.listVersions(current.application_id, { per_page: 100 });
      basicEditVersions.value = page.items ?? [];
      isBasicEditDialogOpen.value = true;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('service.toast.loadFailed'));
    }
  }

  function cancelBasicEditing() {
    isBasicEditDialogOpen.value = false;
    basicEditVersions.value = [];
    Object.assign(basicEditForm, { version_id: '', instance_key: '' });
    Object.assign(basicEditErrors, { version_id: '', instance_key: '' });
    basicEditSubmitError.value = '';
  }

  function setBasicEditDialogOpen(open: boolean) {
    if (open) {
      isBasicEditDialogOpen.value = true;
      return;
    }
    cancelBasicEditing();
  }

  function handleBasicEditVersionChange(value: ComboboxOptionValue) {
    basicEditForm.version_id = String(value || '');
    basicEditErrors.version_id = '';
  }

  async function saveBasicInfo() {
    const versionId = basicEditForm.version_id;
    const instanceKey = basicEditForm.instance_key.trim();
    basicEditSubmitError.value = '';
    basicEditErrors.version_id = versionId ? '' : t('service.create.versionRequired');
    basicEditErrors.instance_key = instanceKey ? '' : t('service.create.instanceKeyRequired');
    if (basicEditErrors.version_id || basicEditErrors.instance_key) return;
    try {
      await executeOperation(async () => {
        const updated = await serviceApi.updateBasic(serviceId, {
          version_id: versionId,
          instance_key: instanceKey,
        });
        setService(updated);
        cancelBasicEditing();
        toast.success(t('service.detail.saved'));
      });
    } catch (error) {
      basicEditSubmitError.value =
        error instanceof Error ? error.message : t('service.detail.saveFailed');
    }
  }

  async function persistEnvironment(entries: EnvironmentVariableEntry[]) {
    try {
      await executeOperation(async () => {
        const updated = await serviceApi.updateEnv(serviceId, {
          env: entries,
        });
        setService(updated);
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

  function openDeployDialog() {
    deployForm.force_recreate = false;
    deploySubmitError.value = '';
    isDeployDialogOpen.value = true;
  }

  function closeDeployDialog() {
    isDeployDialogOpen.value = false;
    deploySubmitError.value = '';
  }

  function handleDeployDialogOpenChange(open: boolean) {
    if (open) {
      isDeployDialogOpen.value = true;
      return;
    }
    closeDeployDialog();
  }

  async function handleDeployOk() {
    if (!service.value) return;
    deploySubmitError.value = '';
    try {
      await executeOperation(async () => {
        const result = await serviceApi.deploy(serviceId, {
          force_recreate: deployForm.force_recreate,
        });
        for (const warning of result.warnings) toast.error(warning);
        toast.success(t('service.toast.deployQueued'));
        closeDeployDialog();
        if (result.deployment_id) await router.push(`/deployment/${result.deployment_id}`);
      });
    } catch (error) {
      deploySubmitError.value =
        error instanceof Error ? error.message : t('service.toast.deployFailed');
    }
  }

  function openDeleteDialog() {
    if (service.value?.status !== 'stopped') return;
    deleteError.value = '';
    isDeleteDialogOpen.value = true;
  }

  function closeDeleteDialog() {
    isDeleteDialogOpen.value = false;
    deleteError.value = '';
  }

  async function deleteService() {
    try {
      await executeOperation(async () => {
        await serviceApi.remove(serviceId);
        closeDeleteDialog();
        toast.success(t('service.toast.deleteSuccess'));
        await router.push('/services');
      });
    } catch (error) {
      deleteError.value = error instanceof Error ? error.message : t('service.toast.deleteFailed');
    }
  }

  onMounted(load);
  onUnmounted(stopLogsAutoRefresh);
</script>
