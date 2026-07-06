<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-simple" :aria-label="t('pipelineTemplate.toolbar')">
      <SearchControl
        v-model="searchText"
        :placeholder="t('pipelineTemplate.searchPlaceholder')"
        :loading="status === 'loading'"
        @search="handleSearch"
      />
      <div class="flex items-center gap-3">
        <ToggleGroupRoot
          v-model="viewMode"
          type="single"
          class="flex h-10 overflow-hidden rounded-md border border-border bg-background"
          :aria-label="t('pipelineTemplate.viewMode')"
        >
          <ToggleGroupItem
            value="card"
            class="flex size-10 items-center justify-center text-muted-foreground outline-none transition-colors hover:bg-muted/50 hover:text-foreground data-[state=on]:bg-primary/10 data-[state=on]:text-primary"
            :aria-label="t('pipelineTemplate.cardView')"
          >
            <LayoutGrid class="size-4" />
          </ToggleGroupItem>
          <ToggleGroupItem
            value="table"
            class="flex size-10 items-center justify-center text-muted-foreground outline-none transition-colors hover:bg-muted/50 hover:text-foreground data-[state=on]:bg-primary/10 data-[state=on]:text-primary"
            :aria-label="t('pipelineTemplate.tableView')"
          >
            <List class="size-4" />
          </ToggleGroupItem>
        </ToggleGroupRoot>
        <button class="app-button-primary px-5" @click="openCreateModal">
          <Plus class="size-4" />
          {{ t('pipelineTemplate.createTemplate') }}
        </button>
      </div>
    </ToolbarRoot>

    <div v-if="status === 'loading'" class="app-surface">
      <AppSpinner class="py-16" />
    </div>

    <div v-else-if="templates.length === 0" class="app-surface">
      <AppEmptyState />
    </div>

    <template v-else-if="viewMode === 'card'">
      <div class="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="tpl in templates"
          :key="tpl.id"
          class="app-surface group cursor-pointer p-5 transition-colors hover:border-primary"
          @click="router.push(`/ci/template/${tpl.id}`)"
        >
          <div class="mb-3 flex items-start justify-between gap-4">
            <h3
              class="min-w-0 truncate text-sm font-medium text-foreground group-hover:text-primary"
            >
              {{ tpl.name }}
            </h3>
            <button
              :disabled="duplicating"
              class="app-link shrink-0 text-sm"
              @click.stop="handleDuplicate(tpl.id)"
            >
              {{ t('common.copy') }}
            </button>
          </div>
          <p class="mb-4 min-h-10 text-sm text-muted-foreground">
            {{ tpl.description || '—' }}
          </p>
          <div class="mb-4 flex flex-wrap gap-2 text-xs text-muted-foreground">
            <AppBadge>v{{ tpl.version }}</AppBadge>
            <AppBadge>
              {{ t('pipelineTemplate.stageCount', { count: tpl.orchestration.length }) }}
            </AppBadge>
            <AppBadge>
              {{ t('pipelineTemplate.variableCount', { count: tpl.variable_declarations.length }) }}
            </AppBadge>
          </div>
          <div class="text-sm text-muted-foreground">
            {{ formatTime(tpl.updated_at) }}
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

    <div v-else class="app-surface">
      <div class="overflow-x-auto">
        <table class="app-table-list min-w-[780px]">
          <thead>
            <tr>
              <th>{{ t('common.name') }}</th>
              <th>{{ t('pipelineTemplate.version') }}</th>
              <th>{{ t('common.description') }}</th>
              <th>{{ t('common.createdAt') }}</th>
              <th>{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="tpl in templates" :key="tpl.id">
              <td class="max-w-64 truncate" :title="tpl.name">
                <router-link :to="`/ci/template/${tpl.id}`" class="app-link">
                  {{ tpl.name }}
                </router-link>
              </td>
              <td>
                <AppBadge>v{{ tpl.version }}</AppBadge>
              </td>
              <td class="max-w-sm truncate text-foreground" :title="tpl.description || undefined">
                {{ tpl.description || '—' }}
              </td>
              <td class="whitespace-nowrap text-foreground">{{ formatTime(tpl.created_at) }}</td>
              <td>
                <router-link :to="`/ci/template/${tpl.id}`" class="app-link">
                  {{ t('application.view') }}
                </router-link>
                <button
                  :disabled="duplicating"
                  class="app-link ml-3"
                  @click="handleDuplicate(tpl.id)"
                >
                  {{ t('common.copy') }}
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
  </div>

  <AppDialog v-model:open="showCreateDialog" :title="t('pipelineTemplate.createTemplate')">
    <div class="space-y-4">
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('pipelineTemplate.templateName') }}</label>
        <input
          v-model="form.name"
          type="text"
          :placeholder="t('pipelineTemplate.templateNamePlaceholder')"
          class="app-input"
          :class="errors.name ? 'app-input-error' : ''"
        />
        <p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
      </div>
      <div class="space-y-1.5">
        <label class="app-field-label block">{{ t('common.description') }}</label>
        <textarea
          v-model="form.description"
          rows="3"
          :placeholder="t('pipelineTemplate.templateDescriptionPlaceholder')"
          class="app-textarea"
        />
      </div>
    </div>
    <template #footer>
      <button class="app-button" @click="showCreateDialog = false">{{ t('common.cancel') }}</button>
      <button class="app-button-primary" :disabled="operating" @click="handleCreateOk">
        {{ t('application.create') }}
      </button>
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { LayoutGrid, List, Plus } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { pipelineTemplateApi } from '@/api/ci';
  import AppDialog from '@/components/AppDialog.vue';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type { PipelineTemplateResp } from '@/gen/proto/orbit/api/v1/template';
  import { formatTime } from '@/utils/time';
  import { ToggleGroupItem, ToggleGroupRoot, ToolbarRoot } from 'reka-ui';

  const router = useRouter();
  const toast = useToast();
  const { t } = useI18n();
  const projectStore = useProjectStore();
  const { status, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();
  const { loading: duplicating, execute: executeDuplicate } = useStatusAsync();

  const templates = ref<PipelineTemplateResp[]>([]);
  const viewMode = ref<'card' | 'table'>('table');
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
  const searchText = ref('');
  const showCreateDialog = ref(false);

  const form = reactive({ name: '', description: '' });
  const errors = reactive({ name: '' });

  async function fetchTemplates() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('pipelineTemplate.toast.selectProjectRequired'));
      return;
    }
    try {
      await execute(async () => {
        const res = await pipelineTemplateApi.list({
          page: pagination.current,
          per_page: pagination.pageSize,
          search: searchText.value || undefined,
          project_id: projectId,
        });
        templates.value = res.items;
        pagination.total = res.total;
      });
    } catch {
      toast.error(t('pipelineTemplate.toast.loadFailed'));
    }
  }

  function handleSearch() {
    if (status.value === 'loading') {
      return;
    }
    pagination.current = 1;
    fetchTemplates();
  }

  function goPage(p: number) {
    pagination.current = p;
    fetchTemplates();
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    fetchTemplates();
  }

  function openCreateModal() {
    Object.assign(form, { name: '', description: '' });
    Object.assign(errors, { name: '' });
    showCreateDialog.value = true;
  }

  async function handleCreateOk() {
    errors.name = form.name.trim() ? '' : t('pipelineTemplate.validation.nameRequired');
    if (errors.name) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('pipelineTemplate.toast.selectProjectRequired'));
      return;
    }
    try {
      await executeOp(async () => {
        const tpl = await pipelineTemplateApi.create(
          {
            name: form.name,
            description: form.description,
            variable_declarations: [],
          },
          { project_id: projectId }
        );
        toast.success(t('pipelineTemplate.toast.createSuccess'));
        showCreateDialog.value = false;
        router.push(`/ci/template/${tpl.id}`);
      });
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('pipelineTemplate.toast.createFailed')
      );
    }
  }

  async function handleDuplicate(id: string) {
    try {
      await executeDuplicate(async () => {
        const newTemplate = await pipelineTemplateApi.duplicate(id);
        toast.success(t('pipelineTemplate.toast.duplicateSuccess'));
        router.push(`/ci/template/${newTemplate.id}`);
      });
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('pipelineTemplate.toast.duplicateFailed')
      );
    }
  }

  onMounted(fetchTemplates);
</script>
