<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-scroll" aria-label="制品工具栏">
      <div class="app-toolbar-row">
        <ComboboxSelect
          :model-value="query.repository_id"
          :options="repoSelectOptions"
          placeholder="筛选项目"
          width-class="app-toolbar-select"
          @update:model-value="handleRepositoryChange"
        />

        <ComboboxSelect
          :model-value="query.template_id"
          :options="templateSelectOptions"
          placeholder="筛选模板"
          width-class="app-toolbar-select"
          @update:model-value="handleTemplateChange"
        />

        <SearchControl
          v-model="query.search"
          placeholder="搜索名称/路径"
          :loading="status === 'loading'"
          class="shrink-0"
          @search="handleSearch"
        />
      </div>
    </ToolbarRoot>

    <div class="app-surface">
      <AppSpinner v-if="status === 'loading'" class="py-16" />
      <div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
        <p class="text-sm">{{ error || '加载失败' }}</p>
      </div>
      <AppEmptyState v-else-if="artifacts.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-table-list table-fixed">
          <colgroup>
            <col class="w-[17%]" />
            <col class="w-[17%]" />
            <col class="w-[12%]" />
            <col class="w-[13%]" />
            <col class="w-[18%]" />
            <col class="w-[16%]" />
            <col class="w-[7%]" />
          </colgroup>
          <thead>
            <tr>
              <th>项目</th>
              <th>模板</th>
              <th>类型</th>
              <th>Stage</th>
              <th>路径/镜像</th>
              <th>创建时间</th>
              <th>运行记录</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in artifacts" :key="a.id">
              <td class="overflow-hidden">
                <router-link
                  :to="`/repository/${a.repository_id}`"
                  class="app-link block truncate"
                  :title="a.repository_name"
                >
                  {{ a.repository_name }}
                </router-link>
              </td>
              <td class="overflow-hidden">
                <router-link
                  :to="`/pipeline/template/${a.template_id}`"
                  class="app-link block truncate"
                  :title="a.template_name"
                >
                  {{ a.template_name }}
                </router-link>
              </td>
              <td>
                <AppBadge variant="pill">
                  {{ a.type }}
                </AppBadge>
              </td>
              <td class="overflow-hidden truncate text-foreground" :title="a.stage_name">
                {{ a.stage_name }}
              </td>
              <td class="overflow-hidden truncate text-foreground" :title="a.path || undefined">
                {{ a.path ?? '—' }}
              </td>
              <td
                class="overflow-hidden truncate text-foreground"
                :title="formatTime(a.created_at)"
              >
                {{ formatTime(a.created_at) }}
              </td>
              <td>
                <router-link :to="`/pipeline-run/${a.pipeline_run_id}`" class="app-link">
                  查看
                </router-link>
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
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref } from 'vue';
  import { ToolbarRoot } from 'reka-ui';
  import { pipelineTemplateApi } from '@/api/pipeline/template';
  import { artifactApi } from '@/api/pipeline_run/artifact';
  import { repositoryApi } from '@/api/repository/repository';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ComboboxSelect from '@/components/ComboboxSelect.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type { ArtifactResp } from '@/gen/proto/orbit/v1/pipeline_run/artifact';
  import type { RepositoryResp } from '@/gen/proto/orbit/v1/repository/repository';
  import type { PipelineTemplateResp } from '@/gen/proto/orbit/v1/pipeline/template';
  import { formatTime } from '@/utils/time';

  const { status, error, execute } = useStatusAsync();
  const toast = useToast();
  const projectStore = useProjectStore();
  const artifacts = ref<ArtifactResp[]>([]);
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

  const query = reactive({ search: '', repository_id: '', template_id: '' });

  const repoOptions = ref<RepositoryResp[]>([]);
  const templateOptions = ref<PipelineTemplateResp[]>([]);

  const repoSelectOptions = computed(() =>
    repoOptions.value.map((repo) => ({
      value: repo.id,
      label: repo.name,
      description: repo.repository_url,
    }))
  );

  const templateSelectOptions = computed(() =>
    templateOptions.value.map((template) => ({
      value: template.id,
      label: template.name,
      description: `v${template.version}`,
    }))
  );

  async function loadRepos() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      return;
    }
    try {
      const resp = await repositoryApi.list({ per_page: 100, project_id: projectId });
      repoOptions.value = resp.items;
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : '获取项目列表失败');
    }
  }

  async function loadTemplates() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      return;
    }
    try {
      const resp = await pipelineTemplateApi.list({ per_page: 100, project_id: projectId });
      templateOptions.value = resp.items;
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : '获取模板列表失败');
    }
  }

  function handleRepositoryChange(value: string | number | boolean) {
    const nextValue = String(value || '');
    if (query.repository_id === nextValue) {
      return;
    }
    query.repository_id = nextValue;
    handleSearch();
  }

  function handleTemplateChange(value: string | number | boolean) {
    const nextValue = String(value || '');
    if (query.template_id === nextValue) {
      return;
    }
    query.template_id = nextValue;
    handleSearch();
  }

  function handleSearch() {
    pagination.current = 1;
    fetchArtifacts();
  }

  async function fetchArtifacts() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    try {
      await execute(async () => {
        const resp = await artifactApi.list({
          page: pagination.current,
          per_page: pagination.pageSize,
          search: query.search || undefined,
          repository_id: query.repository_id || undefined,
          template_id: query.template_id || undefined,
          project_id: projectId,
        });
        artifacts.value = resp.items;
        pagination.total = resp.total;
      });
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : '获取制品列表失败');
    }
  }

  function goPage(p: number) {
    pagination.current = p;
    fetchArtifacts();
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    fetchArtifacts();
  }

  onMounted(async () => {
    fetchArtifacts();
    await loadRepos();
    await loadTemplates();
  });
</script>
