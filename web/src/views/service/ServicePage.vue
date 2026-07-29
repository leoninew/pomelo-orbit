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
      <AppSpinner class="py-16" />
    </div>

    <div v-else-if="services.length === 0" class="app-surface">
      <AppEmptyState :message="t('service.empty')" />
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
                  {{ svc.instance_key || 'default' }}
                </router-link>
              </h2>
              <router-link
                :to="`/application/${svc.application_id}`"
                class="app-link block truncate text-sm"
                :title="svc.application_name || svc.application_id"
              >
                {{ svc.application_name || svc.application_id }}
              </router-link>
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
            <button
              v-if="svc.status === 'stopped'"
              class="app-link-danger"
              :disabled="operating"
              @click="openDeleteDialog(svc)"
            >
              {{ t('service.actions.delete') }}
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
        <table class="app-data-table min-w-[1080px]">
          <thead>
            <tr>
              <th>{{ t('service.fields.instanceKey') }}</th>
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
                <router-link :to="`/service/${svc.id}`" class="app-link">
                  {{ svc.instance_key || 'default' }}
                </router-link>
              </td>
              <td>
                <router-link :to="`/application/${svc.application_id}`" class="app-link">
                  {{ svc.application_name || svc.application_id }}
                </router-link>
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
                  <button
                    v-if="svc.status === 'stopped'"
                    class="app-link-danger"
                    :disabled="operating"
                    @click="openDeleteDialog(svc)"
                  >
                    {{ t('service.actions.delete') }}
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
        <p class="text-sm text-muted-foreground">{{ t('service.deploy.description') }}</p>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('service.fields.instanceKey') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="deployForm.instance_key"
            type="text"
            required
            class="app-input"
            placeholder="default"
          />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">
            {{ t('service.fields.version') }}
            <span class="text-destructive">*</span>
          </label>
          <AppSpinner v-if="deployOptionsLoading" class="py-3" />
          <ComboboxSelect
            v-else
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
        <button
          class="app-button-primary"
          :disabled="operating || deployOptionsLoading"
          @click="handleDeployOk"
        >
          {{ t('service.actions.deploy') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog v-model:open="isCreateDialogOpen" :title="t('service.create.title')">
      <div class="space-y-4">
        <div>
          <label class="app-field-label mb-1.5 block">{{ t('service.fields.application') }}</label>
          <ComboboxSelect
            :model-value="createForm.application_id"
            :options="applicationSelectOptions"
            width-class="w-full"
            @update:model-value="handleCreateApplicationChange"
          />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">{{ t('service.fields.version') }}</label>
          <ComboboxSelect
            :model-value="createForm.version_id"
            :options="createVersionSelectOptions"
            width-class="w-full"
            @update:model-value="createForm.version_id = String($event || '')"
          />
        </div>
        <div>
          <label class="app-field-label mb-1.5 block">{{ t('service.fields.instanceKey') }}</label>
          <input v-model="createForm.instance_key" class="app-input" />
        </div>
        <div class="space-y-2">
          <div
            v-for="(item, index) in createForm.runtime_config"
            :key="index"
            class="grid grid-cols-[1fr_1fr_auto] gap-2"
          >
            <input
              v-model="item.key"
              class="app-input font-mono text-sm"
              :placeholder="t('service.runtimeConfig.key')"
            />
            <input
              v-model="item.value"
              class="app-input text-sm"
              :placeholder="t('service.runtimeConfig.value')"
            />
            <button
              class="app-button-danger size-9"
              :aria-label="t('common.delete')"
              @click="createForm.runtime_config.splice(index, 1)"
            >
              <Trash2 class="size-4" />
            </button>
          </div>
          <button class="app-link" @click="createForm.runtime_config.push({ key: '', value: '' })">
            {{ t('common.add') }}
          </button>
        </div>
        <p v-if="createError" class="app-field-error text-xs">{{ createError }}</p>
      </div>
      <template #footer>
        <button class="app-button" @click="isCreateDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="operating" @click="handleCreateOk">
          {{ t('common.create') }}
        </button>
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
        {{ t('service.delete.confirm', { instance: selectedService?.instance_key || 'default' }) }}
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
  </div>
</template>

<script setup lang="ts">
  import { LayoutGrid, List, Plus, Trash2 } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { ToggleGroupItem, ToggleGroupRoot, ToolbarRoot } from 'reka-ui';
  import { applicationApi } from '@/api/application/application';
  import { serviceApi } from '@/api/service/service';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
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
  const { status, execute } = useStatusAsync();
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
  const deployOptionsLoading = ref(false);
  const deployError = ref('');
  const deployForm = reactive({
    version_id: '',
    instance_key: 'default',
    force_recreate: false,
  });
  const isStopDialogOpen = ref(false);
  const stopRemoveVolumes = ref(false);
  const isDeleteDialogOpen = ref(false);
  const isCreateDialogOpen = ref(false);
  const createError = ref('');
  const createForm = reactive({
    application_id: '',
    version_id: '',
    instance_key: 'default',
    runtime_config: [] as Array<{ key: string; value: string }>,
  });

  const versionSelectOptions = computed(() =>
    versions.value.map((version) => ({
      value: version.id,
      label: version.label,
      description: version.status,
    }))
  );
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

  async function loadVersions(service: ServiceResp) {
    deployOptionsLoading.value = true;
    try {
      const resp = await applicationApi.listVersions(service.application_id, { per_page: 100 });
      versions.value = (resp.items ?? []).filter((version) => version.status === 'published');
      if (
        service.version_id &&
        !versions.value.some((version) => version.id === service.version_id)
      ) {
        versions.value = [
          {
            id: service.version_id,
            application_id: service.application_id,
            label: service.version_label || service.version_id,
            status: 'published',
            created_at: '',
            updated_at: '',
          } as VersionResp,
          ...versions.value,
        ];
      }
    } catch (error) {
      versions.value = [];
      toast.error(error instanceof Error ? error.message : t('service.toast.deployFailed'));
    } finally {
      deployOptionsLoading.value = false;
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
      Object.assign(createForm, {
        application_id: applications.value[0]?.id ?? '',
        version_id: '',
        instance_key: 'default',
        runtime_config: [],
      });
      createError.value = '';
      if (createForm.application_id) {
        await handleCreateApplicationChange(createForm.application_id);
      }
      isCreateDialogOpen.value = true;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('service.toast.loadFailed'));
    }
  }

  async function handleCreateApplicationChange(value: ComboboxOptionValue) {
    createForm.application_id = String(value || '');
    createForm.version_id = '';
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
    const runtime_config: Record<string, string> = {};
    for (const item of createForm.runtime_config) {
      const key = item.key.trim();
      if (!key || Object.prototype.hasOwnProperty.call(runtime_config, key)) {
        createError.value = t('service.runtimeConfig.invalid');
        return;
      }
      runtime_config[key] = item.value;
    }
    if (!createForm.application_id || !createForm.version_id || !createForm.instance_key.trim()) {
      createError.value = t('service.create.required');
      return;
    }
    try {
      await executeOp(async () => {
        const created = await serviceApi.create({
          application_id: createForm.application_id,
          version_id: createForm.version_id,
          instance_key: createForm.instance_key.trim(),
          runtime_config,
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
    versions.value = [];
    deployForm.version_id = service.version_id;
    deployForm.instance_key = 'default';
    deployForm.force_recreate = false;
    deployError.value = '';
    isDeployDialogOpen.value = true;
    await loadVersions(service);
  }

  function handleDeployVersionChange(value: ComboboxOptionValue) {
    deployForm.version_id = String(value || '');
    deployError.value = '';
  }

  async function handleDeployOk() {
    const service = selectedService.value;
    if (!service) {
      return;
    }
    if (!deployForm.version_id) {
      deployError.value = t('service.deploy.versionRequired');
      return;
    }
    try {
      await executeOp(async () => {
        const result = await applicationApi.deploy(service.application_id, {
          version_id: deployForm.version_id,
          instance_key: deployForm.instance_key,
          force_recreate: deployForm.force_recreate,
        });
        toast.success(t('service.toast.deployQueued'));
        isDeployDialogOpen.value = false;
        if (result.deployment_id) {
          await router.push(`/deployment/${result.deployment_id}`);
          return;
        }
        await fetchServices();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('service.toast.deployFailed'));
    }
  }

  function openStopDialog(service: ServiceResp) {
    selectedService.value = service;
    stopRemoveVolumes.value = false;
    isStopDialogOpen.value = true;
  }

  function openDeleteDialog(service: ServiceResp) {
    selectedService.value = service;
    isDeleteDialogOpen.value = true;
  }

  async function handleStopOk() {
    const service = selectedService.value;
    if (!service) {
      return;
    }
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
      toast.error(error instanceof Error ? error.message : t('service.toast.stopFailed'));
    }
  }

  async function handleDeleteOk() {
    const service = selectedService.value;
    if (!service) {
      return;
    }
    try {
      await executeOp(async () => {
        await serviceApi.remove(service.id);
        toast.success(t('service.toast.deleteSuccess'));
        isDeleteDialogOpen.value = false;
        if (services.value.length === 1 && pagination.current > 1) {
          pagination.current--;
        }
        await fetchServices();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('service.toast.deleteFailed'));
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
