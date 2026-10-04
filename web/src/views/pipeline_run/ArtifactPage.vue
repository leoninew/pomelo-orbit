<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-scroll" aria-label="制品工具栏">
      <div class="app-toolbar-row">
        <RepositorySelect
          :model-value="query.repository_id"
          :project-id="projectStore.activeProjectId"
          placeholder="筛选代码仓库"
          width-class="app-toolbar-select"
          @update:model-value="handleRepositoryChange"
        />

        <SearchControl
          v-model="query.search"
          placeholder="搜索名称/位置"
          :loading="status === 'loading'"
          class="shrink-0"
          @search="handleSearch"
        />
      </div>
    </ToolbarRoot>

    <div class="app-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
        <p class="text-sm">{{ error || '加载失败' }}</p>
      </div>
      <AppEmptyState v-else-if="artifacts.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table table-fixed min-w-[960px]">
          <colgroup>
            <col class="w-[20%]" />
            <col class="w-[16%]" />
            <col class="w-[12%]" />
            <col class="w-[16%]" />
            <col class="w-[18%]" />
            <col class="w-[18%]" />
          </colgroup>
          <thead>
            <tr>
              <th>ID</th>
              <th>名称</th>
              <th>收集器</th>
              <th>构建阶段</th>
              <th>仓库</th>
              <th>创建时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in artifacts" :key="a.id">
              <td class="overflow-hidden font-mono text-xs">
                <AppTruncatedText :text="a.id" as-child>
                  <router-link :to="`/pipeline-run/artifact/${a.id}`" class="app-link">
                    {{ a.id }}
                  </router-link>
                </AppTruncatedText>
              </td>
              <td class="text-foreground">
                <AppTruncatedText :text="a.name" />
              </td>
              <td>
                <AppBadge variant="pill">
                  {{ a.collector }}
                </AppBadge>
              </td>
              <td class="text-foreground">
                <AppTruncatedText :text="a.stage_name" />
              </td>
              <td class="overflow-hidden">
                <AppTruncatedText :text="a.repository_name" as-child>
                  <router-link :to="`/repository/${a.repository_id}`" class="app-link">
                    {{ a.repository_name }}
                  </router-link>
                </AppTruncatedText>
              </td>
              <td class="text-foreground">
                <AppTableTime :time="a.created_at" />
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
  import { computed, reactive, ref, watch } from 'vue';
  import { ToolbarRoot } from 'reka-ui';
  import { artifactApi } from '@/api/pipeline_run/artifact';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import AppTableTime from '@/components/AppTableTime.vue';
  import AppTruncatedText from '@/components/AppTruncatedText.vue';
  import RepositorySelect from '@/components/RepositorySelect.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type { ArtifactResp } from '@/gen/proto/orbit/v1/pipeline_run/artifact';

  const { status, error, execute } = useStatusAsync();
  const toast = useToast();
  const projectStore = useProjectStore();
  const artifacts = ref<ArtifactResp[]>([]);
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

  const query = reactive({ search: '', repository_id: '' });

  function handleRepositoryChange(value: string | number | boolean) {
    const nextValue = String(value || '');
    if (query.repository_id === nextValue) {
      return;
    }
    query.repository_id = nextValue;
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
        const resp = await artifactApi.list(projectId, {
          page: pagination.current,
          per_page: pagination.pageSize,
          search: query.search || undefined,
          repository_id: query.repository_id || undefined,
        });
        if (projectStore.activeProjectId === projectId) {
          artifacts.value = resp.items;
          pagination.total = resp.total;
        }
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

  watch(
    () => projectStore.activeProjectId,
    () => {
      query.repository_id = '';
      query.search = '';
      artifacts.value = [];
      pagination.current = 1;
      pagination.total = 0;
      void fetchArtifacts();
    },
    { immediate: true }
  );
</script>
