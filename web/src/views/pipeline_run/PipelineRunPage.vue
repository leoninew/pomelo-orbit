<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-scroll" aria-label="运行记录工具栏">
      <div class="app-toolbar-row">
        <ComboboxSelect
          v-model="repositoryId"
          :options="repositoryOptions"
          placeholder="筛选代码仓库"
          width-class="app-toolbar-select"
          @update:model-value="searchRuns"
        />
        <SearchControl
          v-model="searchText"
          :placeholder="t('pipelineRun.searchPlaceholder')"
          :loading="status === 'loading'"
          class="shrink-0"
          @search="handleSearch"
        />
      </div>
    </ToolbarRoot>
    <div class="app-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <p v-else-if="status === 'error'" class="py-16 text-center text-sm text-destructive">
        {{ error || '加载运行记录失败' }}
      </p>
      <AppEmptyState v-else-if="filteredRuns.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[1120px]">
          <thead>
            <tr>
              <th>ID</th>
              <th>流水线</th>
              <th>仓库</th>
              <th>配置版本</th>
              <th>Ref</th>
              <th>状态</th>
              <th>开始时间</th>
              <th>耗时</th>
              <th>{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="run in filteredRuns" :key="run.id">
              <td class="max-w-52 truncate font-mono text-xs">
                <router-link :to="`/pipeline-run/${run.id}`" class="app-link">
                  {{ run.id }}
                </router-link>
              </td>
              <td>
                <router-link :to="`/pipeline/${run.pipeline_id}`" class="app-link">
                  {{ run.pipeline_name }}
                </router-link>
              </td>
              <td>
                <router-link :to="`/repository/${run.repository_id}`" class="app-link">
                  {{ run.repository_name }}
                </router-link>
              </td>
              <td>
                <AppBadge>v{{ run.pipeline_version }}</AppBadge>
              </td>
              <td class="text-foreground">{{ run.repository_ref }}</td>
              <td>
                <AppBadge variant="status" :tone="statusTone(run.status)">
                  {{ run.status }}
                </AppBadge>
              </td>
              <td class="whitespace-nowrap text-foreground">{{ formatTime(run.started_at) }}</td>
              <td class="whitespace-nowrap text-foreground">
                {{ formatDuration(run.started_at, run.finished_at) }}
              </td>
              <td class="whitespace-nowrap">
                <button
                  v-if="isComplete(run.status)"
                  class="app-link-danger"
                  :disabled="operating"
                  @click="openDeleteDialog(run)"
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
        @change-page-size="changePageSize"
      />
    </div>

    <AppDialog
      v-model:open="isDeleteDialogOpen"
      :title="t('pipelineRun.confirmDelete')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-foreground">
        {{ t('pipelineRun.deleteConfirm', { id: pendingDelete?.id ?? '' }) }}
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
  import { ToolbarRoot } from 'reka-ui';
  import { useI18n } from 'vue-i18n';
  import { pipelineRunApi } from '@/api/pipeline_run/pipeline_run';
  import { repositoryApi } from '@/api/repository/repository';
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
  import type { PipelineRunResp } from '@/gen/proto/orbit/v1/pipeline_run/pipeline_run';
  import type { RepositoryResp } from '@/gen/proto/orbit/v1/repository/repository';
  import { useProjectStore } from '@/stores/project';
  import { isComplete, statusTone } from '@/utils/status';
  import { formatDuration, formatTime } from '@/utils/time';

  const projectStore = useProjectStore();
  const toast = useToast();
  const { t } = useI18n();
  const { status, error, execute } = useStatusAsync();
  const { loading: operating, execute: executeOperation } = useStatusAsync();
  const runs = ref<PipelineRunResp[]>([]);
  const repositories = ref<RepositoryResp[]>([]);
  const repositoryId = ref('');
  const searchText = ref('');
  const appliedSearch = ref('');
  const isDeleteDialogOpen = ref(false);
  const pendingDelete = ref<PipelineRunResp>();
  const deleteSubmitError = ref('');
  const pagination = reactive({ current: 1, pageSize: 20, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
  const repositoryOptions = computed(() =>
    repositories.value.map((repository) => ({
      value: repository.id,
      label: repository.name,
    }))
  );
  const filteredRuns = computed(() => {
    const keyword = appliedSearch.value.trim().toLowerCase();
    if (!keyword) {
      return runs.value;
    }
    return runs.value.filter(
      (run) =>
        run.pipeline_name.toLowerCase().includes(keyword) ||
        run.repository_name.toLowerCase().includes(keyword) ||
        run.repository_ref.toLowerCase().includes(keyword) ||
        run.status.toLowerCase().includes(keyword) ||
        run.id.toLowerCase().includes(keyword)
    );
  });

  async function fetchRuns() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    try {
      await execute(async () => {
        const response = await pipelineRunApi.list(projectId, {
          repository_id: repositoryId.value || undefined,
          page: pagination.current,
          per_page: pagination.pageSize,
        });
        if (projectStore.activeProjectId === projectId) {
          runs.value = response.items;
          pagination.total = response.total;
        }
      });
    } catch (reason) {
      toast.error(reason instanceof Error ? reason.message : '加载运行记录失败');
    }
  }
  async function loadRepositories() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      return;
    }
    const response = await repositoryApi.list(projectId, { per_page: 100 });
    if (projectStore.activeProjectId === projectId) {
      repositories.value = response.items;
    }
  }
  function searchRuns() {
    pagination.current = 1;
    void fetchRuns();
  }
  function handleSearch() {
    appliedSearch.value = searchText.value;
    pagination.current = 1;
    void fetchRuns();
  }
  function goPage(page: number) {
    pagination.current = page;
    void fetchRuns();
  }
  function changePageSize(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    void fetchRuns();
  }
  function openDeleteDialog(run: PipelineRunResp) {
    pendingDelete.value = run;
    deleteSubmitError.value = '';
    isDeleteDialogOpen.value = true;
  }
  function closeDeleteDialog() {
    isDeleteDialogOpen.value = false;
    pendingDelete.value = undefined;
    deleteSubmitError.value = '';
  }
  async function handleDelete() {
    const run = pendingDelete.value;
    const projectId = projectStore.activeProjectId;
    if (!run || !projectId) {
      return;
    }
    deleteSubmitError.value = '';
    try {
      await executeOperation(async () => {
        await pipelineRunApi.delete(projectId, run.id);
        toast.success(t('pipelineRun.toast.deleteSuccess'));
        if (runs.value.length === 1 && pagination.current > 1) {
          pagination.current -= 1;
        }
        closeDeleteDialog();
        await fetchRuns();
      });
    } catch (error) {
      deleteSubmitError.value =
        error instanceof Error ? error.message : t('pipelineRun.toast.deleteFailed');
    }
  }
  onMounted(async () => {
    try {
      await loadRepositories();
    } catch (reason) {
      toast.error(reason instanceof Error ? reason.message : '加载仓库筛选项失败');
    }
    await fetchRuns();
  });
</script>
