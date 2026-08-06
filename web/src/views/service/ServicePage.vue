<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-simple" :aria-label="t('service.toolbar')">
      <div class="flex min-w-0 flex-wrap items-center gap-3">
        <SearchControl
          v-model="query.search"
          :placeholder="t('service.searchPlaceholder')"
          :loading="status === 'loading'"
          class="shrink-0"
          @search="handleSearch"
        />
      </div>
      <div class="ml-auto flex shrink-0 items-center gap-3">
        <ToggleGroupRoot
          v-model="viewMode"
          type="single"
          class="flex h-10 shrink-0 overflow-hidden rounded-md border border-border bg-background"
          :aria-label="t('service.viewMode')"
        >
          <ToggleGroupItem
            value="card"
            class="flex size-10 items-center justify-center text-muted-foreground outline-none transition-colors hover:bg-muted/50 hover:text-foreground data-[state=on]:bg-primary/10 data-[state=on]:text-primary"
            :aria-label="t('service.cardView')"
            :title="t('service.cardView')"
          >
            <LayoutGrid class="size-4" />
          </ToggleGroupItem>
          <ToggleGroupItem
            value="table"
            class="flex size-10 items-center justify-center text-muted-foreground outline-none transition-colors hover:bg-muted/50 hover:text-foreground data-[state=on]:bg-primary/10 data-[state=on]:text-primary"
            :aria-label="t('service.tableView')"
            :title="t('service.tableView')"
          >
            <List class="size-4" />
          </ToggleGroupItem>
        </ToggleGroupRoot>
        <button class="app-button-primary h-10 px-3" @click="openCreateDialog">
          <Plus class="size-4" />
          {{ t('service.actions.create') }}
        </button>
      </div>
    </ToolbarRoot>

    <div v-if="status === 'loading'" class="app-surface">
      <AppLoadingState />
    </div>

    <div v-else-if="status === 'error'" class="app-surface">
      <div class="py-16 text-center text-destructive">
        <p class="text-sm">{{ error || t('service.toast.loadFailed') }}</p>
      </div>
    </div>

    <div v-else-if="services.length === 0" class="app-surface">
      <AppEmptyState />
    </div>

    <div v-else-if="viewMode === 'card'" class="space-y-6">
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        <article
          v-for="svc in services"
          :key="svc.id"
          class="app-surface flex min-h-56 flex-col p-5 transition-colors hover:border-primary"
        >
          <div class="flex items-start justify-between gap-4">
            <div class="min-w-0 space-y-1">
              <h2 class="text-base font-semibold text-foreground">
                <router-link :to="`/service/${svc.id}`" class="app-link block truncate">
                  {{ svc.application_name || svc.application_id }}
                </router-link>
              </h2>
              <p
                v-if="svc.instance_key && svc.instance_key !== 'default'"
                class="truncate text-sm text-muted-foreground"
              >
                {{ svc.instance_key }}
              </p>
            </div>
            <AppBadge variant="status" :tone="appStatusTone(svc.status)">
              {{ svc.status }}
            </AppBadge>
          </div>

          <dl class="mt-5 grid gap-3 text-sm">
            <div class="flex items-center justify-between gap-3">
              <dt class="text-muted-foreground">{{ t('service.fields.version') }}</dt>
              <dd class="min-w-0 text-right">
                <router-link
                  :to="`/version/${svc.version_id}`"
                  class="app-link block truncate"
                  :title="svc.version_label || svc.version_id"
                >
                  {{ svc.version_label || svc.version_id }}
                </router-link>
              </dd>
            </div>
            <div class="flex items-center justify-between gap-3">
              <dt class="text-muted-foreground">{{ t('common.updatedAt') }}</dt>
              <dd class="whitespace-nowrap text-foreground">{{ formatTime(svc.updated_at) }}</dd>
            </div>
          </dl>

          <div
            class="mt-auto flex flex-wrap items-center gap-3 border-t border-border pt-4 text-sm"
          >
            <button
              class="app-link"
              :disabled="operating || svc.status === 'deploying'"
              @click="openDeployDialog(svc)"
            >
              {{ t('service.actions.deploy') }}
            </button>
            <button
              class="app-link-danger"
              :disabled="operating || !canStop(svc)"
              @click="openStopDialog(svc)"
            >
              {{ t('service.actions.stop') }}
            </button>
            <router-link
              :to="{ path: '/deployments', query: { application_id: svc.application_id } }"
              class="app-link"
            >
              {{ t('service.actions.deployments') }}
            </router-link>
          </div>
        </article>
      </div>

      <ListPagination
        standalone
        :current="pagination.current"
        :page-size="pagination.pageSize"
        :total="pagination.total"
        :total-pages="totalPages"
        @change-page="goPage"
        @change-page-size="handlePageSizeChange"
      />
    </div>

    <div v-else class="app-surface">
      <div class="overflow-x-auto">
        <table class="app-data-table min-w-[960px]">
          <thead>
            <tr>
              <th>{{ t('service.fields.application') }}</th>
              <th>{{ t('service.fields.version') }}</th>
              <th>{{ t('common.status') }}</th>
              <th>{{ t('common.updatedAt') }}</th>
              <th>{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="svc in services" :key="svc.id">
              <td>
                <router-link
                  :to="`/service/${svc.id}`"
                  class="app-link"
                  :title="svc.application_name || svc.application_id"
                >
                  {{ svc.application_name || svc.application_id }}
                </router-link>
                <span
                  v-if="svc.instance_key && svc.instance_key !== 'default'"
                  class="ml-2 text-xs text-muted-foreground"
                >
                  {{ svc.instance_key }}
                </span>
              </td>
              <td>
                <router-link :to="`/version/${svc.version_id}`" class="app-link">
                  {{ svc.version_label || svc.version_id }}
                </router-link>
              </td>
              <td>
                <AppBadge variant="status" :tone="appStatusTone(svc.status)">
                  {{ svc.status }}
                </AppBadge>
              </td>
              <td class="whitespace-nowrap text-foreground">{{ formatTime(svc.updated_at) }}</td>
              <td>
                <div class="flex flex-wrap items-center gap-3">
                  <button
                    class="app-link"
                    :disabled="operating || svc.status === 'deploying'"
                    @click="openDeployDialog(svc)"
                  >
                    {{ t('service.actions.deploy') }}
                  </button>
                  <button
                    class="app-link-danger"
                    :disabled="operating || !canStop(svc)"
                    @click="openStopDialog(svc)"
                  >
                    {{ t('service.actions.stop') }}
                  </button>
                  <router-link
                    :to="{ path: '/deployments', query: { application_id: svc.application_id } }"
                    class="app-link"
                  >
                    {{ t('service.actions.deployments') }}
                  </router-link>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <ListPagination
        :current="pagination.current"
        :page-size="pagination.pageSize"
        :total="pagination.total"
        :total-pages="totalPages"
        @change-page="goPage"
        @change-page-size="handlePageSizeChange"
      />
    </div>

    <AppDialog v-model:open="isDeployDialogOpen" :title="t('service.deploy.dialogTitle')">
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
          @cancel="isDeployDialogOpen = false"
          @confirm="handleDeployOk"
        />
      </template>
    </AppDialog>

    <AppDialog v-model:open="isCreateDialogOpen" :title="t('service.create.title')">
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label mb-1.5 block">
            {{ t('service.fields.application') }}
            <span class="text-destructive">*</span>
          </label>
          <ComboboxSelect
            :model-value="createForm.application_id"
            :options="applicationSelectOptions"
            :placeholder="t('service.create.selectApplication')"
            :invalid="Boolean(createErrors.application_id)"
            width-class="w-full"
            @update:model-value="handleCreateApplicationChange"
          />
          <p v-if="createErrors.application_id" class="app-field-error" role="alert">
            {{ createErrors.application_id }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label mb-1.5 block">
            {{ t('service.fields.version') }}
            <span class="text-destructive">*</span>
          </label>
          <ComboboxSelect
            :model-value="createForm.version_id"
            :options="createVersionSelectOptions"
            :placeholder="t('service.create.selectVersion')"
            :disabled="!createForm.application_id"
            :invalid="Boolean(createErrors.version_id)"
            width-class="w-full"
            @update:model-value="
              createForm.version_id = String($event || '');
              createErrors.version_id = '';
            "
          />
          <p v-if="createErrors.version_id" class="app-field-error" role="alert">
            {{ createErrors.version_id }}
          </p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label mb-1.5 block">
            {{ t('service.fields.instanceKey') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="createForm.instance_key"
            class="app-input"
            :class="createErrors.instance_key ? 'app-input-error' : ''"
            :aria-invalid="createErrors.instance_key ? 'true' : undefined"
            @input="createErrors.instance_key = ''"
          />
          <p v-if="createErrors.instance_key" class="app-field-error" role="alert">
            {{ createErrors.instance_key }}
          </p>
        </div>
        <p v-if="createError" class="app-field-error text-xs">{{ createError }}</p>
      </div>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.create')"
          @cancel="isCreateDialogOpen = false"
          @confirm="handleCreateOk"
        />
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isStopDialogOpen"
      :title="t('service.stop.dialogTitle')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <div class="space-y-4">
        <p class="text-sm text-muted-foreground">{{ stopTargetLabel }}</p>
        <p class="text-sm text-muted-foreground">{{ t('service.stop.confirm') }}</p>
        <label class="flex items-center gap-2">
          <input v-model="stopRemoveVolumes" type="checkbox" class="app-checkbox" />
          <span class="text-sm text-foreground">{{ t('service.stop.removeVolumes') }}</span>
        </label>
      </div>
      <p v-if="stopSubmitError" class="app-field-error mt-3" role="alert">
        {{ stopSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.stop')"
          variant="destructive"
          @cancel="isStopDialogOpen = false"
          @confirm="handleStopOk"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { LayoutGrid, List, Plus } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { ToggleGroupItem, ToggleGroupRoot, ToolbarRoot } from 'reka-ui';
  import { applicationApi } from '@/api/application/application';
  import { serviceApi } from '@/api/service/service';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ComboboxSelect, { type ComboboxOptionValue } from '@/components/ComboboxSelect.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { ApplicationResp } from '@/gen/proto/orbit/v1/application/application';
  import type { VersionResp } from '@/gen/proto/orbit/v1/application/version';
  import type { ServiceResp } from '@/gen/proto/orbit/v1/service/service';
  import { useProjectStore } from '@/stores/project';
  import { appStatusTone } from '@/utils/status';
  import { formatTime } from '@/utils/time';

  const { t } = useI18n();
  const toast = useToast();
  const router = useRouter();
  const projectStore = useProjectStore();
  const { status, error, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const services = ref<ServiceResp[]>([]);
  const viewMode = ref<'card' | 'table'>('table');
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize) || 1);

  const query = reactive({
    search: '',
  });

  const selectedService = ref<ServiceResp | null>(null);
  const applications = ref<ApplicationResp[]>([]);
  const versions = ref<VersionResp[]>([]);
  const isDeployDialogOpen = ref(false);
  const deployForm = reactive({
    force_recreate: false,
  });
  const deploySubmitError = ref('');
  const isStopDialogOpen = ref(false);
  const stopRemoveVolumes = ref(false);
  const stopSubmitError = ref('');
  const isCreateDialogOpen = ref(false);
  const createError = ref('');
  const createErrors = reactive({
    application_id: '',
    version_id: '',
    instance_key: '',
  });
  const createForm = reactive({
    application_id: '',
    version_id: '',
    instance_key: 'default',
  });

  const applicationSelectOptions = computed(() =>
    applications.value.map((application) => ({ value: application.id, label: application.name }))
  );
  const createVersionSelectOptions = computed(() =>
    versions.value.map((version) => ({
      value: version.id,
      label: version.label,
      description: version.status,
    }))
  );

  const deployTargetLabel = computed(() => serviceTargetLabel(selectedService.value));
  const stopTargetLabel = computed(() => serviceTargetLabel(selectedService.value));

  async function fetchServices() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      services.value = [];
      pagination.total = 0;
      return;
    }
    try {
      await execute(async () => {
        const resp = await serviceApi.list({
          project_id: projectId,
          page: pagination.current,
          per_page: pagination.pageSize,
          search: query.search || undefined,
        });
        services.value = resp.items ?? [];
        pagination.total = resp.total ?? 0;
      });
    } catch {
      services.value = [];
      pagination.total = 0;
      toast.error(t('service.toast.loadFailed'));
    }
  }

  async function openCreateDialog() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      return;
    }
    try {
      const page = await applicationApi.list({ project_id: projectId, per_page: 100 });
      applications.value = page.items ?? [];
      versions.value = [];
      Object.assign(createForm, {
        application_id: '',
        version_id: '',
        instance_key: 'default',
      });
      createError.value = '';
      Object.assign(createErrors, {
        application_id: '',
        version_id: '',
        instance_key: '',
      });
      isCreateDialogOpen.value = true;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('service.toast.loadFailed'));
    }
  }

  async function handleCreateApplicationChange(value: ComboboxOptionValue) {
    createForm.application_id = String(value || '');
    createForm.version_id = '';
    createErrors.application_id = '';
    createErrors.version_id = '';
    versions.value = [];
    if (!createForm.application_id) {
      return;
    }
    try {
      const page = await applicationApi.listVersions(createForm.application_id, { per_page: 100 });
      versions.value = page.items ?? [];
      createForm.version_id = versions.value[0]?.id ?? '';
    } catch (error) {
      createError.value = error instanceof Error ? error.message : t('service.toast.loadFailed');
    }
  }

  async function handleCreateOk() {
    createError.value = '';
    createErrors.application_id = createForm.application_id
      ? ''
      : t('service.create.applicationRequired');
    createErrors.version_id = createForm.version_id ? '' : t('service.create.versionRequired');
    createErrors.instance_key = createForm.instance_key.trim()
      ? ''
      : t('service.create.instanceKeyRequired');
    if (createErrors.application_id || createErrors.version_id || createErrors.instance_key) {
      return;
    }
    try {
      await executeOp(async () => {
        const created = await serviceApi.create({
          application_id: createForm.application_id,
          version_id: createForm.version_id,
          instance_key: createForm.instance_key.trim(),
        });
        isCreateDialogOpen.value = false;
        toast.success(t('service.create.saved'));
        await router.push(`/service/${created.id}`);
      });
    } catch (error) {
      createError.value = error instanceof Error ? error.message : t('service.toast.deployFailed');
    }
  }

  function serviceTargetLabel(service: ServiceResp | null) {
    if (!service) {
      return '';
    }
    const application = service.application_name || service.application_id;
    const instance = service.instance_key || 'default';
    return t('service.detail.subtitle', {
      instance: `${application} / ${instance}`,
      version: service.version_label || service.version_id,
    });
  }

  function canStop(service: ServiceResp) {
    return service.status === 'running' || service.status === 'faulted';
  }

  async function openDeployDialog(service: ServiceResp) {
    selectedService.value = service;
    deployForm.force_recreate = false;
    deploySubmitError.value = '';
    isDeployDialogOpen.value = true;
  }

  async function handleDeployOk() {
    const service = selectedService.value;
    if (!service) {
      return;
    }
    deploySubmitError.value = '';
    try {
      await executeOp(async () => {
        const result = await serviceApi.deploy(service.id, {
          force_recreate: deployForm.force_recreate,
        });
        for (const warning of result.warnings) toast.error(warning);
        toast.success(t('service.toast.deployQueued'));
        isDeployDialogOpen.value = false;
        if (result.deployment_id) {
          await router.push(`/deployment/${result.deployment_id}`);
          return;
        }
        await fetchServices();
      });
    } catch (error) {
      deploySubmitError.value =
        error instanceof Error ? error.message : t('service.toast.deployFailed');
    }
  }

  function openStopDialog(service: ServiceResp) {
    selectedService.value = service;
    stopRemoveVolumes.value = false;
    stopSubmitError.value = '';
    isStopDialogOpen.value = true;
  }

  async function handleStopOk() {
    const service = selectedService.value;
    if (!service) {
      return;
    }
    stopSubmitError.value = '';
    try {
      await executeOp(async () => {
        const result = await applicationApi.stop(service.application_id, {
          service_id: service.id,
          remove_volumes: stopRemoveVolumes.value,
        });
        toast.success(t('service.toast.stopQueued'));
        isStopDialogOpen.value = false;
        if (result.deployment_id) {
          await router.push(`/deployment/${result.deployment_id}`);
          return;
        }
        await fetchServices();
      });
    } catch (error) {
      stopSubmitError.value =
        error instanceof Error ? error.message : t('service.toast.stopFailed');
    }
  }

  function handleSearch() {
    pagination.current = 1;
    void fetchServices();
  }

  function goPage(page: number) {
    pagination.current = page;
    void fetchServices();
  }

  function handlePageSizeChange(size: number) {
    pagination.pageSize = size;
    pagination.current = 1;
    void fetchServices();
  }

  watch(
    () => projectStore.activeProjectId,
    () => {
      query.search = '';
      pagination.current = 1;
      void fetchServices();
    }
  );

  onMounted(() => {
    void fetchServices();
  });
</script>
