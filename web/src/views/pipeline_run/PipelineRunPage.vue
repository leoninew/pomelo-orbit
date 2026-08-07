<template>
  <div class="space-y-6">
    <div class="app-toolbar-simple">
      <ComboboxSelect
        v-model="pipelineId"
        :options="pipelineOptions"
        placeholder="筛选应用流水线"
        width-class="w-64"
        @update:model-value="searchRuns"
      />
    </div>
    <div class="app-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <p v-else-if="status === 'error'" class="py-16 text-center text-sm text-destructive">
        {{ error || '加载运行记录失败' }}
      </p>
      <AppEmptyState v-else-if="runs.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[1020px]">
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
            </tr>
          </thead>
          <tbody>
            <tr v-for="run in runs" :key="run.id">
              <td class="max-w-52 truncate font-mono">
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
              <td class="text-foreground">{{ run.trigger_ref }}</td>
              <td>
                <AppBadge variant="status" :tone="statusTone(run.status)">
                  {{ run.status }}
                </AppBadge>
              </td>
              <td class="whitespace-nowrap text-foreground">{{ formatTime(run.started_at) }}</td>
              <td class="whitespace-nowrap text-foreground">
                {{ formatDuration(run.started_at, run.finished_at) }}
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
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref } from 'vue';
  import { pipelineApi } from '@/api/pipeline/pipeline';
  import { pipelineRunApi } from '@/api/pipeline_run/pipeline_run';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ComboboxSelect from '@/components/ComboboxSelect.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { PipelineResp } from '@/gen/proto/orbit/v1/pipeline/pipeline';
  import type { PipelineRunResp } from '@/gen/proto/orbit/v1/pipeline_run/pipeline_run';
  import { useProjectStore } from '@/stores/project';
  import { statusTone } from '@/utils/status';
  import { formatDuration, formatTime } from '@/utils/time';

  const projectStore = useProjectStore();
  const toast = useToast();
  const { status, error, execute } = useStatusAsync();
  const runs = ref<PipelineRunResp[]>([]);
  const pipelines = ref<PipelineResp[]>([]);
  const pipelineId = ref('');
  const pagination = reactive({ current: 1, pageSize: 20, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
  const pipelineOptions = computed(() =>
    pipelines.value.map((pipeline) => ({
      value: pipeline.id,
      label: pipeline.name,
      description: `v${pipeline.version}`,
    }))
  );

  async function fetchRuns() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error('请先选择项目');
      return;
    }
    try {
      await execute(async () => {
        const response = await pipelineRunApi.list({
          project_id: projectId,
          pipeline_id: pipelineId.value || undefined,
          page: pagination.current,
          per_page: pagination.pageSize,
        });
        runs.value = response.items;
        pagination.total = response.total;
      });
    } catch (reason) {
      toast.error(reason instanceof Error ? reason.message : '加载运行记录失败');
    }
  }
  async function loadPipelines() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) return;
    const response = await pipelineApi.list({
      project_id: projectId,
      kind: 'application',
      per_page: 100,
    });
    pipelines.value = response.items;
  }
  function searchRuns() {
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
  onMounted(async () => {
    try {
      await loadPipelines();
    } catch (reason) {
      toast.error(reason instanceof Error ? reason.message : '加载流水线筛选项失败');
    }
    await fetchRuns();
  });
</script>
