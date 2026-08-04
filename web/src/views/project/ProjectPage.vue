<template>
  <div class="space-y-6">
    <ToolbarRoot class="app-toolbar-simple" :aria-label="t('project.toolbar')">
      <SearchControl
        v-model="searchText"
        class="shrink-0"
        :placeholder="t('project.searchPlaceholder')"
        :loading="status === 'loading'"
        @search="handleSearch"
      />
      <div class="flex items-center gap-3">
        <button class="app-button-primary px-5" @click="openCreateDialog">
          <Plus class="size-4" />
          {{ t('project.createProject') }}
        </button>
      </div>
    </ToolbarRoot>

    <div class="app-surface">
      <AppLoadingState v-if="status === 'loading'" />
      <div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
        <p class="text-sm">{{ error || t('project.loadFailed') }}</p>
      </div>
      <AppEmptyState v-else-if="filteredProjects.length === 0" />
      <div v-else class="overflow-x-auto">
        <table class="app-data-table min-w-[900px]">
          <colgroup>
            <col class="w-[20%]" />
            <col class="w-[15%]" />
            <col class="w-[10%]" />
            <col class="w-[20%]" />
            <col class="w-[20%]" />
            <col class="w-[15%]" />
          </colgroup>
          <thead>
            <tr>
              <th>{{ t('project.name') }}</th>
              <th>{{ t('project.code') }}</th>
              <th>{{ t('common.status') }}</th>
              <th>{{ t('common.createdAt') }}</th>
              <th>{{ t('common.updatedAt') }}</th>
              <th>{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="project in pagedProjects" :key="project.id">
              <td class="max-w-0 truncate text-foreground" :title="project.name">
                <router-link :to="`/project/${project.id}`" class="app-link">
                  {{ project.name }}
                </router-link>
              </td>
              <td class="whitespace-nowrap text-foreground">{{ project.code }}</td>
              <td>
                <AppBadge v-if="project.is_active" variant="status" tone="success">
                  {{ t('project.active') }}
                </AppBadge>
                <AppBadge v-else variant="status" tone="default">
                  {{ t('project.deprecated') }}
                </AppBadge>
              </td>
              <td class="whitespace-nowrap text-foreground">
                {{ formatTime(project.created_at) }}
              </td>
              <td class="whitespace-nowrap text-foreground">
                {{ formatTime(project.updated_at) }}
              </td>
              <td class="whitespace-nowrap">
                <div class="flex items-center gap-3">
                  <button class="app-link" @click="openEditDialog(project)">
                    {{ t('common.edit') }}
                  </button>
                  <button
                    v-if="project.is_active"
                    class="app-link-danger"
                    @click="openDeprecateDialog(project)"
                  >
                    {{ t('project.deprecate') }}
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
        :total="filteredProjects.length"
        :total-pages="totalPages"
        @change-page="goPage"
        @change-page-size="handlePageSizeChange"
      />
    </div>

    <AppDialog
      v-model:open="isDialogOpen"
      :title="editingProject ? t('project.editProject') : t('project.createProject')"
    >
      <div class="space-y-4">
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('project.name') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.name"
            type="text"
            class="app-input"
            :class="errors.name ? 'app-input-error' : ''"
            placeholder="Default Project"
            :aria-invalid="errors.name ? 'true' : undefined"
            @input="errors.name = ''"
          />
          <p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label block">
            {{ t('project.code') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            v-model="form.code"
            type="text"
            class="app-input"
            :class="errors.code ? 'app-input-error' : ''"
            placeholder="default"
            :aria-invalid="errors.code ? 'true' : undefined"
            @input="errors.code = ''"
          />
          <p v-if="errors.code" class="app-field-error text-xs">{{ errors.code }}</p>
          <p v-else class="app-field-hint">{{ t('project.codeHint') }}</p>
        </div>
      </div>

      <template #footer>
        <AppDialogActions :busy="operating" @cancel="isDialogOpen = false" @confirm="handleSave" />
      </template>
    </AppDialog>

    <AppDialog v-model:open="isDeprecateDialogOpen" :title="t('project.deprecateProject')">
      <p class="text-sm text-foreground">
        {{ t('project.deprecateConfirmPrefix') }}
        <span>{{ deprecatingProject?.name }}</span>
        {{ t('project.deprecateConfirmSuffix') }}
      </p>
      <template #footer>
        <AppDialogActions
          :busy="operating"
          variant="destructive"
          @cancel="isDeprecateDialogOpen = false"
          @confirm="handleDeprecate"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { Plus } from 'lucide-vue-next';
  import { computed, nextTick, onMounted, reactive, ref } from 'vue';
  import { useRouter } from 'vue-router';
  import { useI18n } from 'vue-i18n';
  import { ToolbarRoot } from 'reka-ui';
  import type { ProjectResp } from '@/gen/proto/orbit/v1/project/project';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ListPagination from '@/components/ListPagination.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import { useProjectStore } from '@/stores/project';
  import { formatTime } from '@/utils/time';

  const router = useRouter();
  const { t } = useI18n();
  const toast = useToast();
  const projectStore = useProjectStore();
  const { status, error, execute } = useStatusAsync();
  const { loading: operating, execute: executeOp } = useStatusAsync();

  const searchText = ref('');
  const isDialogOpen = ref(false);
  const isDeprecateDialogOpen = ref(false);
  const editingProject = ref<ProjectResp | null>(null);
  const deprecatingProject = ref<ProjectResp | null>(null);
  const pagination = reactive({ current: 1, pageSize: 10 });
  const form = reactive({ name: '', code: '' });
  const errors = reactive({ name: '', code: '' });

  const filteredProjects = computed(() => {
    const keyword = searchText.value.trim().toLowerCase();
    if (!keyword) {
      return projectStore.projects;
    }
    return projectStore.projects.filter(
      (project) =>
        project.name.toLowerCase().includes(keyword) || project.code.toLowerCase().includes(keyword)
    );
  });
  const totalPages = computed(() =>
    Math.max(1, Math.ceil(filteredProjects.value.length / pagination.pageSize))
  );
  const pagedProjects = computed(() => {
    const start = (pagination.current - 1) * pagination.pageSize;
    return filteredProjects.value.slice(start, start + pagination.pageSize);
  });

  function resetForm(project?: ProjectResp) {
    form.name = project?.name ?? '';
    form.code = project?.code ?? '';
    errors.name = '';
    errors.code = '';
  }

  function validate() {
    errors.name = form.name.trim() ? '' : t('project.nameRequired');
    errors.code = /^[a-z0-9_-]+$/.test(form.code) ? '' : t('project.codeInvalid');
    return !errors.name && !errors.code;
  }

  function openCreateDialog() {
    editingProject.value = null;
    resetForm();
    isDialogOpen.value = true;
  }

  function openEditDialog(project: ProjectResp) {
    editingProject.value = project;
    resetForm(project);
    isDialogOpen.value = true;
  }

  async function openDeprecateDialog(project: ProjectResp) {
    deprecatingProject.value = project;
    (document.activeElement as HTMLElement)?.blur();
    await nextTick();
    isDeprecateDialogOpen.value = true;
  }

  function handleSearch() {
    pagination.current = 1;
  }

  function goPage(page: number) {
    pagination.current = page;
  }

  function handlePageSizeChange(pageSize: number) {
    pagination.pageSize = pageSize;
    pagination.current = 1;
  }

  async function fetchProjects() {
    await execute(async () => {
      await projectStore.fetchProjects();
    });
  }

  async function handleSave() {
    if (!validate()) {
      return;
    }
    await executeOp(async () => {
      if (editingProject.value) {
        await projectStore.updateProject(editingProject.value.id, {
          name: form.name.trim(),
          code: form.code.trim(),
        });
        toast.success(t('project.updated'));
        isDialogOpen.value = false;
      } else {
        const project = await projectStore.createProject({
          name: form.name.trim(),
          code: form.code.trim(),
        });
        toast.success(t('project.created'));
        isDialogOpen.value = false;
        router.push({ name: 'ProjectDetail', params: { id: project.id } });
      }
    });
  }

  async function handleDeprecate() {
    if (!deprecatingProject.value) {
      return;
    }
    const projectId = deprecatingProject.value.id;
    await executeOp(async () => {
      await projectStore.deprecateProject(projectId);
      toast.success(t('project.deprecatedToast'));
      isDeprecateDialogOpen.value = false;
    });
  }

  onMounted(fetchProjects);
</script>
