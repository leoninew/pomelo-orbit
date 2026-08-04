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
      <AppLoadingState />
    </div>

    <div v-else-if="status === 'error'" class="app-surface">
      <div class="py-16 text-center text-destructive">
        <p class="text-sm">{{ error || t('pipelineTemplate.toast.loadFailed') }}</p>
      </div>
    </div>

    <div v-else-if="templates.length === 0" class="app-surface">
      <AppEmptyState />
    </div>

    <template v-else-if="viewMode === 'card'">
      <div class="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="tpl in templates"
          :key="tpl.id"
          class="app-surface p-5 transition-colors hover:border-primary"
        >
          <div class="mb-3 flex items-start justify-between gap-4">
            <h3 class="min-w-0 text-base font-semibold">
              <router-link :to="`/pipeline/template/${tpl.id}`" class="app-link block truncate">
                {{ tpl.name }}
              </router-link>
            </h3>
            <div class="flex shrink-0 items-center gap-3 text-sm">
              <button class="app-link" @click="openEditModal(tpl)">
                {{ t('common.edit') }}
              </button>
              <button :disabled="duplicating" class="app-link" @click="handleDuplicate(tpl.id)">
                {{ t('common.copy') }}
              </button>
            </div>
          </div>
          <p v-if="tpl.description" class="mb-4 min-h-10 text-sm text-muted-foreground">
            {{ tpl.description }}
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
        <table class="app-data-table min-w-[780px]">
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
                <router-link :to="`/pipeline/template/${tpl.id}`" class="app-link">
                  {{ tpl.name }}
                </router-link>
              </td>
              <td>
                <AppBadge>v{{ tpl.version }}</AppBadge>
              </td>
              <td class="max-w-sm truncate text-foreground" :title="tpl.description || undefined">
                {{ tpl.description }}
              </td>
              <td class="whitespace-nowrap text-foreground">{{ formatTime(tpl.created_at) }}</td>
              <td>
                <div class="flex items-center gap-3">
                  <button class="app-link" @click="openEditModal(tpl)">
                    {{ t('common.edit') }}
                  </button>
                  <button :disabled="duplicating" class="app-link" @click="handleDuplicate(tpl.id)">
                    {{ t('common.copy') }}
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
  </div>

  <AppDialog v-model:open="showCreateDialog" :title="t('pipelineTemplate.createTemplate')">
    <div class="space-y-4">
      <div class="space-y-1.5">
        <label class="app-field-label block">
          {{ t('pipelineTemplate.templateName') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          v-model="form.name"
          type="text"
          :placeholder="t('pipelineTemplate.templateNamePlaceholder')"
          class="app-input"
          :class="errors.name ? 'app-input-error' : ''"
          :aria-invalid="errors.name ? 'true' : undefined"
          @input="errors.name = ''"
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
      <AppDialogActions
        :busy="operating"
        @cancel="showCreateDialog = false"
        @confirm="handleCreateOk"
      />
    </template>
  </AppDialog>

  <AppDialog
    :open="showEditDialog"
    :title="t('pipelineTemplate.editBasicInfo')"
    @update:open="handleEditDialogOpenChange"
  >
    <form class="space-y-4" novalidate @submit.prevent="handleEditOk">
      <div class="space-y-1.5">
        <label for="edit-template-name" class="app-field-label block">
          {{ t('pipelineTemplate.templateName') }}
          <span class="text-destructive">*</span>
        </label>
        <input
          id="edit-template-name"
          v-model="editForm.name"
          type="text"
          :placeholder="t('pipelineTemplate.templateNamePlaceholder')"
          class="app-input"
          :class="editErrors.name ? 'app-input-error' : ''"
          :aria-invalid="editErrors.name ? 'true' : undefined"
          :aria-describedby="editErrors.name ? 'edit-template-name-error' : undefined"
          @input="clearEditError('name')"
        />
        <p
          v-if="editErrors.name"
          id="edit-template-name-error"
          class="app-field-error text-xs"
          role="alert"
        >
          {{ editErrors.name }}
        </p>
      </div>
      <div class="space-y-1.5">
        <label for="edit-template-description" class="app-field-label block">
          {{ t('common.description') }}
        </label>
        <textarea
          id="edit-template-description"
          v-model="editForm.description"
          rows="3"
          :placeholder="t('pipelineTemplate.templateDescriptionPlaceholder')"
          class="app-textarea"
        />
      </div>
    </form>
    <template #footer>
      <AppDialogActions :busy="operating" @cancel="closeEditModal" @confirm="handleEditOk" />
    </template>
  </AppDialog>
</template>

<script setup lang="ts">
  import { LayoutGrid, List, Plus } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { pipelineTemplateApi } from '@/api/pipeline/template';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import type { PipelineTemplateResp } from '@/gen/proto/orbit/v1/pipeline/template';
  import { formatTime } from '@/utils/time';
  import { ToggleGroupItem, ToggleGroupRoot, ToolbarRoot } from 'reka-ui';

  const router = useRouter();
  const toast = useToast();
  const { t } = useI18n();
  const projectStore = useProjectStore();
  const { status, error, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();
  const { loading: duplicating, execute: executeDuplicate } = useStatusAsync();

  const templates = ref<PipelineTemplateResp[]>([]);
  const viewMode = ref<'card' | 'table'>('table');
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
  const searchText = ref('');
  const showCreateDialog = ref(false);
  const showEditDialog = ref(false);
  const editingTemplate = ref<PipelineTemplateResp>();

  const form = reactive({ name: '', description: '' });
  const errors = reactive({ name: '' });
  const editForm = reactive({ name: '', description: '' });
  const editErrors = reactive({ name: '' });

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

  function resetEditForm() {
    if (!editingTemplate.value) {
      return;
    }
    Object.assign(editForm, {
      name: editingTemplate.value.name,
      description: editingTemplate.value.description ?? '',
    });
    editErrors.name = '';
  }

  function clearEditError(field: 'name') {
    editErrors[field] = '';
  }

  function openEditModal(template: PipelineTemplateResp) {
    editingTemplate.value = template;
    resetEditForm();
    showEditDialog.value = true;
  }

  function handleEditDialogOpenChange(open: boolean) {
    showEditDialog.value = open;
    if (!open) {
      editErrors.name = '';
    }
  }

  function closeEditModal() {
    handleEditDialogOpenChange(false);
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
        router.push(`/pipeline/template/${tpl.id}`);
      });
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('pipelineTemplate.toast.createFailed')
      );
    }
  }

  async function handleEditOk() {
    const template = editingTemplate.value;
    editErrors.name = editForm.name.trim() ? '' : t('pipelineTemplate.validation.nameNotEmpty');
    if (!template || editErrors.name) {
      return;
    }
    try {
      await executeOp(async () => {
        const updated = await pipelineTemplateApi.update(template.id, {
          name: editForm.name,
          description: editForm.description,
        });
        templates.value = templates.value.map((item) => (item.id === updated.id ? updated : item));
        editingTemplate.value = updated;
        toast.success(t('pipelineTemplate.toast.saveSuccess'));
        closeEditModal();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('pipelineTemplate.toast.saveFailed'));
    }
  }

  async function handleDuplicate(id: string) {
    try {
      await executeDuplicate(async () => {
        const newTemplate = await pipelineTemplateApi.duplicate(id, {});
        toast.success(t('pipelineTemplate.toast.duplicateSuccess'));
        router.push(`/pipeline/template/${newTemplate.id}`);
      });
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('pipelineTemplate.toast.duplicateFailed')
      );
    }
  }

  onMounted(fetchTemplates);
</script>
