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
          v-model="searchText"
          :placeholder="t('deployment.searchPlaceholder')"
          :loading="status === 'loading'"
          class="shrink-0"
          @search="handleSearch"
        />
        <button class="app-button h-9 px-3" type="button" @click="router.push('/dialogue')">
          <MessageSquareText :size="16" aria-hidden="true" />
          <span>{{ t('deploymentDialogue.open') }}</span>
        </button>
      </div>
    </ToolbarRoot>

    <!-- Table Card -->
    <div class="app-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <div v-else-if="status === 'error'" class="py-16 text-center text-destructive">
        <p class="text-sm">{{ error || t('deployment.toast.loadFailed') }}</p>
      </div>
      <AppEmptyState v-else-if="deployments.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[1120px]">
          <thead>
            <tr>
              <th>ID</th>
              <th>{{ t('deployment.fields.service') }}</th>
              <th>{{ t('deployment.fields.operationType') }}</th>
              <th>{{ t('deployment.fields.triggerType') }}</th>
              <th>{{ t('common.status') }}</th>
              <th>{{ t('deployment.fields.startTime') }}</th>
              <th>{{ t('deployment.fields.duration') }}</th>
              <th>{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="deployment in deployments" :key="deployment.id">
              <td class="max-w-64 truncate font-mono text-xs">
                <router-link
                  :to="`/deployment/${deployment.id}`"
                  class="app-link"
                  :title="deployment.id"
                >
                  {{ deployment.id }}
                </router-link>
              </td>
              <td>
                <router-link
                  v-if="deployment.service_id"
                  :to="`/service/${deployment.service_id}`"
                  class="app-link"
                >
                  {{
                    deployment.application_name ||
                    deployment.application_id ||
                    deployment.service_id
                  }}
                </router-link>
                <span v-else class="text-muted-foreground">-</span>
              </td>
              <td>
                <AppBadge variant="pill">{{ deployment.operation_type }}</AppBadge>
              </td>
              <td>
                <AppBadge variant="pill">{{ deployment.trigger_type }}</AppBadge>
              </td>
              <td>
                <AppBadge
                  variant="status"
                  :tone="statusTone(deployment.status)"
                  :title="
                    deployment.status === 'faulted'
                      ? deployment.error_message || undefined
                      : undefined
                  "
                >
                  {{ deployment.status }}
                </AppBadge>
              </td>
              <td class="text-foreground">{{ formatTime(deployment.started_at) }}</td>
              <td class="text-foreground">
                {{ formatDuration(deployment.started_at, deployment.finished_at) }}
              </td>
              <td class="whitespace-nowrap">
                <button
                  v-if="isComplete(deployment.status)"
                  class="app-link-danger"
                  :disabled="operating"
                  @click="openDeleteDialog(deployment)"
                >
                  {{ t('common.delete') }}
                </button>
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
      v-model:open="isDeleteDialogOpen"
      :title="t('deployment.dialog.confirmDelete')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">
        {{ t('deployment.dialog.deleteConfirm', { id: pendingDelete?.id ?? '' }) }}
      </p>
      <p v-if="deleteSubmitError" class="app-field-error mt-3" role="alert">
        {{ deleteSubmitError }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          :confirm-label="t('common.delete')"
          variant="destructive"
          @cancel="closeDeleteDialog"
          @confirm="handleDelete"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { MessageSquareText } from '@lucide/vue';
  import { applicationApi } from '@/api/application/application';
  import { deploymentApi } from '@/api/deployment/deployment';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ComboboxSelect from '@/components/ComboboxSelect.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type { ApplicationResp } from '@/gen/proto/orbit/v1/application/application';
  import type { DeploymentResp } from '@/gen/proto/orbit/v1/deployment/deployment';
  import { isComplete, statusTone } from '@/utils/status';
  import { formatDuration, formatTime } from '@/utils/time';
  import { ToolbarRoot } from 'reka-ui';

  const route = useRoute();
  const router = useRouter();
  const { t } = useI18n();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { status, error, execute } = useStatusAsync();
  const { loading: operating, execute: executeOperation } = useStatusAsync();

  const deployments = ref<DeploymentResp[]>([]);
  const isDeleteDialogOpen = ref(false);
  const pendingDelete = ref<DeploymentResp>();
  const deleteSubmitError = ref('');
  const searchText = ref('');
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

  const query = reactive({
    application_id: (route.query.application_id as string) || '',
  });

  const appOptions = ref<ApplicationResp[]>([]);
  const appSelectOptions = computed(() =>
    appOptions.value.map((app) => ({
      value: app.id,
      label: app.name,
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
          application_id: query.application_id || undefined,
          search: searchText.value.trim() || undefined,
          project_id: projectId,
        });
        deployments.value = res.items;
        pagination.total = res.total;
      });
    } catch {
      toast.error(t('deployment.toast.loadFailed'));
    }
  }

  function handleSearch() {
    pagination.current = 1;
    fetchDeployments();
  }

  function handleApplicationChange(value: string | number | boolean) {
    const nextValue = String(value || '');
    if (query.application_id === nextValue) {
      return;
    }
    query.application_id = nextValue;
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

  function openDeleteDialog(deployment: DeploymentResp) {
    pendingDelete.value = deployment;
    deleteSubmitError.value = '';
    isDeleteDialogOpen.value = true;
  }

  function closeDeleteDialog() {
    isDeleteDialogOpen.value = false;
    pendingDelete.value = undefined;
    deleteSubmitError.value = '';
  }

  async function handleDelete() {
    const deployment = pendingDelete.value;
    if (!deployment) {
      return;
    }
    deleteSubmitError.value = '';
    try {
      await executeOperation(async () => {
        await deploymentApi.delete(deployment.id);
        toast.success(t('deployment.toast.deleteSuccess'));
        if (deployments.value.length === 1 && pagination.current > 1) {
          pagination.current -= 1;
        }
        closeDeleteDialog();
        await fetchDeployments();
      });
    } catch (err: unknown) {
      deleteSubmitError.value =
        err instanceof Error ? err.message : t('deployment.toast.deleteFailed');
    }
  }

  onMounted(async () => {
    fetchDeployments();
    await loadApps();
  });
</script>
