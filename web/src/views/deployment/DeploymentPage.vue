<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-scroll" :aria-label="t('deployment.toolbar')">
      <div class="app-toolbar-row">
        <ComboboxSelect
          :model-value="query.application_id"
          :options="appSelectOptions"
          :placeholder="t('deployment.filterApplication')"
          width-class="app-toolbar-select"
          @update:model-value="handleApplicationChange"
        />
        <SearchControl
          v-model="query.search"
          :placeholder="t('deployment.searchPlaceholder')"
          :loading="status === 'loading'"
          class="shrink-0"
          @search="handleSearch"
        />
      </div>
    </ToolbarRoot>

    <!-- Table Card -->
    <div class="app-surface">
      <AppSpinner v-if="status === 'loading'" class="py-16" />
      <AppEmptyState v-else-if="deployments.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-table-list min-w-[1120px]">
          <thead>
            <tr>
              <th>{{ t('deployment.fields.application') }}</th>
              <th>{{ t('deployment.fields.operationType') }}</th>
              <th>{{ t('deployment.fields.triggerType') }}</th>
              <th>{{ t('common.status') }}</th>
              <th>{{ t('deployment.fields.errorMessage') }}</th>
              <th>{{ t('deployment.fields.startTime') }}</th>
              <th>{{ t('deployment.fields.duration') }}</th>
              <th>{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="deployment in deployments" :key="deployment.id">
              <td>
                <router-link :to="`/application/${deployment.application_id}`" class="app-link">
                  {{ deployment.application_name || deployment.application_id }}
                </router-link>
              </td>
              <td>
                <AppBadge variant="pill">{{ deployment.operation_type }}</AppBadge>
              </td>
              <td>
                <AppBadge variant="pill">{{ deployment.trigger_type }}</AppBadge>
              </td>
              <td>
                <AppBadge variant="status" :tone="statusTone(deployment.status)">
                  {{ deployment.status }}
                </AppBadge>
              </td>
              <td
                class="max-w-56 truncate"
                :class="deployment.error_message ? 'text-destructive' : 'text-muted-foreground'"
                :title="deployment.error_message || undefined"
              >
                {{ deployment.error_message || '—' }}
              </td>
              <td class="text-foreground">{{ formatTime(deployment.started_at) }}</td>
              <td class="text-foreground">
                {{ formatDuration(deployment.started_at, deployment.finished_at) }}
              </td>
              <td>
                <div class="flex items-center gap-3">
                  <router-link :to="`/deployment/${deployment.id}`" class="app-link">
                    {{ t('application.view') }}
                  </router-link>
                  <button
                    v-if="isCancelable(deployment)"
                    class="app-link-danger"
                    :disabled="operating"
                    @click="openCancelDialog(deployment)"
                  >
                    {{ t('common.cancel') }}
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

    <AppDialog
      v-model:open="isCancelDialogOpen"
      :title="t('deployment.dialog.confirmCancel')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">
        {{
          t('deployment.dialog.cancelConfirm', {
            name: deploymentToCancel?.application_name || t('deployment.dialog.currentApplication'),
          })
        }}
      </p>
      <template #footer>
        <button class="app-button" @click="isCancelDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-destructive" :disabled="operating" @click="handleCancelOk">
          {{ t('deployment.dialog.confirmCancel') }}
        </button>
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute } from 'vue-router';
  import { applicationApi } from '@/api/application/application';
  import { deploymentApi } from '@/api/deployment/deployment';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ComboboxSelect from '@/components/ComboboxSelect.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type { ApplicationResp } from '@/gen/proto/orbit/v1/application/application';
  import type { DeploymentResp } from '@/gen/proto/orbit/v1/deployment/deployment';
  import { statusTone } from '@/utils/status';
  import { formatDuration, formatTime } from '@/utils/time';
  import { ToolbarRoot } from 'reka-ui';

  const route = useRoute();
  const { t } = useI18n();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { status, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const deployments = ref<DeploymentResp[]>([]);
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
  const isCancelDialogOpen = ref(false);
  const deploymentToCancel = ref<DeploymentResp | null>(null);

  const query = reactive({
    search: '',
    application_id: (route.query.application_id as string) || '',
  });

  const appOptions = ref<ApplicationResp[]>([]);
  const appSelectOptions = computed(() =>
    appOptions.value.map((app) => ({
      value: app.id,
      label: app.name,
      description: app.code,
    }))
  );

  async function loadApps() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      return;
    }
    try {
      const resp = await applicationApi.list({ per_page: 100, project_id: projectId });
      appOptions.value = resp.items;
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : t('application.toast.loadFailed'));
    }
  }

  async function fetchDeployments() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('application.toast.selectProjectRequired'));
      return;
    }
    try {
      await execute(async () => {
        const res = await deploymentApi.list({
          page: pagination.current,
          per_page: pagination.pageSize,
          search: query.search || undefined,
          application_id: query.application_id || undefined,
          project_id: projectId,
        });
        deployments.value = res.items;
        pagination.total = res.total;
      });
    } catch {
      toast.error(t('deployment.toast.loadFailed'));
    }
  }

  function handleApplicationChange(value: string | number | boolean) {
    const nextValue = String(value || '');
    if (query.application_id === nextValue) {
      return;
    }
    query.application_id = nextValue;
    handleSearch();
  }

  function handleSearch() {
    pagination.current = 1;
    fetchDeployments();
  }

  function goPage(p: number) {
    pagination.current = p;
    fetchDeployments();
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    fetchDeployments();
  }

  function isCancelable(deployment: DeploymentResp) {
    return ['running', 'waiting_to_run'].includes(deployment.status);
  }

  function openCancelDialog(deployment: DeploymentResp) {
    deploymentToCancel.value = deployment;
    isCancelDialogOpen.value = true;
  }

  async function handleCancelOk() {
    if (!deploymentToCancel.value) {
      return;
    }
    const target = deploymentToCancel.value;
    try {
      await executeOp(async () => {
        await deploymentApi.cancel(target.id, {});
        toast.success(t('deployment.toast.cancelSuccess'));
        isCancelDialogOpen.value = false;
        deploymentToCancel.value = null;
        await fetchDeployments();
      });
    } catch {
      toast.error(t('deployment.toast.cancelFailed'));
    }
  }

  onMounted(async () => {
    fetchDeployments();
    await loadApps();
  });
</script>
