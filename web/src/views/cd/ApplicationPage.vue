<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-simple" :aria-label="t('application.toolbar')">
      <SearchControl
        v-model="searchText"
        :placeholder="t('application.searchPlaceholder')"
        :loading="status === 'loading'"
        class="shrink-0"
        @search="handleSearch"
      />
      <div class="flex items-center gap-3">
        <ToggleGroupRoot
          v-model="viewMode"
          type="single"
          class="flex h-10 overflow-hidden rounded-md border border-border bg-background"
          :aria-label="t('application.viewMode')"
        >
          <ToggleGroupItem
            value="card"
            class="flex size-10 items-center justify-center text-muted-foreground outline-none transition-colors hover:bg-muted/50 hover:text-foreground data-[state=on]:bg-primary/10 data-[state=on]:text-primary"
            :aria-label="t('application.cardView')"
          >
            <LayoutGrid class="size-4" />
          </ToggleGroupItem>
          <ToggleGroupItem
            value="table"
            class="flex size-10 items-center justify-center text-muted-foreground outline-none transition-colors hover:bg-muted/50 hover:text-foreground data-[state=on]:bg-primary/10 data-[state=on]:text-primary"
            :aria-label="t('application.tableView')"
          >
            <List class="size-4" />
          </ToggleGroupItem>
        </ToggleGroupRoot>
        <button
          class="app-button-primary px-5"
          :disabled="status === 'loading'"
          @click="openCreateDialog"
        >
          <Plus class="size-4" />
          {{ t('application.createApplication') }}
        </button>
        <button class="app-button px-5" @click="triggerImport">
          <Upload class="size-4" />
          {{ t('application.import') }}
        </button>
        <input
          ref="fileInput"
          type="file"
          accept=".json"
          class="hidden"
          @change="handleFileImport"
        />
      </div>
    </ToolbarRoot>

    <!-- 加载 -->
    <div v-if="status === 'loading'" class="app-surface">
      <AppSpinner class="py-16" />
    </div>

    <!-- 卡片视图 -->
    <div v-else-if="viewMode === 'card'" class="space-y-6">
      <div v-if="applications.length === 0" class="app-surface">
        <AppEmptyState />
      </div>
      <template v-else>
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          <div
            v-for="app in applications"
            :key="app.id"
            class="app-surface group flex min-h-56 cursor-pointer flex-col p-5 transition-colors hover:border-primary"
            @click="router.push(`/cd/applications/${app.id}`)"
          >
            <div class="flex items-start justify-between gap-4">
              <div class="min-w-0 space-y-1">
                <h3
                  class="truncate text-base font-semibold text-foreground group-hover:text-primary"
                >
                  {{ app.name }}
                </h3>
                <div class="flex items-center gap-2">
                  <span class="font-mono text-xs text-muted-foreground">{{ app.code }}</span>
                  <span class="h-1 w-1 rounded-full bg-muted-foreground/40" />
                  <span class="text-xs text-muted-foreground">
                    {{ formatTime(app.created_at) }}
                  </span>
                </div>
              </div>
              <AppBadge variant="status" :tone="appStatusTone(app.status)">
                {{ t('status.' + app.status) }}
              </AppBadge>
            </div>

            <div class="mt-5 grid gap-3 text-sm">
              <div class="flex items-center justify-between gap-3">
                <span class="text-muted-foreground">{{ t('application.imagePull') }}</span>
                <span class="text-foreground">
                  {{ pullPolicyLabel(app.image_pull_policy) }}
                </span>
              </div>
              <div class="flex items-center justify-between gap-3">
                <span class="text-muted-foreground">{{ t('application.routeManaged') }}</span>
                <span class="text-foreground">
                  {{ routeManagedLabel(app.route_managed) }}
                </span>
              </div>
            </div>

            <div
              class="mt-auto flex items-center justify-end gap-3 border-t border-border pt-4 text-sm"
              @click.stop
            >
              <button class="app-link" @click="router.push(`/cd/applications/${app.id}`)">
                {{ t('application.view') }}
              </button>
              <button
                v-if="app.status === 'deployed'"
                class="app-link-danger"
                :disabled="isAppOperating(app)"
                @click="handleStop(app)"
              >
                {{ t('application.stop') }}
              </button>
              <button
                v-else
                class="app-link"
                :disabled="app.status === 'deploying' || isAppOperating(app)"
                @click="handleDeploy(app)"
              >
                {{ t('application.deploy') }}
              </button>
            </div>
          </div>
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
      </template>
    </div>

    <!-- 表格视图 -->
    <template v-else>
      <div class="app-surface">
        <AppEmptyState v-if="applications.length === 0" />
        <div v-else class="overflow-x-auto">
          <table class="app-table-list min-w-[1040px]">
            <thead>
              <tr>
                <th>{{ t('common.name') }}</th>
                <th>{{ t('application.code') }}</th>
                <th>{{ t('application.imagePullPolicy') }}</th>
                <th>{{ t('common.status') }}</th>
                <th>{{ t('application.routeManaged') }}</th>
                <th>{{ t('common.createdAt') }}</th>
                <th>{{ t('common.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="app in applications" :key="app.id">
                <td>
                  <button class="app-link" @click="router.push(`/cd/applications/${app.id}`)">
                    {{ app.name }}
                  </button>
                </td>
                <td class="text-foreground">{{ app.code }}</td>
                <td class="text-foreground">{{ pullPolicyLabel(app.image_pull_policy) }}</td>
                <td>
                  <AppBadge variant="status" :tone="appStatusTone(app.status)" class="font-normal">
                    {{ t('status.' + app.status) }}
                  </AppBadge>
                </td>
                <td class="text-foreground">{{ routeManagedLabel(app.route_managed) }}</td>
                <td class="text-foreground">{{ formatTime(app.created_at) }}</td>
                <td>
                  <div class="flex items-center gap-3">
                    <button class="app-link" @click="router.push(`/cd/applications/${app.id}`)">
                      {{ t('application.view') }}
                    </button>
                    <button
                      v-if="app.status === 'deployed'"
                      class="app-link-danger"
                      :disabled="operating"
                      @click="handleStop(app)"
                    >
                      {{ t('application.stop') }}
                    </button>
                    <button
                      v-else
                      class="app-link"
                      :disabled="app.status === 'deploying' || operating"
                      @click="handleDeploy(app)"
                    >
                      {{ t('application.deploy') }}
                    </button>
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
    </template>

    <AppDialog v-model:open="isCreateDialogOpen" :title="t('application.createApplication')">
      <ApplicationFormFields
        :form="createForm"
        :errors="createErrors"
        @update:form="Object.assign(createForm, $event)"
      />
      <template #footer>
        <button class="app-button" @click="isCreateDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="operating" @click="handleCreateOk">
          {{ t('application.create') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog v-model:open="isImportDialogOpen" :title="t('application.importApplication')">
      <ApplicationFormFields
        :form="importForm"
        :errors="importErrors"
        @update:form="Object.assign(importForm, $event)"
      />
      <div class="app-tip">
        {{ importSummary }}
      </div>
      <template #footer>
        <button class="app-button" @click="isImportDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="operating" @click="handleImportOk">
          {{ t('application.import') }}
        </button>
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { LayoutGrid, List, Plus, Upload } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { applicationApi } from '@/api/cd/application';
  import AppBadge from '@/components/AppBadge.vue';
  import ApplicationFormFields from '@/components/ApplicationFormFields.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type {
    Application,
    ApplicationFormState,
    ApplicationImportReq,
    ApplicationImportState,
  } from '@/types/cd/application';
  import { appStatusTone } from '@/utils/status';
  import { formatTime } from '@/utils/time';
  import { ToggleGroupItem, ToggleGroupRoot, ToolbarRoot } from 'reka-ui';

  const router = useRouter();
  const { t } = useI18n();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { status, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const applications = ref<Application[]>([]);
  const searchText = ref('');
  const viewMode = ref<'card' | 'table'>('card');
  const isCreateDialogOpen = ref(false);
  const isImportDialogOpen = ref(false);
  const fileInput = ref<HTMLInputElement>();
  const operatingAppId = ref<string | null>(null);
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
  const createForm = reactive<ApplicationFormState>({
    name: '',
    code: '',
    image_pull_policy: 'missing',
    route_managed: false,
  });
  const createErrors = reactive({ name: '', code: '' });
  const importForm = reactive<ApplicationImportState>({
    version: undefined,
    name: '',
    code: '',
    image_pull_policy: 'missing',
    route_managed: false,
    config_files: [],
    service_configs: [],
    routes: [],
  });
  const importErrors = reactive({ name: '', code: '' });
  const importSummary = computed(() =>
    t('application.importSummary', {
      configFiles: importForm.config_files.length,
      serviceConfigs: importForm.service_configs.length,
      routes: importForm.routes.length,
    })
  );

  function pullPolicyLabel(policy: string) {
    return t(`application.imagePullPolicyLabels.${policy}`);
  }

  function routeManagedLabel(enabled: boolean) {
    return t(
      enabled ? 'application.routeManagedLabels.enabled' : 'application.routeManagedLabels.disabled'
    );
  }

  function isAppOperating(app: Application) {
    return operating && operatingAppId.value === app.id;
  }

  async function fetchApplications() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('application.toast.selectProjectRequired'));
      return;
    }
    try {
      await execute(async () => {
        const res = await applicationApi.list({
          page: pagination.current,
          per_page: pagination.pageSize,
          search: searchText.value || undefined,
          project_id: projectId,
        });
        applications.value = res.items;
        pagination.total = res.total;
      });
    } catch {
      toast.error(t('application.toast.loadFailed'));
    }
  }

  function handleSearch() {
    pagination.current = 1;
    fetchApplications();
  }

  function goPage(p: number) {
    pagination.current = p;
    fetchApplications();
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    fetchApplications();
  }

  function validateForm(target: ApplicationFormState, errors: { name: string; code: string }) {
    errors.name = target.name.trim() ? '' : t('application.validation.nameRequired');
    errors.code = /^[a-z][a-z0-9-]*$/.test(target.code)
      ? ''
      : t('application.validation.codeInvalid');
    return !errors.name && !errors.code;
  }

  function openCreateDialog() {
    Object.assign(createForm, {
      name: '',
      code: '',
      image_pull_policy: 'missing',
      route_managed: false,
    });
    Object.assign(createErrors, { name: '', code: '' });
    isCreateDialogOpen.value = true;
  }

  async function handleCreateOk() {
    if (!validateForm(createForm, createErrors)) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('application.toast.selectProjectRequired'));
      return;
    }
    try {
      await executeOp(async () => {
        await applicationApi.create(
          {
            name: createForm.name,
            code: createForm.code,
            image_pull_policy: createForm.image_pull_policy,
            route_managed: createForm.route_managed,
          },
          { project_id: projectId }
        );
        toast.success(t('application.toast.createSuccess'));
        isCreateDialogOpen.value = false;
        await fetchApplications();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.createFailed'));
    }
  }

  function triggerImport() {
    fileInput.value?.click();
  }

  async function handleFileImport(event: Event) {
    const target = event.target as HTMLInputElement;
    const file = target.files?.[0];
    if (!file) {
      return;
    }
    try {
      const data = JSON.parse(await file.text()) as ApplicationImportReq;
      if (!data.name || !data.code) {
        toast.error(t('application.validation.importMissingRequiredFields'));
        return;
      }
      Object.assign(importForm, {
        version: data.version,
        name: data.name,
        code: data.code,
        image_pull_policy: data.image_pull_policy,
        route_managed: data.route_managed,
        config_files: data.config_files ?? [],
        service_configs: data.service_configs ?? [],
        routes: data.routes ?? [],
      });
      Object.assign(importErrors, { name: '', code: '' });
      isImportDialogOpen.value = true;
    } catch {
      toast.error(t('application.toast.parseImportFailed'));
    } finally {
      target.value = '';
    }
  }

  async function handleImportOk() {
    if (!validateForm(importForm, importErrors)) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('application.toast.selectProjectRequired'));
      return;
    }
    try {
      await executeOp(async () => {
        await applicationApi.importApplication(
          {
            version: importForm.version,
            name: importForm.name,
            code: importForm.code,
            image_pull_policy: importForm.image_pull_policy,
            route_managed: importForm.route_managed,
            config_files: importForm.config_files,
            service_configs: importForm.service_configs,
            routes: importForm.routes,
          },
          { project_id: projectId }
        );
        toast.success(t('application.toast.importSuccess'));
        isImportDialogOpen.value = false;
        await fetchApplications();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.importFailed'));
    }
  }

  async function handleDeploy(app: Application) {
    operatingAppId.value = app.id;
    try {
      await executeOp(async () => {
        const { deployment_id } = await applicationApi.deploy(app.id);
        toast.success(t('application.toast.deployTriggered', { name: app.name }));
        router.push(`/cd/deployments/${deployment_id}`);
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.deployFailed'));
    } finally {
      operatingAppId.value = null;
    }
  }

  async function handleStop(app: Application) {
    operatingAppId.value = app.id;
    try {
      await executeOp(async () => {
        const { deployment_id } = await applicationApi.stop(app.id);
        toast.success(t('application.toast.stopTriggered', { name: app.name }));
        router.push(`/cd/deployments/${deployment_id}`);
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('application.toast.stopFailed'));
    } finally {
      operatingAppId.value = null;
    }
  }

  onMounted(fetchApplications);
</script>
