<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-simple" :aria-label="t('environment.toolbar')">
      <SearchControl
        v-model="searchText"
        class="shrink-0"
        :placeholder="t('environment.searchPlaceholder')"
        :loading="status === 'loading'"
        @search="handleSearch"
      />
      <div class="flex items-center gap-3">
        <button class="app-button-primary px-5" @click="openCreateModal">
          <Plus class="size-4" />
          {{ t('environment.create') }}
        </button>
      </div>
    </ToolbarRoot>

    <div class="app-surface">
      <AppSpinner v-if="status === 'loading'" class="py-16" />
      <AppEmptyState v-else-if="environments.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-table-list min-w-[960px]">
          <thead>
            <tr>
              <th>{{ t('environment.fields.name') }}</th>
              <th>{{ t('environment.fields.code') }}</th>
              <th>{{ t('common.createdAt') }}</th>
              <th>{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="env in environments" :key="env.id">
              <td class="text-foreground">{{ env.name }}</td>
              <td class="text-foreground">{{ env.code }}</td>
              <td class="whitespace-nowrap text-foreground">
                {{ formatTime(env.created_at) }}
              </td>
              <td class="whitespace-nowrap">
                <div class="flex items-center gap-3">
                  <button class="app-link" @click="openEditModal(env)">
                    {{ t('common.edit') }}
                  </button>
                  <button
                    class="app-link-danger"
                    :disabled="operating || env.code === 'local'"
                    @click="openDeleteModal(env)"
                  >
                    {{ t('common.delete') }}
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
      v-model:open="isFormDialogOpen"
      :title="editingId ? t('environment.dialog.edit') : t('environment.dialog.create')"
      width-class="w-[min(560px,calc(100vw-32px))]"
    >
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('environment.fields.code') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.code"
            type="text"
            class="app-input"
            :class="errors.code ? 'app-input-error' : ''"
            :disabled="!!editingId"
            :placeholder="t('environment.placeholders.code')"
          />
          <p v-if="errors.code" class="app-field-error">{{ errors.code }}</p>
          <p v-else class="app-field-hint">{{ t('environment.hints.code') }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('environment.fields.name') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.name"
            type="text"
            class="app-input"
            :class="errors.name ? 'app-input-error' : ''"
            :placeholder="t('environment.placeholders.name')"
          />
          <p v-if="errors.name" class="app-field-error">{{ errors.name }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">{{ t('common.description') }}</label>
          <input
            v-model="form.description"
            type="text"
            class="app-input"
            :placeholder="t('environment.placeholders.description')"
          />
        </div>
      </div>
      <template #footer>
        <button class="app-button" @click="isFormDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-primary" :disabled="operating" @click="handleSave">
          {{ t('common.save') }}
        </button>
      </template>
    </AppDialog>

    <AppDialog
      v-model:open="isDeleteDialogOpen"
      :title="t('environment.dialog.delete')"
      width-class="w-[min(420px,calc(100vw-32px))]"
    >
      <p class="text-sm text-muted-foreground">
        {{ t('environment.dialog.deleteConfirm', { name: pendingDelete?.name || '' }) }}
      </p>
      <template #footer>
        <button class="app-button" @click="isDeleteDialogOpen = false">
          {{ t('common.cancel') }}
        </button>
        <button class="app-button-destructive" :disabled="operating" @click="handleDelete">
          {{ t('common.delete') }}
        </button>
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { Plus } from 'lucide-vue-next';
  import { computed, onMounted, reactive, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { ToolbarRoot } from 'reka-ui';
  import { environmentApi } from '@/api/environment/environment';
  import AppDialog from '@/components/AppDialog.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppSpinner from '@/components/AppSpinner.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { EnvironmentResp } from '@/gen/proto/orbit/v1/environment/environment';
  import { useProjectStore } from '@/stores/project';
  import { formatTime } from '@/utils/time';

  const toast = useToast();
  const { t } = useI18n();
  const projectStore = useProjectStore();
  const { status, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const environments = ref<EnvironmentResp[]>([]);
  const searchText = ref('');
  const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
  const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize) || 1);

  const isFormDialogOpen = ref(false);
  const isDeleteDialogOpen = ref(false);
  const editingId = ref('');
  const pendingDelete = ref<EnvironmentResp | null>(null);

  const form = reactive({
    code: '',
    name: '',
    description: '',
  });
  const errors = reactive({ code: '', name: '' });

  const codePattern = /^[a-z][a-z0-9-]*$/;

  watch(
    () => projectStore.activeProjectId,
    () => {
      pagination.current = 1;
      fetchData();
    }
  );

  function validateForm() {
    if (editingId.value) {
      errors.code = '';
    } else {
      errors.code = codePattern.test(form.code.trim())
        ? ''
        : t('environment.validation.codeInvalid');
    }
    errors.name = form.name.trim() ? '' : t('environment.validation.nameRequired');
    return !errors.code && !errors.name;
  }

  async function fetchData() {
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      environments.value = [];
      pagination.total = 0;
      toast.error(t('environment.toast.selectProjectRequired'));
      return;
    }
    try {
      await execute(async () => {
        const res = await environmentApi.list({
          project_id: projectId,
          page: pagination.current,
          per_page: pagination.pageSize,
          search: searchText.value || undefined,
        });
        environments.value = res.items ?? [];
        pagination.total = res.total ?? 0;
      });
    } catch {
      toast.error(t('environment.toast.loadFailed'));
    }
  }

  function handleSearch() {
    pagination.current = 1;
    fetchData();
  }

  function goPage(page: number) {
    pagination.current = page;
    fetchData();
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
    fetchData();
  }

  function resetForm() {
    Object.assign(form, {
      code: '',
      name: '',
      description: '',
    });
    Object.assign(errors, { code: '', name: '' });
  }

  function openCreateModal() {
    editingId.value = '';
    resetForm();
    isFormDialogOpen.value = true;
  }

  function openEditModal(env: EnvironmentResp) {
    editingId.value = env.id;
    Object.assign(form, {
      code: env.code,
      name: env.name,
      description: env.description || '',
    });
    Object.assign(errors, { code: '', name: '' });
    isFormDialogOpen.value = true;
  }

  async function handleSave() {
    if (!validateForm()) {
      return;
    }
    const projectId = projectStore.activeProjectId;
    if (!projectId) {
      toast.error(t('environment.toast.selectProjectRequired'));
      return;
    }
    try {
      await executeOp(async () => {
        if (editingId.value) {
          await environmentApi.update(editingId.value, {
            name: form.name.trim(),
            description: form.description.trim() || undefined,
          });
        } else {
          await environmentApi.create(
            {
              project_id: projectId,
              code: form.code.trim(),
              name: form.name.trim(),
              description: form.description.trim() || undefined,
            },
            { project_id: projectId }
          );
        }
        toast.success(t('environment.toast.saveSuccess'));
        isFormDialogOpen.value = false;
        await fetchData();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('environment.toast.saveFailed'));
    }
  }

  function openDeleteModal(env: EnvironmentResp) {
    pendingDelete.value = env;
    isDeleteDialogOpen.value = true;
  }

  async function handleDelete() {
    const target = pendingDelete.value;
    if (!target) {
      return;
    }
    try {
      await executeOp(async () => {
        await environmentApi.delete(target.id);
        toast.success(t('environment.toast.deleteSuccess'));
        isDeleteDialogOpen.value = false;
        pendingDelete.value = null;
        await fetchData();
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('environment.toast.deleteFailed'));
    }
  }

  onMounted(fetchData);
</script>
