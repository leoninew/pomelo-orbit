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
      width-class="w-[min(760px,calc(100vw-32px))]"
      body-class="min-h-0 flex-1 space-y-4 overflow-y-auto px-6 py-4"
      content-class="max-h-[calc(100vh-32px)] flex flex-col"
    >
      <form class="space-y-4" novalidate @submit.prevent="handleSave">
        <div class="space-y-1.5">
          <label class="app-field-label block" for="project-name">
            {{ t('project.name') }}
            <span class="text-destructive">*</span>
          </label>
          <input
            id="project-name"
            v-model="form.name"
            type="text"
            class="app-input"
            :class="errors.name ? 'app-input-error' : ''"
            :aria-invalid="errors.name ? 'true' : undefined"
            @input="errors.name = ''"
          />
          <p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
        </div>
        <template v-if="!editingProject">
          <div class="space-y-1.5">
            <label class="app-field-label block" for="project-code">
              {{ t('project.code') }}
              <span class="text-destructive">*</span>
            </label>
            <input
              id="project-code"
              v-model="form.code"
              type="text"
              class="app-input"
              :class="errors.code ? 'app-input-error' : ''"
              :aria-invalid="errors.code ? 'true' : undefined"
              @input="errors.code = ''"
            />
            <p v-if="errors.code" class="app-field-error text-xs">{{ errors.code }}</p>
            <p v-else class="app-field-hint">{{ t('project.codeHint') }}</p>
          </div>
        </template>
        <button type="submit" class="sr-only" tabindex="-1" aria-hidden="true"></button>
      </form>
      <p v-if="submitError" class="app-field-error mt-3" role="alert">{{ submitError }}</p>
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
      <p v-if="deprecateSubmitError" class="app-field-error mt-3" role="alert">
        {{ deprecateSubmitError }}
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
  import { Plus } from '@lucide/vue';
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
  const appliedSearch = ref('');
  const isDialogOpen = ref(false);
  const isDeprecateDialogOpen = ref(false);
  const editingProject = ref<ProjectResp | null>(null);
  const deprecatingProject = ref<ProjectResp | null>(null);
  const pagination = reactive({ current: 1, pageSize: 10 });
  const form = reactive({
    name: '',
    code: '',
  });
  const errors = reactive({
    name: '',
    code: '',
  });
  const submitError = ref('');
  const deprecateSubmitError = ref('');

  const filteredProjects = computed(() => {
    const keyword = appliedSearch.value.trim().toLowerCase();
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
    Object.keys(errors).forEach((key) => {
      errors[key as keyof typeof errors] = '';
    });
    submitError.value = '';
  }

  function validate() {
    errors.name = form.name.trim() ? '' : t('project.nameRequired');
    if (editingProject.value) return !errors.name;
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
    deprecateSubmitError.value = '';
    (document.activeElement as HTMLElement)?.blur();
    await nextTick();
    isDeprecateDialogOpen.value = true;
  }

  function handleSearch() {
    appliedSearch.value = searchText.value;
    pagination.current = 1;
    void fetchProjects();
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
    submitError.value = '';
    if (!validate()) {
      return;
    }
    try {
      await executeOp(async () => {
        if (editingProject.value) {
          await projectStore.updateProject(editingProject.value.id, { name: form.name.trim() });
          toast.success(t('project.updated'));
          isDialogOpen.value = false;
        } else {
          const project = await projectStore.createProject({
            name: form.name.trim(),
            code: form.code.trim(),
          });
          toast.success(t('project.created'));
          isDialogOpen.value = false;
          projectStore.setActiveProject(project.id);
          await router.push({
            name: 'ProjectInitialization',
            params: { id: project.id },
            query: { redirect: '/gateway' },
          });
        }
      });
    } catch (error: unknown) {
      submitError.value = error instanceof Error ? error.message : t('project.saveFailed');
    }
  }

  async function handleDeprecate() {
    if (!deprecatingProject.value) {
      return;
    }
    deprecateSubmitError.value = '';
    const projectId = deprecatingProject.value.id;
    try {
      await executeOp(async () => {
        await projectStore.deprecateProject(projectId);
        toast.success(t('project.deprecatedToast'));
        isDeprecateDialogOpen.value = false;
      });
    } catch (error: unknown) {
      deprecateSubmitError.value = error instanceof Error ? error.message : t('project.saveFailed');
    }
  }

  onMounted(fetchProjects);
</script>
