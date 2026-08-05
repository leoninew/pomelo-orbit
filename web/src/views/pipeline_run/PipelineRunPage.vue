<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-scroll" :aria-label="t('pipelineRun.toolbar')">
      <div class="app-toolbar-row">
        <ComboboxSelect
          :model-value="query.repository_id"
          :options="repositoryOptions"
          :placeholder="t('pipelineRun.filterRepository')"
          width-class="app-toolbar-select"
          @update:model-value="handleRepositoryChange"
        />
        <ComboboxSelect
          :model-value="query.template_id"
          :options="templateOptions"
          :placeholder="t('pipelineRun.filterTemplate')"
          width-class="app-toolbar-select"
          @update:model-value="handleTemplateChange"
        />
      </div>
    </ToolbarRoot>

    <div class="app-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <div v-else-if="status === 'error'" class="py-16 text-center text-destructive">
        <p class="text-sm">{{ error || t('pipelineRun.toast.loadFailed') }}</p>
      </div>
      <AppEmptyState v-else-if="runs.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table table-fixed min-w-[1320px]">
          <colgroup>
            <col class="w-[22%]" />
            <col class="w-[13%]" />
            <col class="w-[14%]" />
            <col class="w-[7%]" />
            <col class="w-[12%]" />
            <col class="w-[9%]" />
            <col class="w-[13%]" />
            <col class="w-[10%]" />
          </colgroup>
          <thead>
            <tr>
              <th>ID</th>
              <th>{{ t('pipelineRun.fields.repository') }}</th>
              <th>{{ t('pipelineRun.fields.template') }}</th>
              <th>{{ t('pipelineRun.fields.version') }}</th>
              <th>{{ t('pipelineRun.fields.triggerRef') }}</th>
              <th>{{ t('common.status') }}</th>
              <th>{{ t('pipelineRun.fields.startTime') }}</th>
              <th>{{ t('pipelineRun.fields.duration') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="run in runs" :key="run.id">
              <td class="overflow-hidden">
                <router-link
                  :to="`/pipeline-run/${run.id}`"
                  class="app-link block truncate font-mono"
                  :title="run.id"
                >
                  {{ run.id }}
                </router-link>
              </td>
              <td class="overflow-hidden">
                <router-link
                  :to="`/repository/${run.repository_id}`"
                  class="app-link block truncate"
                  :title="run.repository_name"
                >
                  {{ run.repository_name }}
                </router-link>
              </td>
              <td class="overflow-hidden">
                <router-link
                  :to="`/pipeline/template/${run.template_id}`"
                  class="app-link block truncate"
                  :title="run.template_name"
                >
                  {{ run.template_name }}
                </router-link>
              </td>
              <td>
                <AppBadge variant="default">v{{ run.template_version }}</AppBadge>
              </td>
              <td class="overflow-hidden truncate text-foreground" :title="run.trigger_ref">
                {{ run.trigger_ref }}
              </td>
              <td>
                <AppBadge
                  variant="status"
                  :tone="statusTone(run.status)"
                  :title="run.status === 'faulted' ? run.error_message || undefined : undefined"
                >
                  {{ run.status }}
                </AppBadge>
              </td>
              <td class="whitespace-nowrap text-foreground" :title="formatTime(run.started_at)">
                {{ formatTime(run.started_at) }}
              </td>
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
        @change-page-size="handlePageSizeChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute } from 'vue-router';
  import { pipelineTemplateApi } from '@/api/pipeline/template';
  import { pipelineRunApi } from '@/api/pipeline_run/pipeline_run';
  import { repositoryApi } from '@/api/repository/repository';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ComboboxSelect from '@/components/ComboboxSelect.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type { PipelineRunResp } from '@/gen/proto/orbit/v1/pipeline_run/pipeline_run';
  import type { RepositoryResp } from '@/gen/proto/orbit/v1/repository/repository';
  import type { PipelineTemplateResp } from '@/gen/proto/orbit/v1/pipeline/template';
  import { statusTone } from '@/utils/status';
  import { formatTime, formatDuration } from '@/utils/time';
  import { ToolbarRoot } from 'reka-ui';

  const route = useRoute();
  const { t } = useI18n();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { status, error, execute } = useStatusAsync();

  const runs = ref<PipelineRunResp[]>([]);
  const repositories = ref<RepositoryResp[]>([]);
  const templates = ref<PipelineTemplateResp[]>([]);
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
  const query = reactive({ repository_id: '', template_id: '' });

  const repositoryOptions = computed(() =>
    repositories.value.map((repo) => ({
      value: repo.id,
      label: repo.name,
      description: repo.repository_url,
    }))
  );
  const templateOptions = computed(() =>
    templates.value.map((template) => ({
      value: template.id,
      label: template.name,
      description: `v${template.version}`,
    }))
  );

  async function fetchRuns() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('pipelineRun.toast.selectProjectRequired'));
      return;
    }
    try {
      await execute(async () => {
        const res = await pipelineRunApi.list({
          page: pagination.current,
          per_page: pagination.pageSize,
          repository_id: query.repository_id || undefined,
          template_id: query.template_id || undefined,
          project_id: projectId,
        });
        runs.value = res.items;
        pagination.total = res.total;
      });
    } catch {
      toast.error(t('pipelineRun.toast.loadFailed'));
    }
  }

  async function loadRepositories() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      return;
    }
    try {
      const res = await repositoryApi.list({ per_page: 100, project_id: projectId });
      repositories.value = res.items;
    } catch (err: unknown) {
      toast.error(
        err instanceof Error ? err.message : t('pipelineRun.toast.loadRepositoriesFailed')
      );
    }
  }

  async function loadTemplates() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      return;
    }
    try {
      const res = await pipelineTemplateApi.list({ per_page: 100, project_id: projectId });
      templates.value = res.items;
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : t('pipelineRun.toast.loadTemplatesFailed'));
    }
  }

  async function ensureRepositoryOption(id: string) {
    if (!id || repositories.value.some((repo) => repo.id === id)) {
      return;
    }
    try {
      const repo = await repositoryApi.get(id);
      repositories.value = [repo, ...repositories.value];
    } catch {
      // The filter still works by id; missing display text should not block the page.
    }
  }

  async function ensureTemplateOption(id: string) {
    if (!id || templates.value.some((template) => template.id === id)) {
      return;
    }
    try {
      const template = await pipelineTemplateApi.get(id);
      templates.value = [template, ...templates.value];
    } catch {
      // The filter still works by id; missing display text should not block the page.
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
    fetchRuns();
  }

  function goPage(p: number) {
    pagination.current = p;
    fetchRuns();
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    fetchRuns();
  }

  onMounted(async () => {
    const repositoryId = route.query.repository_id as string | undefined;
    const templateId = route.query.template_id as string | undefined;
    query.repository_id = repositoryId ?? '';
    query.template_id = templateId ?? '';

    await Promise.all([loadRepositories(), loadTemplates()]);
    await Promise.all([
      ensureRepositoryOption(query.repository_id),
      ensureTemplateOption(query.template_id),
    ]);
    fetchRuns();
  });
</script>
